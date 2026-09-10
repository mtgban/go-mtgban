package onepiece

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPlainNumber pins the shorthand this game's numbers reduce to. The whole
// number stays on Card.Number and is what names a printing; PlainNumber is
// the ordinal a person types, which is also what a storefront publishes.
func TestPlainNumber(t *testing.T) {
	for _, tt := range []struct {
		number, want string
	}{
		{"OP01-007", "7"},
		{"OP01-120", "120"},
		{"EB01-001", "1"},
		{"ST01-100", "100"},
		{"P-001", "1"},
		{"OP09-051", "51"},
		// DON is a word, not an ordinal, so it reduces to no ordinal.
		{"DON", ""},
		{"", ""},
	} {
		got := Rules{}.PlainNumber(tt.number)
		if got != tt.want {
			t.Errorf("PlainNumber(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}

// TestPlainNumberMatchesLoader pins the rule to what the loader stored, and
// that an ordinal already reduced has nothing left to fold.
func TestPlainNumberMatchesLoader(t *testing.T) {
	loadBackend(t)

	var seen, folded int
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
	// The reduction is the whole point of the rule; a datastore that started
	// publishing bare ordinals would leave this passing while testing nothing.
	if folded == 0 {
		t.Errorf("no number of %d reduced to anything shorter", seen)
	}
}
