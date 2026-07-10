package sealedev

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

const (
	EVAverageRepetition = 5000
	EVFastRepetition    = 10

	defaultConcurrency = 8
)

type SealedEVScraper struct {
	LogCallback      mtgban.LogCallbackFunc
	FastMode         bool
	Affiliate        string
	BuylistAffiliate string
	TargetEdition    string
	TargetProduct    string
	MaxConcurrency   int

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
	SourceStores   []string
	Shorthand      string
	FoundInBuylist bool
	TargetsBuylist bool
	Simulation     bool
}

func passthroughFirst(values []float64) (float64, error) {
	return values[0], nil
}

var evParameters = []evConfig{
	// CK Buylist
	{
		Name:           "Singles Buylist (est.)",
		Shorthand:      "SS",
		StatsFunc:      passthroughFirst,
		SourceStores:   []string{"CK", "SCG"},
		FoundInBuylist: true,
		TargetsBuylist: true,
	},

	// TCG Low
	{
		Name:         "TCG Low EV",
		Shorthand:    "TCGLowEV",
		StatsFunc:    passthroughFirst,
		SourceStores: []string{"TCGLow"},
	},
	{
		Name:      "TCG Low Sim",
		Shorthand: "TCGLowSim",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceStores: []string{"TCGLow"},
		Simulation:   true,
	},

	// TCG Direct (net)
	{
		Name:           "TCG Direct (net) EV",
		Shorthand:      "TCGDirectNetEV",
		StatsFunc:      passthroughFirst,
		SourceStores:   []string{"TCGDirectNet"},
		FoundInBuylist: true,
	},
	{
		Name:      "TCG Direct (net) Sim",
		Shorthand: "TCGDirectNetSim",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceStores:   []string{"TCGDirectNet"},
		FoundInBuylist: true,
		Simulation:     true,
	},

	// Card Trader Zero
	{
		Name:         "CT Zero EV",
		Shorthand:    "CTZeroEV",
		StatsFunc:    passthroughFirst,
		SourceStores: []string{"CT0"},
	},
	{
		Name:      "CT Zero Sim",
		Shorthand: "CTZeroSim",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceStores: []string{"CT0"},
		Simulation:   true,
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
	ss.MaxConcurrency = defaultConcurrency
	return &ss
}

func (ss *SealedEVScraper) printf(format string, a ...interface{}) {
	if ss.LogCallback != nil {
		ss.LogCallback("[SS] "+format, a...)
	}
}

type result struct {
	productId string
	invEntry  *mtgban.InventoryEntry
	buyEntry  *mtgban.BuylistEntry
	err       error
}

// valueFromCache sums the pre-resolved unit prices for a list of picks. When
// probabilities is nil every pick is counted once (used for a single simulated
// draw); otherwise each pick is weighted by its probability.
func valueFromCache(picks []string, unit map[string]float64, probabilities []float64) float64 {
	var total float64
	for i, pick := range picks {
		probability := 1.0
		if probabilities != nil {
			probability = probabilities[i]
		}
		total += unit[pick] * probability
	}
	return total
}

func (ss *SealedEVScraper) runEV(ctx context.Context, uuid string) ([]result, []string) {
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		return nil, []string{err.Error()}
	}

	productUUID := co.UUID
	setCode := co.SetCode

	var allTheErrors []string

	// Enumerate the full universe of possible cards and their probabilities.
	probs, err := mtgmatcher.GetProbabilitiesForSealed(setCode, productUUID)
	if len(probs) == 0 {
		if err == nil {
			err = errors.New("no probabilities found")
		}
		return nil, []string{err.Error()}
	}

	picks := make([]string, len(probs))
	probabilities := make([]float64, len(probs))
	skipped := make(map[string]bool)
	for i := range probs {
		picks[i] = probs[i].UUID
		probabilities[i] = probs[i].Probability

		// Serialized (and unresolvable) cards never count towards the EV.
		co, err := mtgmatcher.GetUUID(probs[i].UUID)
		if err != nil || co.HasPromoType(mtgmatcher.PromoTypeSerialized) {
			skipped[probs[i].UUID] = true
		}
	}

	// Resolve each card's price a single time per parameter (skipped cards
	// resolve to 0). This keeps the price lookups out of the simulation loop,
	// which can run up to EVAverageRepetition times.
	unitPrices := make([]map[string]float64, len(evParameters))
	for i := range evParameters {
		priceSource := ss.prices.Retail
		if evParameters[i].FoundInBuylist {
			priceSource = ss.prices.Buylist
		}

		cache := make(map[string]float64, len(picks))
		for _, pick := range picks {
			if skipped[pick] {
				continue
			}
			cache[pick] = maxStorePrice(pick, priceSource, evParameters[i].SourceStores)
		}
		unitPrices[i] = cache
	}

	datasets := make([][]float64, len(evParameters))

	// Deterministic probability-based EV for the non-simulation parameters.
	for i := range evParameters {
		if evParameters[i].Simulation {
			continue
		}
		datasets[i] = append(datasets[i], valueFromCache(picks, unitPrices[i], probabilities))
	}

	if !mtgmatcher.SealedIsRandom(setCode, productUUID) {
		// Fixed contents: a simulation would always draw the same cards, so its
		// value equals the deterministic probability EV. Copy it instead of
		// running a pointless Monte Carlo.
		for i := range evParameters {
			if !evParameters[i].Simulation {
				continue
			}
			datasets[i] = append(datasets[i], valueFromCache(picks, unitPrices[i], probabilities))
		}
	} else {
		// Random contents: Monte Carlo the simulation parameters.
		repeats := EVAverageRepetition
		if ss.FastMode {
			repeats = EVFastRepetition
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		repeatsChannel := make(chan int)
		locals := make([][][]float64, ss.MaxConcurrency)

		for w := 0; w < ss.MaxConcurrency; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()

				local := make([][]float64, len(evParameters))
				for range repeatsChannel {
					simPicks, err := mtgmatcher.GetPicksForSealed(setCode, productUUID)
					if err != nil {
						mu.Lock()
						if !slices.Contains(allTheErrors, err.Error()) {
							allTheErrors = append(allTheErrors, err.Error())
						}
						mu.Unlock()
						continue
					}

					for i := range evParameters {
						if !evParameters[i].Simulation {
							continue
						}
						local[i] = append(local[i], valueFromCache(simPicks, unitPrices[i], nil))
					}
				}
				locals[w] = local
			}(w)
		}

	feed:
		for j := 0; j < repeats; j++ {
			select {
			case <-ctx.Done():
				break feed
			case repeatsChannel <- j:
			}
		}
		close(repeatsChannel)
		wg.Wait()

		// Merge the per-worker datasets.
		for _, local := range locals {
			for i := range local {
				datasets[i] = append(datasets[i], local[i]...)
			}
		}
	}

	var out []result
	for i, dataset := range datasets {
		if len(dataset) == 0 {
			continue
		}

		price, err := evParameters[i].StatsFunc(dataset)
		if err != nil {
			allTheErrors = append(allTheErrors, err.Error())
			continue
		}
		if price == 0 {
			continue
		}

		res := result{
			productId: productUUID,
		}

		if evParameters[i].TargetsBuylist {
			var link string
			co, err := mtgmatcher.GetUUID(productUUID)
			if err == nil {
				link = "/search?q=contents:" + url.QueryEscape("\""+co.Name+"\"")
			}

			res.buyEntry = &mtgban.BuylistEntry{
				BuyPrice: price,
				URL:      link,
			}
		} else {
			var link string
			tcgID, _ := strconv.Atoi(co.Identifiers["tcgplayerProductId"])
			if tcgID != 0 {
				isDirect := slices.Contains(evParameters[i].SourceStores, "TCGDirectNet")
				link = tcgplayer.GenerateProductURL(tcgID, "", ss.Affiliate, "", "", isDirect)
			}

			res.invEntry = &mtgban.InventoryEntry{
				Price:      price,
				SellerName: evParameters[i].Name,
				URL:        link,
			}

			if evParameters[i].Simulation {
				stdDev, err := stats.StandardDeviation(dataset)
				if err == nil && stdDev > 0 {
					if res.invEntry.ExtraValues == nil {
						res.invEntry.ExtraValues = map[string]float64{}
					}
					res.invEntry.ExtraValues["stdDev"] = stdDev
				}

				iqr, err := stats.InterQuartileRange(dataset)
				if err == nil && iqr > 0 {
					if res.invEntry.ExtraValues == nil {
						res.invEntry.ExtraValues = map[string]float64{}
					}
					res.invEntry.ExtraValues["iqr"] = iqr
				}
			}
		}

		out = append(out, res)
	}

	return out, allTheErrors
}

