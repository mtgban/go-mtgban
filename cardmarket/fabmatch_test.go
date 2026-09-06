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

// fabSpellingDatastore is the published datastore cut down to two cards
// whose sets spell the qualifier both ways, every row copied verbatim from
// it: Valiant Thrust, pitch-qualified in Monarch and bare in the Boltyn
// blitz deck; Rawhide Rumble, bare in the Rhinar armory deck and qualified
// in Heavy Hitters. Whichever spelling a name lookup lands on, the other
// set's printing is only reachable if the treatment tail came off - or, in
// the bare set, only if it did not.
const fabSpellingDatastore = `{
 "game": "fleshandblood",
 "sets": {"MON": {"name": "Monarch", "releaseDate": "2021-04-30"}, "BOL": {"name": "Blitz Deck: Monarch - Boltyn", "releaseDate": "2021-05-14"}, "ADR": {"name": "Armory Deck: Rhinar", "releaseDate": "2025-11-14"}, "HVY": {"name": "Heavy Hitters", "releaseDate": "2024-02-02"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 237741}, "fabId": "MON039", "finish": "1st Edition Normal", "id": "mon039_237741_1e", "name": "Valiant Thrust (Red)", "number": "MON039", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237741}, "fabId": "MON039", "finish": "1st Edition Rainbow Foil", "id": "mon039_237741_1erainbow", "name": "Valiant Thrust (Red)", "number": "MON039", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237741}, "fabId": "MON039", "finish": "Unlimited Edition Normal", "id": "mon039_237741_unl", "name": "Valiant Thrust (Red)", "number": "MON039", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237741}, "fabId": "MON039", "finish": "Unlimited Edition Rainbow Foil", "id": "mon039_237741_unlrainbow", "name": "Valiant Thrust (Red)", "number": "MON039", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 237742}, "fabId": "MON040", "finish": "1st Edition Normal", "id": "mon040_237742_1e", "name": "Valiant Thrust (Yellow)", "number": "MON040", "rarity": "Rare", "setCode": "MON"},
  {"externalLinks": {"tcgPlayerId": 238398}, "fabId": "BOL017", "finish": "Normal", "id": "bol017_238398", "name": "Valiant Thrust", "number": "BOL017", "rarity": "Rare", "setCode": "BOL"},
  {"externalLinks": {"tcgPlayerId": 533460}, "fabId": "HVY023", "finish": "Normal", "id": "hvy023_533460", "name": "Rawhide Rumble (Red)", "number": "HVY023", "rarity": "Rare", "setCode": "HVY"},
  {"externalLinks": {"tcgPlayerId": 533460}, "fabId": "HVY023", "finish": "Rainbow Foil", "id": "hvy023_533460_rainbow", "name": "Rawhide Rumble (Red)", "number": "HVY023", "rarity": "Rare", "setCode": "HVY"},
  {"externalLinks": {"tcgPlayerId": 533461}, "fabId": "HVY024", "finish": "Normal", "id": "hvy024_533461", "name": "Rawhide Rumble (Yellow)", "number": "HVY024", "rarity": "Rare", "setCode": "HVY"},
  {"externalLinks": {"tcgPlayerId": 663031}, "finish": "Normal", "id": "arr012_663031", "name": "Rawhide Rumble", "number": "ARR012", "rarity": "Rare", "setCode": "ADR"}
 ]
}`

// TestMatchProductTreatmentTail pins how the name fallback sees through the
// treatment parenthetical Cardmarket decorates every Flesh and Blood product
// with. The tail has to come off for the sets that spell the card's pitch
// qualifier: "Valiant Thrust (Red) (Regular)" names no card, and the name
// left when the parenthetical is split away, "Valiant Thrust", is the Boltyn
// deck's own card, unreachable from Monarch. And the raw name has to stay
// the fallback for the sets that do not: Rhinar's deck files "Rawhide
// Rumble" bare, so its product's stripped name is Heavy Hitters' card and
// only the decorated one still splits down to the printing.
func TestMatchProductTreatmentTail(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabSpellingDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GameFleshAndBlood}
	for _, tt := range []struct {
		expansion, name, number, want string
	}{
		{"Monarch - First", "Valiant Thrust (Red) (Regular)", "MON039", "mon039_237741_1e"},
		{"Monarch - First", "Valiant Thrust (Red) (Rainbow Foil)", "MON039", "mon039_237741_1erainbow"},
		{"Monarch - Unlimited", "Valiant Thrust (Red) (Regular)", "MON039", "mon039_237741_unl"},
		{"Heavy Hitters", "Rawhide Rumble (Red) (Regular)", "023", "hvy023_533460"},
		{"Heavy Hitters", "Rawhide Rumble (Red) (Rainbow Foil)", "023", "hvy023_533460_rainbow"},
		{"Armory Deck: Rhinar", "Rawhide Rumble (Red) (Regular)", "012", "arr012_663031"},
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

// TestProcessProductByName pins which prices carry the mark that holds them
// back. A product the bridge knows resolves through an id and is not a
// guess; one the bridge has never heard of resolves through its name, and
// every price it produces has to say so, or namedLast has nothing to sort by.
func TestProcessProductByName(t *testing.T) {
	loadFabDatastore(t)

	// Cardmarket 602755, the first-edition Monarch printing of Prismatic
	// Shield (Red), which the datastore keys by TCGplayer id 237847.
	product := MKMProduct{
		IDProduct:     602755,
		Name:          "Prismatic Shield (Red) (Regular)",
		Number:        "MON092",
		ExpansionName: "Monarch - First",
	}

	for _, tt := range []struct {
		name   string
		bridge map[int]int
		want   bool
	}{
		{"a product the bridge knows resolves through its id", map[int]int{602755: 237847}, false},
		{"a product the bridge misses resolves through its name", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mkm := &Index{
				gameID:       GameFleshAndBlood,
				exchangeRate: 1,
				TCGBridge:    tt.bridge,
				priceGuide: map[int]PriceGuide{
					602755: {IDProduct: 602755, LowPrice: 9, TrendPrice: 10},
				},
			}

			channel := make(chan responseChan, len(availableIndexNames))
			err := mkm.processProduct(channel, &product)
			if err != nil {
				t.Fatal(err)
			}
			close(channel)

			var count int
			for result := range channel {
				count++
				if result.cardID != "mon092_237847_1e" {
					t.Errorf("priced %q, want mon092_237847_1e", result.cardID)
				}
				if result.byName != tt.want {
					t.Errorf("byName = %v, want %v", result.byName, tt.want)
				}
			}
			if count != len(availableIndexNames) {
				t.Errorf("got %d prices, want %d", count, len(availableIndexNames))
			}
		})
	}
}
