package cardkingdom

import "testing"

// TestPreprocessGradedNested pins the titles whose edition carries a
// parenthetical of its own. Refusing them dropped 41 of the roughly 950
// listings this storefront grades, silently, and they are the expensive end
// of its inventory.
func TestPreprocessGradedNested(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		title   string
		name    string
		edition string
		variant string
		foil    bool
	}{
		{
			desc:    "the treatment in a parenthetical of its own",
			title:   "Raph & Mikey, Troublemakers (TMNT Foil (Showcase) CGC Pristine 10) #7165",
			name:    "Raph & Mikey, Troublemakers",
			edition: "TMNT", variant: "Showcase", foil: true,
		},
		{
			desc:    "and where the edition carries a word of its own too",
			title:   "Michelangelo, the Heart (TMNT Eternal Foil (Borderless) CGC Pristine 10) #7051",
			name:    "Michelangelo, the Heart",
			edition: "TMNT Eternal", variant: "Borderless", foil: true,
		},
		{
			// The abbreviation reaches the set on its own, but not with
			// "Source Material Cards" behind it: the card falls back to
			// Shadowmoor, where it was first printed.
			desc:    "the edition this storefront abbreviates is spelled out",
			title:   "Plague of Vermin (TMNT Source Material Cards Foil CGC 10) #7067",
			name:    "Plague of Vermin",
			edition: "Teenage Mutant Ninja Turtles Source Material", foil: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatalf("preprocessGraded(%q) = %v", tt.title, err)
			}
			if got.Name != tt.name || got.Edition != tt.edition ||
				got.Variation != tt.variant || got.Foil != tt.foil {
				t.Errorf("preprocessGraded(%q) = %q/%q/%q foil=%v, want %q/%q/%q foil=%v",
					tt.title, got.Name, got.Edition, got.Variation, got.Foil,
					tt.name, tt.edition, tt.variant, tt.foil)
			}
		})
	}
}
