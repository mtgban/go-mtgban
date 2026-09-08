package cardmarket

import (
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// yugiohVersionDatastore is the published Yu-Gi-Oh datastore cut down to the
// one card these tests turn on, the three rows copied verbatim: Winner's
// Pack sells Ghost Ogre & Snow Rabbit three times over, under one number and
// one rarity, told apart only by the programme's stamp.
const yugiohVersionDatastore = `{
 "game": "yugioh",
 "sets": {
  "WI26": {"abbreviation": "WI26", "name": "Winner's Pack 2026-2027", "releaseDate": "2026-07-15"}
 },
 "cards": [
  {"attribute": "LIGHT", "externalLinks": {"konamiId": 59438930, "tcgPlayerId": 708831}, "finish": "Unlimited", "id": "wi26-en001_708831_unl", "name": "Ghost Ogre & Snow Rabbit", "number": "WI26-EN001", "promoTypes": ["ots stamp"], "rarity": "Ultra Rare", "setCode": "WI26", "type": "Tuner/Effect Monster", "variant": "OTS Stamp"},
  {"attribute": "LIGHT", "externalLinks": {"konamiId": 59438930, "tcgPlayerId": 710112}, "finish": "Unlimited", "id": "wi26-en001_710112_unl", "name": "Ghost Ogre & Snow Rabbit", "number": "WI26-EN001", "promoTypes": ["regional qualifier stamp"], "rarity": "Ultra Rare", "setCode": "WI26", "type": "Tuner/Effect Monster", "variant": "Regional Qualifier Stamp"},
  {"attribute": "LIGHT", "externalLinks": {"konamiId": 59438930, "tcgPlayerId": 710152}, "finish": "Unlimited", "id": "wi26-en001_710152_unl", "name": "Ghost Ogre & Snow Rabbit", "number": "WI26-EN001", "promoTypes": ["judge stamp"], "rarity": "Ultra Rare", "setCode": "WI26", "type": "Tuner/Effect Monster", "variant": "Judge Stamp"}
 ]
}`

// TestYugiohVersionVariants pins that each of a shelf's version indices
// reaches its own printing and no other. The three products carry the same
// name, number and rarity, so without the table they alias onto the three
// rows and none of them prices; the point of the rule is not only that they
// land but that they land apart, which is what a shared uuid would break.
func TestYugiohVersionVariants(t *testing.T) {
	if err := mtgmatcher.LoadDatastore(strings.NewReader(yugiohVersionDatastore)); err != nil {
		t.Fatal(err)
	}
	mkm := NewScraperIndex(cm.GameYuGiOh)

	seen := map[string]string{}
	for _, tt := range []struct {
		name string
		want string
	}{
		{"Ghost Ogre & Snow Rabbit (V.1 - Ultra Rare)", "wi26-en001_708831_unl"},
		{"Ghost Ogre & Snow Rabbit (V.2 - Ultra Rare)", "wi26-en001_710112_unl"},
		{"Ghost Ogre & Snow Rabbit (V.3 - Ultra Rare)", "wi26-en001_710152_unl"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, err := mkm.matchYugioh(&cm.Product{
				Name:          tt.name,
				Number:        "001",
				ExpansionName: "Winner's Pack 2026-2027",
			})
			if err != nil {
				t.Fatalf("matchYugioh(%q) = %v", tt.name, err)
			}
			if id != tt.want {
				t.Errorf("matchYugioh(%q) = %s, want %s", tt.name, id, tt.want)
			}
			if other, dup := seen[id]; dup {
				t.Errorf("matchYugioh(%q) landed on %s, already claimed by %q", tt.name, id, other)
			}
			seen[id] = tt.name
		})
	}
}

// TestYugiohVersionUncovered pins that a version the table does not cover is
// left as it was. A fourth printing appearing on the shelf must alias rather
// than borrow the third one's stamp, so that it is seen and named.
func TestYugiohVersionUncovered(t *testing.T) {
	if err := mtgmatcher.LoadDatastore(strings.NewReader(yugiohVersionDatastore)); err != nil {
		t.Fatal(err)
	}
	mkm := NewScraperIndex(cm.GameYuGiOh)
	_, err := mkm.matchYugioh(&cm.Product{
		Name:          "Ghost Ogre & Snow Rabbit (V.4 - Ultra Rare)",
		Number:        "001",
		ExpansionName: "Winner's Pack 2026-2027",
	})
	if err == nil {
		t.Error("a version past the table should refuse rather than reach a printing")
	}
}
