package vegassingles

import "testing"

// TestOversizeHeading pins which listings in the storefront's oversize bin are
// display pieces and which are singles it happens to file there. The bin's own
// code names no set, so a display card left to the matcher reaches whichever
// ordinary printing its name and number find and is published at that card's
// price - an oversize Ambition's Cost at the price of the 8th Edition one.
func TestOversizeHeading(t *testing.T) {
	for _, display := range []string{
		"Abeyance (Arena League) (OVER-) - Oversize Cards",
		"Ambition's Cost (8th Edition) (OVER-118) - Oversize Cards",
		"Chaos Orb (InQuest Magazine) (OVER-) - Oversize Cards",
		"Blacker Lotus (Arena League) (OVER-070) - Oversize Cards",
	} {
		product := VSProduct{DisplayName: display}
		product.ProductData.Set = "over"
		product.ProductData.SetName = "Oversize Cards"
		if _, err := preprocess(product, GameMagic); err == nil {
			t.Errorf("%s: read as a single, want refused as a display card", display)
		}
	}

	// The bin holds real singles too, and their display names say so.
	for _, display := range []string{
		"Aretopolis (OVER-010) - Planechase 2012 Planes",
		"Astral Arena (OVER-011) - Planechase 2012 Planes",
		"Hallowed Fountain (RVR-280) - Ravnica Remastered",
	} {
		product := VSProduct{DisplayName: display}
		product.ProductData.Set = "over"
		product.ProductData.SetName = "Oversize Cards"
		if _, err := preprocess(product, GameMagic); err != nil {
			t.Errorf("%s: refused as a display card, want read as a single (%v)", display, err)
		}
	}
}

// TestDisplaySet pins the storefront's own spelling of a listing's set, which
// is not always one the datastore knows.
func TestDisplaySet(t *testing.T) {
	for _, tt := range []struct{ display, want string }{
		{"Abeyance (Arena League) (OVER-) - Oversize Cards", "Oversize Cards"},
		{"Aretopolis (OVER-010) - Planechase 2012 Planes", "Planechase 2012 Planes"},
		{"Counterspell (MFP-002) - MagicFest Cards Foil", "MagicFest Cards"},
		{"Strixhaven School of Mages draft booster packs", ""},
	} {
		if got := displaySet(tt.display); got != tt.want {
			t.Errorf("displaySet(%q) = %q, want %q", tt.display, got, tt.want)
		}
	}
}
