package starcitygames

import "testing"

// TestGameFromCatalog pins the catalog's own spelling of each game. The
// mapping is the only thing standing between a product and the scraper that
// wants it, and an unrecognized name is indistinguishable from a game we do
// not carry: it maps to 0, the product is skipped, and a scraper configured
// for that game simply finds nothing. Riftbound has now gone out that way
// twice, in both directions: first the catalog said "Riftbound: League of
// Legends TCG" where the mapping expected "Riftbound", then the catalog
// renamed itself back to "Riftbound" and the corrected mapping expected the
// long form. Both spellings answer, so the next rename is one the scraper
// survives.
func TestGameFromCatalog(t *testing.T) {
	tests := []struct {
		catalog string
		want    int
	}{
		{"Magic: The Gathering", GameMagic},
		{"Flesh and Blood", GameFleshAndBlood},
		{"Lorcana", GameLorcana},
		{"Riftbound: League of Legends TCG", GameRiftbound},
		{"Riftbound", GameRiftbound},
		// Shapes the catalog does not use, kept to show the mapping is exact
		// rather than prefix- or substring-based.
		{"Magic", 0},
		{"Flesh And Blood", 0},
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
