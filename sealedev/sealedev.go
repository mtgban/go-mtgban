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
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"golang.org/x/exp/slices"
)

const (
	EVAverageRepetition = 5000

	EVMaxRepickCount = 10

	DefaultRepeatConcurrency = 8
	DefaultSetConcurrency    = 32

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
	Simulation     bool
}

var evParameters = []evConfig{
	{
		Name:       "TCG Low EV",
		SourceName: "TCG Low",
	},
	{
		Name:           "TCG Direct (net) EV",
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
	},
	{
		Name:           "CK Buylist EV for Singles",
		SourceName:     "CK",
		FoundInBuylist: true,
		TargetsBuylist: true,
	},
	{
		Name: "TCG Low Sim Median",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName: "TCG Low",
		Simulation: true,
	},
	{
		Name: "TCG Low Sim StdDev",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.StandardDeviation(values)
		},
		SourceName: "TCG Low",
		Simulation: true,
	},
	{
		Name: "TCG Direct (net) Sim Median",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
		Simulation:     true,
	},
	{
		Name: "TCG Direct (net) Sim StdDev",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.StandardDeviation(values)
		},
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
		Simulation:     true,
	},

	// Card Trader Zero
	{
		Name:       "CT Zero EV",
		SourceName: "Card Trader Zero",
	},
	{
		Name: "CT Zero Sim Median",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName: "Card Trader Zero",
		Simulation: true,
	},
	{
		Name: "CT Zero Sim StdDev",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.StandardDeviation(values)
		},
		SourceName: "Card Trader Zero",
		Simulation: true,
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

type productChan struct {
	setCode string
	index   int
}

func (ss *SealedEVScraper) repeatedPicks(setCode, productUUID string) ([]string, error) {
	pickCount := 0
	for {
		pickCount++
		// Prevent deadlocking
		if pickCount > EVMaxRepickCount {
			return nil, errors.New("repicked too many times")
		}

		picks, err := mtgmatcher.GetPicksForSealed(setCode, productUUID)
		if err != nil {
			return nil, err
		}

		// Repeat booster generation if there is one card type known to skew values
		rePick := false
		for _, pick := range picks {
			co, err := mtgmatcher.GetUUID(pick)
			if err != nil {
				return nil, err
			}
			if co.HasPromoType(mtgjson.PromoTypeSerialized) {
				rePick = true
				break
			}
		}
		if rePick {
			ss.printf("%s - %s: repicking product (%d/%d)", setCode, productUUID, pickCount, EVMaxRepickCount)
			continue
		}

		return picks, nil
	}
}

func (ss *SealedEVScraper) runEV(prod productChan, channelOut chan respChan, prices *BANPriceResponse) {
	sets := mtgmatcher.GetSets()

	setCode := prod.setCode
	i := prod.index
	product := sets[setCode].SealedProduct[i]

	// Skip unsupported types
	if product.Category == "land_station" {
		return
	}

	repeats := EVAverageRepetition
	if ss.FastMode {
		repeats = 10
	}
	if !mtgmatcher.SealedIsRandom(setCode, product.UUID) {
		repeats = 1
	}

	var wg sync.WaitGroup

	datasets := make([][]float64, len(evParameters))
	channel := make(chan resultChan)
	repeatsChannel := make(chan int)

	for j := 0; j < DefaultRepeatConcurrency; j++ {
		wg.Add(2)

		// Simulations
		go func() {
			for _ = range repeatsChannel {
				picks, err := ss.repeatedPicks(setCode, product.UUID)
				if err != nil {
					channel <- resultChan{
						err: fmt.Errorf("[%s] '%s' picks error: %s", setCode, product.Name, err.Error()),
					}
					continue
				}

				for i := range evParameters {
					if !evParameters[i].Simulation {
						continue
					}

					priceSource := prices.Retail
					if evParameters[i].FoundInBuylist {
						priceSource = prices.Buylist
					}

					ev := valueInBooster(picks, priceSource, evParameters[i].SourceName, nil)

					channel <- resultChan{
						i:  i,
						ev: ev,
					}
				}
			}
			wg.Done()
		}()

		// Probability EV
		go func() {
			probabilities, err := mtgmatcher.GetProbabilitiesForSealed(setCode, product.UUID)
			if err != nil {
				channel <- resultChan{
					err: fmt.Errorf("[%s] '%s' probabilities error: %s", setCode, product.Name, err.Error()),
				}
				wg.Done()
				return
			}

			// Split probabilities in two simpler arrays for later reuse
			var probPicks []string
			var probProbs []float64
			for _, probability := range probabilities {
				co, err := mtgmatcher.GetUUID(probability.UUID)
				if err != nil {
					continue
				}

				prob := probability.Probability
				if co.HasPromoType(mtgjson.PromoTypeSerialized) {
					prob = 0
				}

				probProbs = append(probProbs, prob)
				probPicks = append(probPicks, probability.UUID)
			}

			for i := range evParameters {
				if evParameters[i].Simulation {
					continue
				}

				priceSource := prices.Retail
				if evParameters[i].FoundInBuylist {
					priceSource = prices.Buylist
				}

				ev := valueInBooster(probPicks, priceSource, evParameters[i].SourceName, probProbs)

				channel <- resultChan{
					i:  i,
					ev: ev,
				}
			}
			wg.Done()
		}()
	}

	go func(repeatsChannel chan int, channel chan resultChan) {
		for j := 0; j < repeats; j++ {
			repeatsChannel <- j
		}
		close(repeatsChannel)

		wg.Wait()
		close(channel)
	}(repeatsChannel, channel)

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
		var price float64
		if evParameters[i].Simulation {
			var err error
			price, err = evParameters[i].StatsFunc(dataset)
			if err != nil {
				continue
			}
		} else if len(dataset) > 0 {
			price = dataset[0]
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
				link = tcgplayer.TCGPlayerProductURL(tcgID, "", ss.Affiliate, "", "")
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
		// Skip products without Sealed or Booster information
		switch set.Code {
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		}
		if set.SealedProduct == nil {
			continue
		}

		var wgOut sync.WaitGroup
		channelOut := make(chan respChan)
		productChannel := make(chan productChan)

		for e := 0; e < DefaultSetConcurrency; e++ {
			wgOut.Add(1)

			go func() {
				for prod := range productChannel {
					if !ss.FastMode {
						ss.printf("Running sealed EV on %s", sets[prod.setCode].Name)
					}

					ss.runEV(prod, channelOut, prices)
				}
				wgOut.Done()
			}()
		}

		go func(setCode string, productChannel chan productChan, channelOut chan respChan) {
			set := sets[setCode]

			for i := range set.SealedProduct {
				productChannel <- productChan{
					setCode: setCode,
					index:   i,
				}
			}
			close(productChannel)

			wgOut.Wait()
			close(channelOut)
		}(set.Code, productChannel, channelOut)

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
