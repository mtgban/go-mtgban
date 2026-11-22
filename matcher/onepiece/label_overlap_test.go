package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// labelOverlapFixture mirrors the event pairs the catalog prefixes with the
// base set's code. P-135 is the participation card beside the winner's copy
// of it; ST17-004 is the same shape around a treatment word. P-101 is what
// must keep refusing: a set's promotion cards are told apart by nothing the
// wording says. The OP16 DON!! cards are a set's own pair beside the two its
// Double Pack Set prints.
const labelOverlapFixture = `{"data": {
	"game": "onepiece",
	"sets": {
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"},
		"OP16": {"name": "The Time of Battle", "releaseDate": "2026-06-12"}
	},
	"cards": [
		{"id": "p-135_697483", "name": "Monkey.D.Luffy", "number": "P-135", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "promoTypes": ["OP16 Release Event"], "variant": "OP16 Release Event", "image": "x", "externalLinks": {"tcgPlayerId": 697483}},
		{"id": "p-135_697484_foil", "name": "Monkey.D.Luffy", "number": "P-135", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "promoTypes": ["OP16 Release Event Winner"], "variant": "OP16 Release Event Winner", "image": "x", "externalLinks": {"tcgPlayerId": 697484}},
		{"id": "st17-004_623069_foil", "name": "Boa Hancock", "number": "ST17-004", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "promoTypes": ["Illustration Box Vol.1 Textured"], "variant": "Illustration Box Vol.1 Textured", "image": "x", "externalLinks": {"tcgPlayerId": 623069}},
		{"id": "st17-004_712035_foil", "name": "Boa Hancock", "number": "ST17-004", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "promoTypes": ["Dash Pack 2025"], "variant": "Dash Pack 2025", "image": "x", "externalLinks": {"tcgPlayerId": 712035}},
		{"id": "p-101_700001", "name": "Tony Tony.Chopper", "number": "P-101", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "promoTypes": ["Promotion Card Set 2025 Vol. 1"], "variant": "Promotion Card Set 2025 Vol. 1", "image": "x", "externalLinks": {"tcgPlayerId": 700001}},
		{"id": "p-101_700002_foil", "name": "Tony Tony.Chopper", "number": "P-101", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "promoTypes": ["Store Tournament 2025 Vol. 4"], "variant": "Store Tournament 2025 Vol. 4", "image": "x", "externalLinks": {"tcgPlayerId": 700002}},
		{"id": "698313_foil", "name": "DON!! Card", "setCode": "OP16", "rarity": "DON!!", "finish": "Foil", "variant": "Alternate Art", "promoTypes": ["alternateart"], "type": "DON!!", "image": "x", "externalLinks": {"tcgPlayerId": 698313}},
		{"id": "698314_foil", "name": "DON!! Card", "setCode": "OP16", "rarity": "DON!!", "finish": "Foil", "variant": "Alternate Art Gold", "promoTypes": ["alternateart", "gold"], "type": "DON!!", "image": "x", "externalLinks": {"tcgPlayerId": 698314}},
		{"id": "698315_foil", "name": "DON!! Card", "setCode": "OP16", "rarity": "DON!!", "finish": "Foil", "variant": "Double Pack Set Vol. 11 Luffy and Ace", "promoTypes": ["doublepackset"], "watermark": "luffy and ace vol. 11", "type": "DON!!", "image": "x", "externalLinks": {"tcgPlayerId": 698315}},
		{"id": "698316_foil", "name": "DON!! Card", "setCode": "OP16", "rarity": "DON!!", "finish": "Foil", "variant": "Double Pack Set Vol. 11 Impel Down", "promoTypes": ["doublepackset"], "watermark": "impel down vol. 11", "type": "DON!!", "image": "x", "externalLinks": {"tcgPlayerId": 698316}}
	]
}}`

// TestLabelOverlapOnNumberedCards pins the partial-label scoring the numbered
// path falls through to: the catalog prefixes an event label with the
// base set's code, the storefront never writes it, and the whole-label test
// above therefore answers nothing at all. A set's DON!! cards, which carry
// no number, reach the same scoring.
func TestLabelOverlapOnNumberedCards(t *testing.T) {
	b, err := Load(strings.NewReader(labelOverlapFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   matcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "the winner's extra word picks the winner's copy",
			in: matcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-135 Release Event Winner",
				Edition: "Winner Pack"},
			want: "p-135_697484_foil",
		},
		{
			// The winner's label is the plain one with a word appended, and
			// the word it appends is the finishing place it was awarded for.
			// A listing saying the event and not the place is the printing
			// that was not awarded for one: the storefront writes the place
			// on the listings that have it, so leaving it out says which of
			// the pair this is. It is the plain copy that answers, never the
			// winner's, because the winner's is the scarce half and a
			// listing has to ask for it.
			desc: "the place left unsaid is the place not awarded",
			in: matcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-135 Release Event",
				Edition: "Promos"},
			want: "p-135_697483",
		},
		{
			desc: "a label said all but its treatment word wins its pair",
			in: matcher.InputCard{Name: "Boa Hancock", Variation: "ST17-004 Illustration Box Vol.1",
				Edition: "Promos"},
			want: "st17-004_623069_foil",
		},
		{
			desc: "a wording naming neither label settles nothing",
			in: matcher.InputCard{Name: "Tony Tony.Chopper", Variation: "P-101",
				Edition: "Promos"},
			err: true,
		},
		{
			desc: "and neither does one naming less of a label than it leaves out",
			in: matcher.InputCard{Name: "Tony Tony.Chopper", Variation: "P-101 2025",
				Edition: "Promos"},
			err: true,
		},
		{
			desc: "a DON!! naming no Double Pack volume is the set's own",
			in: matcher.InputCard{Name: "DON!! Card", Variation: "V.2",
				Edition: "OP16 - The Time of Battle"},
			want: "698314_foil",
		},
		{
			desc: "and its plain half when no treatment is said",
			in: matcher.InputCard{Name: "DON!! Card",
				Edition: "OP16 - The Time of Battle"},
			want: "698313_foil",
		},
		{
			desc: "but one carrying a Double Pack code is not",
			in: matcher.InputCard{Name: "DON!! Card", Variation: "DP11",
				Edition: "OP16 - The Time of Battle"},
			err: true,
		},
		{
			desc: "in either spelling of the code",
			in: matcher.InputCard{Name: "DON!! Card", Variation: "DP-11",
				Edition: "OP16 - The Time of Battle"},
			err: true,
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
