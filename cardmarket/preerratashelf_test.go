package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// preErrataShelfDatastore is the published One Piece datastore cut down to
// Inuarashi, verbatim: the regular card and its Box Topper, and no pre-errata
// printing, which neither TCGplayer nor CardTrader sells.
const preErrataShelfDatastore = `{"data": {
 "game": "onepiece",
 "sets": {"OP01": {"name": "Romance Dawn", "releaseDate": "2022-12-02"}},
 "cards": [
  {"color": "Green", "externalLinks": {"bandaiId": "OP01-034_p1", "tcgPlayerId": 453513}, "finish": "Foil", "id": "op01-034_453513_foil", "image": "https://static.dotgg.gg/onepiece/card/OP01-034_p1.webp", "name": "Inuarashi", "number": "OP01-034", "promoTypes": ["boxtopper"], "rarity": "C", "setCode": "OP01", "type": "Character", "variant": "Box Topper"},
  {"color": "Green", "externalLinks": {"bandaiId": "OP01-034", "tcgPlayerId": 454555}, "finish": "Normal", "id": "op01-034_454555", "image": "https://static.dotgg.gg/onepiece/card/OP01-034.webp", "name": "Inuarashi", "number": "OP01-034", "rarity": "C", "setCode": "OP01", "type": "Character"}
 ]
}}`

// TestPreErrataShelf pins that a product on Cardmarket's pre-errata shelf
// lands on a pre-errata printing or not at all. Inuarashi's V.2 there had
// reached the Box Topper, which the set's own shelf sells as 690838, because
// offShelf trusts a labelled answer; its V.1 reaches the regular card, which
// offShelf already refused.
func TestPreErrataShelf(t *testing.T) {
	b := datastoreBackend(t, "onepiece", preErrataShelfDatastore)
	shelves := []cm.Expansion{{Name: "Romance Dawn"}, {Name: "Romance Dawn (Pre-Errata)"}}

	for _, tt := range []struct {
		what    string
		product cm.Product
		bridge  map[int]int
		want    string
	}{
		{
			what: "the pre-errata V.2 is not the Box Topper",
			product: cm.Product{
				IDProduct: 768152, Name: "Inuarashi (OP01-034) (V.2)", Number: "034",
				Rarity: "Alternate Art", ExpansionName: "Romance Dawn (Pre-Errata)",
			},
		},
		{
			what: "the pre-errata V.1 is not the regular card",
			product: cm.Product{
				IDProduct: 768151, Name: "Inuarashi (OP01-034) (V.1)", Number: "034",
				Rarity: "Common", ExpansionName: "Romance Dawn (Pre-Errata)",
			},
		},
		{
			what: "a bridge naming the Box Topper is refused on the shelf too",
			product: cm.Product{
				IDProduct: 768152, Name: "Inuarashi (OP01-034) (V.2)", Number: "034",
				Rarity: "Alternate Art", ExpansionName: "Romance Dawn (Pre-Errata)",
			},
			bridge: map[int]int{768152: 453513},
		},
		{
			what: "the set's own shelf keeps the Box Topper",
			product: cm.Product{
				IDProduct: 690838, Name: "Inuarashi (OP01-034) (V.2)", Number: "034",
				Rarity: "Alternate Art", ExpansionName: "Romance Dawn",
			},
			want: "op01-034_453513_foil",
		},
		{
			what: "the set's own shelf keeps the regular card",
			product: cm.Product{
				IDProduct: 690837, Name: "Inuarashi (OP01-034) (V.1)", Number: "034",
				Rarity: "Common", ExpansionName: "Romance Dawn",
			},
			want: "op01-034_454555",
		},
	} {
		t.Run(tt.what, func(t *testing.T) {
			mkm, err := NewScraperIndex(b)
			if err != nil {
				t.Fatalf("NewScraperIndex(b) = %v", err)
			}
			mkm.exchangeRate = 1
			mkm.shelved = shelvedSets(b, shelves)
			mkm.tcgBridge = tt.bridge
			mkm.priceGuide = map[int]cm.PriceGuide{
				tt.product.IDProduct: {IDProduct: tt.product.IDProduct, LowPrice: 1, TrendPrice: 2},
			}
			channel := make(chan responseChan, 8)
			err = mkm.processProduct(channel, &tt.product)
			close(channel)

			if tt.want == "" {
				if !errors.Is(err, errNoPrinting) {
					t.Fatalf("product %d returned %v, want %v", tt.product.IDProduct, err, errNoPrinting)
				}
				if len(channel) != 0 {
					t.Errorf("a refused product priced %d entries, want none", len(channel))
				}
				return
			}
			if err != nil {
				t.Fatalf("product %d returned %v, want a price", tt.product.IDProduct, err)
			}
			var priced int
			for out := range channel {
				priced++
				if out.cardID != tt.want {
					t.Errorf("product %d priced %s, want %s", tt.product.IDProduct, out.cardID, tt.want)
				}
			}
			if priced == 0 {
				t.Errorf("product %d priced nothing, want %s", tt.product.IDProduct, tt.want)
			}
		})
	}
}
