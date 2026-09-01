package gamenerdz

import "testing"

func TestNameSaysFoil(t *testing.T) {
	tests := []struct {
		displayName string
		foil        bool
	}{
		{"Necrologia (7ED-149) - 7th Edition Foil", true},
		{"Necrologia (7ED-149) - 7th Edition", false},
		{"Kilnmouth Dragon (LGN-104) - Legions Foil(MP)", true},
		{"Adarkar Wastes (DMU-243) - Dominaria United Foil (LP)", true},
		{"Aboleth Spawn - Commander Legends: Battle for Baldur's Gate: (Extended Art) Foil", true},
		// A set can be called a foil edition without the product being
		// one, so only the finish the shelf ends in counts.
		{"Richard Garfield, Ph.D. (LIST-017) - The List (Unfinity Foil Edition)", false},
		{"Richard Garfield, Ph.D. (LIST-017) - The List (Unfinity Foil Edition) Foil", true},
	}
	for _, tt := range tests {
		foil := nameSaysFoil(tt.displayName)
		if foil != tt.foil {
			t.Errorf("%q: got %v; want %v", tt.displayName, foil, tt.foil)
		}
	}
}
