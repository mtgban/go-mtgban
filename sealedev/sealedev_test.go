package sealedev

import (
	"context"
	"os"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// The Magic datastore is parsed the first time a test asks for it rather
// than up front. Most of this package prices entries it is handed and never
// reads a card, so the parse is ten seconds against a tenth of a second of
// testing, and charging it to every run made selecting one of those tests as
// slow as running all of them.
var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore loads the published Magic datastore, skipping the test where
// none is configured. The value is drawn from real sealed contents, so there
// is nothing to fake: a hand-built product would be a guess about the shape
// being priced.
func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		backend, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = backend
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// sealedProduct finds a product the datastore holds contents for, either one
// whose contents are drawn at random or one whose contents are fixed. Picking
// it at runtime keeps the test true across a datastore refresh, where a uuid
// written down here would rot.
func sealedProduct(t *testing.T, b *mtgmatcher.Backend, wantRandom bool) (string, string) {
	t.Helper()
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		probs, err := b.GetProbabilitiesForSealed(co.SetCode, uuid)
		if err != nil || len(probs) == 0 {
			continue
		}
		if b.SealedIsRandom(co.SetCode, uuid) == wantRandom {
			return uuid, co.SetCode
		}
	}
	t.Skipf("no sealed product with random=%v in this datastore", wantRandom)
	return "", ""
}

// pricedAt quotes every card the product can contain at the same price, in
// every store the parameters read, so what comes out is arithmetic on the
// contents rather than on which store happened to carry what.
func pricedAt(t *testing.T, b *mtgmatcher.Backend, setCode, uuid string, price float64) *BANPriceResponse {
	t.Helper()
	r := &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
	probs, err := b.GetProbabilitiesForSealed(setCode, uuid)
	if err != nil {
		t.Fatal(err)
	}
	stores := map[string]bool{}
	for _, parameter := range evParameters {
		for _, store := range parameter.SourceStores {
			stores[store] = true
		}
	}
	for _, prob := range probs {
		for store := range stores {
			r.setRetail(b, prob.UUID, store, price)
			r.setBuylist(b, prob.UUID, store, price)
		}
	}
	return r
}

func sldBonusProduct(t *testing.T, b *mtgmatcher.Backend) (string, string, string) {
	t.Helper()
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil || b.SealedIsRandom(co.SetCode, uuid) {
			continue
		}
		probs, err := b.GetProbabilitiesForSealed(co.SetCode, uuid)
		if err != nil {
			continue
		}
		for _, prob := range probs {
			card, err := b.GetUUID(prob.UUID)
			if err == nil && card.HasPromoType(magic.PromoTypeSLDBonus) && prob.Probability < 1 {
				return uuid, co.SetCode, prob.UUID
			}
		}
	}
	t.Skip("no fixed sealed product with a non-guaranteed SLD bonus")
	return "", "", ""
}

func resultPrices(results []result) []float64 {
	prices := make([]float64, 0, len(results))
	for _, result := range results {
		switch {
		case result.invEntry != nil:
			prices = append(prices, result.invEntry.Price)
		case result.buyEntry != nil:
			prices = append(prices, result.buyEntry.BuyPrice)
		}
	}
	sort.Float64s(prices)
	return prices
}

// TestRunEVSkipsUnfixedSLDBonusFromPriceCache proves the skip survives the
// entire run: making a non-fixed bonus absurdly expensive must not alter
// any EV result because that UUID never enters the unit-price cache.
func TestRunEVSkipsUnfixedSLDBonusFromPriceCache(t *testing.T) {
	b := realDatastore(t)
	productUUID, setCode, bonusUUID := sldBonusProduct(t, b)

	base := pricedAt(t, b, setCode, productUUID, 1)
	high := pricedAt(t, b, setCode, productUUID, 1)
	for _, parameter := range evParameters {
		for _, store := range parameter.SourceStores {
			high.setRetail(b, bonusUUID, store, 1000)
			high.setBuylist(b, bonusUUID, store, 1000)
		}
	}

	ss := NewScraper(b, "")
	ss.repetitions = 1
	ss.prices = base
	want, errs := ss.runEV(context.Background(), productUUID)
	if len(errs) != 0 {
		t.Fatalf("baseline runEV reported %v", errs)
	}

	ss.prices = high
	got, errs := ss.runEV(context.Background(), productUUID)
	if len(errs) != 0 {
		t.Fatalf("high-bonus runEV reported %v", errs)
	}
	if !slices.Equal(resultPrices(got), resultPrices(want)) {
		t.Fatalf("non-guaranteed bonus changed EV results: base=%v high=%v", resultPrices(want), resultPrices(got))
	}
}

