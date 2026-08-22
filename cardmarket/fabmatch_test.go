package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// fabDatastore is the published Flesh and Blood datastore cut down to the
// printings these tests turn on, every row copied verbatim from it:
// Prismatic Shield (Red), which Monarch sold in both of its print runs;
// Prism // Ravenous Meataxe, which Cardmarket sells in both runs and the
// datastore carries in the unlimited one alone; and Sink Below (Red), for
// the run Welcome to Rathe spells Alpha. Nine rows rather than the whole
// file keeps the answers below facts about data the test states.
const fabDatastore = `{
 "game": "fleshandblood",
 "sets": {"MON": {"name": "Monarch", "releaseDate": "2021-04-30"}, "WTR": {"name": "Welcome to Rathe", "releaseDate": "2019-10-11"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 237847}, "fabId": "MON092", "finish": "1st Edition Normal", "id": "mon092_237847_1e", "name": "Prismatic Shield (Red)", "number": "MON092", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237847}, "fabId": "MON092", "finish": "1st Edition Rainbow Foil", "id": "mon092_237847_1erainbow", "name": "Prismatic Shield (Red)", "number": "MON092", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237847}, "fabId": "MON092", "finish": "Unlimited Edition Normal", "id": "mon092_237847_unl", "name": "Prismatic Shield (Red)", "number": "MON092", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237847}, "fabId": "MON092", "finish": "Unlimited Edition Rainbow Foil", "id": "mon092_237847_unlrainbow", "name": "Prismatic Shield (Red)", "number": "MON092", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 251042}, "fabId": "MON002", "finish": "Unlimited Edition Normal", "id": "mon002-mon221_251042_unl", "name": "Prism // Ravenous Meataxe", "number": "MON002//MON221", "rarity": "Token", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 225309}, "fabId": "WTR215", "finish": "1st Edition Normal", "id": "wtr215_225309_1e", "name": "Sink Below (Red)", "number": "WTR215", "rarity": "Common", "setCode": "WTR"},
  {"externalLinks": {"tcgPlayerId": 225309}, "fabId": "WTR215", "finish": "1st Edition Rainbow Foil", "id": "wtr215_225309_1erainbow", "name": "Sink Below (Red)", "number": "WTR215", "rarity": "Common", "setCode": "WTR"},
  {"externalLinks": {"tcgPlayerId": 225309}, "fabId": "WTR215", "finish": "Unlimited Edition Normal", "id": "wtr215_225309_unl", "name": "Sink Below (Red)", "number": "WTR215", "rarity": "Common", "setCode": "WTR"},
  {"externalLinks": {"tcgPlayerId": 225309}, "fabId": "WTR215", "finish": "Unlimited Edition Rainbow Foil", "id": "wtr215_225309_unlrainbow", "name": "Sink Below (Red)", "number": "WTR215", "rarity": "Common", "setCode": "WTR"}
 ]
}`

// loadFabDatastore installs the cut-down datastore as the global backend the
// matcher answers from.
func loadFabDatastore(t *testing.T) {
	t.Helper()
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabDatastore))
	if err != nil {
		t.Fatal(err)
	}
}

// TestMatchProductPrintRun pins what the name fallback answers for a
// Cardmarket expansion that names a print run. Two things have to hold at
// once: the set has to be looked up under the name left when the run suffix
// comes off, since no set of ours is called "Monarch - First"; and the
// printing that comes back has to be in the run the expansion named, since
// Match answers a card the datastore keeps in one run only with that run
// whichever was asked for - and the other run's expansion sells the very
// same card.
func TestMatchProductPrintRun(t *testing.T) {
	loadFabDatastore(t)

	mkm := &Index{gameID: GameFleshAndBlood}
	for _, tt := range []struct {
		expansion, name, number, want string
	}{
		// The run named by the expansion and the treatment named by the
		// product cross into one printing.
		{"Monarch - First", "Prismatic Shield (Red) (Regular)", "MON092", "mon092_237847_1e"},
		{"Monarch - First", "Prismatic Shield (Red) (Rainbow Foil)", "MON092", "mon092_237847_1erainbow"},
		{"Monarch - Unlimited", "Prismatic Shield (Red) (Regular)", "MON092", "mon092_237847_unl"},
		{"Monarch - Unlimited", "Prismatic Shield (Red) (Rainbow Foil)", "MON092", "mon092_237847_unlrainbow"},
		// Welcome to Rathe's first run is the one Cardmarket calls Alpha.
		{"Welcome to Rathe - Alpha", "Sink Below (Red) (Regular)", "WTR215-C", "wtr215_225309_1e"},
		{"Welcome to Rathe - Unlimited", "Sink Below (Red) (Regular)", "WTR215-C", "wtr215_225309_unl"},
		// The datastore has no first edition of this pair, so the product
		// the first-edition expansion sells resolves to nothing rather
		// than to the unlimited printing its own expansion prices.
		{"Monarch - First", "Prism // Ravenous Meataxe (Regular)", "002/221", ""},
		{"Monarch - Unlimited", "Prism // Ravenous Meataxe (Regular)", "002/221", "mon002-mon221_251042_unl"},
		// An expansion naming no set of ours resolves to nothing, run
		// suffix or not.
		{"Monarch - Boltyn Blitz Deck", "Prismatic Shield (Red) (Regular)", "MON092", ""},
	} {
		product := MKMProduct{
			Name:          tt.name,
			Number:        tt.number,
			ExpansionName: tt.expansion,
		}
		got := mkm.matchProduct(&product)
		if got != tt.want {
			t.Errorf("matchProduct(%q, %q) = %q, want %q", tt.expansion, tt.name, got, tt.want)
		}
	}
}
