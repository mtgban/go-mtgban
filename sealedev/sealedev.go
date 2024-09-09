package sealedev

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	TargetEdition    string
	TargetProduct    string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	banpriceKey string
	prices      *BANPriceResponse
}

type evConfig struct {
	Name           string
	StatsFunc      func(values []float64) (float64, error)
	SourceName     string
	Shorthand      string
	FoundInBuylist bool
	TargetsBuylist bool
	Simulation     bool
}

var evParameters = []evConfig{
	// CK Buylist
	{
		Name:           "CK Buylist EV for Singles",
		Shorthand:      "SS",
		SourceName:     "CK",
		FoundInBuylist: true,
		TargetsBuylist: true,
	},

	// TCG Low
	{
		Name:       "TCG Low EV",
		Shorthand:  "TCGLowEV",
		SourceName: "TCGLow",
	},
	{
		Name:      "TCG Low Sim Median",
		Shorthand: "TCGLowSimMed",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName: "TCGLow",
		Simulation: true,
	},
	{
		Name:      "TCG Low Sim StdDev",
		Shorthand: "TCGLowSimStd",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.StandardDeviation(values)
		},
		SourceName: "TCGLow",
		Simulation: true,
	},

	// TCG Direct (net)
	{
		Name:           "TCG Direct (net) EV",
		Shorthand:      "TCGDirectNetEV",
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
	},
	{
		Name:      "TCG Direct (net) Sim Median",
		Shorthand: "TCGDirectNetSimMed",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName:     "TCGDirectNet",
		FoundInBuylist: true,
		Simulation:     true,
	},
	{
		Name:      "TCG Direct (net) Sim StdDev",
		Shorthand: "TCGDirectNetSimStd",
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
		Shorthand:  "CTZeroEV",
		SourceName: "CT0",
	},
	{
		Name:      "CT Zero Sim Median",
		Shorthand: "CTZeroSimMed",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceName: "CT0",
		Simulation: true,
	},
	{
		Name:      "CT Zero Sim StdDev",
		Shorthand: "CTZeroSimStd",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.StandardDeviation(values)
		},
		SourceName: "CT0",
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

type result struct {
	productId string
	invEntry  *mtgban.InventoryEntry
	buyEntry  *mtgban.BuylistEntry
	err       error
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

func (ss *SealedEVScraper) runEV(uuid string) ([]result, []string) {
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		return nil, []string{err.Error()}
	}

	productUUID := co.UUID
	setCode := co.SetCode

	repeats := EVAverageRepetition
	if ss.FastMode {
		repeats = 10
	}
	if !mtgmatcher.SealedIsRandom(setCode, productUUID) {
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
				picks, err := ss.repeatedPicks(setCode, productUUID)
				if err != nil {
					channel <- resultChan{
						err: err,
					}
					continue
				}

				for i := range evParameters {
					if !evParameters[i].Simulation {
						continue
					}

					priceSource := ss.prices.Retail
					if evParameters[i].FoundInBuylist {
						priceSource = ss.prices.Buylist
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
			probabilities, err := mtgmatcher.GetProbabilitiesForSealed(setCode, productUUID)
			if err != nil {
				channel <- resultChan{
					err: err,
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

				priceSource := ss.prices.Retail
				if evParameters[i].FoundInBuylist {
					priceSource = ss.prices.Buylist
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

	var allTheErrors []string
	for resp := range channel {
		// Collect all the errors from this product
		if resp.err != nil && !slices.Contains(allTheErrors, resp.err.Error()) {
			allTheErrors = append(allTheErrors, resp.err.Error())
			continue
		}

		datasets[resp.i] = append(datasets[resp.i], resp.ev)
	}

	var out []result
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

		res := result{
			productId: productUUID,
		}

		if evParameters[i].TargetsBuylist {
			link := ckBuylistLink
			if ss.BuylistAffiliate != "" {
				link += fmt.Sprintf("?partner=%s&utm_campaign=%s&utm_medium=affiliate&utm_source=%s", ss.BuylistAffiliate, ss.BuylistAffiliate, ss.BuylistAffiliate)
			}

			res.buyEntry = &mtgban.BuylistEntry{
				BuyPrice: price,
				URL:      link,
			}
		} else {
			var link string
			tcgID, _ := strconv.Atoi(co.Identifiers["tcgplayerProductId"])
			if tcgID != 0 {
				isDirect := evParameters[i].SourceName == "TCGDirectNet"
				link = tcgplayer.GenerateProductURL(tcgID, "", ss.Affiliate, "", "", isDirect)
			}

			res.invEntry = &mtgban.InventoryEntry{
				Price:      price,
				SellerName: evParameters[i].Name,
				URL:        link,
			}
		}

		out = append(out, res)
	}

	return out, allTheErrors
}

func (ss *SealedEVScraper) scrape() error {
	var selected string

	ss.printf("Loading products")
	sets := mtgmatcher.GetAllSets()
	var uuids []string
	for _, code := range sets {
		set, _ := mtgmatcher.GetSet(code)

		switch set.Code {
		// Skip products without Sealed or Booster information
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		default:
			// Skip filtered editions if set
			if ss.TargetEdition != "" && set.Code != ss.TargetEdition && set.Name != ss.TargetEdition {
				continue
			}
		}

		for _, product := range set.SealedProduct {
			// Skip unsupported types
			if product.Category == "land_station" {
				continue
			}

			// Skip filtered products if set
			if ss.TargetProduct != "" && product.Name != ss.TargetProduct && product.UUID != ss.TargetProduct {
				continue
			}

			uuids = append(uuids, product.UUID)
		}

		// Keep track of what was selected to reduce price calls
		if ss.TargetEdition != "" {
			selected = "/" + set.Code
		}
	}
	ss.printf("Found %d products over %d sets", len(uuids), len(sets))

	ss.printf("Loading BAN prices")
	prices, err := loadPrices(ss.banpriceKey, selected)
	if err != nil {
		return err
	}
	ss.printf("Retrieved %d+%d prices", len(prices.Retail), len(prices.Buylist))
	ss.prices = prices

	start := time.Now()

	for i, uuid := range uuids {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}

		if !ss.FastMode {
			ss.printf("Running EV on [%s] %s (%d/%d)", co.SetCode, co.Name, i+1, len(uuids))
		}

		results, messages := ss.runEV(uuid)

		// Print errors if necessary
		if len(messages) > 0 {
			ss.printf("runEV error: %s", strings.Join(messages, " | "))
		}

		for _, result := range results {
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

func (ss *SealedEVScraper) InfoForScraper(name string) mtgban.ScraperInfo {
	info := ss.Info()
	info.Name = name
	for _, param := range evParameters {
		if param.Name == name {
			info.Shorthand = param.Shorthand

			// Only the retail side is metadata only
			info.MetadataOnly = !param.TargetsBuylist
			break
		}
	}
	return info
}

func (ss *SealedEVScraper) Info() (info mtgban.ScraperInfo) {
	info.Name = "Sealed EV Scraper"
	info.Shorthand = "SS"
	info.InventoryTimestamp = &ss.inventoryDate
	info.BuylistTimestamp = &ss.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.3
	info.Family = "EV"
	return
}
