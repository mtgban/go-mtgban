package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestIsToken pins that the names only Magic knows as tokens still answer
// through the rules hook. The datastore's own token list is checked before
// this is asked, so what is pinned here is the half that list does not hold:
// the rules tips, the checklists, the storefront spellings.
func TestIsToken(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Rules Tip Card", true},
		{"Emblem of the Warmind", false},
		{"Tip: Something", true},
		{"Build a Deck: Something", true},
		{"Blank", true},
		{"On An Adventure", true},
		{"Kavu Monarch", false},
		{"Our Market Research", false},
		{"Lightning Bolt", false},
	} {
		if got := (Rules{}).IsToken(nil, tt.name); got != tt.want {
			t.Errorf("IsToken(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
	// And the same answers through the loaded backend, which is how Match
	// asks: the hook is only reached once the rules are attached.
	if !mtgmatcher.IsToken("Rules Tip Card") {
		t.Error("the backend does not reach the game's own token names")
	}
	if mtgmatcher.IsToken("Lightning Bolt") {
		t.Error("a real card reads as a token")
	}
	var _ mtgmatcher.GameRules = Rules{}
}
