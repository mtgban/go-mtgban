package yugioh

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// loosePrefixFixture holds what makes the relaxation both useful and
// dangerous. DL files its Duelist League volumes as DL1- and DL5-, and
// cardtrader writes those volumes as "1-" and "5-"; MC1 numbers an unrelated
// card MC1-EN002, whose set code carries the very same digit. The Duelist
// League volume cardtrader calls "1-E002" does not exist in DL at all, so
// nothing but the edition stands between that listing and the Master
// Collection card.
const loosePrefixFixture = `{
	"game": "yugioh",
	"sets": {
		"DL":   {"name": "Duelist League Promo", "releaseDate": "2010-09-01", "type": "promo"},
		"MC1":  {"name": "Master Collection Volume 1", "releaseDate": "2004-03-01", "type": "promo"},
		"G358": {"name": "Yu-Gi-Oh! R Manga Promo", "releaseDate": "2005-01-01", "type": "promo"}
	},
	"cards": [
		{"id": "dl5-en001_25392_lim", "name": "Restructer Revolution", "number": "DL5-EN001", "setCode": "DL", "rarity": "Super Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 25392}},
		{"id": "dl1-001_25300_lim", "name": "Thousand-Eyes Restrict", "number": "DL1-001", "setCode": "DL", "rarity": "Super Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 25300}},
		{"id": "mc1-en002_25375_lim", "name": "Barrel Dragon", "number": "MC1-EN002", "setCode": "MC1", "rarity": "Secret Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 25375}},
		{"id": "yr05-en001_38640_lim", "name": "Alector, Sovereign of Birds", "number": "YR05-EN001", "setCode": "G358", "rarity": "Ultra Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 38640}}
	]
}`

// TestLoosePrefixNeedsItsEdition pins both halves of the rule: a volume index
// written where the set code belongs reaches its printing inside the set the
// listing names, and reaches nothing at all outside it.
func TestLoosePrefixNeedsItsEdition(t *testing.T) {
	b, err := Load(strings.NewReader(loosePrefixFixture))
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
			desc: "the volume index reads as the set code's digits",
			in: mtgmatcher.InputCard{Name: "Restructer Revolution", Variation: "5-001 Super Rare",
				Edition: "Duelist League Promo"},
			want: "dl5-en001_25392_lim",
		},
		{
			desc: "a two-digit index loses its padding the way any number does",
			in: mtgmatcher.InputCard{Name: "Alector, Sovereign of Birds", Variation: "05-001 Ultra Rare",
				Edition: "Yu-Gi-Oh! R Manga Promo"},
			want: "yr05-en001_38640_lim",
		},
		{
			desc: "a card its named set never printed stays refused",
			in: mtgmatcher.InputCard{Name: "Barrel Dragon", Variation: "1-E002 Super Rare",
				Edition: "Duelist League Promo"},
			err: true,
		},
		{
			desc: "and an edition naming no set buys nothing",
			in: mtgmatcher.InputCard{Name: "Barrel Dragon", Variation: "1-E002 Super Rare",
				Edition: "Promo"},
			err: true,
		},
		{
			desc: "the set's own dashed numbering is untouched",
			in: mtgmatcher.InputCard{Name: "Thousand-Eyes Restrict", Variation: "DL1-001 Super Rare",
				Edition: "Duelist League Promo"},
			want: "dl1-001_25300_lim",
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
