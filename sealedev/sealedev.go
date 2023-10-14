package sealedev

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"golang.org/x/exp/slices"
)

const (
	EVAverageRepetition = 5000

	ckBuylistLink = "https://www.cardkingdom.com/purchasing/mtg_singles"
)

type SealedEVScraper struct {
	LogCallback      mtgban.LogCallbackFunc
	FastMode         bool
	Affiliate        string
	BuylistAffiliate string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
	buylist     mtgban.BuylistRecord

	banpriceKey string
}

type evConfig struct {
	Name           string
	StatsFunc      func(values []float64) (float64, error)
	SourceName     string
	FoundInBuylist bool
	TargetsBuylist bool
}

var evParameters = []evConfig{
	{
		Name: "TCG Low EV Mean",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Mean(values)
		},
		SourceName: "TCG Low",
	},
	{
		Name: "TCG Low EV Median",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName: "TCG Low",
	},
	{
		Name: "TCG Direct (net) EV Mean",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Mean(values)
		},
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
	},
	{
		Name: "TCG Direct (net) EV Median",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
	},
	{
		Name: "CK Buylist for Singles",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Mean(values)
		},
		SourceName:     "CK",
		FoundInBuylist: true,
		TargetsBuylist: true,
	},
}

type evOutputStash struct {
	Total   float64
	Dataset []float64
}

func NewScraper(sig string) *SealedEVScraper {
	ss := SealedEVScraper{}
	ss.inventory = mtgban.InventoryRecord{}
	ss.marketplace = map[string]mtgban.InventoryRecord{}
	ss.buylist = mtgban.BuylistRecord{}
	ss.banpriceKey = sig
	return &ss
}

func (ss *SealedEVScraper) printf(format string, a ...interface{}) {
	if ss.LogCallback != nil {
		ss.LogCallback("[SS] "+format, a...)
	}
}

type resultChan struct {
	i   int
	ev  float64
	err error
}

type respChan struct {
	productId string
	invEntry  *mtgban.InventoryEntry
	buyEntry  *mtgban.BuylistEntry
	err       error
}

