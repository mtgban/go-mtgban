package gamenerdz

import (
	"testing"

	"github.com/mtgban/go-mtgban/internal/jsonflex"
)

func product(sku, display, set string) GNProduct {
	return GNProduct{
		DisplayName:    display,
		ProductData:    GNProductData{Set: jsonflex.String(set)},
		RetailVariants: []GNRetailVariant{{SKU: sku}},
	}
}

func TestSkuFamily(t *testing.T) {
	for _, tt := range []struct{ sku, want string }{
		{"MTG-RNA-249-F-6NUARTU9FH", "MTG-RNA-249-6NUARTU9FH"},
		{"MTG-RNA-249-6NUARTU9FH", "MTG-RNA-249-6NUARTU9FH"},
		{"MTG-MH2-389-EF-7NTYUWLWDO", "MTG-MH2-389-7NTYUWLWDO"},
		// A hash that happens to read F or EF is not the finish segment,
		// which only ever sits between the number and the hash.
		{"MTG-DFT-092-VYAO8V2VJ9", "MTG-DFT-092-VYAO8V2VJ9"},
		{"", ""},
	} {
		got := skuFamily(GNProduct{RetailVariants: []GNRetailVariant{{SKU: tt.sku}}})
		if got != tt.want {
			t.Errorf("skuFamily(%q) = %q, want %q", tt.sku, got, tt.want)
		}
	}
	if got := skuFamily(GNProduct{}); got != "" {
		t.Errorf("skuFamily with no variant = %q, want empty", got)
	}
}

func TestBodyNamesOwnSet(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product GNProduct
		want    bool
	}{
		{
			"the plain listing, whose body is its own set",
			product("MTG-RNA-249-F-6NUARTU9FH", "Gruul Guildgate (RNA-249) - Ravnica Allegiance Foil", "rna"),
			true,
		},
		{
			// The body Game Nerdz joined to this product belongs to a
			// Crucible of Worlds prerelease, and says so.
			"the listing carrying another product's body",
			product("MTG-RNA-249-6NUARTU9FH", "Gruul Guildgate (RNA-249) - Ravnica Allegiance", "pre"),
			false,
		},
		{
			// A promo pack is coded ppm21 and named PPM21, on both
			// finishes, so the two never disagree.
			"a shelf this storefront codes its own way",
			product("MTG-PM21-179P-F-WNWC8O2ID2", "Elder Gargaroth (PPM21-179P) - Core Set 2021 Promos Foil", "ppm21"),
			true,
		},
		{
			"a name carrying no set tag at all",
			product("MTG-MH2-436-F-BEUFVXJ561", "Arid Mesa - Modern Horizons 2 Etched Foil", "mh2"),
			true,
		},
	} {
		if got := bodyNamesOwnSet(tt.product); got != tt.want {
			t.Errorf("%s: bodyNamesOwnSet = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

// The crawl holds a product whose body names another set and decides it once
// every product has been seen: the body belongs to another product where the
// two finishes disagree about it, and is this storefront's own shelf code
// where they do not.
func TestReleaseHeldProducts(t *testing.T) {
	realDatastore(t)

	corrupt := product("MTG-RNA-249-6NUARTU9FH",
		"Gruul Guildgate (RNA-249) - Ravnica Allegiance", "pre")
	corrupt.BuyVariants = []GNBuyVariant{{Title: "Default Title", OfferPrice: 34.60}}
	shelf := product("MTG-DFT-092-F-VYAO8V2VJ9",
		"Chrome Mox (Borderless) (DFT-092) - Special Guests Foil", "spg")
	shelf.BuyVariants = []GNBuyVariant{{Title: "Default Title", OfferPrice: 157.69}}

	gn := NewScraper(GameMagic)
	state := &crawlState{
		seen:     map[string]bool{},
		rarities: map[string]bool{},
		finishes: map[string]bool{},
		bodies: map[string]map[string]bool{
			// the twin of the Guildgate says "rna" where this one says "pre"
			skuFamily(corrupt): {"pre": true, "rna": true},
			// the twin of the Chrome Mox says "spg" too
			skuFamily(shelf): {"spg": true},
		},
		held: []GNProduct{corrupt, shelf},
	}
	gn.release(modeBuylist, state)

	if len(state.held) != 0 {
		t.Errorf("release left %d products held", len(state.held))
	}
	var guildgate, chromeMox int
	for _, entries := range gn.Buylist() {
		for _, entry := range entries {
			switch entry.BuyPrice {
			case 34.60:
				guildgate++
			case 157.69:
				chromeMox++
			}
		}
	}
	if guildgate != 0 {
		t.Errorf("the mis-joined listing was priced %d times, want 0", guildgate)
	}
	if chromeMox != 1 {
		t.Errorf("the Special Guest was priced %d times, want 1", chromeMox)
	}
}
