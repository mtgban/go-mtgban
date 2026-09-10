package palworld

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// tailFixture holds two cards sold in three rarities apiece, the parallel
// tiers numbered apart from the card they parallel, every row the
// datastore's.
const tailFixture = `{
	"game": "palworld",
	"sets": {
		"BP01": {"name": "BP01: Dawn of Palpagos", "releaseDate": "2026-07-30"},
		"TD01": {"name": "TD01: Dawn of Palpagos Red•Blue", "releaseDate": "2026-07-30"}
	},
	"cards": [
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713879}, "finish": "Foil", "id": "ebp01-001_713879_foil", "image": "x", "name": "Jormuntide Ignis - Savage Lava Dragon", "number": "EBP01-001", "rarity": "Double Rare", "setCode": "BP01", "type": "Pal"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713880}, "finish": "Foil", "id": "ebp01-001osr_713880_foil", "image": "x", "name": "Jormuntide Ignis - Savage Lava Dragon", "number": "EBP01-001OSR", "rarity": "Over Super Rare", "setCode": "BP01", "type": "Pal"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713881}, "finish": "Foil", "id": "ebp01-001ssp_713881_foil", "image": "x", "name": "Jormuntide Ignis - Savage Lava Dragon", "number": "EBP01-001SSP", "rarity": "Super Special Parallel", "setCode": "BP01", "type": "Pal"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713801}, "finish": "Normal", "id": "etd01-001_713801", "image": "x", "name": "Grizzbolt - Rumbling Tank", "number": "ETD01-001", "rarity": "Trial Deck Rare", "setCode": "TD01", "type": "Pal"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713802}, "finish": "Foil", "id": "etd01-001tsp_713802_foil", "image": "x", "name": "Grizzbolt - Rumbling Tank", "number": "ETD01-001TSP", "rarity": "Trial Deck Super Parallel", "setCode": "TD01", "type": "Pal"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 713803}, "finish": "Foil", "id": "etd01-001tsr_713803_foil", "image": "x", "name": "Grizzbolt - Rumbling Tank", "number": "ETD01-001TSR", "rarity": "Trial Deck Super Deck Rare", "setCode": "TD01", "type": "Pal"}
	]
}`

// TestTailSaidLongest pins that a rarity spelled inside a longer one is not
// the one named: "Over Super Rare" says "Super Rare" and "Trial Deck Super
// Parallel" says "Super Parallel", and reading both as two codes named
// answered the base printing for the 15 OSR and 4 TSP cards.
func TestTailSaidLongest(t *testing.T) {
	for _, tt := range []struct{ wording, want string }{
		{"EBP01-001 Over Super Rare", "OSR"},
		{"EBP01-001 Super Rare", "SR"},
		{"ETD01-001 Trial Deck Super Parallel", "TSP"},
		{"ETD01-001 Super Parallel", "SP"},
		{"ETD01-001 TSR", "TSR"},
		{"EBP01-001 SR OSR", ""},
		{"EBP01-001", ""},
	} {
		if got := tailSaid(tt.wording); got != tt.want {
			t.Errorf("tailSaid(%q) = %q, want %q", tt.wording, got, tt.want)
		}
	}

	b, err := Load(strings.NewReader(tailFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, variation, want string }{
		{"Jormuntide Ignis - Savage Lava Dragon", "EBP01-001 Over Super Rare", "ebp01-001osr_713880_foil"},
		{"Jormuntide Ignis - Savage Lava Dragon", "EBP01-001", "ebp01-001_713879_foil"},
		{"Grizzbolt - Rumbling Tank", "ETD01-001 Trial Deck Super Parallel", "etd01-001tsp_713802_foil"},
	} {
		in := mtgmatcher.InputCard{Name: tt.name, Variation: tt.variation}
		got, err := b.Match(&in)
		if err != nil || got != tt.want {
			t.Errorf("Match(%q, %q) = %q, %v; want %s", tt.name, tt.variation, got, err, tt.want)
		}
	}
}
