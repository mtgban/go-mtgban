package manaleak

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
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

func TestMatchAmbiguousID(t *testing.T) {
	b := realDatastore(t)
	ml := NewScraper(b)

	tests := []struct {
		name    string
		product MLProduct
		set     string
		number  string
		finish  string
	}{
		{
			name:    "double-faced card by its front face",
			product: MLProduct{Name: "Bruna, The Fading Light", TCGProductID: "119686", AmbiguousID: true},
			set:     "EMN", number: "15", finish: "nonfoil",
		},
		{
			name:    "double-faced card, foil",
			product: MLProduct{Name: "Bruna, The Fading Light - Foil", TCGProductID: "119686", AmbiguousID: true},
			set:     "EMN", number: "15", finish: "foil",
		},
		{
			name:    "misspelled name",
			product: MLProduct{Name: "Dread Sanctuary", TCGProductID: "121810", AmbiguousID: true},
			set:     "CN2", number: "217", finish: "nonfoil",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := ml.match(tc.product)
			if err != nil {
				t.Fatal(err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tc.set || co.Number != tc.number || co.Finish != tc.finish {
				t.Errorf("landed on %s %s %s, want %s %s %s",
					co.SetCode, co.Number, co.Finish, tc.set, tc.number, tc.finish)
			}
		})
	}
}

// TestMatchProductPage pins rows whose listing image names no card, read off
// their product page: an id behind a set code, a bare id in either id space,
// the Promos shelf, where only a collector number picks the printing, and a
// set's own promo shelf, where the promo printing is the one that counts.
func TestMatchProductPage(t *testing.T) {
	b := realDatastore(t)
	ml := NewScraper(b)

	tests := []struct {
		name     string
		category string
		code     string
		product  MLProduct
		set      string
		number   string
		finish   string
	}{
		{
			name: "TCGplayer id behind a set code", category: "The List",
			code: "mtgTLP269131", product: MLProduct{Name: "Elspeth, Undaunted Hero"},
			set: "PLST", number: "THB-270", finish: "nonfoil",
		},
		{
			name: "bare multiverse id", category: "Battle for Zendikar",
			code: "mtg401991", product: MLProduct{Name: "Plains (252) - Full Art"},
			set: "BFZ", number: "252", finish: "nonfoil",
		},
		{
			name: "collector number on the Promos shelf", category: "Promos",
			code: "mtgGRC_12-f", product: MLProduct{Name: "Dauntless Dourbark - Foil"},
			set: "DCI", number: "12", finish: "foil",
		},
		{
			name: "intro pack promo of a set", category: "Khans of Tarkir",
			code: "mtgMBP_93-f", product: MLProduct{Name: "Ankle Shanker - Intro Pack Foil"},
			set: "PKTK", number: "164", finish: "foil",
		},
		{
			name: "box promo spelling", category: "Zendikar",
			code: "mtgMBP_22-f", product: MLProduct{Name: "Day Of Judgment - Box Promo Foil"},
			set: "PZEN", number: "9★", finish: "foil",
		},
		{
			name: "storefront index on the Promos shelf", category: "Promos",
			code: "mtgFNMP_122-f", product: MLProduct{Name: "Anathemancer - Foil"},
			set: "F10", number: "7", finish: "foil",
		},
		{
			name: "player rewards index, nonfoil", category: "Promos",
			code: "mtgMPRP_39", product: MLProduct{Name: "Terminate"},
			set: "P09", number: "9", finish: "nonfoil",
		},
		{
			name: "generic promo name on the prerelease shelf", category: "Zendikar",
			code: "mtgPTC_42-f", product: MLProduct{Name: "Rampaging Baloths - Promo Foil"},
			set: "PZEN", number: "178★", finish: "foil",
		},
		{
			name: "duels promo filed under its set's core category", category: "Magic 2014 Core Set",
			code: "mtgMBP_68-f", product: MLProduct{Name: "Bonescythe Sliver - Promo Foil"},
			set: "PDP14", number: "1", finish: "foil",
		},
		{
			name: "card with a single promo printing on the Promos shelf", category: "Promos",
			code: "mtgMBP_7", product: MLProduct{Name: "Windseeker Centaur"},
			set: "PHPR", number: "3", finish: "nonfoil",
		},
		{
			name: "release shelf row named prerelease", category: "Magic 2011",
			code: "mtgMLP_11-f", product: MLProduct{Name: "Ancient Hellkite - Prerelease Promo Foil"},
			set: "PM11", number: "122★", finish: "foil",
		},
		{
			name: "promo shelf id naming the promo", category: "Aether Revolt",
			code: "mtgMBP125841-f", product: MLProduct{Name: "Scrap Trawler - Buy a Box Promo Foil"},
			set: "PAER", number: "175", finish: "foil",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.product.readPage(tc.category, tc.code)
			id, err := ml.match(tc.product)
			if err != nil {
				t.Fatal(err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tc.set || co.Number != tc.number || co.Finish != tc.finish {
				t.Errorf("landed on %s %s %s, want %s %s %s",
					co.SetCode, co.Number, co.Finish, tc.set, tc.number, tc.finish)
			}
		})
	}

	// Rows no printing is provable for stay unpriced: FNM promos of one card
	// in two years, and a card whose lone answer from the matcher is not the
	// printing the shelf and number name.
	for _, tc := range []struct {
		name, category, code string
	}{
		{"Goblin Warchief - Foil", "Promos", "mtgFNMP_72-f"},
		{"Niv-Mizzet, The Firemind - Foil", "Promos", "mtgCP_2-f"},
	} {
		product := MLProduct{Name: tc.name}
		product.readPage(tc.category, tc.code)
		_, err := ml.match(product)
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("%s: an unprovable promo row answered %v, want ErrUnsupported", tc.name, err)
		}
	}
}

func TestNameAgrees(t *testing.T) {
	b := realDatastore(t)
	ml := NewScraper(b)

	id := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "119686")
	cardID, err := b.MatchID(id, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !ml.nameAgrees(cardID, "Bruna, The Fading Light") {
		t.Error("a double-faced card's front face should agree")
	}
	if ml.nameAgrees(cardID, "Lightning Bolt") {
		t.Error("an unrelated name should not agree")
	}
	if ml.nameAgrees(cardID, "") {
		t.Error("an empty name should not agree")
	}
}
