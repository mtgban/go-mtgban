package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaBridgeDatastore is the published datastore cut down to the rows the
// bridge settles: Fabled's Stitch - Rock Star beside the Discover promo
// numbered the same, and the two DLPC printings of Mickey Mouse - True
// Friend at 25, whose names differ only by their decoration.
const lorcanaBridgeDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "9": {"name": "Fabled", "type": "expansion", "releaseDate": "2025-09-05"},
    "DLPC": {"name": "Disney Lorcana Promo Cards", "type": "promo", "releaseDate": "2023-07-01"}
  },
  "cards": [
    {"id": 1939, "name": "Stitch", "version": "Rock Star", "fullName": "Stitch - Rock Star", "setCode": "9", "number": "3", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "1939_foil"}, {"finish": "Normal", "id": "1939"}], "externalLinks": {"cardmarketId": 843959, "tcgPlayerId": 649952}},
    {"id": 3245, "name": "Stitch", "version": "Rock Star", "fullName": "Stitch - Rock Star", "setCode": "9", "number": "3", "total": "DIS", "rarity": "Special", "type": "Character", "promoTypes": ["disneyparksstores"], "printings": [{"finish": "Holofoil", "id": "3245_holofoil", "promoTypes": ["freeform"]}], "externalLinks": {"cardmarketId": 864947, "tcgPlayerId": 668575}},
    {"id": -588123, "name": "Mickey Mouse - True Friend (CS Exclusive)", "fullName": "Mickey Mouse - True Friend (CS Exclusive)", "setCode": "DLPC", "number": "25", "rarity": "Promo", "type": "Character", "promoTypes": ["csexclusive"], "printings": [{"finish": "Cold Foil", "id": "m-588123_foil"}], "externalLinks": {"tcgPlayerId": 588123}},
    {"id": -588124, "name": "Mickey Mouse - True Friend (JP Exclusive)", "fullName": "Mickey Mouse - True Friend (JP Exclusive)", "setCode": "DLPC", "number": "25", "rarity": "Promo", "type": "Character", "promoTypes": ["jpexclusive"], "printings": [{"finish": "Cold Foil", "id": "m-588124_foil"}], "externalLinks": {"tcgPlayerId": 588124}}
  ]
}}`

// TestLorcanaBridge pins that the bridge names a Lorcana product's printing
// where the wording cannot, as long as the name still agrees: a promo whose
// number a set card shares, and one our datastore decorates. The products
// and bridge links are the catalog's and CardTrader's own.
func TestLorcanaBridge(t *testing.T) {
	b := datastoreBackend(t, "lorcana", lorcanaBridgeDatastore)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		desc      string
		id, tcgID int
		expansion string
		product   cm.CatalogProduct
		want      string
	}{
		{
			desc: "the promo, not the set card at its number",
			id:   864947, tcgID: 668575, expansion: "Discover Promo",
			product: cm.CatalogProduct{Name: "Stitch - Rock Star", Number: "3", Rarity: "Promo"},
			want:    "3245_holofoil",
		},
		{
			desc: "a decorated name still agrees",
			id:   791307, tcgID: 588124, expansion: "Promos Year 1",
			product: cm.CatalogProduct{Name: "Mickey Mouse - True Friend", Number: "25", Rarity: "Promo"},
			want:    "m-588124_foil",
		},
		{
			desc: "a link to another card is refused and the wording answers",
			id:   843959, tcgID: 588124, expansion: "Fabled",
			product: cm.CatalogProduct{Name: "Stitch - Rock Star", Number: "3", Rarity: "Super Rare"},
			want:    "1939",
		},
	} {
		mkm.tcgBridge = map[int]int{tt.id: tt.tcgID}
		got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: tt.expansion})
		if got.err != nil || got.cardID != tt.want {
			t.Errorf("%s: resolveMapped(%d) = (%q, %v), want %q", tt.desc, tt.id, got.cardID, got.err, tt.want)
		}
	}
}
