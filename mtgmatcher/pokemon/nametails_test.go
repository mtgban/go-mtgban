package pokemon

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNameTailsTheCatalogWrites pins the spellings the catalog itself puts
// behind a dash, which a storefront copying the product name carries along:
// a number written with a dash for its slash, a bracketed set behind the
// number, and the two halves' numbers of a LEGEND pair printed as one card.
// The datastore names none of them into the card any more.
func TestNameTailsTheCatalogWrites(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a dash for the slash", mtgmatcher.InputCard{
			Name: "Morpeko - 072-167", Edition: "Prize Pack Series Cards"}, "072-167_619142"},
		{"a bracketed set behind the number", mtgmatcher.InputCard{
			Name: "Pichu - 45/106 [Platinum]", Variation: "045/100", Edition: "Burger King Promos"}, "045-100_188126_reverseholofoil"},
		{"and behind a number the catalog's own field contradicts", mtgmatcher.InputCard{
			Name: "Piplup - 93/130 [Diamond and Pearl]", Variation: "093/100", Edition: "Burger King Promos"}, "093-100_235080_reverseholofoil"},
		{"a LEGEND pair's two numbers", mtgmatcher.InputCard{
			Name: "Darkrai & Cresselia Legend - 99/102 & 100/102", Variation: "Single Oversized Promo", Edition: "Jumbo Cards"}, "099-102-100-102_211448_holofoil"},
		{"joined with a plus", mtgmatcher.InputCard{
			Name: "Palkia & Dialga Legends - 101/102 + 102/102", Variation: "Single Oversized Promo", Edition: "Jumbo Cards"}, "101-102-102-102_211449_holofoil"},
		{"read as one number in the wording too, not as the halves", mtgmatcher.InputCard{
			Name: "Darkrai & Cresselia Legend", Variation: "099/102 & 100/102"}, "099-102-100-102_211448_holofoil"},
		{"a name carrying a bracket of its own stays whole", mtgmatcher.InputCard{
			Name: "Greninja V-UNION [Set of 4]", Edition: "SWSH Promos"}, "248896_holofoil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestLegendPairNeverAnswersAHalf pins the refusal: a pair's number names
// the one card printed with both halves' numbers, and the storefronts shelve
// that card under Triumphant, where the edition admits the halves alone.
func TestLegendPairNeverAnswersAHalf(t *testing.T) {
	b := loadBackend(t)

	in := mtgmatcher.InputCard{
		Name: "Darkrai & Cresselia Legend - 99/102 & 100/102", Variation: "Single Oversized Promo", Edition: "Triumphant"}
	got, err := b.Match(&in)
	if err == nil {
		t.Fatalf("Match(%+v) = %s, want a refusal", in, got)
	}
	if !errors.Is(err, mtgmatcher.ErrCardWrongVariant) {
		t.Errorf("Match(%+v) = %v, want %v", in, err, mtgmatcher.ErrCardWrongVariant)
	}
}
