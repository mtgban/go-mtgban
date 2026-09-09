package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNamedID pins which of a blueprint's two ids answers for it. The ids are
// the surest thing Card Trader publishes and the scryfall one is preferred,
// but they are not checked against each other on the way in: eight Strixhaven
// promos each carry the scryfall id of the card before them on the shelf, so
// "Exponential Growth" priced as Ecological Appreciation for as long as the
// listing stood.
func TestNamedID(t *testing.T) {
	realDatastore(t)
	uuid := func(space, id string) string {
		out := mtgmatcher.ConvertID(space, id)
		if out == "" {
			t.Fatalf("%s id %q resolves to nothing", space, id)
		}
		return out
	}
	// The shifted blueprint: its scryfall id is Ecological Appreciation's and
	// its TCGplayer id is its own.
	shifted := uuid(mtgmatcher.IDSpaceScryfall, "357ce522-4872-4386-b409-264e23953a53")
	itsOwn := uuid(mtgmatcher.IDSpaceTCGplayer, "237134")
	// A blueprint whose ids agree on the card, which must not move.
	plainScryfall := uuid(mtgmatcher.IDSpaceScryfall, "e939194a-6993-412c-bbd8-540b9872a721")
	plainTCGplayer := uuid(mtgmatcher.IDSpaceTCGplayer, "237341")

	for _, tt := range []struct {
		desc                    string
		scryfallID, tcgplayerID string
		cardName                string
		want                    string
	}{
		{"neither id says anything", "", "", "Exponential Growth", ""},
		{"only the scryfall id", shifted, "", "Exponential Growth", shifted},
		{"only the TCGplayer id", "", itsOwn, "Exponential Growth", itsOwn},
		{
			// Both name Verdant Mastery, so the order stands and the
			// preferred id answers as it always has.
			desc:       "ids agreeing on the card leave the order alone",
			scryfallID: plainScryfall, tcgplayerID: plainTCGplayer,
			cardName: "Verdant Mastery", want: plainScryfall,
		},
		{
			// The wording is the third thing the blueprint says, and it is
			// what breaks a tie the ids cannot.
			desc:       "the name picks the id that names its card",
			scryfallID: shifted, tcgplayerID: itsOwn,
			cardName: "Exponential Growth", want: itsOwn,
		},
		{
			desc:       "and leaves the preferred id where the name agrees with it",
			scryfallID: shifted, tcgplayerID: itsOwn,
			cardName: "Ecological Appreciation", want: shifted,
		},
		{
			// A pair of tokens sold under one blueprint names neither, and
			// guessing between them would be worse than keeping the order.
			desc:       "a name settling nothing changes nothing",
			scryfallID: shifted, tcgplayerID: itsOwn,
			cardName: "Soldier // Spirit", want: shifted,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := namedID(tt.scryfallID, tt.tcgplayerID, tt.cardName); got != tt.want {
				t.Errorf("namedID = %q, want %q", got, tt.want)
			}
		})
	}
}
