package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaOwnedDatastore is the published datastore cut down to the rows a
// V.N lands on by name: Ariel, whose misprint Cardmarket sells as a V.2;
// Simba, whose starter foil is the Holofoil beside the V.1's two finishes;
// and Moana and Vaiana, the two cards claiming Cardmarket product 801862.
const lorcanaOwnedDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"},
    "7": {"name": "Archazia's Island", "type": "expansion", "releaseDate": "2025-03-21"},
    "10": {"name": "Whispers in the Well", "type": "expansion", "releaseDate": "2025-11-14"}
  },
  "cards": [
    {"id": 2, "name": "Ariel", "version": "Spectacular Singer", "fullName": "Ariel - Spectacular Singer", "setCode": "1", "number": "2", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "2_foil"}, {"finish": "Normal", "id": "2"}], "externalLinks": {"cardTraderId": 258920, "cardmarketId": 727082, "tcgPlayerId": 504451}},
    {"id": 1433, "name": "Moana", "version": "Adventurer of Land and Sea", "fullName": "Moana - Adventurer of Land and Sea", "setCode": "7", "number": "26", "total": "P2", "rarity": "Special", "type": "Character", "promoGrouping": "P2", "promoSourceCategory": "Promo", "printings": [{"finish": "Cold Foil", "id": "1433_foil"}], "externalLinks": {"cardTraderId": 311906, "cardmarketId": 801862, "tcgPlayerId": 601112}},
    {"id": 1663, "name": "Vaiana", "version": "Adventurer of Land and Sea", "fullName": "Vaiana - Adventurer of Land and Sea", "setCode": "7", "number": "26", "total": "P2", "rarity": "Special", "type": "Character", "promoGrouping": "P2", "promoSourceCategory": "Promo", "printings": [{"finish": "Cold Foil", "id": "1663_foil"}], "externalLinks": {"cardTraderId": 311906, "cardmarketId": 801862}},
    {"id": 2209, "name": "Simba", "version": "King in the Making", "fullName": "Simba - King in the Making", "setCode": "10", "number": "20", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "2209_foil", "promoTypes": ["freeform"]}, {"finish": "Holofoil", "id": "2209_holofoil", "promoTypes": ["rainbowpillars"]}, {"finish": "Normal", "id": "2209"}], "externalLinks": {"cardTraderId": 354170, "cardmarketId": 856011, "tcgPlayerId": 657894}}
  ]
}}`

// TestLorcanaOwnedElsewhere pins that a V.N the wording lands on a printing
// another product's id prices is skipped, while one moved to a finish the id
// does not price, or one whose owner no single card claims, still lands.
// The products are the catalog's own.
func TestLorcanaOwnedElsewhere(t *testing.T) {
	b := datastoreBackend(t, "lorcana", lorcanaOwnedDatastore)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		desc               string
		id                 int
		expansion          string
		product            cm.CatalogProduct
		wantCard, wantFoil string
	}{
		{"the owner lands by its id", 727082, "The First Chapter",
			cm.CatalogProduct{Name: "Ariel - Spectacular Singer (V.1)", Number: "2", Rarity: "Super Rare", Version: 1}, "2", "2_foil"},
		{"its misprint is skipped", 832568, "The First Chapter",
			cm.CatalogProduct{Name: "Ariel - Spectacular Singer (V.2)", Number: "2", Rarity: "Super Rare", Version: 2}, "", ""},
		{"a starter foil keeps the Holofoil", 864479, "Whispers in the Well",
			cm.CatalogProduct{Name: "Simba - King in the Making (V.3)", Number: "020", Rarity: "Super Rare", Version: 3}, "2209_holofoil", "2209_holofoil"},
		{"an id two cards claim owns nothing", 804164, "Promos Year 2",
			cm.CatalogProduct{Name: "Vaiana - Adventurer of Land and Sea", Number: "26", Rarity: "Promo"}, "1663_foil", "1663_foil"},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: tt.expansion})
		if got.err != nil || got.cardID != tt.wantCard || got.cardIDFoil != tt.wantFoil {
			t.Errorf("%s: resolveMapped(%d) = (%q, %q, %v), want (%q, %q)",
				tt.desc, tt.id, got.cardID, got.cardIDFoil, got.err, tt.wantCard, tt.wantFoil)
		}
	}
}
