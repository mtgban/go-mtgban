// Package sealedev prices sealed product by the expected value of its
// contents, simulating openings against singles prices instead of reading
// a storefront.
package sealedev

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

const (
	// defaultRepetitions is how many openings are simulated before an
	// average settles, where the scraper is not told otherwise.
	defaultRepetitions = 5000

	defaultConcurrency = 8
)

// Scraper prices sealed product by what opening it is worth, drawing
// its contents against singles prices rather than reading any storefront.
type Scraper struct {
	logCallback    mtgban.LogCallbackFunc
	affiliate      string
	targetEdition  string
	targetProduct  string
	maxConcurrency int
	// Repetitions is how many openings of a random product are simulated
	// before its average settles. NewScraper sets the default; a caller
	// wanting a quick answer lowers it, and a test wanting a run that
	// cannot finish raises it.
	repetitions int

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	backend     *mtgmatcher.Backend
	banpriceKey string
	prices      *priceSnapshot

	// held marks the products another product holds whole, and kept is what
	// opening each of those came to once it has been simulated, so the
	// product holding it draws from those openings rather than opening it
	// again.
	held map[string]bool
	kept map[string]opened
}

// opened is what simulating one product came to: a dataset per parameter,
// and what went wrong along the way.
type opened struct {
	datasets [][]float64
	errs     []string
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

	// Mana Pool
	{
		Name:         "Mana Pool EV",
		Shorthand:    "MPEV",
		StatsFunc:    passthroughFirst,
		SourceStores: []string{"MP"},
	},
	{
		Name:      "Mana Pool Sim",
		Shorthand: "MPSim",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceStores: []string{"MP"},
		Simulation:   true,
	},

	// Cardmarket. The cards its market scraper never polled are priced from
	// the published guide instead; see mkm.go for what that is worth, and
	// for the games it is not worth enough in.
	{
		Name:         "Cardmarket EV",
		Shorthand:    "MKMEV",
		StatsFunc:    passthroughFirst,
		SourceStores: []string{"MKM"},
	},
	{
		Name:      "Cardmarket Sim",
		Shorthand: "MKMSim",
		StatsFunc: func(values []float64) (float64, error) {
			return stats.Median(values)
		},
		SourceStores: []string{"MKM"},
		Simulation:   true,
	},

	// Custom buylist
	{
		Name:           "TCG Direct SYP (net) EV",
		Shorthand:      "TCGDirectSYPNetEV",
		StatsFunc:      passthroughFirst,
		SourceStores:   []string{"TCGDirectSYPNet"},
		FoundInBuylist: true,
	},
}

// NewScraper returns an EV scraper, signing its price lookups with sig.
func NewScraper(b *mtgmatcher.Backend, sig string) *Scraper {
	ss := Scraper{}
	ss.backend = b
	ss.inventory = mtgban.InventoryRecord{}
	ss.buylist = mtgban.BuylistRecord{}
	ss.banpriceKey = sig
	ss.maxConcurrency = defaultConcurrency
	ss.repetitions = defaultRepetitions
	return &ss
}

func (ss *Scraper) printf(format string, a ...any) {
	if ss.logCallback != nil {
		ss.logCallback("[SS] "+format, a...)
	}
}

