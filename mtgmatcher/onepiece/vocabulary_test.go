package onepiece

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// vocabularyFixture mirrors the numbers whose printings a storefront and the
// catalog call by different names, and the one number that files both names
// at once - the case the fallback has to leave alone.
//
// The collector numbers, product ids and labels are the catalog's.
const vocabularyFixture = `{
	"game": "onepiece",
	"sets": {
		"OP13": {"name": "Carrying On His Will", "releaseDate": "2025-11-21"},
		"OP16": {"name": "The Time of Battle", "releaseDate": "2026-05-22"},
		"OP17": {"name": "The World's Strongest Warriors", "releaseDate": "2026-08-14"},
		"OP09": {"name": "Emperors in the New World", "releaseDate": "2024-12-27"},
		"PRB-02": {"name": "Premium Booster -The Best- Vol. 2", "releaseDate": "2025-08-01"}
	},
	"cards": [
		{"id": "op13-118_657400_foil", "name": "Monkey.D.Luffy", "number": "OP13-118", "setCode": "OP13", "rarity": "SEC", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 657400}},
		{"id": "op13-118_657401_foil", "name": "Monkey.D.Luffy", "number": "OP13-118", "setCode": "OP13", "rarity": "SEC", "finish": "Foil", "image": "x", "variant": "Red Super Alternate Art", "externalLinks": {"tcgPlayerId": 657401}},
		{"id": "op13-118_657402_foil", "name": "Monkey.D.Luffy", "number": "OP13-118", "setCode": "OP13", "rarity": "SEC", "finish": "Foil", "variant": "Super Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 657402}},
		{"id": "op13-118_657403_foil", "name": "Monkey.D.Luffy", "number": "OP13-118", "setCode": "OP13", "rarity": "SEC", "finish": "Foil", "variant": "Parallel", "image": "x", "externalLinks": {"tcgPlayerId": 657403}},
		{"id": "op16-011_695995_foil", "name": "Vista", "number": "OP16-011", "setCode": "OP16", "rarity": "SR", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 695995}},
		{"id": "op16-011_695996_foil", "name": "Vista", "number": "OP16-011", "setCode": "OP16", "rarity": "SR", "finish": "Foil", "variant": "TR", "image": "x", "externalLinks": {"tcgPlayerId": 695996}},
		{"id": "op17-062_700001_foil", "name": "Kaido", "number": "OP17-062", "setCode": "OP17", "rarity": "SR", "finish": "Foil", "variant": "Manga", "image": "x", "externalLinks": {"tcgPlayerId": 700001}},
		{"id": "op17-062_700002_foil", "name": "Kaido", "number": "OP17-062", "setCode": "OP17", "rarity": "SR", "finish": "Foil", "variant": "Super Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 700002}},
		{"id": "op09-078_597018_foil", "name": "Gum-Gum Giant", "number": "OP09-078", "setCode": "OP09", "rarity": "R", "finish": "Foil", "image": "x", "externalLinks": {"tcgPlayerId": 597018}},
		{"id": "op09-078_653835_foil", "name": "Gum-Gum Giant", "number": "OP09-078", "setCode": "PRB-02", "rarity": "R", "finish": "Foil", "variant": "Alternate Art", "image": "x", "externalLinks": {"tcgPlayerId": 653835}}
	]
}`

// TestCatalogVocabulary pins the words a storefront uses for a treatment the
// catalog files under another name, and the one number that carries both
// names at once: the fallback has to stand aside there, or a wording that
// could not have been plainer aliases away.
func TestCatalogVocabulary(t *testing.T) {
	b, err := Load(strings.NewReader(vocabularyFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "the manga art the catalog files as a super alternate art",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "OP13-118 Alternate Art Manga",
				Edition: "OP13 - Carrying On His Will", Foil: true,
			},
			want: "op13-118_657402_foil",
		},
		{
			// The same listing with the treatment written behind it: the
			// storefront sells this at $8400 and the one above at $1950.
			// The catalog crosses the two into one label, "Red Super
			// Alternate Art", and that is the printing being sold - not
			// the plain Parallel standing beside it, which the storefront
			// sells for $25. The words of the crossed label are all said,
			// and no other label carrying the manga art says more of them.
			desc: "and the parallel of it is told apart by what it added",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "OP13-118 Alternate Art Manga Red Parallel",
				Edition: "OP13 - Carrying On His Will", Foil: true,
			},
			want: "op13-118_657401_foil",
		},
		{
			desc: "and the plain printing beside it is still the plain one",
			in: mtgmatcher.InputCard{
				Name: "Monkey.D.Luffy", Variation: "OP13-118",
				Edition: "OP13 - Carrying On His Will", Foil: true,
			},
			want: "op13-118_657400_foil",
		},
		{
			desc: "the treasure rare the catalog files by its two letters",
			in: mtgmatcher.InputCard{
				Name: "Vista", Variation: "OP16-011 OP16 Treasure Rare",
				Edition: "OP16 - The Time of Battle", Foil: true,
			},
			want: "op16-011_695996_foil",
		},
		{
			// Both names at one number: the storefront's own word names a
			// printing outright, so the catalog's spelling is never asked
			// for and the two do not tie.
			desc: "a number filing both names answers the word that was said",
			in: mtgmatcher.InputCard{
				Name: "Kaido", Variation: "OP17-062 Manga",
				Edition: "OP17 - The World's Strongest Warriors", Foil: true,
			},
			want: "op17-062_700001_foil",
		},
		{
			// Premium Booster Vol. 2 reprints in manga art and files every
			// one as an alternate art. The edition names the set the card
			// was first printed in, which is the printing it is not.
			desc: "a set's own word for the treatment reaches its printing",
			in: mtgmatcher.InputCard{
				Name: "Gum-Gum Giant", Variation: "Manga",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-078_653835_foil",
		},
		{
			desc: "and a listing naming no treatment stays where the edition says",
			in: mtgmatcher.InputCard{
				Name: "Gum-Gum Giant", Variation: "OP09-078",
				Edition: "OP09 - Emperors in the New World", Foil: true,
			},
			want: "op09-078_597018_foil",
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
