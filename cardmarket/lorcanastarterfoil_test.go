package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaStarterFoilDatastore is Simba - King in the Making as published: a
// Normal, a Cold Foil, and the starter deck's exclusive foil as a Holofoil.
const lorcanaStarterFoilDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {"10": {"name": "Whispers in the Well", "type": "expansion", "releaseDate": "2025-11-14"}},
  "cards": [
    {"id": 2209, "name": "Simba", "version": "King in the Making", "fullName": "Simba - King in the Making", "setCode": "10", "number": "20", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "2209_foil", "promoTypes": ["freeform"]}, {"finish": "Holofoil", "id": "2209_holofoil", "promoTypes": ["rainbowpillars"]}, {"finish": "Normal", "id": "2209"}], "externalLinks": {"cardmarketId": 856011, "tcgPlayerId": 657894}}
  ]
}}`

// TestLorcanaStarterFoil pins that the V.N Cardmarket sells beside a card at
// its own number, the starter deck's exclusive foil, lands on the Holofoil,
// while the card's own V.1 keeps its Normal and Cold Foil.
func TestLorcanaStarterFoil(t *testing.T) {
	b := datastoreBackend(t, "lorcana", lorcanaStarterFoilDatastore)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		id                 int
		product            cm.CatalogProduct
		wantCard, wantFoil string
	}{
		{856011, cm.CatalogProduct{Name: "Simba - King in the Making (V.1)", Number: "020", Rarity: "Super Rare", Version: 1}, "2209", "2209_foil"},
		{864479, cm.CatalogProduct{Name: "Simba - King in the Making (V.3)", Number: "020", Rarity: "Super Rare", Version: 3}, "2209_holofoil", "2209_holofoil"},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: "Whispers in the Well"})
		if got.err != nil || got.cardID != tt.wantCard || got.cardIDFoil != tt.wantFoil {
			t.Errorf("%d %q: resolveMapped = (%q, %q, %v), want (%q, %q)",
				tt.id, tt.product.Name, got.cardID, got.cardIDFoil, got.err, tt.wantCard, tt.wantFoil)
		}
	}
}