type result struct {
	productID string
	invEntry  *mtgban.InventoryEntry
	buyEntry  *mtgban.BuylistEntry
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

// skipFromEV identifies cards that should not contribute to sealed-product
// EV. Serialized and cosmic-foil printings have no usable EV; SLD bonuses are
// skipped only when their published distribution is not fixed: fewer than one
// copy, on average, in one copy of the product holding them.
func skipFromEV(co *mtgmatcher.CardObject, err error, expectedCountPerCopy float64) bool {
	return err != nil || co.HasPromoType(magic.PromoTypeSerialized) ||
		co.HasPromoType(magic.PromoTypeCosmicFoil) ||
		(co.HasPromoType(magic.PromoTypeSLDBonus) && expectedCountPerCopy < 1)
}

// open simulates a product, or hands back what simulating it came to
// already where another product holding it asked first.
func (ss *Scraper) open(ctx context.Context, co *mtgmatcher.CardObject) opened {
	kept, found := ss.kept[co.UUID]
	if found {
		return kept
	}
	op := ss.simulate(ctx, co.SetCode, co.UUID)
	if ss.held[co.UUID] {
		ss.kept[co.UUID] = op
	}
	return op
}

// simulate values a product under every parameter: by its probabilities, and
// by simulated openings where the parameter simulates.
func (ss *Scraper) simulate(ctx context.Context, setCode, productUUID string) opened {
	var allTheErrors []string

	// Enumerate the full universe of possible cards and their probabilities.
	probs, err := sealed.ProductCounts(ss.backend, setCode, productUUID)
	if len(probs) == 0 {
		if err == nil {
			err = errors.New("no probabilities found")
		}
		return opened{errs: []string{err.Error()}}
	}

	picks := make([]string, len(probs))
	probabilities := make([]float64, len(probs))
	skipped := make(map[string]bool)
	for i := range probs {
		picks[i] = probs[i].UUID
		probabilities[i] = probs[i].ExpectedCount * float64(probs[i].Copies)

		// SLD bonuses without a fixed published distribution never count towards the EV.
		co, err := ss.backend.GetUUID(probs[i].UUID)
		if skipFromEV(co, err, probs[i].ExpectedCount) {
			skipped[probs[i].UUID] = true
		}
	}

	// Resolve each card's price a single time per parameter (skipped cards
	// resolve to 0). This keeps the price lookups out of the simulation loop,
	// which can run Repetitions times.
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
			cache[pick] = maxStorePrice(ss.backend, pick, priceSource, evParameters[i].SourceStores)
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

	if !sealed.IsRandom(ss.backend, setCode, productUUID) {
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
		repeats := ss.repetitions
		held := ss.heldOpenings(ctx, setCode, productUUID)

		var mu sync.Mutex
		var wg sync.WaitGroup
		repeatsChannel := make(chan int)
		locals := make([][][]float64, ss.maxConcurrency)

		for w := 0; w < ss.maxConcurrency; w++ {
			wg.Go(func() {
				local := make([][]float64, len(evParameters))
				for range repeatsChannel {
					if held != nil {
						for i := range evParameters {
							if evParameters[i].Simulation {
								local[i] = append(local[i], held.draw(i))
							}
						}
						continue
					}

					simPicks, err := sealed.ProductPicks(ss.backend, setCode, productUUID)
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
			})
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

	return opened{datasets: datasets, errs: allTheErrors}
}

// heldContents is one product another holds whole, as many times as it
// holds it.
type heldContents struct {
	datasets [][]float64
	count    int
}

type heldOpenings []heldContents

// draw values one opening of the holding product for parameter i, as one
// opening of each product it holds, each drawn from that product's own.
func (h heldOpenings) draw(i int) float64 {
	var total float64
	for _, content := range h {
		dataset := content.datasets[i]
		for range content.count {
			total += dataset[rand.IntN(len(dataset))]
		}
	}
	return total
}

// heldOpenings returns the openings of the products a product holds, where
// whole sealed products are all it holds and each of them opened, and nil
// where it must be opened card by card instead, which is also what decides
// whether an unopenable sample pack may be left out.
func (ss *Scraper) heldOpenings(ctx context.Context, setCode, productUUID string) heldOpenings {
	set, err := ss.backend.GetSet(setCode)
	if err != nil {
		return nil
	}
	idx := slices.IndexFunc(set.SealedProduct, func(p mtgmatcher.SealedProduct) bool {
		return p.UUID == productUUID
	})
	if idx < 0 {
		return nil
	}
	contents := set.SealedProduct[idx].Contents
	if !holdsOnlySealed(contents) {
		return nil
	}

	var out heldOpenings
	for _, content := range contents["sealed"] {
		co, err := ss.backend.GetUUID(content.UUID)
		if err != nil {
			return nil
		}
		op := ss.open(ctx, co)
		if !opensForEveryParameter(op) {
			return nil
		}
		out = append(out, heldContents{datasets: op.datasets, count: content.Count})
	}
	return out
}

// opensForEveryParameter reports whether a simulation came to at least one
// opening for each parameter that simulates.
func opensForEveryParameter(op opened) bool {
	if len(op.datasets) == 0 {
		return false
	}
	for i := range evParameters {
		if evParameters[i].Simulation && len(op.datasets[i]) == 0 {
			return false
		}
	}
	return true
}

func (ss *Scraper) runEV(ctx context.Context, uuid string) ([]result, []string) {
	co, err := ss.backend.GetUUID(uuid)
	if err != nil {
		return nil, []string{err.Error()}
	}
	productUUID := co.UUID

	op := ss.open(ctx, co)
	datasets := op.datasets
	allTheErrors := op.errs

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
			productID: productUUID,
		}

		if evParameters[i].TargetsBuylist {
			var link string
			co, err := ss.backend.GetUUID(productUUID)
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
				link = tcgplayer.GenerateProductURL(tcgID, "", ss.affiliate, "", "", isDirect)
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

// heldWhole marks the products that another product made only of whole
// sealed products holds, the ones whose openings are worth keeping.
func heldWhole(b *mtgmatcher.Backend) map[string]bool {
	held := map[string]bool{}
	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		for _, product := range set.SealedProduct {
			if !holdsOnlySealed(product.Contents) {
				continue
			}
			for _, content := range product.Contents["sealed"] {
				held[content.UUID] = true
			}
		}
	}
	return held
}

// holdsOnlySealed reports whether every card a product's contents can give
// comes from a whole sealed product inside it.
func holdsOnlySealed(contents map[string][]mtgmatcher.SealedContent) bool {
	for kind, entries := range contents {
		if kind != "sealed" && kind != "other" && len(entries) > 0 {
			return false
		}
	}
	return len(contents["sealed"]) > 0
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (ss *Scraper) Load(ctx context.Context) error {
	var selected string

	ss.printf("Loading products")
	sets := ss.backend.GetAllSets()
	var uuids []string
	for _, code := range sets {
		set, _ := ss.backend.GetSet(code)

		switch set.Code {
		// Skip products without Sealed or Booster information
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		default:
			// Skip filtered editions if set
			if ss.targetEdition != "" && !strings.EqualFold(set.Code, ss.targetEdition) && !strings.EqualFold(set.Name, ss.targetEdition) {
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
			if ss.targetProduct != "" && product.Name != ss.targetProduct && product.UUID != ss.targetProduct {
				continue
			}

			uuids = append(uuids, product.UUID)
		}

		// Keep track of what was selected to reduce price calls
		if ss.targetEdition != "" {
			selected = "/" + set.Code
		}
	}
	ss.printf("Found %d products over %d sets", len(uuids), len(sets))
	if len(uuids) == 0 {
		return errors.New("no product loaded")
	}

	ss.printf("Loading BAN prices")
	prices, err := loadPrices(ctx, ss.backend, ss.banpriceKey, selected)
	if err != nil {
		return err
	}
	ss.printf("Retrieved %d+%d prices", len(prices.Retail), len(prices.Buylist))
	ss.prices = prices
	ss.held = heldWhole(ss.backend)
	ss.kept = make(map[string]opened, len(ss.held))

	start := time.Now()

	for i, uuid := range uuids {
		if ctx.Err() != nil {
			break
		}

		co, err := ss.backend.GetUUID(uuid)
		if err != nil {
			continue
		}

		ss.printf("Running EV on [%s] %s (%d/%d)", co.SetCode, co.Name, i+1, len(uuids))

		results, messages := ss.runEV(ctx, uuid)

		// Print errors if necessary
		if len(messages) > 0 {
			ss.printf("%s - runEV error: %s", co.Name, strings.Join(messages, " | "))
		}

		for _, result := range results {
			if result.invEntry != nil {
				ss.inventory.Add(result.productID, result.invEntry)
			}
			if result.buyEntry != nil {
				ss.buylist.Add(result.productID, result.buyEntry)
			}
		}
	}

	ss.printf("Took %v", time.Since(start))

	ss.inventoryDate = time.Now()
	ss.buylistDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (ss *Scraper) Inventory() mtgban.InventoryRecord {
	return ss.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (ss *Scraper) Buylist() mtgban.BuylistRecord {
	return ss.buylist
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (ss *Scraper) MarketNames() []string {
	var names []string
	for _, param := range evParameters {
		if param.TargetsBuylist {
			continue
		}
		names = append(names, param.Name)
	}
	return names
}

// InfoForScraper describes one of the sub-scrapers named above.
func (ss *Scraper) InfoForScraper(name string) mtgban.ScraperInfo {
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

// Info describes this scraper. See mtgban.Scraper.
func (ss *Scraper) Info() (info mtgban.ScraperInfo) {
	info.Name = "Sealed EV Scraper"
	info.Shorthand = "SS"
	info.InventoryTimestamp = &ss.inventoryDate
	info.BuylistTimestamp = &ss.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.3
	info.Family = "EV"
	info.Game = mtgmatcher.GameMagic
	return
}
