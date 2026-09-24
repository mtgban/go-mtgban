package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// namesSourceSet must also match a colon-subtitled set's short form and
// the singular "Duel Deck" against the datastore's plural "Duel Decks".
func TestPLSTVariationNamesSourceSet(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		name      string
		variation string
		want      string
	}{
		// Storm-Kiln Artist: OTC-181 (Outlaws of Thunder Junction
		// Commander) vs STX-115 (Strixhaven: School of Mages).
		{"Storm-Kiln Artist", "Strixhaven", "d4a3a951-a4b6-57e5-89cd-5510c316c40e"},
		// Dark Ritual: A25-82 (Masters 25) vs DDE-18 (Duel Decks:
		// Phyrexia vs. the Coalition), named in the singular.
		{"Dark Ritual", "Duel Deck: Phyrexia vs The Coalition", "b602d317-e08f-52b4-85ea-1e9ad046caef"},
		// Goblin Bombardment: DDN-24 (Duel Decks: Speed vs. Cunning) vs
		// MH2-279 (Modern Horizons 2).
		{"Goblin Bombardment", "Duel Deck: Speed vs Cunning", "fdd675d2-f65a-5791-add1-00e0078286bf"},
	} {
		in := &mtgmatcher.InputCard{Name: tc.name, Edition: "Mystery Booster", Variation: tc.variation}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%s %q: unexpected error: %v", tc.name, tc.variation, err)
			continue
		}
		if id != tc.want {
			t.Errorf("%s %q: id = %q, want %q", tc.name, tc.variation, id, tc.want)
		}
	}

	// "Dark Ascension" must keep naming DKA-127 alone, and not also spare
	// PDKA-127 (Dark Ascension Promos) just because its own name starts
	// with the same words - a bare prefix is a different, related set,
	// not a short form of it.
	in := &mtgmatcher.InputCard{Name: "Strangleroot Geist", Edition: "Mystery Booster/The List", Variation: "Dark Ascension"}
	id, err := b.Match(in)
	if err != nil {
		t.Errorf("Strangleroot Geist: unexpected error: %v", err)
	} else if want := "f3f73551-7892-5527-a2d6-0534974eecde"; id != want {
		t.Errorf("Strangleroot Geist: id = %q, want %q", id, want)
	}
}
