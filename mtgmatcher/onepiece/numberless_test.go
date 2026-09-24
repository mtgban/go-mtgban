package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// numberlessFixture carries a card the catalog files with no number beside
// one it numbers. The trophy card is the builder's own entry for product
// 719668, which carries no Number and no CardType, verbatim.
const numberlessFixture = `{"data": {
	"game": "onepiece",
	"sets": {
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02", "type": "promo"}
	},
	"cards": [
		{"id": "p-041_531486", "name": "Monkey.D.Luffy", "number": "P-041", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 531486}},
		{"color": "", "externalLinks": {"tcgPlayerId": 719668}, "finish": "Foil", "id": "719668_foil", "image": "https://tcgplayer-cdn.tcgplayer.com/product/719668_400w.jpg", "name": "Flame-Flame Fruit Trophy Card", "promoTypes": ["flameflamefruitcoliseum"], "rarity": "None", "setCode": "OP-PR", "type": "", "variant": "Flame-Flame Fruit Coliseum"}
	]
}}`

// TestLoadReadsANumberlessCard: one card with no number is a card, not a
// wrong file. Refusing it refused the whole datastore, every card with it.
func TestLoadReadsANumberlessCard(t *testing.T) {
	b, err := Load(strings.NewReader(numberlessFixture))
	if err != nil {
		t.Fatalf("Load = %v; a card with no number must not refuse the file", err)
	}

	for _, test := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "the numberless card is reached by its name",
			in: mtgmatcher.InputCard{
				Name: "Flame-Flame Fruit Trophy Card", Variation: "Flame-Flame Fruit Coliseum",
				Edition: "One Piece Promotion Cards", Foil: true,
			},
			want: "719668_foil",
		},
		{
			desc: "and the numbered card beside it is untouched",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "P-041", Edition: "One Piece Promotion Cards",
			},
			want: "p-041_531486",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				t.Errorf("Match = %q, want %q", uuid, test.want)
			}
		})
	}
}

// TestLoadStillRefusesAWrongFile: the number was the one optional field. A
// card with no id, name or finish is still not a One Piece datastore.
func TestLoadStillRefusesAWrongFile(t *testing.T) {
	for _, field := range []string{`"id": "719668_foil", `, `"name": "Flame-Flame Fruit Trophy Card", `, `"finish": "Foil", `} {
		broken := strings.Replace(numberlessFixture, field, "", 1)
		if broken == numberlessFixture {
			t.Fatalf("fixture does not carry %s", field)
		}
		if _, err := Load(strings.NewReader(broken)); err == nil {
			t.Errorf("Load accepted a card without %s", strings.TrimSuffix(field, ", "))
		}
	}
}
