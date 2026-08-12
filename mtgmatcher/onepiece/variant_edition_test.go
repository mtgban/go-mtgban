package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// variantFixture mirrors the shape of the real catalog that makes One
// Piece different from the other games: a variant printing shares its
// collector number with the base card but is filed in another set - the
// alternate arts in PRB-01, the event printings in OP-PR - while
// storefronts file all of them under the base card's set. Brannew is the
// other half of the contract: two plain printings of one number in
// different sets, which only the edition can tell apart.
const variantFixture = `{
	"game": "onepiece",
	"sets": {
		"OP01":   {"name": "Romance Dawn", "releaseDate": "2022-12-02"},
		"OP03":   {"name": "Pillars of Strength", "releaseDate": "2023-06-30"},
		"ST-19":  {"name": "Smoker", "releaseDate": "2024-03-08"},
		"OP-PR":  {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02"},
		"PRB-01": {"name": "Premium Booster -The Best-", "releaseDate": "2024-08-30"},
		"EB-01":  {"name": "Extra Booster: Memorial Collection", "releaseDate": "2024-05-31"},
		"OP07":   {"name": "500 Years in the Future", "releaseDate": "2024-06-28"},
		"ST-24":  {"name": "Starter Deck 24: GREEN Jewelry Bonney", "releaseDate": "2024-11-01"},
		"PRB-02": {"name": "Premium Booster -The Best- Vol. 2", "releaseDate": "2025-08-01"}
	},
	"cards": [
		{"id": "op01-016_base", "name": "Nami", "number": "OP01-016", "setCode": "OP01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op01-016_manga", "name": "Nami", "number": "OP01-016", "setCode": "PRB-01", "rarity": "C", "finish": "Normal", "variant": "Manga", "image": "x"},
		{"id": "op01-084_base", "name": "Mr.2.Bon.Kurei (Bentham)", "number": "OP01-084", "setCode": "OP01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op01-084_promo", "name": "Mr.2.Bon.Kurei (Bentham)", "number": "OP01-084", "setCode": "OP-PR", "rarity": "P", "finish": "Normal", "variant": "Store Championship Participation Pack Vol. 2", "image": "x"},
		{"id": "op03-089_base", "name": "Brannew", "number": "OP03-089", "setCode": "OP03", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op03-089_deck", "name": "Brannew", "number": "OP03-089", "setCode": "ST-19", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "eb01-012_base", "name": "Cavendish", "number": "EB01-012", "setCode": "EB-01", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "eb01-012_promo", "name": "Cavendish", "number": "EB01-012", "setCode": "OP-PR", "rarity": "P", "finish": "Normal", "variant": "Treasure Cup 2024", "image": "x"},
		{"id": "op07-031_base", "name": "Bartolomeo", "number": "OP07-031", "setCode": "OP07", "rarity": "C", "finish": "Normal", "image": "x"},
		{"id": "op07-031_st", "name": "Bartolomeo", "number": "OP07-031", "setCode": "ST-24", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"},
		{"id": "op07-031_prb", "name": "Bartolomeo", "number": "OP07-031", "setCode": "PRB-02", "rarity": "C", "finish": "Normal", "variant": "Reprint", "image": "x"}
	]
}`

func variantBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(variantFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestEditionKeepsVariantPrintings pins the interaction between the
// edition and the variant tiering. CoolStuffInc names the base card's set
// for every printing of it, so an edition that resolves would otherwise
// narrow the candidates to that set and delete the very printing the rest
// of the wording asks for - before FilterCards ever tiers them.
func TestEditionKeepsVariantPrintings(t *testing.T) {
	b := variantBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "qualifier reaches the variant filed in another set",
			in:   mtgmatcher.InputCard{Name: "Nami (OP01-016) (Manga)", Edition: "OP01 - Romance Dawn"},
			want: "op01-016_manga",
		},
		{
			desc: "the same wording without an edition is unchanged",
			in:   mtgmatcher.InputCard{Name: "Nami (OP01-016) (Manga)"},
			want: "op01-016_manga",
		},
		{
			desc: "a bare input still keeps the base printing",
			in:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016", Edition: "OP01 - Romance Dawn"},
			want: "op01-016_base",
		},
		{
			desc: "a letter tail reaches the variant despite the edition",
			in:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham)", Variation: "OP01-084a", Edition: "OP01 - Romance Dawn"},
			want: "op01-084_promo",
		},
		{
			desc: "an event qualifier reaches the promo despite the edition",
			in:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham) (Store Championship Participation Pack Vol. 2)", Variation: "OP01-084", Edition: "OP01 - Romance Dawn"},
			want: "op01-084_promo",
		},
		{
			// A storefront's edition need not equal the set name to have
			// narrowed the candidates: "Memorial Collection" is merely
			// contained in "Extra Booster: Memorial Collection", and the
			// looser match deleted the promo just as thoroughly.
			desc: "a partially spelled edition still reaches the promo",
			in:   mtgmatcher.InputCard{Name: "Cavendish (Treasure Cup 2024)", Variation: "EB01-012", Edition: "EB01 - Memorial Collection"},
			want: "eb01-012_promo",
		},
		{
			// The edition must keep doing the job it was widened for:
			// telling two plain printings of one number apart.
			desc: "the edition still separates two base printings",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089", Edition: "OP03 - Pillars of Strength"},
			want: "op03-089_base",
		},
		{
			desc: "the other set's reprint resolves through its own edition",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089", Edition: "ST-19 - Smoker"},
			want: "op03-089_deck",
		},
		{
			desc: "negative: without an edition the two base printings alias",
			in:   mtgmatcher.InputCard{Name: "Brannew", Variation: "OP03-089"},
			err:  true,
		},
		{
			// One label printed in several sets: the widened pool holds both
			// Reprints, and only the edition can hand back the named one.
			desc: "the edition tiebreaks a label shared across sets",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "PRB-02 - Premium Booster -The Best- Vol. 2"},
			want: "op07-031_prb",
		},
		{
			desc: "the same label resolves to the other set through its edition",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "ST-24 - Starter Deck 24: GREEN Jewelry Bonney"},
			want: "op07-031_st",
		},
		{
			// cardtrader demands the variant with a letter tail and names the
			// variant's own set; the tail widens the pool past the edition,
			// and the tiebreak brings it back.
			desc: "a letter tail with the variant's own edition stays in that set",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo", Variation: "OP07-031a", Edition: "ST-24 - Starter Deck 24: GREEN Jewelry Bonney"},
			want: "op07-031_st",
		},
		{
			desc: "the base printing still resolves through its own edition",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo", Variation: "OP07-031", Edition: "OP07 - 500 Years in the Future"},
			want: "op07-031_base",
		},
		{
			desc: "negative: a shared label with a foreign edition stays ambiguous",
			in:   mtgmatcher.InputCard{Name: "Bartolomeo (Reprint)", Variation: "OP07-031", Edition: "OP07 - 500 Years in the Future"},
			err:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if test.err {
				if err == nil {
					t.Fatalf("Match = %q, want an error", uuid)
				}
				return
			}
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
