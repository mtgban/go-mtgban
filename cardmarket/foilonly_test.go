package cardmarket

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// foilOnlyDatastore carries two printings of one Lorcana set: a card sold
// both plain and foiled, and an Enchanted sold in a holofoil alone. The
// second is the shape marketFoilOnly exists for - Cardmarket sells it as one
// product, so it resolves to a single id in both of a product's slots.
const foilOnlyDatastore = `{
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {"13": {"name": "Attack of the Vine!", "type": "expansion", "releaseDate": "2026-08-14"}},
  "cards": [
    {
      "id": 1, "name": "Meilin Lee", "fullName": "Meilin Lee - Popular Red Panda",
      "setCode": "13", "number": "125", "rarity": "Legendary", "type": "Character",
      "color": "Amber", "story": "Turning Red", "foilTypes": ["None", "Cold Foil"],
      "externalLinks": {"cardmarketId": 897323}
    },
    {
      "id": 2, "name": "Meilin Lee", "fullName": "Meilin Lee - Popular Red Panda",
      "setCode": "13", "number": "240", "rarity": "Enchanted", "type": "Character",
      "color": "Amber", "story": "Turning Red", "foilTypes": ["Holofoil"],
      "externalLinks": {"cardmarketId": 897451}
    }
  ]
}`

// TestMarketFoilOnly pins which finish a product's first query asks for. The
// plain printing is asked for as a plain card and its foil twin separately;
// the Enchanted, which answers both of a product's ids with one uuid, is
// asked for as a foil, because a query for its non-foil listings rejects
// every listing the product has.
func TestMarketFoilOnly(t *testing.T) {
	b := datastoreBackend(t, "lorcana", foilOnlyDatastore)

	// The ids the resolver hands queryPrintings, as the datastore mints
	// them: the plain printing and its foil twin are two, the Enchanted is
	// one that stands in both slots.
	for _, tt := range []struct {
		name               string
		cardID, cardIDFoil string
		want               bool
	}{
		{"a printing with both finishes asks for its plain listings", "1", "1_foil", false},
		{"a foil-only printing asks for its foil listings", "2_holofoil", "2_holofoil", true},
		{"an empty foil slot names that same single printing", "2_holofoil", "", true},
		{"a plain id with no foil twin stays a plain query", "1", "", false},
		{"an unknown id asks for nothing in particular", "nope", "nope", false},
	} {
		if got := marketFoilOnly(b, tt.cardID, tt.cardIDFoil); got != tt.want {
			t.Errorf("%s: marketFoilOnly(%q, %q) = %v, want %v",
				tt.name, tt.cardID, tt.cardIDFoil, got, tt.want)
		}
	}
}
