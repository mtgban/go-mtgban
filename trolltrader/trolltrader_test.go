package trolltrader

import (
	"encoding/json"
	"maps"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

var (
	magicOnce sync.Once
	magicErr  error
	magicB    *matcher.Backend
)

// withMagic loads AllPrintings the first time a test asks for it, and skips
// where the run carries none.
func withMagic(t *testing.T) *matcher.Backend {
	t.Helper()
	magicOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		magicB, magicErr = datastore.Read("magic", path)
	})
	if magicErr != nil {
		t.Fatal(magicErr)
	}
	if magicB == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return magicB
}

func TestTitleRE(t *testing.T) {
	for title, want := range map[string][2]string{
		"Raugrin Triome (251)":                               {"Raugrin Triome", "251"},
		"Spider-Sense (284) - BaB Promo":                     {"Spider-Sense", "284"},
		"Mirko, Obsessive Theorist (50) (Display Commander)": {"Mirko, Obsessive Theorist", "50"},
		"Counterspell - Release Promo(457)":                  {"Counterspell - Release Promo", "457"},
		"Emblem - Jace, Unraveler of Secrets (17)":           {"Emblem - Jace, Unraveler of Secrets", "17"},
	} {
		m := titleRE.FindStringSubmatch(title)
		if m == nil || m[1] != want[0] || m[2] != want[1] {
			t.Errorf("%q: got %q, want %q", title, m, want)
		}
	}
}

func TestTTCondition(t *testing.T) {
	for grade, want := range map[string]mtgban.Condition{
		"Near Mint":    mtgban.NM,
		"Excellent":    mtgban.SP,
		"Good":         mtgban.MP,
		"Light Played": mtgban.MP,
		"Played":       mtgban.HP,
		"Poor":         mtgban.PO,
	} {
		got, err := ttCondition(grade)
		if err != nil || got != want {
			t.Errorf("%s: got %s (%v), want %s", grade, got, err, want)
		}
	}
	_, err := ttCondition("Signed")
	if err == nil {
		t.Error("Signed read as a grade")
	}
}

func readProducts(t *testing.T) []shopifyProduct {
	t.Helper()
	data, err := os.ReadFile("testdata/products.json")
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Products []shopifyProduct `json:"products"`
	}
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatal(err)
	}
	return out.Products
}

// TestProcessProduct lists each sample product's English variants in stock
// on the printing its SKU's set and its title's number name.
func TestProcessProduct(t *testing.T) {
	b := withMagic(t)
	tt := NewScraper(b)
	tt.rate = 1
	tt.logCallback = t.Logf

	for _, product := range readProducts(t) {
		if isMagicSingle(product) {
			tt.processProduct(product)
		}
	}

	type landing struct {
		set, number  string
		foil, etched bool
		cond         mtgban.Condition
	}
	got := map[landing]float64{}
	for cardID, entries := range tt.inventory {
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			got[landing{co.SetCode, co.Number, co.Foil, co.Etched, entry.Conditions}] = entry.Price
		}
	}
	// Out of stock, French and signed variants are left out, and the
	// Cyberpunk single is not Magic's. The store numbers a foil the
	// datastore files apart the same as the plain card and an etched one
	// with an e, files an emblem and a two-sided token under the set's own
	// code, and titles a Secret Lair crossover by its flavor name first.
	want := map[landing]float64{
		{"IKO", "251", false, false, mtgban.NM}:      13.43,
		{"SPM", "284", true, false, mtgban.NM}:       0.81,
		{"GTC", "247", true, false, mtgban.NM}:       13.35,
		{"GTC", "247", false, false, mtgban.MP}:      6.24,
		{"WC98", "rb57", false, false, mtgban.NM}:    1.48,
		{"WC98", "rb57", false, false, mtgban.MP}:    1.33,
		{"DMR", "457", true, false, mtgban.NM}:       6.33,
		{"MKC", "50", false, true, mtgban.NM}:        0.45,
		{"8ED", "64", false, false, mtgban.NM}:       5.05,
		{"8ED", "64", false, false, mtgban.MP}:       4.55,
		{"8ED", "64★", true, false, mtgban.NM}:       18.3,
		{"MH2", "273", false, true, mtgban.NM}:       1.66,
		{"ALL", "88a", false, false, mtgban.MP}:      0.53,
		{"SLD", "1559★", true, false, mtgban.NM}:     25.03,
		{"TAFR", "16", false, false, mtgban.NM}:      0.37,
		{"TAFR", "5 // 22", false, false, mtgban.NM}: 0.35,
		{"TAFR", "5 // 22", true, false, mtgban.NM}:  0.35,
	}
	if !maps.Equal(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}
