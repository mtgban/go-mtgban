package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// onePieceDonDatastore is the published datastore's three Zoro DON!! cards
// of The Best, copied verbatim: TCGplayer's one product in both finishes,
// and the gold one of its own.
const onePieceDonDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"PRB-01": {"name": "Premium Booster -The Best-", "releaseDate": "2024-11-08"}},
 "cards": [
  {"color": "", "externalLinks": {"tcgPlayerId": 587192}, "finish": "Foil", "id": "don_587192_foil", "name": "DON!! Card", "number": "DON", "promoTypes": ["gold"], "rarity": "DON!!", "setCode": "PRB-01", "type": "DON!!", "variant": "Zoro Gold", "watermark": "zoro"},
  {"color": "", "externalLinks": {"tcgPlayerId": 593830}, "finish": "Normal", "id": "don_593830", "name": "DON!! Card", "number": "DON", "rarity": "DON!!", "setCode": "PRB-01", "type": "DON!!", "variant": "Zoro", "watermark": "zoro"},
  {"color": "", "externalLinks": {"tcgPlayerId": 593830}, "finish": "Foil", "id": "don_593830_foil", "name": "DON!! Card", "number": "DON", "rarity": "DON!!", "setCode": "PRB-01", "type": "DON!!", "variant": "Zoro", "watermark": "zoro"}
 ]
}}`

// TestOnePieceDonFoilVersion pins that the V.2 of a The Best DON!! is its
// foil: CardTrader's one blueprint links the V.1 and V.2 to one TCGplayer
// product. The products and links are the catalog's and CardTrader's own.
func TestOnePieceDonFoilVersion(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceDonDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.tcgBridge = map[int]int{799541: 593830, 799542: 593830, 799543: 587192}

	exp := cm.Expansion{IDExpansion: 5805, Name: "The Best"}
	products := map[int]cm.CatalogProduct{
		799541: {ExpansionID: 5805, Name: "Don!! (PRB Zoro) (V.1)", Rarity: "DON!!", Version: 1},
		799542: {ExpansionID: 5805, Name: "Don!! (PRB Zoro) (V.2)", Rarity: "DON!!", Version: 2},
		799543: {ExpansionID: 5805, Name: "Don!! (PRB Zoro) (V.3)", Rarity: "DON!!", Version: 3},
	}
	mkm.resolver.claimByID(map[int][]int{5805: {799541, 799542, 799543}}, products, []cm.Expansion{exp})

	for id, want := range map[int]string{
		799541: "don_593830",
		799542: "don_593830_foil",
		799543: "don_587192_foil",
	} {
		got := mkm.resolver.resolveMapped(id, products[id], exp)
		if got.err != nil || got.cardID != want {
			t.Errorf("%d %q: resolveMapped = (%q, %v), want %q", id, products[id].Name, got.cardID, got.err, want)
		}
	}
}
