package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// pokemonEmitDatastore is the published Pokemon datastore cut down to the
// four shapes a product's columns land on, every row copied from it: the
// SM04 Pikachu sold holo and in nothing else, the Rowlet of SM Base Set
// sold plain beside a reverse holo, the Shiinotic of the same set sold
// holo beside a reverse holo, and the Mew of Southern Islands sold as a
// reverse holo and in nothing else.
const pokemonEmitDatastore = `{
 "game": "pokemon",
 "sets": {"SMP": {"abbreviation": "SMP", "name": "SM Promos", "releaseDate": "2016-11-18", "type": "promo"}, "SM01": {"abbreviation": "SM01", "name": "SM Base Set", "releaseDate": "2017-02-03"}, "SI": {"abbreviation": "SI", "name": "Southern Islands", "releaseDate": "2001-07-31"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 46466}, "finish": "Reverse Holofoil", "id": "01-18_46466_reverseholofoil", "name": "Mew", "number": "01", "rarity": "Promo", "setCode": "SI", "total": "18"},
  {"externalLinks": {"tcgPlayerId": 127160}, "finish": "Holofoil", "id": "sm04_127160_holofoil", "name": "Pikachu", "number": "SM04", "rarity": "Promo", "setCode": "SMP"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Normal", "id": "9-149_126880", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Reverse Holofoil", "id": "9-149_126880_reverseholofoil", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126888}, "finish": "Holofoil", "id": "17-149_126888_holofoil", "name": "Shiinotic", "number": "17", "rarity": "Holo Rare", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126888}, "finish": "Reverse Holofoil", "id": "17-149_126888_reverseholofoil", "name": "Shiinotic", "number": "17", "rarity": "Holo Rare", "setCode": "SM01"}
 ]
}`

// TestEmitPokemonColumns pins where a Pokemon product's two pairs of columns
// land: the product's own on its printing, holo or plain, and the guide's
// holo pair on the reverse holo sold beside it - or on the printing itself
// where the product is sold as a reverse holo and nothing else, the own
// columns being whatever other listings the storefront holds. A holo rare
// is a foil to the flag, and reading the flag priced the holo from the
// reverse's columns and the reverse from nothing; a promo sold holo and in
// nothing else, whose guide carries no holo pair at all, went unpriced.
func TestEmitPokemonColumns(t *testing.T) {
	installDatastore(t, "pokemon", pokemonEmitDatastore)

	mkm := &Index{
		gameID:       cm.GamePokemon,
		exchangeRate: 1,
		priceGuide: map[int]cm.PriceGuide{
			1: {IDProduct: 1, LowPrice: 1, TrendPrice: 2},
			2: {IDProduct: 2, LowPrice: 3, TrendPrice: 4, HoloLowPrice: 5, HoloTrendPrice: 6},
			3: {IDProduct: 3, LowPrice: 7, TrendPrice: 8, HoloLowPrice: 9, HoloTrendPrice: 10},
			4: {IDProduct: 4, LowPrice: 11, TrendPrice: 12, HoloLowPrice: 13, HoloTrendPrice: 14},
		},
	}
	products := []cm.Product{
		{IDProduct: 1, Name: "Pikachu ", Number: "04", ExpansionName: "SM Black Star Promos"},
		{IDProduct: 2, Name: "Rowlet ", Number: "9", ExpansionName: "Sun & Moon"},
		{IDProduct: 3, Name: "Shiinotic ", Number: "17", ExpansionName: "Sun & Moon"},
		{IDProduct: 4, Name: "Mew ", Number: "1", ExpansionName: "Southern Islands"},
	}
	channel := make(chan responseChan, 16)
	for i := range products {
		if err := mkm.processProduct(channel, &products[i]); err != nil {
			t.Fatalf("processProduct(%q): %v", products[i].Name, err)
		}
	}
	close(channel)

	got := map[string][]float64{}
	for result := range channel {
		got[result.cardID] = append(got[result.cardID], result.entry.Price)
	}
	want := map[string][]float64{
		"sm04_127160_holofoil":          {1, 2},
		"9-149_126880":                  {3, 4},
		"9-149_126880_reverseholofoil":  {5, 6},
		"17-149_126888_holofoil":        {7, 8},
		"17-149_126888_reverseholofoil": {9, 10},
		"01-18_46466_reverseholofoil":   {13, 14},
	}
	for uuid, prices := range want {
		if len(got[uuid]) != len(prices) {
			t.Errorf("%s priced %v, want %v", uuid, got[uuid], prices)
			continue
		}
		for i := range prices {
			if got[uuid][i] != prices[i] {
				t.Errorf("%s priced %v, want %v", uuid, got[uuid], prices)
				break
			}
		}
	}
	for uuid := range got {
		if _, expected := want[uuid]; !expected {
			t.Errorf("%s was priced %v and should not have been", uuid, got[uuid])
		}
	}
}
