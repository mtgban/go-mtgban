package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Reality Fracture set names do not claim the fracture foil treatment;
// the wording around them does.
func TestFractureFoilTag(t *testing.T) {
	element := promoTypeElementFor(t, PromoTypeFractureFoil)

	for _, tt := range []struct {
		edition, variation string
		want               bool
	}{
		{"Reality Fracture", "", false},
		{"Reality Fracture Commander", "", false},
		{"FRA", "Reality Fracture", false},
		{"Reality Fracture", "Fracture Foil", true},
		{"Secret Lair Drop", "Fractal Foil", true},
	} {
		got := element.claimedBy(nil, &mtgmatcher.InputCard{Edition: tt.edition, Variation: tt.variation})
		if got != tt.want {
			t.Errorf("%q / %q: claims fracture foil %v, want %v", tt.edition, tt.variation, got, tt.want)
		}
	}
}

// Facet foil is claimed by its own wording, whatever the border is called.
func TestFacetFoilTag(t *testing.T) {
	element := promoTypeElementFor(t, PromoTypeFacetFoil)

	for _, tt := range []struct {
		edition, variation string
		want               bool
	}{
		{"Reality Fracture", "", false},
		{"Reality Fracture", "Borderless", false},
		{"Reality Fracture", "Borderless Facet Foil", true},
		{"Reality Fracture", "Facet Foil", true},
	} {
		got := element.claimedBy(nil, &mtgmatcher.InputCard{Edition: tt.edition, Variation: tt.variation})
		if got != tt.want {
			t.Errorf("%q / %q: claims facet foil %v, want %v", tt.edition, tt.variation, got, tt.want)
		}
	}
}

func promoTypeElementFor(t *testing.T, promoType string) promoTypeElement {
	t.Helper()
	i := slices.IndexFunc(promoTypeElements, func(e promoTypeElement) bool {
		return e.PromoType == promoType
	})
	if i < 0 {
		t.Fatalf("no promo type element for %q", promoType)
	}
	return promoTypeElements[i]
}
