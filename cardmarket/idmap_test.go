package cardmarket

import (
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
func loadCatalogDatastore(t *testing.T) {
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
	installBackend(t, b)
}

// The uuids are real entries of the published Magic map, chosen for their
// shapes: a plain and foil pair, a double-faced card whose second uuid is
// its back face the datastore never indexes, and a product whose printings
// share a number that therefore settles nothing.
func TestResolveUUIDs(t *testing.T) {
	loadCatalogDatastore(t)

	mkm := &Index{gameID: cm.GameMagic}

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
			co, err := mtgmatcher.GetUUID(cardID)
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
				foilCo, err := mtgmatcher.GetUUID(cardIDFoil)
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
		gameID  int
		catalog *cm.Catalog
		usable  bool
	}{
		{"one piece coded", cm.GameOnePiece, coded, true},
		{"one piece bare", cm.GameOnePiece, bare, false},
		{"yugioh bare", cm.GameYuGiOh, bare, false},
		{"magic bare", cm.GameMagic, bare, true},
		{"magic none", cm.GameMagic, nil, false},
	} {
		mkm := NewScraperIndex(tt.gameID)
		mkm.Catalog = tt.catalog
		err := mkm.checkCatalog()
		if (err == nil) != tt.usable {
			t.Errorf("%s: checkCatalog() = %v, want usable %v", tt.name, err, tt.usable)
		}
	}
}
