package pokemon

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestLabelsBeforeMarks pins which copy a wording names when it names a
// label beside a mark, a number, or another label. Every case is a product
// the catalog sells under its own TCGplayer id, and the id is what the
// answer is graded by.
func TestLabelsBeforeMarks(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a placement outranks the year the plain copy is marked with", mtgmatcher.InputCard{
			Name: "Champions Festival", Edition: "XY Promos", Variation: "XY91 2015 Quarter Finalist"},
			"xy91_199317"},
		{"a placement wearing the year too is still the placement", mtgmatcher.InputCard{
			Name: "Champions Festival - XY27", Edition: "XY Promos", Variation: "XY27 2014 Staff"},
			"xy27_96485_holofoil"},
		{"a placement spelled out is the figure its label carries", mtgmatcher.InputCard{
			Name: "Champions Festival - BW95 [Top Thirty-Two]", Edition: "Black and White Promos", Variation: "BW95 Worlds 13"},
			"bw95_96567"},
		{"a label is named by its words with a year between them", mtgmatcher.InputCard{
			Name: "Fighting Energy", Edition: "League & Championship Cards", Variation: "WotC 2002 League Promo"},
			"125568_holofoil"},
		{"a label named keeps the copy the number tier would drop", mtgmatcher.InputCard{
			Name: "Leafeon - 11/116", Edition: "League & Championship Cards", Variation: "11 Regional Championship Promo"},
			"11_183200_reverseholofoil"},
		{"the mark named most fully is the one meant", mtgmatcher.InputCard{
			Name: "Code Card - Legends of Galar Tin [Zacian V]", Edition: "SWSH02: Rebel Clash", Variation: "International Version"},
			"279260"},
		{"a wording writing no number means the unnumbered twin", mtgmatcher.InputCard{
			Name: "Metal Energy - 2008", Edition: "World Championship Decks", Variation: "Tristan Robinson"},
			"479797"},
		{"a stamp names the stamped copy before its mark or label", mtgmatcher.InputCard{
			Name: "Archaludon - 107/142", Edition: "Miscellaneous Cards & Products", Variation: "107/142 Cosmos Holo Stellar Crown Stamp"},
			"107-142_588382_holofoil"},
		{"an accent does not hide a label", mtgmatcher.InputCard{
			Name: "Magneton - 159", Edition: "SV: Scarlet & Violet Promo Cards", Variation: "159 Pokémon Center Exclusive"},
			"159_594468_holofoil"},
		{"two words opening a label name it", mtgmatcher.InputCard{
			Name: "Flutter Mane - 097", Edition: "SV: Scarlet & Violet Promo Cards", Variation: "097 Pokemon Center"},
			"097_543947_holofoil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			got, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%+v) = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Match(%+v) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

// TestPlacementNoSurvivorWears pins the refusal: a wording naming a
// placement prices a copy the plain printing is not, and where the copies
// the number reaches wear none of it the plain one used to answer. The
// catalog numbers this staff card 177 in its number field and 171 in its
// name, so the number reaches the plain card alone.
func TestPlacementNoSurvivorWears(t *testing.T) {
	b := loadBackend(t)

	in := mtgmatcher.InputCard{
		Name: "Professor Turo's Scenario - 171/182 [Staff]", Edition: "League & Championship Cards", Variation: "177/182 Regional Championships"}
	got, err := b.Match(&in)
	if err == nil {
		t.Fatalf("Match(%+v) = %s, want a refusal", in, got)
	}
	if !errors.Is(err, mtgmatcher.ErrCardWrongVariant) {
		t.Errorf("Match(%+v) = %v, want %v", in, err, mtgmatcher.ErrCardWrongVariant)
	}
}