// TestRunEVValuesAProduct pins that a product with contents and prices comes
// back valued, on every parameter, and in the shape each one asks for: the
// buylist parameter offers to buy it, the rest price it for sale.
func TestRunEVValuesAProduct(t *testing.T) {
	b := realDatastore(t)
	uuid, setCode := sealedProduct(t, b, true)

	ss := NewScraper(b, "")
	ss.repetitions = 10
	ss.prices = pricedAt(t, b, setCode, uuid, 1)

	results, errs := ss.runEV(context.Background(), uuid)
	if len(errs) != 0 {
		t.Fatalf("runEV reported %v", errs)
	}
	if len(results) == 0 {
		t.Fatal("runEV valued a product it has both contents and prices for at nothing")
	}

	var sawBuylist, sawRetail bool
	for _, res := range results {
		if res.productID != uuid {
			t.Errorf("a result carries %q, want the product %q", res.productID, uuid)
		}
		switch {
		case res.buyEntry != nil:
			sawBuylist = true
			if res.buyEntry.BuyPrice <= 0 {
				t.Errorf("a buylist result offers %v", res.buyEntry.BuyPrice)
			}
			if res.buyEntry.URL == "" {
				t.Error("a buylist result carries no link to what it is buying")
			}
		case res.invEntry != nil:
			sawRetail = true
			if res.invEntry.Price <= 0 {
				t.Errorf("a retail result prices it at %v", res.invEntry.Price)
			}
			if res.invEntry.SellerName == "" {
				t.Error("a retail result does not say which measure produced it")
			}
		default:
			t.Error("a result carries neither side")
		}
	}
	if !sawBuylist || !sawRetail {
		t.Errorf("saw buylist=%v retail=%v, want both", sawBuylist, sawRetail)
	}
}

// TestRunEVSkipsTheSimulationForFixedContents pins the shortcut: a product
// whose contents never vary would draw the same cards every time, so its
// simulated value is its probability value rather than five thousand
// identical openings.
func TestRunEVSkipsTheSimulationForFixedContents(t *testing.T) {
	b := realDatastore(t)
	uuid, setCode := sealedProduct(t, b, false)

	ss := NewScraper(b, "")
	ss.repetitions = 10
	ss.prices = pricedAt(t, b, setCode, uuid, 1)

	results, _ := ss.runEV(context.Background(), uuid)
	byName := map[string]float64{}
	for _, res := range results {
		if res.invEntry != nil {
			byName[res.invEntry.SellerName] = res.invEntry.Price
		}
	}

	// Each simulated measure reads the same store as a deterministic one;
	// with fixed contents the two must agree.
	var checked int
	for _, sim := range evParameters {
		if !sim.Simulation {
			continue
		}
		for _, det := range evParameters {
			if det.Simulation || det.FoundInBuylist != sim.FoundInBuylist {
				continue
			}
			if len(det.SourceStores) != 1 || det.SourceStores[0] != sim.SourceStores[0] {
				continue
			}
			simPrice, ok := byName[sim.Name]
			detPrice, ok2 := byName[det.Name]
			if !ok || !ok2 {
				continue
			}
			if simPrice != detPrice {
				t.Errorf("%q gave %v but %q gave %v; fixed contents must agree",
					sim.Name, simPrice, det.Name, detPrice)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Skip("no simulated measure shares a store with a deterministic one")
	}
}

// A product the datastore has no contents for is reported rather than valued
// at nothing silently.
func TestRunEVReportsAProductItCannotOpen(t *testing.T) {
	b := realDatastore(t)

	ss := NewScraper(b, "")
	ss.repetitions = 10
	ss.prices = &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}

	results, errs := ss.runEV(context.Background(), "not-a-uuid")
	if len(results) != 0 {
		t.Errorf("runEV valued a product it cannot name: %v", results)
	}
	if len(errs) == 0 {
		t.Error("runEV said nothing about a product it cannot name")
	}
}

// Priced at nothing, a product is worth nothing, and says so by returning no
// rows rather than rows of zeroes.
func TestRunEVDropsAProductWorthNothing(t *testing.T) {
	b := realDatastore(t)
	uuid, _ := sealedProduct(t, b, true)

	ss := NewScraper(b, "")
	ss.repetitions = 10
	ss.prices = &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}

	results, _ := ss.runEV(context.Background(), uuid)
	if len(results) != 0 {
		t.Errorf("runEV returned %d rows for a product with no prices", len(results))
	}
}

