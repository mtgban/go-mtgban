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

// TestPreprocessGradedEditions pins three title shapes that used to leave
// the edition either wrong or cluttered with grade-tier noise: this
// storefront's own "Breaking New" typo, the correctly-spelled showcase
// variant of the same shelf, and a rules-status word that belongs on the
// edition, not stripped from it.
func TestPreprocessGradedEditions(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		title   string
		name    string
		edition string
		variant string
		foil    bool
	}{
		{
			desc:    "the storefront's own typo for Breaking News",
			title:   "Commandeer (Outlaws of Thunder Junction Breaking New Foil CGC Pristine 10) #8003",
			name:    "Commandeer",
			edition: "Breaking News", foil: true,
		},
		{
			desc:    "the correctly-spelled Breaking News Showcase",
			title:   "Path to Exile (Outlaws of Thunder Junction Breaking News Showcase Foil CGC Pristine 10) #8017",
			name:    "Path to Exile",
			edition: "Breaking News", variant: "Showcase", foil: true,
		},
		{
			desc:    "Eternal-Legal is CK's own name for the datastore's Eternal",
			title:   "Spider-Man, Miles Morales (Marvel's Spider-Man Eternal-Legal Foil CGC 9.5) #3066",
			name:    "Spider-Man, Miles Morales",
			edition: "Marvel's Spider-Man Eternal", foil: true,
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

// TestMatchGradedSecretLairCountdown pins matchGraded retrying an unknown
// "Secret Lair" edition against Secret Lair Countdown, and proves a card
// genuinely filed under the plain drop, ambiguous or not, still refuses
// exactly as it did before.
func TestMatchGradedSecretLairCountdown(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc  string
		title string
		want  string // "" means still refused
	}{
		{
			desc:  "a halofoil printing only Secret Lair Countdown carries",
			title: "Hymn to Tourach (Secret Lair Halo Foil CGC Pristine 10) #9065",
			want:  "5435a718-eefc-504e-8316-b187336a83f1",
		},
		{
			desc:  "a nonfoil printing only Secret Lair Countdown carries",
			title: "Elite Spellbinder (Secret Lair CGC Pristine 10) #7125",
			want:  "a568851d-5da2-5089-8dab-a85db1796c3f",
		},
		{
			// Dark Ritual has two ordinary Secret Lair printings, so the
			// retry must not fire and mask the genuine ambiguity.
			desc:  "an ambiguous plain Secret Lair card stays refused",
			title: "Dark Ritual (Secret Lair Foil PSA 10) #5692",
			want:  "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatalf("preprocessGraded(%q) = %v", tt.title, err)
			}
			cardID, err := matchGraded(b, theCard)
			if tt.want == "" {
				if err == nil {
					co, _ := b.GetUUID(cardID)
					t.Errorf("matchGraded(%v) = %s (%v), want it to stay refused", theCard, cardID, co)
				}
				return
			}
			if err != nil {
				t.Fatalf("matchGraded(%v) = %v", theCard, err)
			}
			if cardID != tt.want {
				co, _ := b.GetUUID(cardID)
				t.Errorf("matchGraded(%v) = %s (%v), want %s", theCard, cardID, co, tt.want)
			}
		})
	}
}
