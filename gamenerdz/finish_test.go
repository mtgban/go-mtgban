package gamenerdz

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/pokemon"
)

func TestNameSaysFoil(t *testing.T) {
	tests := []struct {
		displayName string
		foil        bool
	}{
		{"Necrologia (7ED-149) - 7th Edition Foil", true},
		{"Necrologia (7ED-149) - 7th Edition", false},
		{"Kilnmouth Dragon (LGN-104) - Legions Foil(MP)", true},
		{"Adarkar Wastes (DMU-243) - Dominaria United Foil (LP)", true},
		{"Aboleth Spawn - Commander Legends: Battle for Baldur's Gate: (Extended Art) Foil", true},
		// A set can be called a foil edition without the product being
		// one, so only the finish the shelf ends in counts.
		{"Richard Garfield, Ph.D. (LIST-017) - The List (Unfinity Foil Edition)", false},
		{"Richard Garfield, Ph.D. (LIST-017) - The List (Unfinity Foil Edition) Foil", true},
	}
	for _, tt := range tests {
		foil := nameSaysFoil(tt.displayName)
		if foil != tt.foil {
			t.Errorf("%q: got %v; want %v", tt.displayName, foil, tt.foil)
		}
	}
}

// tiebreakDatastore is the published Pokemon datastore cut down to the cards
// the tiebreak turns on, every row and set copied verbatim from it: POP
// Series 1's Armaldo ex in both finishes, which share a set and a number, and
// Seel at number 41 in Base Set and in its Shadowless run.
const tiebreakDatastore = `{"data": {
 "game": "pokemon",
 "sets": {
  "BS": {"abbreviation": "BS", "baseSetSize": 102, "name": "Base Set", "releaseDate": "1999-01-09", "symbol": "https://images.pokemontcg.io/base1/symbol.png"},
  "BSS": {"abbreviation": "BSS", "baseSetSize": 102, "name": "Base Set (Shadowless)", "releaseDate": "1999-01-09"},
  "PL": {"abbreviation": "PL", "baseSetSize": 127, "name": "Platinum", "releaseDate": "2009-02-11", "symbol": "https://assets.tcgdex.net/en/pl/pl1/symbol.png"},
  "POP-1422": {"abbreviation": "POP", "baseSetSize": 17, "name": "POP Series 1", "releaseDate": "2004-09-01", "symbol": "https://assets.tcgdex.net/en/pop/pop1/symbol.png"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 83623, "tcgdexId": "pop1-16"}, "finish": "Normal", "id": "016-017_83623", "image": "https://assets.tcgdex.net/en/pop/pop1/16/high.webp", "name": "Armaldo ex", "number": "016", "rarity": "Ultra Rare", "setCode": "POP-1422", "total": "017", "type": "Fighting", "types": ["Fighting"], "variant": "Non-Holo", "watermark": "non-holo"},
  {"externalLinks": {"tcgPlayerId": 97806, "tcgdexId": "pop1-16"}, "finish": "Holofoil", "id": "016-017_97806_holofoil", "image": "https://assets.tcgdex.net/en/pop/pop1/16/high.webp", "name": "Armaldo ex", "number": "016", "rarity": "Ultra Rare", "setCode": "POP-1422", "total": "017", "type": "Fighting", "types": ["Fighting"], "variant": "Holo", "watermark": "holo"},
  {"externalLinks": {"tcgPlayerId": 42384, "tcgdexId": "base1-41"}, "finish": "Normal", "id": "041-102_42384", "image": "https://assets.tcgdex.net/en/base/base1/41/high.webp", "name": "Seel", "number": "041", "rarity": "Uncommon", "setCode": "BS", "total": "102", "type": "Water", "types": ["Water"]},
  {"externalLinks": {"tcgPlayerId": 107038}, "finish": "1st Edition", "id": "041-102_107038_1stedition", "image": "https://tcgplayer-cdn.tcgplayer.com/product/107038_400w.jpg", "name": "Seel", "number": "041", "rarity": "Uncommon", "setCode": "BSS", "total": "102", "type": "Water", "types": ["Water"]},
  {"externalLinks": {"tcgPlayerId": 107038}, "finish": "Unlimited", "id": "041-102_107038_unlimited", "image": "https://tcgplayer-cdn.tcgplayer.com/product/107038_400w.jpg", "name": "Seel", "number": "041", "rarity": "Uncommon", "setCode": "BSS", "total": "102", "type": "Water", "types": ["Water"]}
 ]
}}`

// TestFinishTiebreak pins that a listing the wording leaves between two
// finishes of one printing is placed by its own finish, and that the finish
// does not choose between printings the wording merely failed to tell apart.
func TestFinishTiebreak(t *testing.T) {
	b, err := matcher.Open("pokemon", strings.NewReader(tiebreakDatastore))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		displayName string
		setName     string
		finish      string
		uuid        string
	}{
		{"Armaldo ex 16/17 - POP Series 1", "POP Series 1", "Normal", "016-017_83623"},
		{"Armaldo ex 16/17 - POP Series 1 Holofoil", "POP Series 1", "Holofoil", "016-017_97806_holofoil"},
		// The body names Platinum, so the one plain Seel is not the card
		// this Shadowless listing stands for.
		{"Seel 41 - Base Set Shadowless Unlimited(DMG)", "Platinum", "Normal", ""},
	}
	for _, tt := range tests {
		gn, err := NewScraper(b)
		if err != nil {
			t.Fatal(err)
		}
		product := GNProduct{
			ID:             tt.displayName,
			DisplayName:    tt.displayName,
			SelectedFinish: tt.finish,
			ProductData:    GNProductData{SetName: tt.setName},
		}
		got, err := gn.resolveProduct(modeBuylist, product)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tt.displayName, err)
			continue
		}
		if got != tt.uuid {
			t.Errorf("%q: got %q; want %q", tt.displayName, got, tt.uuid)
		}
	}
}
