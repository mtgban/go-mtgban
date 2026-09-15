package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A World Championship listing names the set the deck's card was printed
// from before its collector number, and ExtractNumber reads its argument and
// nothing else, so "7ED" is the first thing in this wording that looks like
// a number. Which codes exist is a question for the backend the callback
// already holds, so the drop happens there.
func TestWorldChampSetCodeIsNotTheNumber(t *testing.T) {
	sets := map[string]*mtgmatcher.Set{
		"7ED":  {Name: "Seventh Edition"},
		"WC01": {Name: "World Championship Decks 2001"},
	}
	b := &mtgmatcher.Backend{Sets: sets}
	in := &mtgmatcher.InputCard{
		Name:      "Mountain",
		Edition:   "World Championship",
		Variation: "2001 Tom van de Logt 7ED 337",
	}

	// Tom van de Logt's Mountain is the one the listing names; Jan
	// Tomcani's is what aliasing settled on once that one was refused.
	named := &mtgmatcher.Card{Name: "Mountain", SetCode: "WC01", Number: "tvdl337"}
	other := &mtgmatcher.Card{Name: "Mountain", SetCode: "WC01", Number: "jt337"}
	if wcdNumberCompare(b, in, named) {
		t.Errorf("wcdNumberCompare refused %s, the printing the listing names", named.Number)
	}
	if !wcdNumberCompare(b, in, other) {
		t.Errorf("wcdNumberCompare admitted %s, another player's Mountain", other.Number)
	}

	// The drop reads the backend it is handed, not a global: one that does
	// not carry 7ED leaves the code in the wording, and the number read is
	// the code again.
	partial := &mtgmatcher.Backend{Sets: map[string]*mtgmatcher.Set{"WC01": sets["WC01"]}}
	if !wcdNumberCompare(partial, in, named) {
		t.Errorf("wcdNumberCompare read %s from a backend that does not carry 7ED", named.Number)
	}

	// ExtractNumber stays pure, and this is what that costs.
	if got := mtgmatcher.ExtractNumber(in.Variation); got != "7ed" {
		t.Errorf("ExtractNumber(%q) = %q, want %q", in.Variation, got, "7ed")
	}
}
