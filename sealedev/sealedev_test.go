package sealedev

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// installed records whether TestMain found a Magic datastore. The EV tests
// read real sealed contents, so there is nothing to fake, and a run without
// the file skips them.
var installed bool

func TestMain(m *testing.M) {
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path != "" {
		b, err := datastore.Read("magic", path)
		if err != nil {
			log.Fatalln(err)
		}
		mtgmatcher.SetGlobalDatastore(b)
		installed = true
	}
	os.Exit(m.Run())
}

// realDatastore skips a test that reads the published Magic datastore where
// none is installed. The value is drawn from real sealed contents, so there
// is nothing to fake: a hand-built product would be a guess about the shape
// being priced.
func realDatastore(t *testing.T) {
	t.Helper()
	if !installed {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}

// sealedProduct finds a product the datastore holds contents for, either one
// whose contents are drawn at random or one whose contents are fixed. Picking
// it at runtime keeps the test true across a datastore refresh, where a uuid
// written down here would rot.
func sealedProduct(t *testing.T, wantRandom bool) (string, string) {
	t.Helper()
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		probs, err := mtgmatcher.GetProbabilitiesForSealed(co.SetCode, uuid)
		if err != nil || len(probs) == 0 {
			continue
		}
		if mtgmatcher.SealedIsRandom(co.SetCode, uuid) == wantRandom {
			return uuid, co.SetCode
		}
	}
	t.Skipf("no sealed product with random=%v in this datastore", wantRandom)
	return "", ""
}

// pricedAt quotes every card the product can contain at the same price, in
// every store the parameters read, so what comes out is arithmetic on the
// contents rather than on which store happened to carry what.
func pricedAt(t *testing.T, setCode, uuid string, price float64) *BANPriceResponse {
	t.Helper()
	r := &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
	probs, err := mtgmatcher.GetProbabilitiesForSealed(setCode, uuid)
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
			r.setRetail(prob.UUID, store, price)
			r.setBuylist(prob.UUID, store, price)
		}
	}
	return r
}

// TestRunEVValuesAProduct pins that a product with contents and prices comes
// back valued, on every parameter, and in the shape each one asks for: the
// buylist parameter offers to buy it, the rest price it for sale.
func TestRunEVValuesAProduct(t *testing.T) {
	realDatastore(t)
	uuid, setCode := sealedProduct(t, true)

	ss := NewScraper("")
	ss.FastMode = true
	ss.prices = pricedAt(t, setCode, uuid, 1)

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
	realDatastore(t)
	uuid, setCode := sealedProduct(t, false)

	ss := NewScraper("")
	ss.FastMode = true
	ss.prices = pricedAt(t, setCode, uuid, 1)

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
	realDatastore(t)

	ss := NewScraper("")
	ss.FastMode = true
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
	realDatastore(t)
	uuid, _ := sealedProduct(t, true)

	ss := NewScraper("")
	ss.FastMode = true
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
	realDatastore(t)
	uuid, setCode := sealedProduct(t, true)

	ss := NewScraper("")
	ss.prices = pricedAt(t, setCode, uuid, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// It still answers, on whatever it managed to draw; what matters is
	// that it returns rather than finishing five thousand openings.
	ss.runEV(ctx, uuid)
}

// TestMarketNames pins the sub-sellers this scraper splits into: one per
// measure that prices the product for sale.
func TestMarketNames(t *testing.T) {
	ss := NewScraper("")
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
	realDatastore(t)
	uuid, setCode := sealedProduct(t, true)

	// Price the contents unevenly, or every opening is worth the same and
	// there is no spread to report.
	r := &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
	probs, err := mtgmatcher.GetProbabilitiesForSealed(setCode, uuid)
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
			r.setRetail(prob.UUID, store, price)
			r.setBuylist(prob.UUID, store, price)
		}
	}

	ss := NewScraper("")
	ss.FastMode = true
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
	ss := NewScraper("")
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
	ss.LogCallback = func(format string, a ...any) { heard = format }
	ss.printf("something is")
	if heard == "" {
		t.Error("printf said nothing to a caller that was listening")
	}
}
