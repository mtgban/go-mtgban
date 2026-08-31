package cardmarket

import "testing"

// TestSealedShelfHoldsNoProduct pins which of the catalog's own headings are
// read as holding nothing to price. Each game writes its own name in front of
// the heading, so the match is on the tail and not the whole.
func TestSealedShelfHoldsNoProduct(t *testing.T) {
	for _, tt := range []struct {
		category string
		want     bool
	}{
		{"Pokémon Coins", true},
		{"Yugioh Event Tickets", true},
		{"Magic Event Tickets", true},
		// The lots hold the odd real product, so they are priced like
		// anything else and refused one at a time where they are not.
		{"One Piece Lots", false},
		{"Yugioh Lot", false},
		{"Pokémon Lot", false},
		// What the datastore is missing is not what the catalog is not
		// selling, and only the second is shelved.
		{"Riftbound Champion Decks", false},
		{"Pokémon Booster", false},
		{"Lorcana Sets", false},
		// The heading is the tail, not a word inside it.
		{"Coins of Elsewhere", false},
		{"Coins", false},
		{"", false},
	} {
		if _, got := sealedShelfHoldsNoProduct(tt.category); got != tt.want {
			t.Errorf("sealedShelfHoldsNoProduct(%q) = %v, want %v", tt.category, got, tt.want)
		}
	}
}
