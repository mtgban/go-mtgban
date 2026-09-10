package pokemon

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNameTailsTheCatalogWrites pins two spellings the catalog itself puts
// behind a dash, which a storefront copying the product name carries along:
// a number written with a dash for its slash, and a bracketed set behind
// the number. The datastore names neither into the card any more.
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
