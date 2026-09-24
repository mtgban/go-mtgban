package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaOversizedDatastore is the published datastore cut down to the rows
// this turns on: the four oversized printings Cardmarket sells, each beside
// the ordinary card at its number, and Baymax, whose oversized product has no
// printing of ours.
const lorcanaOversizedDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"},
    "3": {"name": "Into the Inklands", "type": "expansion", "releaseDate": "2024-03-08"},
    "6": {"name": "Azurite Sea", "type": "expansion", "releaseDate": "2024-11-25"}
  },
  "cards": [
    {"id": 5, "name": "Hades", "version": "King of Olympus", "fullName": "Hades - King of Olympus", "setCode": "1", "number": "5", "total": "204", "rarity": "Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "5_foil"}, {"finish": "Normal", "id": "5"}], "externalLinks": {"cardmarketId": 727085, "tcgPlayerId": 485364}},
    {"id": 118, "name": "Mulan", "version": "Imperial Soldier", "fullName": "Mulan - Imperial Soldier", "setCode": "1", "number": "118", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "118_foil"}, {"finish": "Normal", "id": "118"}], "externalLinks": {"cardmarketId": 729305, "tcgPlayerId": 485365}},
    {"id": 525, "name": "Stitch", "version": "Covert Agent", "fullName": "Stitch - Covert Agent", "setCode": "3", "number": "89", "total": "204", "rarity": "Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "525_foil"}, {"finish": "Normal", "id": "525"}], "externalLinks": {"cardmarketId": 757320, "tcgPlayerId": 539083}},
    {"id": 593, "name": "Tinker Bell", "version": "Very Clever Fairy", "fullName": "Tinker Bell - Very Clever Fairy", "setCode": "3", "number": "157", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "593_foil"}, {"finish": "Normal", "id": "593"}], "externalLinks": {"cardmarketId": 755619, "tcgPlayerId": 536268}},
    {"id": 1356, "name": "Baymax", "version": "Armored Companion", "fullName": "Baymax - Armored Companion", "setCode": "6", "number": "157", "total": "204", "rarity": "Legendary", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "1356_foil"}, {"finish": "Normal", "id": "1356"}], "externalLinks": {"cardmarketId": 794871, "tcgPlayerId": 578165}},
    {"id": -516775, "name": "Hades - King of Olympus (Oversized)", "fullName": "Hades - King of Olympus (Oversized)", "setCode": "1", "number": "5", "total": "204", "rarity": "Rare", "type": "Character", "promoTypes": ["oversized"], "printings": [{"finish": "Cold Foil", "id": "m-516775_foil"}], "externalLinks": {"tcgPlayerId": 516775}},
    {"id": -516778, "name": "Mulan - Imperial Soldier (Oversized)", "fullName": "Mulan - Imperial Soldier (Oversized)", "setCode": "1", "number": "118", "total": "204", "rarity": "Super Rare", "type": "Character", "promoTypes": ["oversized"], "printings": [{"finish": "Cold Foil", "id": "m-516778_foil"}], "externalLinks": {"tcgPlayerId": 516778}},
    {"id": -539145, "name": "Stitch - Covert Agent (Oversized)", "fullName": "Stitch - Covert Agent (Oversized)", "setCode": "3", "number": "89", "total": "204", "rarity": "Rare", "type": "Character", "promoTypes": ["oversized"], "printings": [{"finish": "Cold Foil", "id": "m-539145_foil"}], "externalLinks": {"tcgPlayerId": 539145}},
    {"id": -539148, "name": "Tinker Bell - Very Clever Fairy (Oversized)", "fullName": "Tinker Bell - Very Clever Fairy (Oversized)", "setCode": "3", "number": "157", "total": "204", "rarity": "Super Rare", "type": "Character", "promoTypes": ["oversized"], "printings": [{"finish": "Cold Foil", "id": "m-539148_foil"}], "externalLinks": {"tcgPlayerId": 539148}}
  ]
}}`

// TestLorcanaOversized pins that a catalog product of Cardmarket's
// "Oversized" rarity lands on the oversized printing at its number, not the
// ordinary card beside it, and is skipped where the datastore carries none.
// The products are the catalog's own, verbatim.
func TestLorcanaOversized(t *testing.T) {
	b := datastoreBackend(t, "lorcana", lorcanaOversizedDatastore)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		id        int
		expansion string
		product   cm.CatalogProduct
		want      string
	}{
		{729405, "The First Chapter", cm.CatalogProduct{Name: "Hades - King of Olympus (V.3)", Number: "5", Rarity: "Oversized", Version: 3}, "m-516775_foil"},
		{729406, "The First Chapter", cm.CatalogProduct{Name: "Mulan - Imperial Soldier (V.2)", Number: "118", Rarity: "Oversized", Version: 2}, "m-516778_foil"},
		{770257, "Into the Inklands", cm.CatalogProduct{Name: "Stitch - Covert Agent (V.2)", Number: "89", Rarity: "Oversized", Version: 2}, "m-539145_foil"},
		{770258, "Into the Inklands", cm.CatalogProduct{Name: "Tinker Bell - Very Clever Fairy (V.2)", Number: "157", Rarity: "Oversized", Version: 2}, "m-539148_foil"},
		// No oversized printing of ours: skipped, not priced on the Legendary.
		{852841, "Azurite Sea", cm.CatalogProduct{Name: "Baymax - Armored Companion (V.2)", Number: "157", Rarity: "Oversized", Version: 2}, ""},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: tt.expansion})
		if got.err != nil || got.cardID != tt.want || got.cardIDFoil != tt.want {
			t.Errorf("%d %q: resolveMapped = (%q, %q, %v), want (%q, %q, nil)",
				tt.id, tt.product.Name, got.cardID, got.cardIDFoil, got.err, tt.want, tt.want)
		}
	}

	// The ordinary product at the same number keeps its own card.
	got := mkm.resolveMapped(727085, cm.CatalogProduct{Name: "Hades - King of Olympus (V.1)", Number: "5", Rarity: "Rare", Version: 1}, cm.Expansion{Name: "The First Chapter"})
	if got.err != nil || got.cardID != "5" || got.cardIDFoil != "5_foil" {
		t.Errorf("727085 Hades (V.1): resolveMapped = (%q, %q, %v), want (\"5\", \"5_foil\", nil)", got.cardID, got.cardIDFoil, got.err)
	}
}
