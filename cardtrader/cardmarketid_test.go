package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// seaDatastore is the published Pokemon datastore cut down to one collector
// number's printings, copied verbatim: the stamped promo was minted from the
// Cardmarket product that sells it, and no TCGplayer product does.
const seaDatastore = `{"data": {
 "game": "pokemon",
 "sets": {
  "SVI": {"abbreviation": "SVI", "baseSetSize": 198, "name": "SV01: Scarlet & Violet Base Set", "releaseDate": "2023-03-31"},
  "G22880": {"name": "Prize Pack Series Cards", "releaseDate": "2022-11-30"},
  "PR-1840": {"abbreviation": "PR", "name": "Deck Exclusives", "releaseDate": "2016-09-01"},
  "WCD2024": {"name": "2024 World Championship Decks", "releaseDate": "2024-08-01"},
  "SEA": {"abbreviation": "SEA", "name": "Southeast Asia Exclusives", "releaseDate": "2023-08-05"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 488007, "tcgdexId": "sv01-118"}, "finish": "Holofoil", "id": "118-198_488007_holofoil", "name": "Hawlucha", "number": "118", "rarity": "Rare", "setCode": "SVI", "total": "198", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 488007, "tcgdexId": "sv01-118"}, "finish": "Reverse Holofoil", "id": "118-198_488007_reverseholofoil", "name": "Hawlucha", "number": "118", "rarity": "Rare", "setCode": "SVI", "total": "198", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 515385}, "finish": "Normal", "id": "118-198_515385", "name": "Hawlucha", "number": "118", "rarity": "Rare", "setCode": "G22880", "total": "198", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 583734}, "finish": "Normal", "id": "118-198_583734", "name": "Hawlucha", "number": "118", "rarity": "Rare", "setCode": "PR-1840", "total": "198", "type": "Fighting"},
  {"externalLinks": {"tcgPlayerId": 637564}, "finish": "Normal", "id": "118-198_637564", "name": "Hawlucha", "number": "118", "rarity": "Rare", "setCode": "WCD2024", "total": "198", "type": "Fighting", "variant": "Evan Pavelski", "watermark": "evan pavelski"},
  {"externalLinks": {"cardmarketId": 779978}, "finish": "Holofoil", "id": "118-198_mkm779978_holofoil", "name": "Hawlucha", "number": "118", "rarity": "Promo", "setCode": "SEA", "total": "198"}
 ]
}}`

// TestBlueprintCardmarketID pins that a blueprint no TCGplayer product
// sells lands through the Cardmarket product it names, where the name alone
// aliased the stamped promo with every other printing of its number.
func TestBlueprintCardmarketID(t *testing.T) {
	b, err := mtgmatcher.Open("pokemon", strings.NewReader(seaDatastore))
	if err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		ID:            311508,
		Name:          "Hawlucha",
		Version:       "SVI 118",
		GameID:        GamePokemon,
		CategoryID:    CategoryPokemonSingles,
		CardMarketIDs: []int{779978},
	}
	bp.Expansion.Name = "Southeast Asia Gym Promos"
	bp.Properties.Number = "SVI 118"

	ct := &Market{
		backend:     b,
		gameID:      GamePokemon,
		blueprints:  map[int]*Blueprint{bp.ID: bp},
		logCallback: func(string, ...any) {},
	}

	var p Product
	p.ID = 1
	p.BlueprintID = bp.ID
	p.Quantity = 1
	p.Price.Cents = 100
	p.Price.Currency = "USD"
	p.Properties.Condition = "Near Mint"
	p.Properties.PokemonLanguage = "en"

	ch := make(chan resultChan, 1)
	ct.processProducts(ch, bp.ID, []Product{p})
	close(ch)

	var got string
	for result := range ch {
		got = result.cardID
	}
	if got != "118-198_mkm779978_holofoil" {
		t.Errorf("the SEA Hawlucha blueprint landed on %q, want 118-198_mkm779978_holofoil", got)
	}
}
