package sealedev

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

const (
	EVAverageRepetition = 5000

	ckBuylistLink = "https: //www.cardkingdom.com/purchasing/mtg_singles"
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

func needsRandom(setCode, sealedUUID string) bool {
	set, err := mtgmatcher.GetSet(setCode)
	if err != nil {
		return false
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		if product.Contents == nil {
			return true
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
				case "pack":
					return true
				case "sealed":
					if needsRandom(content.Set, content.UUID) {
						return true
					}
				case "deck":
				case "variable":
					return true
				case "other":
				default:
					return true
				}
			}
		}
	}

	return false
}

func getPicksForDeck(setCode, deckName string) ([]string, error) {
	var picks []string

	set, err := mtgmatcher.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}

		for _, card := range deck.Cards {
			uuid, err := mtgmatcher.MatchId(card.UUID, card.Finish == "foil", card.Finish == "etched")
			if err != nil {
				return nil, err
			}

			for i := 0; i < card.Count; i++ {
				picks = append(picks, uuid)
			}
		}
	}

	return picks, nil
}

func getPicksForSealed(setCode, sealedUUID string) ([]string, error) {
	var picks []string

	set, err := mtgmatcher.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
					uuid, err := mtgmatcher.MatchId(content.UUID, content.Foil)
					if err != nil {
						return nil, err
					}
					picks = append(picks, uuid)
				case "pack":
					boosterPicks, err := mtgmatcher.BoosterGen(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					picks = append(picks, boosterPicks...)
				case "sealed":
					for i := 0; i < content.Count; i++ {
						sealedPicks, err := getPicksForSealed(content.Set, content.UUID)
						if err != nil {
							return nil, err
						}
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := getPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					picks = append(picks, deckPicks...)
				case "variable":
					variableIndex := rand.Intn(len(content.Configs))

					for _, deck := range content.Configs[variableIndex].Deck {
						deckPicks, err := getPicksForDeck(deck.Set, deck.Name)
						if err != nil {
							return nil, err
						}
						picks = append(picks, deckPicks...)
					}
				case "other":
				default:
					return nil, errors.New("unknown key")
				}
			}
		}

		if product.Contents == nil {
			picks = append(picks, classicSealedCalc(set, product.UUID)...)
		}
	}

	if len(picks) == 0 {
		return nil, errors.New("nothing was picked")
	}

	return picks, nil
}

// This function can be dropped once all Sealed has the Contents array in place
func classicSealedCalc(set *mtgjson.Set, sealedUUID string) []string {
	var picks []string

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		// Only keep booster-like items
		switch product.Category {
		case "booster_box",
			"booster_pack",
			"draft_set":
		default:
			continue
		}

		subType := product.Subtype
		switch subType {
		case "default",
			"collector",
			"jumpstart",
			"set":
		case "topper":
			subType = "box-topper"
		case "theme":
			if product.Category != "booster_pack" {
				continue
			}
			for boosterType, booster := range set.Booster {
				if mtgmatcher.SealedEquals(booster.Name, product.Name) {
					subType = boosterType
					break
				}
			}
		default:
			continue
		}

		// Derive how many boosters the product contains
		productSize := 1
		if product.ProductSize != 0 {
			productSize = product.ProductSize
		}

		for e := 0; e < productSize; e++ {
			boosterPicks, err := mtgmatcher.BoosterGen(set.Code, subType)
			if err != nil {
				continue
			}
			picks = append(picks, boosterPicks...)
		}
	}

	return picks
}

func (ss *SealedEVScraper) scrape() error {
	ss.printf("Loading BAN prices")
	prices, err := loadPrices(ss.banpriceKey)
	if err != nil {
		return err
	}
	ss.printf("Retrieved %d+%d prices", len(prices.Retail), len(prices.Buylist))

	sets := mtgmatcher.GetSets()
	for _, set := range sets {
		// Skip products without Sealed or Booster information
		switch set.Code {
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		}
		if set.SealedProduct == nil || set.Booster == nil {
			continue
		}

		if !ss.FastMode {
			ss.printf("Running sealed EV on %s", set.Name)
		}

		for _, product := range set.SealedProduct {
			var errPrinted bool

			repeats := EVAverageRepetition
			if ss.FastMode {
				repeats = 10
			}
			if !needsRandom(set.Code, product.UUID) {
				repeats = 1
			}

			datasets := make([][]float64, len(evParameters))

			for j := 0; j < repeats; j++ {
				picks, err := getPicksForSealed(set.Code, product.UUID)
				if err != nil {
					if product.Contents != nil && !errPrinted && product.Category != "land_station" {
						ss.printf("[%s] '%s' error: %s", set.Code, product.Name, err.Error())
						errPrinted = true
					}
					continue
				}

				for i := range evParameters {
					priceSource := prices.Retail
					if evParameters[i].FoundInBuylist {
						priceSource = prices.Buylist
					}
					ev := valueInBooster(picks, priceSource, evParameters[i].SourceName)
					datasets[i] = append(datasets[i], ev)
				}
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
					ss.buylist.Add(product.UUID, &mtgban.BuylistEntry{
						Conditions: "INDEX",
						BuyPrice:   price,
						URL:        link,
					})
				} else {
					var link string
					tcgID, _ := strconv.Atoi(product.Identifiers["tcgplayerProductId"])
					if tcgID != 0 {
						link = tcgplayer.TCGPlayerProductURL(tcgID, "", ss.Affiliate, "")
					}

					ss.inventory.Add(product.UUID, &mtgban.InventoryEntry{
						Conditions: "INDEX",
						Price:      price,
						SellerName: evParameters[i].Name,
						URL:        link,
					})
				}
			}
		}
	}

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
