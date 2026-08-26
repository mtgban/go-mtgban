package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// numberedNameFixture mirrors the catalog's habit of writing a promo's
// occasion inside the name. Both P-110 printings are "Monkey.D.Luffy (4th
// Anniversary)" while every storefront writes the bare "Monkey.D.Luffy",
// which is a canonical name in its own right - OP01-001 is filed under it,
// so the lookup succeeds and the number then finds nothing. P-138 is the
// control: two names share it, so the number says nothing about which was
// meant.
const numberedNameFixture = `{
	"game": "onepiece",
	"sets": {
		"OP01":  {"name": "Romance Dawn", "releaseDate": "2022-12-02"},
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"}
	},
	"cards": [
		{"id": "op01-001_500001", "name": "Monkey.D.Luffy", "number": "OP01-001", "setCode": "OP01", "rarity": "L", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 500001}},
		{"id": "p-110_712742", "name": "Monkey.D.Luffy (4th Anniversary)", "number": "P-110", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 712742}},
		{"id": "p-110_712743_foil", "name": "Monkey.D.Luffy (4th Anniversary)", "number": "P-110", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Winner", "image": "x", "externalLinks": {"tcgPlayerId": 712743}},
		{"id": "p-138_712800", "name": "Monkey.D.Luffy (5th Anniversary)", "number": "P-138", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 712800}},
		{"id": "p-138_712801", "name": "Roronoa Zoro (5th Anniversary)", "number": "P-138", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 712801}}
	]
}`

// TestNumberedNameAdopted pins that a canonical name with no printing at the
// input's number takes the catalog's decorated spelling of the card the
// number names, and that the guards nameAtNumber already carries keep it
// from renaming anything else.
func TestNumberedNameAdopted(t *testing.T) {
	b, err := Load(strings.NewReader(numberedNameFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "the number names the decorated printing",
			in: mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-110 OP-17 Release Event",
				Edition: "Promos"},
			want: "p-110_712742",
		},
		{
			desc: "and the label beside it still picks the winner's copy",
			in: mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-110 OP-17 Release Event | Winner",
				Edition: "Winner Pack"},
			want: "p-110_712743_foil",
		},
		{
			desc: "cardmarket's positional wording reaches it too",
			in:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy (P-110) (V.1)"},
			want: "p-110_712742",
		},
		{
			desc: "a name with a printing at the number is left as it is",
			in:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "OP01-001", Edition: "Romance Dawn"},
			want: "op01-001_500001",
		},
		{
			desc: "a number two names answer renames nothing",
			in:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-138", Edition: "Promos"},
			err:  true,
		},
		{
			desc: "and neither does a bare tail, which every set has one of",
			in:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "110", Edition: "Promos"},
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if tt.err {
				if err == nil {
					t.Fatalf("Match(%q, %q) = %q, want an error",
						tt.in.Name, tt.in.Variation, id)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%q, %q) = %v", tt.in.Name, tt.in.Variation, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q, %q) = %q, want %q", tt.in.Name, tt.in.Variation, id, tt.want)
			}
		})
	}
}
