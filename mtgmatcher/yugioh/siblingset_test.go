package yugioh

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// siblingSetFixture mirrors a set family: one name split into editions filed
// as sets of their own, whose codes extend the family's and whose numbers say
// which edition a printing belongs to.
//
// The set codes, collector numbers and rarities are the catalog's.
const siblingSetFixture = `{
	"game": "yugioh",
	"sets": {
		"MVP1":     {"name": "The Dark Side of Dimensions Movie Pack", "releaseDate": "2017-03-10"},
		"MVP1-ENG": {"name": "The Dark Side of Dimensions Movie Pack: Gold Edition", "releaseDate": "2017-09-15"},
		"MVP1-ENS": {"name": "The Dark Side of Dimensions Movie Pack: Secret Edition", "releaseDate": "2018-03-09"}
	},
	"cards": [
		{"id": "mvp1-en054_unl", "name": "Dark Magician", "number": "MVP1-EN054", "setCode": "MVP1", "rarity": "Ultra Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 120956}},
		{"id": "mvp1-eng54_unl", "name": "Dark Magician", "number": "MVP1-ENG54", "setCode": "MVP1-ENG", "rarity": "Gold Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 126694}},
		{"id": "mvp1-engv3_unl", "name": "Dark Magician", "number": "MVP1-ENGV3", "setCode": "MVP1-ENG", "rarity": "Gold Secret Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 126695}},
		{"id": "mvp1-ens54_unl", "name": "Dark Magician", "number": "MVP1-ENS54", "setCode": "MVP1-ENS", "rarity": "Secret Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 207865}}
	]
}`

// TestSiblingSetNamed pins the edition a number names when the shelf names
// the family. A storefront sells every edition under the family's name and
// says which is which in the number alone, so the family's own printing
// answered all of them - one card carrying four prices.
func TestSiblingSetNamed(t *testing.T) {
	b, err := Load(strings.NewReader(siblingSetFixture))
	if err != nil {
		t.Fatal(err)
	}

	const shelf = "The Dark Side of Dimensions Movie Pack"
	tests := []struct {
		desc      string
		variation string
		want      string
	}{
		{"the family's own printing", "MVP1-EN054 Ultra Rare", "mvp1-en054_unl"},
		{"the gold edition it is sold beside", "MVP1-ENG54 Gold Rare", "mvp1-eng54_unl"},
		{"its secret rare, numbered apart again", "MVP1-ENGV3 Gold Secret Rare", "mvp1-engv3_unl"},
		{"and the secret edition", "MVP1-ENS54 Secret Rare", "mvp1-ens54_unl"},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := mtgmatcher.InputCard{Name: "Dark Magician", Edition: shelf, Variation: test.variation}
			uuid, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%q) = %v, want %q", test.variation, err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match(%q) = %q (%v), want %q", test.variation, uuid, co, test.want)
			}
		})
	}
}
