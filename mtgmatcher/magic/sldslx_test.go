package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A Secret Lair Drop number the SLD set holds names that printing, even when
// the card also has a plain SLX printing, and a Japanese foil reaches its own
// foil row.
func TestSLDNumberBeatsSLX(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		name, number, set, want string
		foil                    bool
	}{
		{"Cecily, Haunted Mage", "343", "SLD", "", false},
		{"Themberchaud", "728", "SLD", "", false},
		{"Cecily, Haunted Mage", "", "SLX", "", false},
		{"Chord of Calling", "1595 Japanese", "SLD", "1595★jpn", true},
		{"Chord of Calling", "1595 Japanese", "SLD", "1595jpn", false},
	} {
		in := &mtgmatcher.InputCard{Name: tc.name, Edition: "Secret Lair Drop", Variation: tc.number, Foil: tc.foil}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%s %q: unexpected error: %v", tc.name, tc.number, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set {
			t.Errorf("%s %q: landed in %s %s, want %s", tc.name, tc.number, co.SetCode, co.Number, tc.set)
		}
		if tc.want != "" && co.Number != tc.want {
			t.Errorf("%s %q: landed on %s, want %s", tc.name, tc.number, co.Number, tc.want)
		}
	}
}
