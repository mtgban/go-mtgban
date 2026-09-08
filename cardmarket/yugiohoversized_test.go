package cardmarket

import (
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// yugiohOversizedDatastore is the published datastore cut down to the shapes
// this turns on, every row copied verbatim: Machina Fortress as the oversized
// card the value box holds and as the ordinary card of the deck it is
// numbered for, and Dark Magician as an oversized card whose set is a
// collector box of its own.
const yugiohOversizedDatastore = `{
 "game": "yugioh",
 "sets": {
  "SDMM": {"abbreviation": "SDMM", "name": "Structure Deck: Machina Mayhem", "releaseDate": "2010-03-09"},
  "VBX": {"abbreviation": "VBX", "name": "Yu-Gi-Oh! Value Boxes", "releaseDate": "2011-11-01"},
  "YUCB": {"abbreviation": "YUCB", "name": "Yugi's Collector Box", "releaseDate": "2017-09-15"}
 },
 "cards": [
  {"attribute": "DARK", "externalLinks": {"tcgPlayerId": 146151}, "finish": "Limited", "id": "yucb-en001_146151_limited", "name": "Dark Magician", "number": "YUCB-EN001", "promoTypes": ["oversized"], "rarity": "Promo", "setCode": "YUCB", "type": "Normal Monster", "variant": "Oversized"},
  {"attribute": "EARTH", "externalLinks": {"konamiId": 5556499, "tcgPlayerId": 181002}, "finish": "Limited", "id": "sdmm-en001_181002_limited", "name": "Machina Fortress", "number": "SDMM-EN001", "promoTypes": ["oversized", "machine madness"], "rarity": "Promo", "setCode": "VBX", "type": "Effect Monster", "variant": "Oversized Machine Madness"},
  {"attribute": "EARTH", "externalLinks": {"konamiId": 5556499, "tcgPlayerId": 34641}, "finish": "1st Edition", "id": "sdmm-en001_34641_1stedition", "name": "Machina Fortress", "number": "SDMM-EN001", "rarity": "Ultra Rare", "setCode": "SDMM", "type": "Effect Monster"}
 ]
}`

// TestYugiohOversized pins that an oversized product reaches the printing the
// datastore files in the box it came in, not the deck the marketplace shelves
// it under, and that the ordinary card of that same deck and number is not
// what it reaches.
func TestYugiohOversized(t *testing.T) {
	if err := mtgmatcher.LoadDatastore(strings.NewReader(yugiohOversizedDatastore)); err != nil {
		t.Fatal(err)
	}
	mkm := NewScraperIndex(cm.GameYuGiOh)

	for _, tt := range []struct {
		desc, name, expansion, number, want string
	}{
		{
			// Shelved under the deck, filed by us in the value box.
			desc: "the shelf is not the set", name: "Machina Fortress (V.2 - Oversized)",
			expansion: "Structure Deck: Machina Mayhem", number: "001",
			want: "sdmm-en001_181002_limited",
		},
		{
			desc: "a box of its own still answers", name: "Dark Magician (V.2 - Oversized)",
			expansion: "Yugi's Collector Box", number: "001",
			want: "yucb-en001_146151_limited",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := mkm.matchYugioh(&cm.Product{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion})
			if err != nil {
				t.Fatalf("matchYugioh(%q) = %v", tt.name, err)
			}
			if id != tt.want {
				t.Errorf("matchYugioh(%q) = %s, want %s", tt.name, id, tt.want)
			}
		})
	}

	// The ordinary printing of the same deck and number is a different card
	// and must keep answering for the product that is not oversized.
	id, err := mkm.matchYugioh(&cm.Product{
		Name: "Machina Fortress (V.1 - Ultra Rare)", Number: "001",
		ExpansionName: "Structure Deck: Machina Mayhem",
	})
	if err != nil {
		t.Fatalf("the plain product: %v", err)
	}
	if id != "sdmm-en001_34641_1stedition" {
		t.Errorf("the plain product reached %s, want sdmm-en001_34641_1e", id)
	}
}
