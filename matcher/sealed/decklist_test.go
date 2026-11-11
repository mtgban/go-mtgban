package sealed_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/sealed"
)

// A product holding a single fixed card, the shape of the Secret Lair Promo
// drops, lists that card as its decklist, and one mixing a card with packs
// lists only the card; one holding only packs has none.
func TestProductDecklistOfFixedCards(t *testing.T) {
	var set matcher.Set
	err := json.Unmarshal([]byte(`{
		"code": "SLD",
		"sealedProduct": [
			{"uuid": "promo-product", "contents": {"card": [
				{"uuid": "promo_f", "foil": true, "set": "sld"}]}},
			{"uuid": "mixed-product", "contents": {
				"card": [{"uuid": "promo_f", "foil": true, "set": "sld"}],
				"pack": [{"code": "default", "set": "sld"}]}},
			{"uuid": "pack-product", "contents": {"pack": [
				{"code": "default", "set": "sld"}]}}
		]}`), &set)
	if err != nil {
		t.Fatal(err)
	}
	b := &matcher.Backend{
		Sets: map[string]*matcher.Set{"SLD": &set},
		UUIDs: map[string]*matcher.CardObject{
			"promo_f": {Card: matcher.Card{UUID: "promo_f"}, Foil: true},
		},
	}

	for _, product := range []string{"promo-product", "mixed-product"} {
		if !sealed.HasDecklist(b, "SLD", product) {
			t.Errorf("%s has no decklist", product)
		}
		decklist, err := sealed.ProductDecklist(b, "SLD", product)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"promo_f"}; !slices.Equal(decklist, want) {
			t.Errorf("%s decklist %v, want %v", product, decklist, want)
		}
	}

	if sealed.HasDecklist(b, "SLD", "pack-product") {
		t.Error("pack-product has a decklist")
	}
}
