package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// variantTiebreakFixture mirrors the two shapes a storefront's wording ties
// on: a shelf name that spells a label worn by another printing of the same
// number, and a treatment written behind the category it belongs to.
//
// The collector numbers, product ids and labels are the catalog's.
const variantTiebreakFixture = `{
	"game": "onepiece",
	"sets": {
		"OP05": {"name": "Awakening of the New Era", "releaseDate": "2023-09-08"},
		"OP09": {"name": "Emperors in the New World", "releaseDate": "2024-12-27"},
		"OP11": {"name": "A Fist of Divine Speed", "releaseDate": "2025-06-27"},
		"OP13": {"name": "Carrying On His Will", "releaseDate": "2025-11-21"}
	},
	"cards": [
		{"id": "op05-119_527024_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP05", "rarity": "SEC", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 527024}},
		{"id": "op05-119_596915_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP09", "rarity": "SEC", "finish": "Foil", "variant": "Wanted Poster", "image": "x", "externalLinks": {"tcgPlayerId": 596915}},
		{"id": "op05-119_632503_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP11", "rarity": "SEC", "finish": "Foil", "variant": "SP", "image": "x", "externalLinks": {"tcgPlayerId": 632503}},
		{"id": "op13-091_657366_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 657366}},
		{"id": "op13-091_657367_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "variant": "Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 657367}},
		{"id": "op13-091_657368_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "variant": "Parallel", "image": "x", "externalLinks": {"tcgPlayerId": 657368}}
	]
}`

// TestVariantTiebreaks pins the two ties a label-naming wording leaves. Both
// answered with an aliasing error, which drops the listing: the shelf tie
// costs a $425 buy price and the category tie a $345 one.
func TestVariantTiebreaks(t *testing.T) {
	b, err := Load(strings.NewReader(variantTiebreakFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			// The shelf spells a label another printing of this number
			// wears, and the variation spells the one this listing is.
			desc: "the variation outranks the shelf that names a label too",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "OP05-119 Wanted Poster",
				Edition: "One Piece: SP", Foil: true,
			},
			want: "op05-119_596915_foil",
		},
		{
			desc: "and the shelf still answers when the variation names nothing",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "OP05-119",
				Edition: "One Piece: SP", Foil: true,
			},
			want: "op05-119_632503_foil",
		},
		{
			// The store sells this at $12 and the one below at $345.
			desc: "a wording naming only the category is the category's own printing",
			in: mtgmatcher.InputCard{
				Name: "St. Marcus Mars", Variation: "OP13-091 Alternate Art",
				Edition: "OP13 - Carrying On His Will", Foil: true,
			},
			want: "op13-091_657367_foil",
		},
		{
			desc: "and the treatment written behind it is what tells the two apart",
			in: mtgmatcher.InputCard{
				Name: "St. Marcus Mars", Variation: "OP13-091 Alternate Art Red Parallel",
				Edition: "OP13 - Carrying On His Will", Foil: true,
			},
			want: "op13-091_657368_foil",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match = %q (%v), want %q", uuid, co, test.want)
			}
		})
	}
}
