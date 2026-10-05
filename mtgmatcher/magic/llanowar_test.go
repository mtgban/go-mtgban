package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Llanowar Elves reaches Dominaria Promos only by a generic promo wording;
// a Gateway or MagicFest promo keeps its own set.
func TestLlanowarElvesGenericPromo(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		variation, set string
	}{
		{"Open House Promo", "PDOM"},
		{"DCI Gateway Promo", "DCI"},
		{"Future Sight Frame - MagicCon Amsterdam Festival in a Box Promo", "PF26"},
	} {
		in := &mtgmatcher.InputCard{Name: "Llanowar Elves", Edition: "Promo", Variation: tc.variation, Foil: true}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.variation, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set {
			t.Errorf("%q: landed in %s %s, want %s", tc.variation, co.SetCode, co.Number, tc.set)
		}
	}
}
