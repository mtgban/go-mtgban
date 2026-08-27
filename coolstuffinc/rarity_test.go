package coolstuffinc

import "testing"

// TestCatalogRarity pins the spellings the storefront and the catalog
// disagree on. Everything else passes through, so a rarity the storefront
// adds later reaches the matcher as written rather than being swallowed.
func TestCatalogRarity(t *testing.T) {
	for _, tt := range []struct {
		desc, in, want string
	}{
		{"the storefront writes starfoil as two words and drops the tier",
			"Star Foil", "Starfoil Rare"},
		{"and drops the tier on shatterfoil too",
			"Shatterfoil", "Shatterfoil Rare"},
		{"and drops the possessive the catalog keeps on collector's",
			"Collector Rare", "Collector's Rare"},
		{"a rarity both spell alike passes through", "Mosaic Rare", "Mosaic Rare"},
		{"so does the plainest one", "Common", "Common"},
		{"and one neither has ever printed", "Quarter Century Secret Rare", "Quarter Century Secret Rare"},
		{"a row with no rarity says nothing", "", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := catalogRarity(tt.in); got != tt.want {
				t.Errorf("catalogRarity(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
