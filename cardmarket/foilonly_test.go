package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

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

// TestFoilOnlyShelf pins the closed set of shelves whose guide entry prices
// the shown treatment's foil alone - verbatim product names from the shelves
// the README documents, plus the edge cases the closed table is built
// around: M3C Extras' own "(V.1)" printing is not foil-only, and the
// Holiday Release name is shared by LTC's foil-only silverfoil box topper
// and LTR's own (also foil-only) showcase cards. LTR is left off the table
// not because it sells both finishes - Boromir (V.2) below is foil-only
// same as LTC's box topper - but because Preprocess already resolves its
// "(V.2)" rows onto their foil printing by variant, before foilOnlyShelf is
// ever asked, so an entry here would be a no-op for all 181 of them.
func TestFoilOnlyShelf(t *testing.T) {
	for _, tt := range []struct {
		name    string
		product cm.Product
		setCode string
		want    bool
	}{
		{"FIC Collector's Edition", cm.Product{Name: "Summon: Esper Valigarmanda", ExpansionName: "Commander: Magic: The Gathering - FINAL FANTASY: Collector's Edition"}, "FIC", true},
		{"MSC Collector's Edition", cm.Product{Name: "Iron Man, Armored Avenger", ExpansionName: "Commander: Marvel Super Heroes: Collector's Edition"}, "MSC", true},
		{"TMC Extras", cm.Product{Name: "Baxter, Fly in the Ointment", ExpansionName: "Commander: Teenage Mutant Ninja Turtles: Extras"}, "TMC", true},
		{"M3C Extras with no (V.N) suffix", cm.Product{Name: "Drowner of Hope", ExpansionName: "Commander: Modern Horizons 3: Extras"}, "M3C", true},
		{"M3C Extras (V.2)", cm.Product{Name: "Localized Destruction (V.2)", ExpansionName: "Commander: Modern Horizons 3: Extras"}, "M3C", true},
		{"M3C Extras (V.1) is the ordinary nonfoil printing", cm.Product{Name: "Sunken Palace (V.1)", ExpansionName: "Commander: Modern Horizons 3: Extras"}, "M3C", false},
		{"LTC Holiday Release (V.2) is the silverfoil box topper", cm.Product{Name: "Kenrith, the Returned King (V.2)", ExpansionName: "The Lord of the Rings: Tales of Middle-earth Holiday Release"}, "LTC", true},
		{"LTR Holiday Release (V.2) already resolved to foil by Preprocess", cm.Product{Name: "Boromir, Warden of the Tower (V.2)", ExpansionName: "The Lord of the Rings: Tales of Middle-earth Holiday Release"}, "LTR", false},
		{"LTC Holiday Release (V.1) is the ordinary printing", cm.Product{Name: "Kenrith, the Returned King (V.1)", ExpansionName: "The Lord of the Rings: Tales of Middle-earth Holiday Release"}, "LTC", false},
		{"an unrelated shelf", cm.Product{Name: "Lightning Bolt", ExpansionName: "Modern Horizons 2"}, "MH2", false},
	} {
		if got := foilOnlyShelf(&tt.product, tt.setCode); got != tt.want {
			t.Errorf("%s: foilOnlyShelf(%+v, %q) = %v, want %v", tt.name, tt.product, tt.setCode, got, tt.want)
		}
	}
}
