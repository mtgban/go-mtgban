package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoWordingNamesItsSet pins the promos a storefront files under a bare
// "Promo" heading and describes by the event or insert series that issued
// them. The series is the only thing that says which set the listing means:
// left unread, every one of these cards also has a prerelease printing whose
// promo set outranks the one the wording asked for.
func TestPromoWordingNamesItsSet(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		in       mtgmatcher.InputCard
		wantSet  string
		wantNum  string
		wantFoil bool
	}{
		{
			desc:    "a Regional Championship promo is the qualifier printing, not the prerelease one",
			in:      mtgmatcher.InputCard{Name: "Gideon, Ally of Zendikar", Edition: "Promo", Variation: "Regional Championship", Foil: true},
			wantSet: "PRCQ", wantNum: "1", wantFoil: true,
		},
		{
			desc:    "and the nonfoil of the same listing keeps its finish",
			in:      mtgmatcher.InputCard{Name: "Gideon, Ally of Zendikar", Edition: "Promo", Variation: "Regional Championship"},
			wantSet: "PRCQ", wantNum: "1",
		},
		{
			desc:    "the year the wording spells still chooses between the two qualifier sets",
			in:      mtgmatcher.InputCard{Name: "Mystical Dispute", Edition: "Promo", Variation: "Regional Championship Qualifiers 2023"},
			wantSet: "PR23", wantNum: "1",
		},
		{
			desc:    "a Regional Championship promo filed under Pro Tour Promos stays there",
			in:      mtgmatcher.InputCard{Name: "Aerith Gainsborough", Edition: "Pro Tour Promos", Variation: "Regional Championship 1"},
			wantSet: "PPRO", wantNum: "2025-3",
		},
		{
			desc:    "a Marvel Legends Series insert is the insert, not the prerelease printing",
			in:      mtgmatcher.InputCard{Name: "Iron Spider, Stark Upgrade", Edition: "Promo", Variation: "Borderless Marvel Legends Series", Foil: true},
			wantSet: "LMAR", wantNum: "4", wantFoil: true,
		},
		{
			desc:    "the insert wins even where the base set numbers the card differently",
			in:      mtgmatcher.InputCard{Name: "Spectacular Spider-Man", Edition: "Promo", Variation: "Borderless Marvel Legends Series", Foil: true},
			wantSet: "LMAR", wantNum: "2", wantFoil: true,
		},
		{
			desc:    "the wording the edition table already knew still lands on the insert",
			in:      mtgmatcher.InputCard{Name: "Anti-Venom, Horrifying Healer", Edition: "Marvel Legends Foil", Foil: true},
			wantSet: "LMAR", wantNum: "1", wantFoil: true,
		},
		{
			desc:    "a listing that says prerelease instead is still the prerelease printing",
			in:      mtgmatcher.InputCard{Name: "Iron Spider, Stark Upgrade", Edition: "Marvel's Spider-Man Promos", Variation: "Prerelease", Foil: true},
			wantSet: "PSPM", wantNum: "166s", wantFoil: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum || co.Foil != tt.wantFoil {
				t.Errorf("Match(%v) = %s|%s|foil=%v, want %s|%s|foil=%v", tt.in, co.SetCode, co.Number, co.Foil, tt.wantSet, tt.wantNum, tt.wantFoil)
			}
		})
	}
}
