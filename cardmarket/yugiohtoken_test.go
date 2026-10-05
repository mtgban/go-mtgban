package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// yugiohTokenDatastore is the published Yu-Gi-Oh datastore cut down to
// tokens of the three Token Promos runs, which share their number tails
// within one set and are named for the card they stand for, and a Legendary
// Duelists art token, images left out.
const yugiohTokenDatastore = `{"data": {
 "game": "yugioh",
 "sets": {
  "TKN": {"name": "Yu-Gi-Oh! Tokens", "releaseDate": "2006-07-20"},
  "LDS1": {"name": "Legendary Duelists: Season 1", "releaseDate": "2020-07-03"}
 },
 "cards": [
  {"attribute": "", "externalLinks": {"konamiId": 73915052, "tcgPlayerId": 81303}, "finish": "Unlimited", "id": "tkn1-en001_81303_unlimited", "name": "Token: Sheep", "number": "TKN1-EN001", "rarity": "Common", "setCode": "TKN", "type": "", "variant": "Blue"},
  {"attribute": "", "externalLinks": {"tcgPlayerId": 81307}, "finish": "Unlimited", "id": "tkn3-en001_81307_unlimited", "name": "Token: Grinder Golem", "number": "TKN3-EN001", "rarity": "Common", "setCode": "TKN", "type": ""},
  {"attribute": "", "externalLinks": {"tcgPlayerId": 81320}, "finish": "Unlimited", "id": "tkn4-en001_81320_unlimited", "name": "Yu-Gi-Oh 5D's 2009 National Championship Token", "number": "TKN4-EN001", "rarity": "Ultra Rare", "setCode": "TKN", "type": ""},
  {"attribute": "", "externalLinks": {"tcgPlayerId": 217527}, "finish": "Unlimited", "id": "217527_unlimited", "name": "Art Token: Mai Valentine", "rarity": "Promo", "setCode": "LDS1", "type": "Token"}
 ]
}}`

// TestMatchYugiohTokens pins that a Token Promos token, named by its art,
// lands on the print its number names, and that an "X Art Token" reads as
// the datastore's "Art Token: X". The products are the catalog's own.
func TestMatchYugiohTokens(t *testing.T) {
	b := datastoreBackend(t, "yugioh", yugiohTokenDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	for _, tt := range []struct {
		expansion, name, number, want string
	}{
		{"Token Promos 3", "Grinder Token", "001", "tkn3-en001_81307_unlimited"},
		{"Legendary Duelists: Season 1", "Mai Valentine Art Token", "", "217527_unlimited"},
	} {
		product := cm.Product{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchYugioh(&product)
		if got != tt.want || err != nil {
			t.Errorf("%q in %q (%s) = %q, %v; want %q", tt.name, tt.expansion, tt.number, got, err, tt.want)
		}
	}
}
