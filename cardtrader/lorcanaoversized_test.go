package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaOversizedDatastore is the published datastore cut down to Hades,
// whose oversized printing TCGplayer sells and the datastore carries beside
// the ordinary card, and Baymax, whose oversized card it does not carry.
const lorcanaOversizedDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"},
    "6": {"name": "Azurite Sea", "type": "expansion", "releaseDate": "2024-11-25"}
  },
  "cards": [
    {"id": 5, "name": "Hades", "version": "King of Olympus", "fullName": "Hades - King of Olympus", "setCode": "1", "number": "5", "total": "204", "rarity": "Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "5_foil"}, {"finish": "Normal", "id": "5"}], "externalLinks": {"cardTraderId": 258949, "cardmarketId": 727085, "tcgPlayerId": 485364}},
    {"id": 1356, "name": "Baymax", "version": "Armored Companion", "fullName": "Baymax - Armored Companion", "setCode": "6", "number": "157", "total": "204", "rarity": "Legendary", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "1356_foil"}, {"finish": "Normal", "id": "1356"}], "externalLinks": {"cardTraderId": 302044, "cardmarketId": 794871, "tcgPlayerId": 578165}},
    {"id": -516775, "name": "Hades - King of Olympus (Oversized)", "fullName": "Hades - King of Olympus (Oversized)", "setCode": "1", "number": "5", "total": "204", "rarity": "Rare", "type": "Character", "promoTypes": ["oversized"], "printings": [{"finish": "Cold Foil", "id": "m-516775_foil"}], "externalLinks": {"tcgPlayerId": 516775}}
  ]
}}`

// TestLorcanaOversized pins that a blueprint in Card Trader's oversized
// category lands on the oversized printing at its number, by name as by its
// TCGplayer id, and is skipped where the datastore carries none, rather than
// priced on the ordinary card.
func TestLorcanaOversized(t *testing.T) {
	b, err := mtgmatcher.Open("lorcana", strings.NewReader(lorcanaOversizedDatastore))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		desc                    string
		bpID, category, tcgID   int
		name, expansion, number string
		want                    string
	}{
		{"the ordinary card", 302044, CategoryLorcanaSingles, 578165, "Baymax - Armored Companion", "Azurite Sea", "157", "1356"},
		{"no oversized printing of ours", 353325, CategoryLorcanaOversized, 0, "Baymax - Armored Companion", "Azurite Sea", "157", ""},
		{"an oversized printing by its id", 278385, CategoryLorcanaOversized, 516775, "Hades - King of Olympus", "The First Chapter", "005", "m-516775_foil"},
		{"an oversized printing by name", 278385, CategoryLorcanaOversized, 0, "Hades - King of Olympus", "The First Chapter", "005", "m-516775_foil"},
	} {
		bp := &Blueprint{
			ID:          tt.bpID,
			Name:        tt.name,
			Version:     "Oversized",
			GameID:      GameLorcana,
			CategoryID:  tt.category,
			TCGplayerID: tt.tcgID,
		}
		if tt.category == CategoryLorcanaSingles {
			bp.Version = ""
		}
		bp.Expansion.Name = tt.expansion
		bp.Properties.Number = tt.number

		ct := &Market{
			backend:     b,
			gameID:      GameLorcana,
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
		p.Properties.LorcanaLanguage = "en"

		ch := make(chan resultChan, 1)
		ct.processProducts(ch, bp.ID, []Product{p})
		close(ch)

		var got string
		for result := range ch {
			got = result.cardID
		}
		if got != tt.want {
			t.Errorf("%s: blueprint %d landed on %q, want %q", tt.desc, tt.bpID, got, tt.want)
		}
	}
}