// A cancelled context stops the openings rather than running all of them.
func TestRunEVStopsWhenCancelled(t *testing.T) {
	b := realDatastore(t)
	uuid, setCode := sealedProduct(t, b, true)

	ss := NewScraper(b, "")
	ss.prices = pricedAt(t, b, setCode, uuid, 1)

	// It still answers, on whatever it managed to draw; what matters is
	// that it returns rather than finishing the openings asked for - which
	// are made more than a run could ever finish, so that returning at all
	// is the cancellation and nothing else.
	ss.repetitions = 1 << 30

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ss.runEV(ctx, uuid)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("runEV kept opening after its context was cancelled")
	}
}

// TestMarketNames pins the sub-sellers this scraper splits into: one per
// measure that prices the product for sale.
func TestMarketNames(t *testing.T) {
	// No datastore lookup is reached on this path, so an empty backend is
	// enough.
	ss := NewScraper(&mtgmatcher.Backend{}, "")
	names := ss.MarketNames()
	if len(names) == 0 {
		t.Fatal("the scraper names no measures")
	}
	for _, name := range names {
		info := ss.InfoForScraper(name)
		if info.Name != name {
			t.Errorf("InfoForScraper(%q) is named %q", name, info.Name)
		}
		if info.Shorthand == "" {
			t.Errorf("InfoForScraper(%q) carries no shorthand", name)
		}
	}
}

// TestRunEVReportsHowMuchOpeningsVaried pins the spread a simulated measure
// carries beside its price. Openings of the same product are not worth the
// same, and a caller deciding whether to buy one wants to know by how much,
// so the deviation and the interquartile range ride along with the row.
func TestRunEVReportsHowMuchOpeningsVaried(t *testing.T) {
	b := realDatastore(t)
	uuid, setCode := sealedProduct(t, b, true)

	// Price the contents unevenly, or every opening is worth the same and
	// there is no spread to report.
	r := &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
	probs, err := b.GetProbabilitiesForSealed(setCode, uuid)
	if err != nil {
		t.Fatal(err)
	}
	stores := map[string]bool{}
	for _, parameter := range evParameters {
		for _, store := range parameter.SourceStores {
			stores[store] = true
		}
	}
	for i, prob := range probs {
		price := float64(1 + i%50)
		for store := range stores {
			r.setRetail(b, prob.UUID, store, price)
			r.setBuylist(b, prob.UUID, store, price)
		}
	}

	ss := NewScraper(b, "")
	ss.repetitions = 10
	ss.prices = r

	results, _ := ss.runEV(context.Background(), uuid)

	simulated := map[string]bool{}
	for _, parameter := range evParameters {
		if parameter.Simulation {
			simulated[parameter.Name] = true
		}
	}

	var sawSpread bool
	for _, res := range results {
		if res.invEntry == nil || !simulated[res.invEntry.SellerName] {
			continue
		}
		if res.invEntry.ExtraValues["stdDev"] > 0 && res.invEntry.ExtraValues["iqr"] > 0 {
			sawSpread = true
		}
	}
	if !sawSpread {
		t.Error("no simulated measure reported how much its openings varied")
	}
}

// The records a caller reads are the ones the load filled in.
func TestInventoryAndBuylistAreWhatWasLoaded(t *testing.T) {
	// No datastore lookup is reached on this path, so an empty backend is
	// enough.
	ss := NewScraper(&mtgmatcher.Backend{}, "")
	if got := len(ss.Inventory()); got != 0 {
		t.Errorf("a fresh scraper holds %d inventory rows, want none", got)
	}
	if got := len(ss.Buylist()); got != 0 {
		t.Errorf("a fresh scraper holds %d buylist rows, want none", got)
	}

	ss.inventory["product"] = []mtgban.InventoryEntry{{Price: 5}}
	ss.buylist["product"] = []mtgban.BuylistEntry{{BuyPrice: 3}}
	if got := ss.Inventory()["product"][0].Price; got != 5 {
		t.Errorf("Inventory read back %v, want 5", got)
	}
	if got := ss.Buylist()["product"][0].BuyPrice; got != 3 {
		t.Errorf("Buylist read back %v, want 3", got)
	}

	// The log callback is what a caller hears; without one it says nothing
	// rather than writing somewhere nobody asked for.
	ss.printf("nothing is listening")
	var heard string
	ss.logCallback = func(format string, a ...any) { heard = format }
	ss.printf("something is")
	if heard == "" {
		t.Error("printf said nothing to a caller that was listening")
	}
}
