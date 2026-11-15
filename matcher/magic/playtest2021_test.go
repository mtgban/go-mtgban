package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
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
		in := &matcher.InputCard{Name: "Rift", Edition: tc.edition, Variation: "119"}
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

// A Mystery Booster/The List listing names the List reprint's own number and
// lands on it, while Mystery wording with no number keeps the playtest card.
func TestPlaytestMysteryListNumber(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		variation, set, number string
	}{
		{"MKM-170", "PLST", "MKM-170"},
		{"", "CMB1", "97"},
		{"98", "CMB1", "97"},
		{"97-f", "CMB1", "97"},
		{"cmb1-97", "CMB1", "97"},
	} {
		in := &matcher.InputCard{Name: "Pick Your Poison", Edition: "Mystery Booster/The List", Variation: tc.variation}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.variation, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set || co.Number != tc.number {
			t.Errorf("%q: landed in %s %s, want %s %s", tc.variation, co.SetCode, co.Number, tc.set, tc.number)
		}
	}
}
