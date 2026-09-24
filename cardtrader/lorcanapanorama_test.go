package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// lorcanaPanoramaDatastore is Huey - Reliable Leader as published: TCGplayer
// sells the Normal as 631351 and the Panorama foil apart as 633429, which the
// datastore files on the same row as its Cold Foil.
const lorcanaPanoramaDatastore = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {"8": {"name": "The Reign of Jafar", "type": "expansion", "releaseDate": "2025-06-06"}},
  "cards": [
    {"id": 1667, "name": "Huey", "version": "Reliable Leader", "fullName": "Huey - Reliable Leader", "setCode": "8", "number": "3", "total": "204", "rarity": "Uncommon", "type": "Character", "printings": [{"finish": "Cold Foil", "id": "1667_foil"}, {"finish": "Normal", "id": "1667"}], "externalLinks": {"cardTraderId": 334319, "cardmarketId": 826338, "tcgPlayerId": 631351, "tcgPlayerExtraIds": [633429]}}
  ]
}}`

// TestLorcanaPanoramaFoil pins that a blueprint sold as the Panorama foil's
// own TCGplayer product lands on the foil whatever its listing's flag, while
// the plain card's blueprint keeps the flag's say.
func TestLorcanaPanoramaFoil(t *testing.T) {
	b, err := mtgmatcher.Open("lorcana", strings.NewReader(lorcanaPanoramaDatastore))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		bpID, tcgID int
		version     string
		foil        bool
		want        string
	}{
		{334319, 631351, "", false, "1667"},
		{334319, 631351, "", true, "1667_foil"},
		{332000, 633429, "Panorama", false, "1667_foil"},
		{332000, 633429, "Panorama", true, "1667_foil"},
	} {
		bp := &Blueprint{
			ID:          tt.bpID,
			Name:        "Huey - Reliable Leader",
			Version:     tt.version,
			GameID:      GameLorcana,
			CategoryID:  CategoryLorcanaSingles,
			TCGplayerID: tt.tcgID,
		}
		bp.Expansion.Name = "Reign of Jafar"
		bp.Properties.Number = "003"

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
