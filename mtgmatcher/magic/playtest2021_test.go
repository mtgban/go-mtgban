package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// An edition naming the 2021 playtest set reaches CMB2, and the 2019 one
// keeps CMB1, for a card both sets print.
func TestPlaytestEditionPicksItsSet(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		edition, set string
	}{
		{"Mystery Booster Playtest Cards 2021", "CMB2"},
		{"Mystery Booster Playtest Cards 2019", "CMB1"},
	} {
		in := &mtgmatcher.InputCard{Name: "Rift", Edition: tc.edition, Variation: "119"}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.edition, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set || co.Number != "119" {
			t.Errorf("%q: landed in %s %s, want %s 119", tc.edition, co.SetCode, co.Number, tc.set)
		}
	}
}
