package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// fusedSlashFixture mirrors the two spellings the catalog writes a fused
// card's pair in. Frostbite and Teklovossen are filed with a single slash,
// Spectral Shield with two, and the standalone Ash beside DRO002 is the
// printing a face number must keep reaching on its own.
const fusedSlashFixture = `{
	"game": "fleshandblood",
	"sets": {
		"UPR": {"name": "Uprising", "releaseDate": "2022-10-14"},
		"EVO": {"name": "Bright Lights", "releaseDate": "2023-08-25"},
		"DRO": {"name": "Blitz Deck: Uprising - Dromai", "releaseDate": "2022-10-14"},
		"MST": {"name": "Part the Mistveil", "releaseDate": "2024-09-13"}
	},
	"cards": [
		{"id": "upr150-upr183_275838", "name": "Frostbite // Helio's Mitre", "number": "UPR150/UPR183", "setCode": "UPR", "rarity": "Token", "finish": "Normal", "image": "x"},
		{"id": "evo008-evo006_510040", "name": "Teklovossen // Banksy", "number": "EVO008/EVO006", "setCode": "EVO", "rarity": "Token", "finish": "Normal", "image": "x"},
		{"id": "mst026-mst027_600001", "name": "Slither // Enigma", "number": "MST026 // MST027", "setCode": "MST", "rarity": "Token", "finish": "Normal", "image": "x"},
		{"id": "dro002-dro003_276817", "name": "Ash // Aether Ashwing", "number": "DRO002/DRO003", "setCode": "DRO", "rarity": "Token", "finish": "Normal", "image": "x"},
		{"id": "dro002_704533", "name": "Ash", "number": "DRO002", "setCode": "DRO", "rarity": "Token", "finish": "Normal", "image": "x"}
	]
}`

// TestFusedFaceSingleSlash pins that a face number reaches its fused
// printing whichever separator the catalog wrote the pair with. numberMatches
// and sortedPair have always read one slash as a pair separator; fusedFaceAt
// demanded two, so the eight printings filed with one were unreachable from
// either of their own faces.
func TestFusedFaceSingleSlash(t *testing.T) {
	b, err := Load(strings.NewReader(fusedSlashFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "a face of a single-slash pair reaches its printing",
			in:   mtgmatcher.InputCard{Name: "Frostbite", Variation: "UPR150", Edition: "Uprising"},
			want: "upr150-upr183_275838",
		},
		{
			desc: "and so does the other face, numbered out of order",
			in:   mtgmatcher.InputCard{Name: "Banksy", Variation: "EVO006", Edition: "Bright Lights"},
			want: "evo008-evo006_510040",
		},
		{
			desc: "a double-slash pair keeps answering as it did",
			in:   mtgmatcher.InputCard{Name: "Enigma", Variation: "MST027", Edition: "Part the Mistveil"},
			want: "mst026-mst027_600001",
		},
		{
			desc: "a face with a standalone printing at its number keeps it",
			in:   mtgmatcher.InputCard{Name: "Ash", Variation: "DRO002", Edition: "Blitz Deck: Uprising - Dromai"},
			want: "dro002_704533",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%q, %q) = %v", tt.in.Name, tt.in.Variation, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q, %q) = %q, want %q", tt.in.Name, tt.in.Variation, id, tt.want)
			}
		})
	}
}
