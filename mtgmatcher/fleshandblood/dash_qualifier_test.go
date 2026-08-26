package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// dashQualifierFixture holds the two shapes the respell has to cross: a card
// the catalog files unqualified in one set and pitch-qualified in another
// (Hyper Driver), and a pitch cycle whose members sit on consecutive numbers
// (Impenetrable Belief), where the number and the storefront's own name
// disagree.
const dashQualifierFixture = `{
	"game": "fleshandblood",
	"sets": {
		"ARC": {"name": "Arcane Rising", "releaseDate": "2020-06-05"},
		"DYN": {"name": "Dynasty", "releaseDate": "2022-03-25"},
		"MON": {"name": "Monarch", "releaseDate": "2021-05-07"}
	},
	"cards": [
		{"id": "arc036_225629_unl", "name": "Hyper Driver", "number": "ARC036", "setCode": "ARC", "rarity": "Rare", "finish": "Normal", "image": "x"},
		{"id": "dyn110_452817_unl", "name": "Hyper Driver (Red)", "number": "DYN110", "setCode": "DYN", "rarity": "Rare", "finish": "Normal", "image": "x"},
		{"id": "mon075_237817_unl", "name": "Impenetrable Belief (Red)", "number": "MON075", "setCode": "MON", "rarity": "Common", "finish": "Normal", "image": "x"},
		{"id": "mon076_237818_unl", "name": "Impenetrable Belief (Yellow)", "number": "MON076", "setCode": "MON", "rarity": "Common", "finish": "Normal", "image": "x"},
		{"id": "mon077_237819_unl", "name": "Impenetrable Belief (Blue)", "number": "MON077", "setCode": "MON", "rarity": "Common", "finish": "Normal", "image": "x"}
	]
}`

// TestDashSpelledQualifier pins that a pitch qualifier written with a dash
// answers exactly as the parenthesised spelling of the same listing does.
// Normalize folds the two together, so the dashed name passes the canonical
// lookup untouched and the respell that reads the qualifier never sees one.
func TestDashSpelledQualifier(t *testing.T) {
	b, err := Load(strings.NewReader(dashQualifierFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc       string
		dash       mtgmatcher.InputCard
		paren      mtgmatcher.InputCard
		want       string
		wantRename string
	}{
		{
			desc:       "a dashed qualifier drops onto the unqualified printing its number names",
			dash:       mtgmatcher.InputCard{Name: "Hyper Driver - Red", Variation: "ARC036", Edition: "Arcane Rising"},
			paren:      mtgmatcher.InputCard{Name: "Hyper Driver (Red)", Variation: "ARC036", Edition: "Arcane Rising"},
			want:       "arc036_225629_unl",
			wantRename: "Hyper Driver",
		},
		{
			desc:       "and keeps the qualified printing where the number names that one",
			dash:       mtgmatcher.InputCard{Name: "Hyper Driver - Red", Variation: "DYN110", Edition: "Dynasty"},
			paren:      mtgmatcher.InputCard{Name: "Hyper Driver (Red)", Variation: "DYN110", Edition: "Dynasty"},
			want:       "dyn110_452817_unl",
			wantRename: "Hyper Driver (Red)",
		},
		{
			desc:       "the number outranks the storefront's own pitch, as it always has",
			dash:       mtgmatcher.InputCard{Name: "Impenetrable Belief - Red", Variation: "MON076", Edition: "Monarch"},
			paren:      mtgmatcher.InputCard{Name: "Impenetrable Belief (Red)", Variation: "MON076", Edition: "Monarch"},
			want:       "mon076_237818_unl",
			wantRename: "Impenetrable Belief (Yellow)",
		},
		{
			desc:       "a dashed name agreeing with its number is left where it is",
			dash:       mtgmatcher.InputCard{Name: "Impenetrable Belief - Blue", Variation: "MON077", Edition: "Monarch"},
			paren:      mtgmatcher.InputCard{Name: "Impenetrable Belief (Blue)", Variation: "MON077", Edition: "Monarch"},
			want:       "mon077_237819_unl",
			wantRename: "Impenetrable Belief (Blue)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dash := tt.dash
			id, err := b.Match(&dash)
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.dash.Name, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q) = %q, want %q", tt.dash.Name, id, tt.want)
			}
			if dash.Name != tt.wantRename {
				t.Errorf("Match(%q) renamed to %q, want %q", tt.dash.Name, dash.Name, tt.wantRename)
			}
			paren := tt.paren
			other, err := b.Match(&paren)
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.paren.Name, err)
			}
			if other != id {
				t.Errorf("Match(%q) = %q, but Match(%q) = %q; the two spellings must agree",
					tt.paren.Name, other, tt.dash.Name, id)
			}
		})
	}
}
