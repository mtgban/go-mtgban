package cardmarket

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// loadCatalogDatastore installs testdata/idmap_datastore.json, the published
// Magic datastore cut down to the printings these tests turn on, every row
// copied verbatim: Laughing Hyena and its starred foil, both faces of Order
// of Midnight, the two same-numbered Growth Charms of the Mystery Booster
// playtest sets, and the dagger variants under one Cardmarket id of Fifth
// Edition's Shivan Dragon and Happy Holidays' foil-only Gifts Given.
func loadCatalogDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	reader, err := os.Open("testdata/idmap_datastore.json")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	b, err := mtgmatcher.Open("magic", reader)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The uuids are real entries of the published Magic map, chosen for their
// shapes: a plain and foil pair, a double-faced card whose second uuid is
// its back face the datastore never indexes, and a product whose printings
// share a number that therefore settles nothing.
func TestResolveUUIDs(t *testing.T) {
	b := loadCatalogDatastore(t)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	tests := []struct {
		name     string
		product  cm.Product
		uuids    []string
		wantSame bool // both ids answer the same printing
		wantFoil bool // the foil id differs from the plain one
		foilOnly bool // one foil printing answers both columns
		wantNone bool
		wantUUID string // the plain id, where it has to be a given one
	}{
		{
			name:    "a plain and foil pair splits across the columns",
			product: cm.Product{IDProduct: 14866, Name: "Laughing Hyena", Number: "103"},
			uuids: []string{
				"aec9aca7-ae93-5d6a-9070-4f6829af218c",
				"f3567523-c62d-5903-aee8-7bdfcfc5d993",
			},
			wantFoil: true,
		},
		{
			name:    "a back face is passed over, not resolved",
			product: cm.Product{IDProduct: 398679, Name: "Order of Midnight // Alter Fate", Number: "99"},
			uuids: []string{
				"a9879e18-ba17-5577-b5da-7e6b433804b7",
				"c45098e6-b676-5758-ac75-f83d41fe7afa",
			},
		},
		{
			name:    "printings a number cannot settle still answer one",
			product: cm.Product{IDProduct: 414959, Name: "Growth Charm (V.1)"},
			uuids: []string{
				"20b40fba-a649-5004-a43e-69ab8b393d87",
				"7beff333-8949-5af6-9851-92cdf3ccb60d",
			},
		},
		{
			name:    "a dagger variant under the same id loses to the undecorated printing",
			product: cm.Product{IDProduct: 9639, Name: "Shivan Dragon"},
			uuids: []string{
				"2f118c2e-da99-54a6-894b-5c948d74b0f4",
				"7a24a4dd-a3e5-5b06-b133-ad4fd6293eb0",
			},
			wantUUID: "7a24a4dd-a3e5-5b06-b133-ad4fd6293eb0",
		},
		{
			name:    "a foil-only dagger variant loses to the undecorated printing",
			product: cm.Product{IDProduct: 20040, Name: "Gifts Given"},
			uuids: []string{
				"87e5a278-5cfd-5abf-bc89-5d6a50d91fac",
				"b9c93bde-7e6a-5a9d-b078-2cf886f7007e",
			},
			foilOnly: true,
			wantUUID: "b9c93bde-7e6a-5a9d-b078-2cf886f7007e",
		},
		{
			name:     "no known uuid decides nothing",
			product:  cm.Product{IDProduct: 1, Name: "Unknown"},
			uuids:    []string{"not-a-uuid"},
			wantNone: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cardID, cardIDFoil := mkm.resolveUUIDs(&test.product, test.uuids)
			if test.wantNone {
				if cardID != "" || cardIDFoil != "" {
					t.Fatalf("resolved (%q, %q), want nothing", cardID, cardIDFoil)
				}
				return
			}
			if cardID == "" {
				t.Fatal("resolved nothing")
			}
			if test.wantUUID != "" && cardID != test.wantUUID {
				t.Errorf("plain column landed on %s, want %s", cardID, test.wantUUID)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatalf("plain id %q: %v", cardID, err)
			}
			if test.foilOnly {
				if cardIDFoil != cardID {
					t.Errorf("foil column got %q, want %s", cardIDFoil, cardID)
				}
				return
			}
			if co.Foil || co.Etched {
				t.Errorf("plain column landed on a foil printing %s", cardID)
			}
			if test.wantFoil {
				if cardIDFoil == "" || cardIDFoil == cardID {
					t.Fatalf("foil column got %q, want a distinct printing", cardIDFoil)
				}
				foilCo, err := b.GetUUID(cardIDFoil)
				if err != nil {
					t.Fatalf("foil id %q: %v", cardIDFoil, err)
				}
				if !foilCo.Foil && !foilCo.Etched {
					t.Errorf("foil column landed on a plain printing %s", cardIDFoil)
				}
			}
		})
	}
}

