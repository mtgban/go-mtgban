package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPlainNumberKeepsTheListNumbers pins the numbers The List is filed
// under. It names a card by the set it was drawn from, "ARB-1", and those end
// in a digit, so the letters trimmed off a number's tail never reach them:
// 5,582 of its 5,584 numbers keep the plain form they had. The two this does
// reach are here as well, so what the change touches is written down rather
// than assumed.
func TestPlainNumberKeepsTheListNumbers(t *testing.T) {
	for _, tt := range []struct {
		number, want string
	}{
		{"ARB-1", "ARB-1"},
		{"BBD-1", "BBD-1"},
		{"CSP-1", "CSP-1"},
		{"MID-123", "MID-123"},
		{"M19-185", "M19-185"},
		{"POR-57", "POR-57"},
		// The two The List numbers a tail reaches, each a treatment of the
		// number standing beside it rather than a number of its own.
		{"POR-57s", "POR-57"},
		{"M19-185j", "M19-185"},
		// And a mark, which the older rule already reached.
		{"JUD-78†", "JUD-78"},
	} {
		t.Run(tt.number, func(t *testing.T) {
			got := plainNumber(tt.number)
			if got != tt.want {
				t.Errorf("plainNumber(%q) = %q, want %q", tt.number, got, tt.want)
			}
		})
	}
}

// TestPlainNumberMatchesLoader pins the rules to the loader. PlainNumber is
// what folds a number a person typed, and the card carries what it is
// compared against, so the two spelling a number differently finds nothing
// and raises nothing - the failure a caller reads as "no such card".
func TestPlainNumberMatchesLoader(t *testing.T) {
	realDatastore(t)
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
