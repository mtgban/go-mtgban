package starcitygames

import "testing"

// TestGameFromCatalog pins the catalog's own spelling of each game. The
// mapping is the only thing standing between a product and the scraper that
// wants it, and an unrecognized name is indistinguishable from a game we do
// not carry: it maps to 0, the product is skipped, and a scraper configured
// for that game simply finds nothing. Riftbound went out that way, since the
// catalog calls it "Riftbound: League of Legends TCG" rather than the
// "Riftbound" the mapping expected, and all 1754 of its products were
// dropped without a word.
func TestGameFromCatalog(t *testing.T) {
	tests := []struct {
		catalog string
		want    int
	}{
		{"Magic: The Gathering", GameMagic},
		{"Lorcana", GameLorcana},
		{"Riftbound: League of Legends TCG", GameRiftbound},
		// Shapes the catalog does not use, kept to show the mapping is exact
		// rather than prefix- or substring-based.
		{"Riftbound", 0},
		{"Magic", 0},
		{"Flesh and Blood", 0},
		{"", 0},
	}
	for _, test := range tests {
		t.Run(test.catalog, func(t *testing.T) {
			if got := gameFromCatalog(test.catalog); got != test.want {
				t.Errorf("gameFromCatalog(%q) = %d, want %d", test.catalog, got, test.want)
			}
		})
	}
}
