package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestColorsAreNames holds Magic's cards to naming their colors and color
// identity as every game's do, from the colors its sets list in order.
func TestColorsAreNames(t *testing.T) {
	realDatastore(t)

	for _, co := range testBackend.UUIDs {
		for _, color := range append(slices.Clone(co.Colors), co.ColorIdentity...) {
			if !slices.Contains(mtgColors[:5], color) {
				t.Fatalf("%s carries the color %q", co.UUID, color)
			}
		}
	}
}

// TestTokenColorPicksThePrinting pins the token rule that reads a color off
// the listing: it answers with the token of that color.
func TestTokenColorPicksThePrinting(t *testing.T) {
	realDatastore(t)

	for _, tt := range []struct {
		name, edition, variation, number string
	}{
		{"Astartes Warrior", "Warhammer 40,000 Commander Tokens", "White", "1"},
		{"Astartes Warrior", "Warhammer 40,000 Commander Tokens", "Black", "12"},
		{"Thopter", "Double Masters Tokens", "Blue", "8"},
		{"Spirit", "Double Masters 2022 Tokens", "White", "8"},
	} {
		in := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.variation}
		id, err := testBackend.Match(&in)
		if err != nil {
			t.Errorf("%s %s: %v", tt.name, tt.variation, err)
			continue
		}
		co, err := testBackend.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number != tt.number {
			t.Errorf("%s %s = #%s %v, want #%s", tt.name, tt.variation, co.Number, co.Colors, tt.number)
		}
	}
}
