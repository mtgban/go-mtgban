package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
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
		got := element.claimedBy(nil, &matcher.InputCard{Edition: tt.edition, Variation: tt.variation})
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
		got := element.claimedBy(nil, &matcher.InputCard{Edition: tt.edition, Variation: tt.variation})
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

// A facet foil listing lands on the facet foil printing, not on the plain
// borderless foil at its own number.
func TestFacetFoilMatch(t *testing.T) {
	realDatastore(t)
	b := testBackend

	for _, variation := range []string{"Borderless Facet Foil", "Facet Foil"} {
		in := &matcher.InputCard{Name: "Enlightened Confidant", Edition: "Reality Fracture", Variation: variation, Foil: true}
		id, err := b.Match(in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", variation, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != "FRA" || co.Number != "449" {
			t.Errorf("%q: landed on %s %s, want FRA 449", variation, co.SetCode, co.Number)
		}
	}
}
