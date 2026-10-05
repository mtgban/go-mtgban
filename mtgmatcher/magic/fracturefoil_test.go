package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Reality Fracture set names do not claim the fracture foil treatment;
// the wording around them does.
func TestFractureFoilTag(t *testing.T) {
	i := slices.IndexFunc(promoTypeElements, func(e promoTypeElement) bool {
		return e.PromoType == PromoTypeFractureFoil
	})
	element := promoTypeElements[i]

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
