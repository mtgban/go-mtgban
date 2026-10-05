package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A Secret Lair Drop number the SLD set holds names that printing, even when
// the card also has a plain SLX printing.
func TestSLDNumberBeatsSLX(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, tc := range []struct {
		name, number, set string
	}{
		{"Cecily, Haunted Mage", "343", "SLD"},
		{"Themberchaud", "728", "SLD"},
		{"Cecily, Haunted Mage", "", "SLX"},
	} {
		in := &mtgmatcher.InputCard{Name: tc.name, Edition: "Secret Lair Drop", Variation: tc.number}
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
	}
}
