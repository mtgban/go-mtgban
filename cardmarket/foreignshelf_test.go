package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// TestForeignShelf pins which expansion names are a catalog of their own.
// Cardmarket shelves a set's non-English printings beside the English ones
// under the same card names, and the datastore carries only the English, so a
// price from one of those shelves lands on a printing it is not.
func TestForeignShelf(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Metal Raiders (Japanese)", true},
		{"Metal Raiders (Korean)", true},
		{"Metal Raiders (PMT)", true},
		{"Metal Raiders", false},
		{"Metal Raiders (25th Anniversary Edition)", false},
		{"Legend of Blue Eyes White Dragon (25th Anniversary Edition)", false},
		// The tail has to end the name: a set spelling one of those words
		// somewhere else is still the English catalog.
		{"Japanese Collection Tin", false},
		{"", false},
	} {
		if got := foreignShelf(tt.name); got != tt.want {
			t.Errorf("foreignShelf(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestForeignExpansion pins which expansions each game drops before the
// walk: Flesh and Blood by its own five foreign-only programs, One Piece and
// Yu-Gi-Oh by the shared -JP suffix and shelf-name test.
func TestForeignExpansion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		gameID  cm.Game
		exp     cm.Expansion
		foreign bool
	}{
		{"FaB Black Label history pack", cm.GameFleshAndBlood, cm.Expansion{Name: "History Pack 2 - Black Label", SetCode: "2HP-BL"}, true},
		{"FaB English set", cm.GameFleshAndBlood, cm.Expansion{Name: "Welcome to Rathe", SetCode: "WTR"}, false},
		{"One Piece Japanese shelf", cm.GameOnePiece, cm.Expansion{Name: "Romance Dawn (Japanese)", SetCode: "OP01"}, true},
		{"One Piece -JP code", cm.GameOnePiece, cm.Expansion{Name: "Romance Dawn", SetCode: "OP01-JP"}, true},
		{"One Piece English set", cm.GameOnePiece, cm.Expansion{Name: "Romance Dawn", SetCode: "OP01"}, false},
		{"Pokemon is never filtered", cm.GamePokemon, cm.Expansion{Name: "Base Set (Japanese)", SetCode: "BS"}, false},
	} {
		if got := foreignExpansion(tt.gameID, tt.exp); got != tt.foreign {
			t.Errorf("%s: foreignExpansion(%v, %v) = %v, want %v", tt.name, tt.gameID, tt.exp, got, tt.foreign)
		}
	}
}
