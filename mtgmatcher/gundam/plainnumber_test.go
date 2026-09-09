package gundam

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPlainNumberMatchesLoader pins the rules to the loader. PlainNumber is
// what folds a number a person typed, and the card carries what it is
// compared against, so the two spelling a number differently finds nothing
// and raises nothing - the failure a caller reads as "no such card".
func TestPlainNumberMatchesLoader(t *testing.T) {
	loadBackend(t)

	var seen int
	for _, code := range mtgmatcher.GetAllSets() {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			plain := Rules{}.PlainNumber(card.Number)
			if plain != card.PlainNumber {
				t.Errorf("%s %q: the rule folds to %q, the card carries %q",
					code, card.Number, plain, card.PlainNumber)
			}
			// Folding a number already plain has nothing left to do.
			again := Rules{}.PlainNumber(plain)
			if again != plain {
				t.Errorf("%s %q: folding %q again gives %q", code, card.Number, plain, again)
			}
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no cards to check")
	}
}
