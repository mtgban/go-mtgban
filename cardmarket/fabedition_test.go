package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// TestFabEdition pins the translation from Cardmarket's expansion names to
// ours. Every shape here is one the production log showed resolving to
// nothing: the promo programmes the datastore keeps as number prefixes in a
// single set, the decks both catalogs name with the same words in a different
// order, and the History packs Cardmarket spells "History" where the datastore
// spells "Historic".
func TestFabEdition(t *testing.T) {
	for _, tt := range []struct{ expansion, setName, prefix string }{
		// The seven promo programmes, which are one set of ours.
		{"FAB Promos", "Flesh and Blood: Promo Cards", "FAB"},
		{"Hero Promos", "Flesh and Blood: Promo Cards", "HER"},
		{"Judge Promos", "Flesh and Blood: Promo Cards", "JDG"},
		{"LGS Promos", "Flesh and Blood: Promo Cards", "LGS"},
		{"Tournament Pack", "Flesh and Blood: Promo Cards", "TNP"},
		// A blitz deck keeps the set it was sold with; a hero deck is named
		// for its hero alone, and the epithet Cardmarket writes beside it
		// belongs to the card rather than to the deck.
		{"Monarch - Chane Blitz Deck", "Blitz Deck: Monarch - Chane", ""},
		{"Heavy Hitters - Victor Blitz Deck", "Blitz Deck: Heavy Hitters - Victor", ""},
		{"Welcome to Rathe - Bravo, Showstopper Hero Deck", "Hero Deck: Bravo", ""},
		{"Welcome to Rathe - Dorinthea Ironsong Hero Deck", "Hero Deck: Dorinthea", ""},
		{"History Pack 1 - Dash Blitz Deck", "Historic Pack 1 Blitz Deck: Dash", ""},
		{"Archive Mastery Pack - Guardian", "Mastery Pack Guardian", ""},
		{"Armory Deck Origins: Jarl Vetreidi", "Armory Deck: Jarl Vetreidi", ""},
		{"Armory Deck Legends: Prism, Sculptor of Arc Light", "Armory Deck: Legends Prism", ""},
		{"Ira Welcome Deck", "Welcome Deck: Ira", ""},
		// An expansion named nothing in particular is handed back as it
		// came, which is what keeps the rewrite from reaching a set that
		// answered for itself.
		{"Monarch", "Monarch", ""},
		{"Welcome to Rathe", "Welcome to Rathe", ""},
	} {
		setName, prefix := fabEdition(tt.expansion)
		if setName != tt.setName || prefix != tt.prefix {
			t.Errorf("fabEdition(%q) = %q, %q; want %q, %q",
				tt.expansion, setName, prefix, tt.setName, tt.prefix)
		}
	}
}

// fabPromoDatastore carries the two promo rows the test below turns on, copied
// from the published datastore: one numbered by the FAB programme and one by
// HER, which is the whole reason the programme has to be put back before the
// number means anything.
const fabPromoDatastore = `{
 "game": "fleshandblood",
 "sets": {"PR": {"name": "Flesh and Blood: Promo Cards", "releaseDate": "2019-10-11"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 604800}, "finish": "Cold Foil", "id": "fab012_cold", "name": "Energy Potion", "number": "FAB012", "rarity": "Promo", "setCode": "PR"},
  {"externalLinks": {"tcgPlayerId": 604846}, "finish": "Cold Foil", "id": "her009_cold", "name": "Dash, Inventor Extraordinaire", "number": "HER009", "rarity": "Promo", "setCode": "PR"}
 ]
}`

// TestFabPromoProduct walks a promo product the whole way, because the
// translation is only worth anything if the number it hands back reaches the
// printing: both of these are "012" and "009" to Cardmarket, and the
// programme it sells them under is the only thing telling them apart.
func TestFabPromoProduct(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabPromoDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GameFleshAndBlood}
	for _, tt := range []struct{ name, number, expansion, want string }{
		{"Energy Potion (Cold Foil)", "012", "FAB Promos", "fab012_cold"},
		{"Dash, Inventor Extraordinaire (Cold Foil)", "009", "Hero Promos", "her009_cold"},
		// The same number under the programme that does not issue it stays
		// unresolved rather than landing on the other one's card.
		{"Energy Potion (Cold Foil)", "009", "FAB Promos", ""},
	} {
		product := MKMProduct{
			Name:          tt.name,
			Number:        tt.number,
			ExpansionName: tt.expansion,
		}
		if got := mkm.matchProduct(&product); got != tt.want {
			t.Errorf("matchProduct(%q in %s, %s) = %q, want %q",
				tt.name, tt.expansion, tt.number, got, tt.want)
		}
	}
}
