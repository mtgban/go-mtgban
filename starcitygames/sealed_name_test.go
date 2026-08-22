package starcitygames

import "testing"

// TestSealedProductName pins the three ways Star City Games spells a sealed
// product differently from the datastore. The name-resolution rule needs
// every word of the datastore's name accounted for by the vendor's, so a
// word only one side writes loses the product outright.
func TestSealedProductName(t *testing.T) {
	for _, tt := range []struct{ game, in, want string }{
		{"Flesh and Blood",
			"Flesh and Blood - Welcome to Rathe (Unlimited) Booster Box",
			"Welcome to Rathe Booster Box [Unlimited Edition]"},
		{"Flesh and Blood",
			"Flesh and Blood - Monarch (1st Edition) Booster Case",
			"Monarch Booster Box Case [1st Edition]"},
		{"Flesh and Blood",
			"Flesh and Blood - Uprising Booster Case",
			"Uprising Booster Box Case"},
		// A game prefix that is not this product's is left alone, and a
		// product naming no run keeps its name as it is.
		{"Lorcana", "Lorcana - Rise of the Floodborn Booster Box",
			"Rise of the Floodborn Booster Box"},
		{"Flesh and Blood", "Flesh and Blood - Dynasty Booster Box",
			"Dynasty Booster Box"},
		// The dash inside a product's own name is not the game prefix.
		{"Flesh and Blood",
			"Flesh and Blood - Hero Decks: Welcome to Rathe - Bravo",
			"Hero Decks: Welcome to Rathe - Bravo"},
	} {
		got := sealedProductName(CatalogProduct{Name: tt.in, Game: tt.game})
		if got != tt.want {
			t.Errorf("sealedProductName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