func (ss *SealedEVScraper) scrape() error {
	ss.printf("Loading BAN prices")
	prices, err := loadPrices(ss.banpriceKey)
	if err != nil {
		return err
	}
	ss.printf("Retrieved %d+%d prices", len(prices.Retail), len(prices.Buylist))

	start := time.Now()

	sets := mtgmatcher.GetSets()
	for _, set := range sets {
		var wgOut sync.WaitGroup
		channelOut := make(chan respChan)

		// Skip products without Sealed or Booster information
		switch set.Code {
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		}
		if set.SealedProduct == nil {
			continue
		}

		if !ss.FastMode {
			ss.printf("Running sealed EV on %s", set.Name)
		}

		for _, product := range set.SealedProduct {
			wgOut.Add(1)
			go func() {
				defer wgOut.Done()

				repeats := EVAverageRepetition
				if ss.FastMode {
					repeats = 10
				}
				if !mtgmatcher.SealedIsRandom(set.Code, product.UUID) {
					repeats = 1
				}

				var wg sync.WaitGroup

				datasets := make([][]float64, len(evParameters))
				channel := make(chan resultChan)

				for j := 0; j < repeats; j++ {
					wg.Add(1)

					go func() {
						defer wg.Done()

						picks, err := mtgmatcher.GetPicksForSealed(set.Code, product.UUID)
						if err != nil {
							if product.Category != "land_station" {
								channel <- resultChan{
									err: fmt.Errorf("[%s] '%s' error: %s", set.Code, product.Name, err.Error()),
								}
							}
							return
						}

						for i := range evParameters {
							priceSource := prices.Retail
							if evParameters[i].FoundInBuylist {
								priceSource = prices.Buylist
							}
							ev := valueInBooster(picks, priceSource, evParameters[i].SourceName)
							channel <- resultChan{
								i:  i,
								ev: ev,
							}
						}
					}()
				}

				go func() {
					wg.Wait()
					close(channel)
				}()

				for resp := range channel {
					if resp.err != nil {
						channelOut <- respChan{
							err: resp.err,
						}
						continue
					}

					datasets[resp.i] = append(datasets[resp.i], resp.ev)
				}

				for i, dataset := range datasets {
					price, err := evParameters[i].StatsFunc(dataset)
					if err != nil {
						continue
					}

					if price == 0 {
						continue
					}

					if evParameters[i].TargetsBuylist {
						link := ckBuylistLink
						if ss.BuylistAffiliate != "" {
							link += fmt.Sprintf("?partner=%s&utm_campaign=%s&utm_medium=affiliate&utm_source=%s", ss.BuylistAffiliate, ss.BuylistAffiliate, ss.BuylistAffiliate)
						}
						channelOut <- respChan{
							productId: product.UUID,
							buyEntry: &mtgban.BuylistEntry{
								Conditions: "INDEX",
								BuyPrice:   price,
								TradePrice: price * 1.3,
								URL:        link,
							},
						}
					} else {
						var link string
						tcgID, _ := strconv.Atoi(product.Identifiers["tcgplayerProductId"])
						if tcgID != 0 {
							link = tcgplayer.TCGPlayerProductURL(tcgID, "", ss.Affiliate, "")
						}

						channelOut <- respChan{
							productId: product.UUID,
							invEntry: &mtgban.InventoryEntry{
								Conditions: "INDEX",
								Price:      price,
								SellerName: evParameters[i].Name,
								URL:        link,
							},
						}
					}
				}
			}()
		}

		go func() {
			wgOut.Wait()
			close(channelOut)
		}()

		var printedErrors []string
		for result := range channelOut {
			if result.err != nil && !slices.Contains(printedErrors, result.err.Error()) {
				ss.printf("%s", result.err.Error())
				printedErrors = append(printedErrors, result.err.Error())
				continue
			}

			if result.invEntry != nil {
				ss.inventory.Add(result.productId, result.invEntry)
			}
			if result.buyEntry != nil {
				ss.buylist.Add(result.productId, result.buyEntry)
			}
		}

	}

	ss.printf("Took %v", time.Since(start))

	ss.inventoryDate = time.Now()
	ss.buylistDate = time.Now()

	return nil
}

func (ss *SealedEVScraper) Inventory() (mtgban.InventoryRecord, error) {
	if len(ss.inventory) > 0 {
		return ss.inventory, nil
	}

	err := ss.scrape()
	if err != nil {
		return nil, err
	}

	return ss.inventory, nil
}

func (ss *SealedEVScraper) Buylist() (mtgban.BuylistRecord, error) {
	if len(ss.buylist) > 0 {
		return ss.buylist, nil
	}

	err := ss.scrape()
	if err != nil {
		return nil, err
	}

	return ss.buylist, nil
}

func (ss *SealedEVScraper) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(ss.inventory) == 0 {
		_, err := ss.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := ss.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range ss.inventory {
		for i := range ss.inventory[card] {
			if ss.inventory[card][i].SellerName == sellerName {
				if ss.inventory[card][i].Price == 0 {
					continue
				}
				if ss.marketplace[sellerName] == nil {
					ss.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				ss.marketplace[sellerName][card] = append(ss.marketplace[sellerName][card], ss.inventory[card][i])
			}
		}
	}

	if len(ss.marketplace[sellerName]) == 0 {
		return nil, errors.New("seller not found")
	}
	return ss.marketplace[sellerName], nil
}

func (ss *SealedEVScraper) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return errors.New("nothing was loaded")
	}

	ss.marketplace = market
	ss.inventory = inventory

	ss.printf("Loaded inventory from file")

	return nil
}

func (tcg *SealedEVScraper) MarketNames() []string {
	var names []string
	for _, param := range evParameters {
		if param.TargetsBuylist {
			continue
		}
		names = append(names, param.Name)
	}
	return names
}

func (ss *SealedEVScraper) Info() (info mtgban.ScraperInfo) {
	info.Name = "Sealed EV Scraper"
	info.Shorthand = "SS"
	info.InventoryTimestamp = &ss.inventoryDate
	info.BuylistTimestamp = &ss.buylistDate
	info.SealedMode = true
	info.MetadataOnly = true
	return
}
