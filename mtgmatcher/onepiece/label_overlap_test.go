package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// labelOverlapFixture mirrors the event pairs the catalog prefixes with the
// base set's code. P-135 is the participation card beside the winner's copy
// of it; ST17-004 is the same shape around a treatment word. P-101 is what
// must keep refusing: a set's promotion cards are told apart by nothing the
// wording says.
const labelOverlapFixture = `{
	"game": "onepiece",
	"sets": {
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"}
	},
	"cards": [
		{"id": "p-135_697483", "name": "Monkey.D.Luffy", "number": "P-135", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "variant": "OP16 Release Event", "image": "x", "externalLinks": {"tcgPlayerId": 697483}},
		{"id": "p-135_697484_foil", "name": "Monkey.D.Luffy", "number": "P-135", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "OP16 Release Event Winner", "image": "x", "externalLinks": {"tcgPlayerId": 697484}},
		{"id": "st17-004_623069_foil", "name": "Boa Hancock", "number": "ST17-004", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Illustration Box Vol.1 Textured", "image": "x", "externalLinks": {"tcgPlayerId": 623069}},
		{"id": "st17-004_712035_foil", "name": "Boa Hancock", "number": "ST17-004", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Dash Pack 2025", "image": "x", "externalLinks": {"tcgPlayerId": 712035}},
		{"id": "p-101_700001", "name": "Tony Tony.Chopper", "number": "P-101", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "variant": "Promotion Card Set 2025 Vol. 1", "image": "x", "externalLinks": {"tcgPlayerId": 700001}},
		{"id": "p-101_700002_foil", "name": "Tony Tony.Chopper", "number": "P-101", "setCode": "OP-PR", "rarity": "PR", "finish": "Foil", "variant": "Store Tournament 2025 Vol. 4", "image": "x", "externalLinks": {"tcgPlayerId": 700002}}
	]
}`

// TestLabelOverlapOnNumberedCards pins the partial-label scoring the numbered
// path now falls through to: the catalog prefixes an event label with the
// base set's code, the storefront never writes it, and the whole-label test
// above therefore answers nothing at all.
func TestLabelOverlapOnNumberedCards(t *testing.T) {
	b, err := Load(strings.NewReader(labelOverlapFixture))
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
			desc: "the winner's extra word picks the winner's copy",
			in: mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-135 Release Event Winner",
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
			in: mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "P-135 Release Event",
				Edition: "Promos"},
			want: "p-135_697483",
		},
		{
			desc: "a label said all but its treatment word wins its pair",
			in: mtgmatcher.InputCard{Name: "Boa Hancock", Variation: "ST17-004 Illustration Box Vol.1",
				Edition: "Promos"},
			want: "st17-004_623069_foil",
		},
		{
			desc: "a wording naming neither label settles nothing",
			in: mtgmatcher.InputCard{Name: "Tony Tony.Chopper", Variation: "P-101",
				Edition: "Promos"},
			err: true,
		},
		{
			desc: "and neither does one naming less of a label than it leaves out",
			in: mtgmatcher.InputCard{Name: "Tony Tony.Chopper", Variation: "P-101 2025",
				Edition: "Promos"},
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