func (ss *SealedEVScraper) Load(ctx context.Context) error {
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
			if ss.TargetEdition != "" && strings.ToLower(set.Code) != strings.ToLower(ss.TargetEdition) && strings.ToLower(set.Name) != strings.ToLower(ss.TargetEdition) {
				continue
			}
		}

		for _, product := range set.SealedProduct {
			// Skip unsupported types
			if product.Category == "land_station" {
				continue
			}

			// Skip unsupported languages
			if strings.Contains(product.Name, "Japanese") {
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
	if len(uuids) == 0 {
		return errors.New("no product loaded")
	}

	ss.printf("Loading BAN prices")
	prices, err := loadPrices(ctx, ss.banpriceKey, selected)
	if err != nil {
		return err
	}
	ss.printf("Retrieved %d+%d prices", len(prices.Retail), len(prices.Buylist))
	ss.prices = prices

	start := time.Now()

	for i, uuid := range uuids {
		if ctx.Err() != nil {
			break
		}

		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}

		if !ss.FastMode {
			ss.printf("Running EV on [%s] %s (%d/%d)", co.SetCode, co.Name, i+1, len(uuids))
		}

		results, messages := ss.runEV(ctx, uuid)

		// Print errors if necessary
		if len(messages) > 0 {
			ss.printf("%s - runEV error: %s", co.Name, strings.Join(messages, " | "))
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

func (ss *SealedEVScraper) Inventory() mtgban.InventoryRecord {
	return ss.inventory
}

func (ss *SealedEVScraper) Buylist() mtgban.BuylistRecord {
	return ss.buylist
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
