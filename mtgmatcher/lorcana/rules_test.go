package lorcana

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestIsUnsupportedShadowsDefault pins that the game's own answer wins over
// the embedded default: DefaultRules says nothing is unsupported, and
// Lorcana has to keep saying its puzzle inserts and lots are.
func TestIsUnsupportedShadowsDefault(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Puzzle Insert - Piece 1", true},
		{"Disney Cruise Promos (Set of 5)", true},
		{"Mickey Mouse - True Friend", false},
	} {
		got := Rules{}.IsUnsupported(nil, &mtgmatcher.InputCard{Name: tt.name})
		if got != tt.want {
			t.Errorf("IsUnsupported(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
