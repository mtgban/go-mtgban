package magic

import (
	"testing"
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

func TestPlainNumberDropsNumericPadding(t *testing.T) {
	for _, tt := range []struct {
		number, want string
	}{
		{"071", "71"},
		{"001", "1"},
		{"000", "0"},
		{"071★", "71"},
		{"071a", "71"},
		{"M19-001", "M19-001"},
		{"ARB-1", "ARB-1"},
	} {
		t.Run(tt.number, func(t *testing.T) {
			got := plainNumber(tt.number)
			if got != tt.want {
				t.Errorf("plainNumber(%q) = %q, want %q", tt.number, got, tt.want)
			}
		})
	}
}

// TestPlainNumberMatchesLoader pins the rule to what the loader stored on
// each card. The two spelling a number differently finds nothing and raises
// nothing, which a caller reads as "no such card".
func TestPlainNumberMatchesLoader(t *testing.T) {
	realDatastore(t)
	var seen, folded int
	for _, code := range testBackend.GetAllSets() {
		set, err := testBackend.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			plain := Rules{}.PlainNumber(card.Number)
			if plain != card.PlainNumber {
				t.Errorf("%s %q: the rule folds to %q, the card carries %q",
					code, card.Number, plain, card.PlainNumber)
			}
			if len(plain) > len(card.Number) {
				t.Errorf("%s: PlainNumber %q is wider than Number %q",
					code, plain, card.Number)
			}
			again := Rules{}.PlainNumber(plain)
			if again != plain {
				t.Errorf("%s %q: folding %q again gives %q",
					code, card.Number, plain, again)
			}
			if plain != card.Number {
				folded++
			}
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no cards to check")
	}
	// A datastore publishing only bare ordinals would leave the checks above
	// passing while testing nothing.
	if folded == 0 {
		t.Errorf("no number of %d reduced to anything shorter", seen)
	}
}
