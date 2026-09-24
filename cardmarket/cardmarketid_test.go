package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// preErrataDatastore is the published One Piece datastore cut down to two
// cards' pre-errata printings, copied verbatim. The hand-carried ones carry
// the Cardmarket product they were minted for, where Cardmarket sells one;
// the plain pre-errata Kin'emon is carried by neither.
const preErrataDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP01": {"name": "Romance Dawn", "releaseDate": "2022-12-02"}},
 "cards": [
  {"color": "Green", "externalLinks": {"bandaiId": "OP01-040", "tcgPlayerId": 453514}, "finish": "Foil", "id": "op01-040_453514_foil", "name": "Kin'emon", "number": "OP01-040", "rarity": "SR", "setCode": "OP01", "type": "Character"},
  {"color": "Green", "externalLinks": {"bandaiId": "OP01-040_p1", "tcgPlayerId": 454564}, "finish": "Foil", "id": "op01-040_454564_foil", "name": "Kin'emon", "number": "OP01-040", "promoTypes": ["parallel"], "rarity": "SR", "setCode": "OP01", "type": "Character", "variant": "Parallel"},
  {"color": "Green", "externalLinks": {"cardTraderId": 319051, "cardmarketId": 768200}, "finish": "Foil", "id": "op01-040_ct319051_foil", "name": "Kin'emon", "number": "OP01-040", "promoTypes": ["parallel", "preerrata"], "rarity": "SR", "setCode": "OP01", "type": "Character", "variant": "Parallel Pre-Errata"},
  {"color": "Blue;Purple", "externalLinks": {"bandaiId": "OP01-061", "tcgPlayerId": 453517}, "finish": "Normal", "id": "op01-061_453517", "name": "Kaido", "number": "OP01-061", "rarity": "L", "setCode": "OP01", "type": "Leader"},
  {"color": "Blue;Purple", "externalLinks": {"bandaiId": "OP01-061_p1", "tcgPlayerId": 454587}, "finish": "Foil", "id": "op01-061_454587_foil", "name": "Kaido", "number": "OP01-061", "promoTypes": ["parallel"], "rarity": "L", "setCode": "OP01", "type": "Leader", "variant": "Parallel"},
  {"color": "Blue;Purple", "externalLinks": {"cardTraderId": 277515}, "finish": "Normal", "id": "op01-061_ct277515", "name": "Kaido", "number": "OP01-061", "promoTypes": ["preerrata"], "rarity": "L", "setCode": "OP01", "type": "Leader", "variant": "Pre-Errata"},
  {"color": "Blue;Purple", "externalLinks": {"cardTraderId": 277514, "cardmarketId": 755419}, "finish": "Foil", "id": "op01-061_ct277514_foil", "name": "Kaido", "number": "OP01-061", "promoTypes": ["parallel", "preerrata"], "rarity": "L", "setCode": "OP01", "type": "Leader", "variant": "Parallel Pre-Errata"}
 ]
}}`

// TestCardmarketIDNamesThePrinting pins that a product the datastore records
// by id lands on that printing, where the name reached the plain pre-errata
// Kaido, and that a name reaching a printing some other product's id owns is
// refused: the plain pre-errata Kin'emon had been pricing the parallel.
func TestCardmarketIDNamesThePrinting(t *testing.T) {
	b := datastoreBackend(t, "onepiece", preErrataDatastore)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		desc    string
		product cm.Product
		want    string
		wantErr error
	}{
		{
			desc:    "the id names the parallel",
			product: cm.Product{IDProduct: 755419, Name: "Kaido (OP01-061) (V.2)", Number: "061", Rarity: "Alternate Art"},
			want:    "op01-061_ct277514_foil",
		},
		{
			desc:    "a product with no id is still named",
			product: cm.Product{IDProduct: 768170, Name: "Kaido (OP01-061) (V.1)", Number: "061", Rarity: "Leader"},
			want:    "op01-061_ct277515",
		},
		{
			desc:    "the owner lands by its id",
			product: cm.Product{IDProduct: 768200, Name: "Kin'emon (OP01-040) (V.2)", Number: "040", Rarity: "Alternate Art"},
			want:    "op01-040_ct319051_foil",
		},
		{
			desc:    "a sibling named onto the owner's printing is refused",
			product: cm.Product{IDProduct: 768155, Name: "Kin'emon (OP01-040) (V.1)", Number: "040", Rarity: "Super Rare"},
			wantErr: errNoPrinting,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			tt.product.ExpansionName = "Romance Dawn (Pre-Errata)"
			cardID, _, _, err := mkm.resolveProduct(&tt.product)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveProduct(%q) = %q, %v; want error %v", tt.product.Name, cardID, err, tt.wantErr)
			}
			if cardID != tt.want {
				t.Errorf("resolveProduct(%q) = %q, want %q", tt.product.Name, cardID, tt.want)
			}
		})
	}
}
