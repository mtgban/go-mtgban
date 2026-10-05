package cardmarket

import (
	"maps"
	"os"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// loadCatalogDatastore installs testdata/idmap_datastore.json, the published
// Magic datastore cut down to the six printings these tests turn on, every
// row copied verbatim: Laughing Hyena and its starred foil, both faces of
// Order of Midnight, and the two same-numbered Growth Charms of the Mystery
// Booster playtest sets.
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
		wantNone bool
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
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatalf("plain id %q: %v", cardID, err)
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
