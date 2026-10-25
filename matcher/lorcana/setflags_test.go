package lorcana

import (
	"strings"
	"testing"
)

// TestSetFinishFlags pins that a set's finish flags say what its cards are
// sold in: a set with a card sold plain is not foil-only, and one with a
// card sold foil is not nonfoil-only.
func TestSetFinishFlags(t *testing.T) {
	b, err := Load(strings.NewReader(finishNamesData))
	if err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]struct{ foilOnly, nonfoilOnly bool }{
		"1":  {false, false},
		"P1": {true, false},
	} {
		set := b.Sets[code]
		if set.IsFoilOnly != want.foilOnly || set.IsNonFoilOnly != want.nonfoilOnly {
			t.Errorf("set %s: foil-only %v, nonfoil-only %v; want %v, %v", code, set.IsFoilOnly, set.IsNonFoilOnly, want.foilOnly, want.nonfoilOnly)
		}
	}
}