// TestCheckCatalog pins the guard on a code-less map for the games that
// shelve whole foreign catalogs: without the expansion codes the map cannot
// say which shelves those are, and Load refuses to walk it rather than
// price the foreign printings onto the English ones.
func TestCheckCatalog(t *testing.T) {
	coded := &cm.Catalog{Data: cm.CatalogData{Expansions: map[int]cm.CatalogExpansion{
		1: {Name: "Romance Dawn", Code: "OP01"},
		2: {Name: "Romance Dawn", Code: "OP01-JP"},
	}}}
	bare := &cm.Catalog{Data: cm.CatalogData{Expansions: map[int]cm.CatalogExpansion{
		1: {Name: "Romance Dawn"},
	}}}

	for _, tt := range []struct {
		name    string
		game    mtgmatcher.Game
		catalog *cm.Catalog
		usable  bool
	}{
		{"one piece coded", mtgmatcher.GameOnePiece, coded, true},
		{"one piece bare", mtgmatcher.GameOnePiece, bare, false},
		{"yugioh bare", mtgmatcher.GameYuGiOh, bare, false},
		{"fab coded", mtgmatcher.GameFleshAndBlood, coded, true},
		{"fab bare", mtgmatcher.GameFleshAndBlood, bare, false},
		{"magic bare", mtgmatcher.GameMagic, bare, true},
		{"magic none", mtgmatcher.GameMagic, nil, false},
	} {
		mkm, err := NewScraperIndex(&mtgmatcher.Backend{Game: tt.game})
		if err != nil {
			t.Fatalf("%s: NewScraperIndex(%v) = %v", tt.name, tt.game, err)
		}
		mkm.catalog = tt.catalog
		err = mkm.checkCatalog()
		if (err == nil) != tt.usable {
			t.Errorf("%s: checkCatalog() = %v, want usable %v", tt.name, err, tt.usable)
		}
	}
}

// TestMergeList pins how the product list fills out the catalog, for every
// game: a catalog product stays as the catalog has it, a product only the
// list carries joins on a shelf the catalog names, with no number, and one on
// a shelf the catalog does not name is dropped.
func TestMergeList(t *testing.T) {
	catalog := &cm.Catalog{}
	catalog.Data.Expansions = map[int]cm.CatalogExpansion{1: {Name: "Alpha", Code: "LEA"}}
	catalog.Data.Products = map[int]cm.CatalogProduct{
		10: {ExpansionID: 1, Name: "Black Lotus", Number: "232", Rarity: "Rare"},
	}
	list := []cm.ProductList{
		{IDProduct: 10, Name: "Black Lotus (renamed)", ExpansionID: 1},
		{IDProduct: 11, Name: "Mox Pearl", ExpansionID: 1},
		{IDProduct: 12, Name: "Alfie's Token", ExpansionID: 5571},
	}

	got := mergeList(catalog, list, nil)
	want := map[int]cm.CatalogProduct{
		10: {ExpansionID: 1, Name: "Black Lotus", Number: "232", Rarity: "Rare"},
		11: {ExpansionID: 1, Name: "Mox Pearl"},
	}
	same := func(a, b cm.CatalogProduct) bool {
		return a.ExpansionID == b.ExpansionID && a.Name == b.Name && a.Number == b.Number && a.Rarity == b.Rarity
	}
	if !maps.EqualFunc(got, want, same) {
		t.Errorf("mergeList = %v, want %v", got, want)
	}
}

// TestUnpricedSummary pins that a product the guide has no row for yet is
// no refusal: of two products that resolve by their ids, the one the guide
// prices is added and the other is named once, in a summary line, instead of
// once each as an error.
func TestUnpricedSummary(t *testing.T) {
	b := loadCatalogDatastore(t)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	var lines []string
	mkm.logCallback = func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) }
	mkm.exchangeRate = 1
	mkm.maxConcurrency = 1
	mkm.catalog = &cm.Catalog{}
	mkm.catalog.Data.Expansions = map[int]cm.CatalogExpansion{1: {Name: "Visions", Code: "VIS"}}
	mkm.priceGuide = map[int]cm.PriceGuide{14866: {IDProduct: 14866, LowPrice: 1, TrendPrice: 1}}
	products := map[int]cm.CatalogProduct{
		14866:  {ExpansionID: 1, Name: "Laughing Hyena", Number: "103"},
		398679: {ExpansionID: 1, Name: "Order of Midnight // Alter Fate", Number: "99"},
	}

	err = mkm.priceProducts(context.Background(), products)
	if err != nil {
		t.Fatal(err)
	}

	var pending int
	var refusedZero bool
	for _, line := range lines {
		if strings.Contains(line, "returned") || strings.Contains(line, "PriceGuide") {
			t.Errorf("unpriced product reported as an error: %q", line)
		}
		if strings.Contains(line, "Walked 2 products, 0 of which") {
			refusedZero = true
		}
		if strings.Contains(line, "not in the price guide yet") && strings.Contains(line, "398679") {
			pending++
		}
	}
	if pending != 1 {
		t.Errorf("got %d summary lines in %q, want 1", pending, lines)
	}
	if !refusedZero {
		t.Errorf("the unpriced product counted as refused: %q", lines)
	}
	if len(mkm.inventory) != 1 {
		t.Errorf("priced %d printings, want the one product the guide lists", len(mkm.inventory))
	}
}
