package gamenerdz

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// snorlaxDatastore is the published Pokemon datastore cut down to the two
// printings this test turns on, both rows copied verbatim from it: the same
// name, set, number and finish, told apart by nothing but the wording the
// storefront brackets. Two rows rather than the whole file keeps the answer
// below a fact about data the test states.
const snorlaxDatastore = `{
 "game": "pokemon",
 "sets": {"SVP": {"abbreviation": "SVP", "name": "SV: Scarlet & Violet Promo Cards", "releaseDate": "2023-03-31", "type": "promo"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 517175}, "finish": "Holofoil", "id": "051_517175_holo", "name": "Snorlax", "number": "051", "rarity": "Promo", "setCode": "SVP", "type": "Colorless"},
  {"externalLinks": {"tcgPlayerId": 517270}, "finish": "Holofoil", "id": "051_517270_holo", "name": "Snorlax", "number": "051", "promoTypes": ["pokemon center exclusive"], "rarity": "Promo", "setCode": "SVP", "type": "Colorless", "variant": "Pokemon Center Exclusive"}
 ]
}`

// TestPreprocessPokemonQualifier pins that the wording behind the number
// reaches the printing it names. The storefront sells these two at $15.31
// and $251.59, and the number they share is all the scraper used to read.
func TestPreprocessPokemonQualifier(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(snorlaxDatastore))
	if err != nil {
		t.Fatal(err)
	}
	// The rest of the package reads the Magic datastore TestMain loaded, and
	// the global one is what a probe asks.
	t.Cleanup(func() {
		err := mtgmatcher.LoadDatastoreFile(os.Getenv("ALLPRINTINGS5_PATH"))
		if err != nil {
			log.Fatalln(err)
		}
	})

	tests := []struct {
		displayName string
		uuid        string
	}{
		{"Snorlax - 051 (Pokemon Center Exclusive)  - SV Scarlet  Violet Promo Cards Holofoil", "051_517270_holo"},
		{"Snorlax - 051  - SV Scarlet  Violet Promo Cards Holofoil", "051_517175_holo"},
		// Wording the catalog cannot place falls back on the number it was
		// read from, so it costs the listing nothing.
		{"Snorlax - 051 (Sealed In A Jar)  - SV Scarlet  Violet Promo Cards Holofoil", "051_517175_holo"},
	}
	for _, tt := range tests {
		product := GNProduct{
			DisplayName:    tt.displayName,
			SelectedFinish: "Holofoil",
			ProductData:    GNProductData{SetName: "SV: Scarlet & Violet Promo Cards"},
		}
		card, err := preprocess(product, GamePokemon)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		uuid, err := mtgmatcher.Match(card)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		if uuid != tt.uuid {
			t.Errorf("%q: got %q; want %q", tt.displayName, uuid, tt.uuid)
		}
	}
}
