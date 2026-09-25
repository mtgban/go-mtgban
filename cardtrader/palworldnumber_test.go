package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/palworld"
)

// palworldNumberDatastore is the rows the placeholder cases reach, as
// published on 2026-09-25.
const palworldNumberDatastore = `{"meta": {"date": "2026-09-25", "version": "1"}, "data": {
  "game": "palworld",
  "sets": {
    "BP01": {"name": "BP01: Dawn of Palpagos", "releaseDate": "2026-07-30"},
    "PR": {"name": "Palworld Promo Cards", "releaseDate": "2026-07-30", "type": "promo"},
    "TD01": {"name": "TD01: Dawn of Palpagos Red•Blue", "releaseDate": "2026-07-30"}
  },
  "cards": [
    {"id": "etd01-008_713812", "name": "Stone Pit", "number": "ETD01-008", "setCode": "TD01", "rarity": "Trial Deck", "finish": "Normal", "externalLinks": {"tcgPlayerId": 713812}},
    {"id": "etd01-008tsr_713813_foil", "name": "Stone Pit", "number": "ETD01-008TSR", "setCode": "TD01", "rarity": "Trial Deck Super Deck Rare", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713813}},
    {"id": "etd01-012_713818", "name": "Elphidran Aqua - Gentle Ripples", "number": "ETD01-012", "setCode": "TD01", "rarity": "Trial Deck Rare", "finish": "Normal", "externalLinks": {"tcgPlayerId": 713818}},
    {"id": "etd01-012tsp_713819_foil", "name": "Elphidran Aqua - Gentle Ripples", "number": "ETD01-012TSP", "setCode": "TD01", "rarity": "Trial Deck Super Parallel", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713819}},
    {"id": "etd01-012tsr_713820_foil", "name": "Elphidran Aqua - Gentle Ripples", "number": "ETD01-012TSR", "setCode": "TD01", "rarity": "Trial Deck Super Deck Rare", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713820}},
    {"id": "etd01-023_713835", "name": "Lamball - My First Pal", "number": "ETD01-023", "setCode": "TD01", "rarity": "Trial Deck", "finish": "Normal", "externalLinks": {"tcgPlayerId": 713835}},
    {"id": "etd01-023tsr_713836_foil", "name": "Lamball - My First Pal", "number": "ETD01-023TSR", "setCode": "TD01", "rarity": "Trial Deck Super Deck Rare", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713836}},
    {"id": "ebp01-001_713879_foil", "name": "Jormuntide Ignis - Savage Lava Dragon", "number": "EBP01-001", "setCode": "BP01", "rarity": "Double Rare", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713879}},
    {"id": "ebp01-001ssp_713881_foil", "name": "Jormuntide Ignis - Savage Lava Dragon", "number": "EBP01-001SSP", "setCode": "BP01", "rarity": "Super Special Parallel", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713881}},
    {"id": "ebp01-039_713941_foil", "name": "Antique Curtain", "number": "EBP01-039", "setCode": "BP01", "rarity": "Rare", "finish": "Foil", "externalLinks": {"tcgPlayerId": 713941}},
    {"id": "ebp01-041_713944", "name": "Hot Spring", "number": "EBP01-041", "setCode": "BP01", "rarity": "Uncommon", "finish": "Normal", "externalLinks": {"tcgPlayerId": 713944}},
    {"id": "epr-001_714388", "name": "Lamball - My First Pal", "number": "EPR-001", "setCode": "PR", "rarity": "Promo", "finish": "Normal", "externalLinks": {"tcgPlayerId": 714388}}
  ],
  "sealed": []
}}`

// TestPalworldPlaceholderNumber pins that a trial-deck blueprint carrying
// Jormuntide's EBP01-001SSP lands on its own card's printing on the shelf,
// and that a card with two parallels, or a number from its own set, is
// still refused rather than guessed.
func TestPalworldPlaceholderNumber(t *testing.T) {
	b, err := mtgmatcher.Open("palworld", strings.NewReader(palworldNumberDatastore))
	if err != nil {
		t.Fatal(err)
	}

	const td01 = "Dawn of Palpagos Red・Blue"
	for _, tt := range []struct {
		bpID                     int
		name, version, expansion string
		number                   string
		want                     string
	}{
		{410331, "Stone Pit", "Trial Deck", td01, "EBP01-001SSP", "etd01-008_713812"},
		{410332, "Stone Pit", "Trial Deck | Special Rare", td01, "EBP01-001SSP", "etd01-008tsr_713813_foil"},
		// Lamball's promo is another run; the shelf names the trial deck's.
		{410355, "Lamball - My First Pal", "Trial Deck | Special Rare", td01, "EBP01-001SSP", "etd01-023tsr_713836_foil"},
		{410337, "Elphidran Aqua - Gentle Ripples", "Trial Deck", td01, "EBP01-001SSP", "etd01-012_713818"},
		{410338, "Elphidran Aqua - Gentle Ripples", "Trial Deck | Special Rare", td01, "EBP01-001SSP", ""},
		{409925, "Jormuntide Ignis - Savage Lava Dragon", "Super Special Parallel", "Dawn of Palpagos", "EBP01-001SSP", "ebp01-001ssp_713881_foil"},
		{410220, "Antique Curtain", "", "Dawn of Palpagos", "EBP01-041", ""},
	} {
		bp := &Blueprint{
			ID:         tt.bpID,
			Name:       tt.name,
			Version:    tt.version,
			GameID:     GamePalworld,
			CategoryID: CategoryPalworldSingles,
		}
		bp.Expansion.Name = tt.expansion
		bp.Properties.Number = tt.number

		ct := &Market{
			backend:     b,
			gameID:      GamePalworld,
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
		p.Properties.PalworldLanguage = "en"

		ch := make(chan resultChan, 1)
		ct.processProducts(ch, bp.ID, []Product{p})
		close(ch)
		got := ""
		for r := range ch {
			got = r.cardID
		}
		if got != tt.want {
			t.Errorf("%d %s %q: got %q, want %q", tt.bpID, tt.name, tt.version, got, tt.want)
		}
	}
}
