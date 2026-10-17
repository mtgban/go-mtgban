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
	b := realDatastore(t)

	corrupt := product("MTG-RNA-249-6NUARTU9FH",
		"Gruul Guildgate (RNA-249) - Ravnica Allegiance", "pre")
	corrupt.BuyVariants = []GNBuyVariant{{Title: "Default Title", OfferPrice: 34.60}}
	shelf := product("MTG-DFT-092-F-VYAO8V2VJ9",
		"Chrome Mox (Borderless) (DFT-092) - Special Guests Foil", "spg")
	shelf.BuyVariants = []GNBuyVariant{{Title: "Default Title", OfferPrice: 157.69}}

	gn, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
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

// A plain listing whose finish field alone says foil is plain where its price
// is not its foil sibling's, and the sibling's duplicate where it is.
func TestSettleFoilBySibling(t *testing.T) {
	gn, err := NewScraper(realDatastore(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		desc, set             string
		plainSKU, foilSKU     string
		display               string
		plainPrice, foilPrice float64
		want                  string
	}{
		{"priced as the plain card", "9ed", "MTG-9ED-289-AMGOSOAL3W", "MTG-9ED-289-F-AMGOSOAL3W", "Booby Trap (9ED-289) - 9th Edition", 0.48, 9.89, "nonfoil"},
		{"a second listing of the foil", "8ed", "MTG-8ED-064-LZWABIF543", "MTG-8ED-064-F-LZWABIF543", "Bribery (8ED-064) - 8th Edition", 24.13, 24.13, "foil"},
	} {
		state := &crawlState{bodies: map[string]map[string]bool{}, foilPrices: map[string]float64{}}
		foil := product(tt.foilSKU, tt.display+" Foil", tt.set)
		foil.SelectedFinish, foil.Price = "foil", tt.foilPrice
		plain := product(tt.plainSKU, tt.display, tt.set)
		plain.SelectedFinish, plain.Price = "foil", tt.plainPrice
		if gn.admit(foil, state) {
			t.Errorf("%s: the foil listing should not wait", tt.desc)
		}
		if !gn.admit(plain, state) {
			t.Errorf("%s: the plain listing should wait for its sibling", tt.desc)
		}
		settled, ok := state.settle(plain)
		if !ok || settled.SelectedFinish != tt.want {
			t.Errorf("%s: settled to %q (kept %v), want %q", tt.desc, settled.SelectedFinish, ok, tt.want)
		}
	}
}

// A body naming another set than the display's own known one is the price of
// another product, and a refusal of it is not worth reporting.
func TestForeignBody(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc    string
		product GNProduct
		want    bool
	}{
		{"another set's body", product("MTG-LEA-031", "Personal Incarnation (LEA-031) - Limited Edition Alpha", "tla"), true},
		{"its own body", product("MTG-LEA-031", "Personal Incarnation (LEA-031) - Limited Edition Alpha", "lea"), false},
		{"a shelf code of the storefront's", product("MTG-PM21-179P-F-WNWC8O2ID2", "Elder Gargaroth (PPM21-179P) - Core Set 2021 Promos Foil", "ppm21"), false},
		{"the storefront's prerelease shelf", product("MTG-PXLN-213S-F", "Verdant Sun's Avatar (PXLN-213S) - Ixalan Promos Foil (LP)", "pre"), false},
		{"the List shelf", product("MTG-PLST-C16-52", "Boompile (PLST-C16-52) - The List", "list"), false},
		{"the List's origin set", product("MTG-PLST-JMP-18", "Witch of the Moors (PLST-JMP-18) - The List", "jmp"), false},
		{"a leading body tag", product("MTG-SPE-MAR-005", "Alchemax Slayer-Bots (SPE, MAR-005) - Marvel's Spider-Man: Eternal-Legal", "spe"), false},
		{"no tag to disagree with", product("MTG-DMU-160-LP", "Defiler of Vigor - Dominaria United (LP)", "pre"), false},
	} {
		got := foreignBody(b, tt.product)
		if got != tt.want {
			t.Errorf("%s: foreignBody = %v, want %v", tt.desc, got, tt.want)
		}
	}
}
