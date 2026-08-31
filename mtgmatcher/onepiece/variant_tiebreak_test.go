package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// variantTiebreakFixture mirrors the three shapes a storefront's wording
// leaves the tiering: a shelf name that spells a label worn by another
// printing of the same number, a treatment written behind the category it
// belongs to, and a label the wording shortens beside the set it names.
//
// The collector numbers, product ids and labels are the catalog's.
const variantTiebreakFixture = `{
	"game": "onepiece",
	"sets": {
		"OP05": {"name": "Awakening of the New Era", "releaseDate": "2023-09-08"},
		"OP09": {"name": "Emperors in the New World", "releaseDate": "2024-12-27"},
		"OP11": {"name": "A Fist of Divine Speed", "releaseDate": "2025-06-27"},
		"OP13": {"name": "Carrying On His Will", "releaseDate": "2025-11-21"},
		"ST-01": {"name": "Starter Deck 1: Straw Hat Crew", "releaseDate": "2022-12-02"}
	},
	"cards": [
		{"id": "op05-119_527024_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP05", "rarity": "SEC", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 527024}},
		{"id": "op05-119_596915_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP09", "rarity": "SEC", "finish": "Foil", "variant": "Wanted Poster", "image": "x", "externalLinks": {"tcgPlayerId": 596915}},
		{"id": "op05-119_632503_foil", "name": "Monkey.D.Luffy", "number": "OP05-119", "setCode": "OP11", "rarity": "SEC", "finish": "Foil", "variant": "SP", "image": "x", "externalLinks": {"tcgPlayerId": 632503}},
		{"id": "op13-091_657366_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 657366}},
		{"id": "op13-091_657367_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "variant": "Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 657367}},
		{"id": "op13-091_657368_foil", "name": "St. Marcus Mars", "number": "OP13-091", "setCode": "OP13", "rarity": "SR", "finish": "Foil", "variant": "Parallel", "image": "x", "externalLinks": {"tcgPlayerId": 657368}},
		{"id": "st01-012_288241_foil", "name": "Monkey.D.Luffy", "number": "ST01-012", "setCode": "ST-01", "rarity": "SR", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 288241}},
		{"id": "st01-012_527027_foil", "name": "Monkey.D.Luffy", "number": "ST01-012", "setCode": "OP05", "rarity": "SR", "finish": "Foil", "variant": "Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 527027}},
		{"id": "st01-012_529850_foil", "name": "Monkey.D.Luffy", "number": "ST01-012", "setCode": "OP05", "rarity": "SR", "finish": "Foil", "variant": "Alternate Art Gold-Stamped Signature", "image": "x", "externalLinks": {"tcgPlayerId": 529850}}
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
		{
			// The storefront shortens the label to the signature and drops
			// the treatment it is printed on, and names the set the
			// printing is filed in. The edition names the deck the card was
			// first printed in, which is the one printing it is not.
			desc: "a shortened label reaches the set the wording names",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "ST01-012 OP05 1st Anniversary Gold-Stamped Signature",
				Edition: "ST01 - Starter Deck: Straw Hat Crew", Foil: true,
			},
			want: "st01-012_529850_foil",
		},
		{
			desc: "and a listing naming no label at all stays in that deck",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "ST01-012",
				Edition: "ST01 - Starter Deck: Straw Hat Crew", Foil: true,
			},
			want: "st01-012_288241_foil",
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
