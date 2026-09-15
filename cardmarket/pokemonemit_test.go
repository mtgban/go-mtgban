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
// holo beside a reverse holo, the Mew of Southern Islands sold as a
// reverse holo and in nothing else, and the Team Rocket Charizard sold as
// Unlimited Holofoil beside 1st Edition Holofoil - the print-run axis, no
// reverse holo involved at all (mtgban/go-mtgban#641; matches Team
// Rocket's real #4/#84572 checked live against Cardmarket).
const pokemonEmitDatastore = `{
 "game": "pokemon",
 "sets": {"SMP": {"abbreviation": "SMP", "name": "SM Promos", "releaseDate": "2016-11-18", "type": "promo"}, "SM01": {"abbreviation": "SM01", "name": "SM Base Set", "releaseDate": "2017-02-03"}, "SI": {"abbreviation": "SI", "name": "Southern Islands", "releaseDate": "2001-07-31"}, "TR": {"abbreviation": "TR", "name": "Team Rocket", "releaseDate": "2000-04-24"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 46466}, "finish": "Reverse Holofoil", "id": "01-18_46466_reverseholofoil", "name": "Mew", "number": "01", "rarity": "Promo", "setCode": "SI", "total": "18"},
  {"externalLinks": {"tcgPlayerId": 127160}, "finish": "Holofoil", "id": "sm04_127160_holofoil", "name": "Pikachu", "number": "SM04", "rarity": "Promo", "setCode": "SMP"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Normal", "id": "9-149_126880", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126880}, "finish": "Reverse Holofoil", "id": "9-149_126880_reverseholofoil", "name": "Rowlet", "number": "9", "rarity": "Common", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126888}, "finish": "Holofoil", "id": "17-149_126888_holofoil", "name": "Shiinotic", "number": "17", "rarity": "Holo Rare", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 126888}, "finish": "Reverse Holofoil", "id": "17-149_126888_reverseholofoil", "name": "Shiinotic", "number": "17", "rarity": "Holo Rare", "setCode": "SM01"},
  {"externalLinks": {"tcgPlayerId": 84572}, "finish": "Unlimited Holofoil", "id": "04-82_84572_unlimitedholofoil", "name": "Charizard", "number": "4", "rarity": "Holo Rare", "setCode": "TR"},
  {"externalLinks": {"tcgPlayerId": 84572}, "finish": "1st Edition Holofoil", "id": "04-82_84572_1steditionholofoil", "name": "Charizard", "number": "4", "rarity": "Holo Rare", "setCode": "TR"}
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
// The Team Rocket Charizard pins the print-run axis specifically
// (mtgban/go-mtgban#641): the guide has no column of its own for 1st
// Edition, so both it and the Unlimited Holofoil sibling now share the
// product's own first pair rather than only one of them getting it and
// the other nothing.
func TestEmitPokemonColumns(t *testing.T) {
	b := datastoreBackend(t, "pokemon", pokemonEmitDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.exchangeRate = 1
	mkm.priceGuide = map[int]cm.PriceGuide{
		1: {IDProduct: 1, LowPrice: 1, TrendPrice: 2},
		2: {IDProduct: 2, LowPrice: 3, TrendPrice: 4, HoloLowPrice: 5, HoloTrendPrice: 6},
		3: {IDProduct: 3, LowPrice: 7, TrendPrice: 8, HoloLowPrice: 9, HoloTrendPrice: 10},
		4: {IDProduct: 4, LowPrice: 11, TrendPrice: 12, HoloLowPrice: 13, HoloTrendPrice: 14},
		// A holo pair is present but must go unpriced entirely: neither
		// sibling is Reverse Holofoil, so cardIDFoil never resolves, the
		// same as product 1's Pikachu above - this just also has a second
		// uuid (the 1st Edition Holofoil sibling) that the first pair now
		// reaches too.
		5: {IDProduct: 5, LowPrice: 15, TrendPrice: 16, HoloLowPrice: 17, HoloTrendPrice: 18},
	}
	products := []cm.Product{
		{IDProduct: 1, Name: "Pikachu ", Number: "04", ExpansionName: "SM Black Star Promos"},
		{IDProduct: 2, Name: "Rowlet ", Number: "9", ExpansionName: "Sun & Moon"},
		{IDProduct: 3, Name: "Shiinotic ", Number: "17", ExpansionName: "Sun & Moon"},
		{IDProduct: 4, Name: "Mew ", Number: "1", ExpansionName: "Southern Islands"},
		{IDProduct: 5, Name: "Charizard ", Number: "4", ExpansionName: "Team Rocket"},
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
		"sm04_127160_holofoil":           {1, 2},
		"9-149_126880":                   {3, 4},
		"9-149_126880_reverseholofoil":   {5, 6},
		"17-149_126888_holofoil":         {7, 8},
		"17-149_126888_reverseholofoil":  {9, 10},
		"01-18_46466_reverseholofoil":    {13, 14},
		"04-82_84572_unlimitedholofoil":  {15, 16},
		"04-82_84572_1steditionholofoil": {15, 16},
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
