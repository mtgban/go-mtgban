package cardmarket

import (
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// loadIDMapDatastore installs testdata/idmap_datastore.json, the published
// Magic datastore cut down to the six printings these tests turn on, every
// row copied verbatim: Laughing Hyena and its starred foil, both faces of
// Order of Midnight, and the two same-numbered Growth Charms of the Mystery
// Booster playtest sets.
func loadIDMapDatastore(t *testing.T) {
	t.Helper()
	reader, err := os.Open("testdata/idmap_datastore.json")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := mtgmatcher.LoadDatastore(reader); err != nil {
		t.Fatal(err)
	}
}

const idMapFixture = `{
	"meta": {"date": "2026-09-03", "version": "5.3.0"},
	"data": {
		"expansions": {
			"1": {"name": "Alpha", "setCodes": ["LEA"]}
		},
		"products": {
			"14866": {
				"expansionId": 1,
				"name": "Laughing Hyena",
				"number": "103",
				"uuids": ["aaa", "bbb"]
			},
			"999999": {
				"expansionId": 1,
				"name": "No Printing Of Ours"
			}
		}
	}
}`

func TestLoadIDMap(t *testing.T) {
	idMap, err := LoadIDMap(strings.NewReader(idMapFixture))
	if err != nil {
		t.Fatalf("LoadIDMap: %v", err)
	}
	if len(idMap.Products) != 2 || len(idMap.Expansions) != 1 {
		t.Fatalf("got %d products and %d expansions, want 2 and 1",
			len(idMap.Products), len(idMap.Expansions))
	}
	product, found := idMap.Products[14866]
	if !found {
		t.Fatal("the string key 14866 did not become the number")
	}
	if product.Number != "103" || len(product.UUIDs) != 2 {
		t.Errorf("product 14866 = %+v", product)
	}
	if idMap.Expansions[1].Name != "Alpha" {
		t.Errorf("expansion 1 = %+v", idMap.Expansions[1])
	}

	if _, err := LoadIDMap(strings.NewReader(`{"data":{}}`)); err == nil {
		t.Error("an empty map should refuse to load")
	}
}

// The uuids are real entries of the published Magic map, chosen for their
// shapes: a plain and foil pair, a double-faced card whose second uuid is
// its back face the datastore never indexes, and a product whose printings
// share a number that therefore settles nothing.
func TestResolveUUIDs(t *testing.T) {
	loadIDMapDatastore(t)

	mkm := &Index{gameID: GameMagic}

	tests := []struct {
		name     string
		product  MKMProduct
		uuids    []string
		wantSame bool // both ids answer the same printing
		wantFoil bool // the foil id differs from the plain one
		wantNone bool
	}{
		{
			name:    "a plain and foil pair splits across the columns",
			product: MKMProduct{IDProduct: 14866, Name: "Laughing Hyena", Number: "103"},
			uuids: []string{
				"aec9aca7-ae93-5d6a-9070-4f6829af218c",
				"f3567523-c62d-5903-aee8-7bdfcfc5d993",
			},
			wantFoil: true,
		},
		{
			name:    "a back face is passed over, not resolved",
			product: MKMProduct{IDProduct: 398679, Name: "Order of Midnight // Alter Fate", Number: "99"},
			uuids: []string{
				"a9879e18-ba17-5577-b5da-7e6b433804b7",
				"c45098e6-b676-5758-ac75-f83d41fe7afa",
			},
		},
		{
			name:    "printings a number cannot settle still answer one",
			product: MKMProduct{IDProduct: 414959, Name: "Growth Charm (V.1)"},
			uuids: []string{
				"20b40fba-a649-5004-a43e-69ab8b393d87",
				"7beff333-8949-5af6-9851-92cdf3ccb60d",
			},
		},
		{
			name:     "no known uuid decides nothing",
			product:  MKMProduct{IDProduct: 1, Name: "Unknown"},
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

// TestCheckIDMap pins the guard on a code-less map for the games that
// shelve whole foreign catalogs: without the expansion codes the map cannot
// say which shelves those are, and Load refuses to walk it rather than
// price the foreign printings onto the English ones.
func TestCheckIDMap(t *testing.T) {
	coded := &IDMap{Expansions: map[int]IDMapExpansion{
		1: {Name: "Romance Dawn", Code: "OP01"},
		2: {Name: "Romance Dawn", Code: "OP01-JP"},
	}}
	bare := &IDMap{Expansions: map[int]IDMapExpansion{
		1: {Name: "Romance Dawn"},
	}}

	for _, tt := range []struct {
		name   string
		gameID int
		idMap  *IDMap
		usable bool
	}{
		{"one piece coded", GameOnePiece, coded, true},
		{"one piece bare", GameOnePiece, bare, false},
		{"yugioh bare", GameYuGiOh, bare, false},
		{"magic bare", GameMagic, bare, true},
		{"magic none", GameMagic, nil, false},
	} {
		mkm := NewScraperIndex(tt.gameID)
		mkm.IDMap = tt.idMap
		err := mkm.checkIDMap()
		if (err == nil) != tt.usable {
			t.Errorf("%s: checkIDMap() = %v, want usable %v", tt.name, err, tt.usable)
		}
	}
}
