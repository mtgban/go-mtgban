package yugioh

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// qualifiedNameFixture mirrors the two shapes the catalog spells a qualifier
// inside a name in. LC02 files the alternate art as "Elemental HERO Avian
// (Alternate Art)" beside the plain card at another number, and SS03 keeps
// the deck letter in "White Elephant's Gift (A)" beside a sibling that had
// its own letter distilled into a variant. SGX1 is the shape that must keep
// refusing: the number compare reads only the trailing digits, so ENA12 and
// ENC12 are one number to it and nothing tells them apart. SBC1 is the same
// shape rescued by exactness - a storefront that writes the full number has
// said which deck letter it means, and only the shorthand is left guessing.
const qualifiedNameFixture = `{
	"game": "yugioh",
	"sets": {
		"LC02":  {"name": "Legendary Collection 2: Mega Pack", "releaseDate": "2012-10-02"},
		"SS03":  {"name": "Speed Duel Decks: Ultimate Predators", "releaseDate": "2019-08-15"},
		"G2970": {"name": "Speed Duel GX: Duel Academy Box", "releaseDate": "2022-06-09"},
		"SBC1":  {"name": "Speed Duel: Streets of Battle City", "releaseDate": "2024-01-01"},
		"CRV":   {"name": "Cybernetic Revolution", "releaseDate": "2005-08-17"}
	},
	"cards": [
		{"id": "lcgx-en001_56799_unl", "name": "Elemental HERO Avian", "number": "LCGX-EN001", "setCode": "LC02", "rarity": "Common", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 56799}},
		{"id": "lcgx-en002_56800_unl", "name": "Elemental HERO Avian (Alternate Art)", "number": "LCGX-EN002", "setCode": "LC02", "rarity": "Secret Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 56800}},
		{"id": "lcgx-en182_56745_unl", "name": "Cyber End Dragon (Alternate Art)", "number": "LCGX-EN182", "setCode": "LC02", "rarity": "Secret Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 56745}},
		{"id": "crv-en036_36001_1e", "name": "Cyber End Dragon", "number": "CRV-EN036", "setCode": "CRV", "rarity": "Ultra Rare", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 36001}},
		{"id": "ss03-ena22_196260_1e", "name": "White Elephant's Gift (A)", "number": "SS03-ENA22", "setCode": "SS03", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 196260}},
		{"id": "ss03-enb24_196261_1e", "name": "White Elephant's Gift", "number": "SS03-ENB24", "setCode": "SS03", "rarity": "Common", "variant": "B", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 196261}},
		{"id": "sgx1-ena12_286001_1e", "name": "Polymerization (A)", "number": "SGX1-ENA12", "setCode": "G2970", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 286001}},
		{"id": "sgx1-enc12_286002_1e", "name": "Polymerization", "number": "SGX1-ENC12", "setCode": "G2970", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 286002}},
		{"id": "sbc1-ena01_512195_1e", "name": "Dark Magician (A)", "number": "SBC1-ENA01", "setCode": "SBC1", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 512195}},
		{"id": "sbc1-ena01_512196_1e", "name": "Dark Magician (A)", "number": "SBC1-ENA01", "setCode": "SBC1", "rarity": "Secret Rare", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 512196}},
		{"id": "sbc1-eng01_512387_1e", "name": "Dark Magician", "number": "SBC1-ENG01", "setCode": "SBC1", "rarity": "Secret Rare", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 512387}},
		{"id": "sbc1-eng10_512388_1e", "name": "Dark Magician", "number": "SBC1-ENG10", "setCode": "SBC1", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 512388}}
	]
}`

// TestQualifiedNameAdoptedByNumber pins that a name the catalog keeps its
// qualifier inside is reachable from the bare name a storefront writes, and
// only where the number leaves nothing to guess.
func TestQualifiedNameAdoptedByNumber(t *testing.T) {
	b, err := Load(strings.NewReader(qualifiedNameFixture))
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
			desc: "the number picks the decorated sibling",
			in: mtgmatcher.InputCard{Name: "Elemental HERO Avian", Variation: "002sec Secret Rare",
				Edition: "Legendary Collection 2: Mega Pack"},
			want: "lcgx-en002_56800_unl",
		},
		{
			desc: "and leaves the plain one at its own number",
			in: mtgmatcher.InputCard{Name: "Elemental HERO Avian", Variation: "001 Common",
				Edition: "Legendary Collection 2: Mega Pack"},
			want: "lcgx-en001_56799_unl",
		},
		{
			desc: "a deck letter kept inside the name reads the same way",
			in: mtgmatcher.InputCard{Name: "White Elephant's Gift", Variation: "022 Common",
				Edition: "Speed Duel Decks: Ultimate Predators"},
			want: "ss03-ena22_196260_1e",
		},
		{
			desc: "the sibling spelled plainly keeps its own number",
			in: mtgmatcher.InputCard{Name: "White Elephant's Gift", Variation: "024 Common",
				Edition: "Speed Duel Decks: Ultimate Predators"},
			want: "ss03-enb24_196261_1e",
		},
		{
			desc: "a set printing the bare name at that very number keeps it",
			in: mtgmatcher.InputCard{Name: "Polymerization", Variation: "012 Common",
				Edition: "Speed Duel GX: Duel Academy Box"},
			want: "sgx1-enc12_286002_1e",
		},
		{
			desc: "a full number the set spells exactly reaches its own deck letter",
			in: mtgmatcher.InputCard{Name: "Dark Magician (Purple Armor)", Variation: "SBC1-ENA01 Secret Rare",
				Edition: "Speed Duel: Streets of Battle City"},
			want: "sbc1-ena01_512196_1e",
		},
		{
			desc: "written bare it adopts the same printing",
			in: mtgmatcher.InputCard{Name: "Dark Magician", Variation: "SBC1-ENA01 Common",
				Edition: "Speed Duel: Streets of Battle City"},
			want: "sbc1-ena01_512195_1e",
		},
		{
			desc: "while the bare name at its own exact number keeps it",
			in: mtgmatcher.InputCard{Name: "Dark Magician (Red Armor)", Variation: "SBC1-ENG01 Secret Rare",
				Edition: "Speed Duel: Streets of Battle City"},
			want: "sbc1-eng01_512387_1e",
		},
		{
			desc: "the decorated name still answers to itself",
			in: mtgmatcher.InputCard{Name: "Cyber End Dragon (Alternate Art)", Variation: "182sec Secret Rare",
				Edition: "Legendary Collection 2: Mega Pack"},
			want: "lcgx-en182_56745_unl",
		},
		{
			desc: "an input naming no number is never renamed",
			in: mtgmatcher.InputCard{Name: "Cyber End Dragon", Variation: "",
				Edition: "Legendary Collection 2: Mega Pack"},
			want: "crv-en036_36001_1e",
		},
		{
			desc: "and neither is one naming no set",
			in: mtgmatcher.InputCard{Name: "Cyber End Dragon", Variation: "182sec Secret Rare",
				Edition: "Promo"},
			err: true,
		},
		{
			desc: "the number reaches it once the set is named",
			in: mtgmatcher.InputCard{Name: "Cyber End Dragon", Variation: "182sec Secret Rare",
				Edition: "Legendary Collection 2: Mega Pack"},
			want: "lcgx-en182_56745_unl",
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
