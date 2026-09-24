package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaStarterFoilDatastore is two cards as published, each with the
// starter deck's exclusive foil filed as its Holofoil beside the Normal and
// the Cold Foil.
const lorcanaStarterFoilDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "10": {"name": "Whispers in the Well", "type": "expansion", "releaseDate": "2025-11-14"},
    "12": {"name": "Wilds Unknown", "type": "expansion", "releaseDate": "2026-05-15"}
  },
  "cards": [
    {"id": 2209, "name": "Simba", "version": "King in the Making", "fullName": "Simba - King in the Making", "setCode": "10", "number": "20", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "2209_foil", "promoTypes": ["freeform"]}, {"finish": "Holofoil", "id": "2209_holofoil", "promoTypes": ["rainbowpillars"]}, {"finish": "Normal", "id": "2209"}], "externalLinks": {"cardTraderId": 354170, "cardmarketId": 856011, "tcgPlayerId": 657894}},
    {"id": 2735, "name": "Jessie", "version": "Lively Cowgirl", "fullName": "Jessie - Lively Cowgirl", "setCode": "12", "number": "20", "total": "204", "rarity": "Super Rare", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "2735_foil"}, {"finish": "Holofoil", "id": "2735_holofoil", "promoTypes": ["rainbowpillars"]}, {"finish": "Normal", "id": "2735"}], "externalLinks": {"cardTraderId": 388534, "cardmarketId": 885598, "tcgPlayerId": 690204}}
  ]
}}`

// TestLorcanaStarterFoil pins that a blueprint whose version names a starter
// deck's exclusive foil lands on its card's Holofoil whatever its listing's
// flag, under either wording, while the card's own blueprint keeps the flag.
func TestLorcanaStarterFoil(t *testing.T) {
	b, err := mtgmatcher.Open("lorcana", strings.NewReader(lorcanaStarterFoilDatastore))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		bpID, tcgID       int
		name, version     string
		expansion, number string
		foil              bool
		want              string
	}{
		{354170, 657894, "Simba - King in the Making", "", "Whispers in the Well", "020", false, "2209"},
		{354170, 657894, "Simba - King in the Making", "", "Whispers in the Well", "020", true, "2209_foil"},
		{372385, 0, "Simba - King in the Making", "Starter Deck Exclusive Foil", "Whispers in the Well", "020s", false, "2209_holofoil"},
		{372385, 0, "Simba - King in the Making", "Starter Deck Exclusive Foil", "Whispers in the Well", "020s", true, "2209_holofoil"},
		{395142, 0, "Jessie - Lively Cowgirl", "Rainbow Foil", "Promos Year 3", "020", false, "2735_holofoil"},
	} {
		bp := &Blueprint{
			ID:          tt.bpID,
			Name:        tt.name,
			Version:     tt.version,
			GameID:      GameLorcana,
			CategoryID:  CategoryLorcanaSingles,
			TCGplayerID: tt.tcgID,
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
		p.Properties.LorcanaFoil = tt.foil

		ch := make(chan resultChan, 1)
		ct.processProducts(ch, bp.ID, []Product{p})
		close(ch)

		var got string
		for result := range ch {
			got = result.cardID
		}
		if got != tt.want {
			t.Errorf("blueprint %d (%q, foil %v) landed on %q, want %q", tt.bpID, tt.version, tt.foil, got, tt.want)
		}
	}
}
