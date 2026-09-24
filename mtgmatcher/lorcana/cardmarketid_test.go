package lorcana

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// cardmarketIDData cuts the datastore to three cards: one that names its
// Cardmarket product alone, and Moana and Vaiana, which upstream files under
// the same one.
const cardmarketIDData = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-08-18"},
    "7": {"name": "Archazia's Island", "type": "expansion", "releaseDate": "2025-03-21"}
  },
  "cards": [
    {"id": 101, "name": "Dr. Facilier's Cards", "fullName": "Dr. Facilier's Cards", "setCode": "1", "number": "101", "rarity": "Uncommon", "type": "Item", "color": "Emerald", "story": "The Princess And The Frog",
     "printings": [{"finish": "Cold Foil", "id": "101_foil"}, {"finish": "Normal", "id": "101"}],
     "externalLinks": {"cardTraderId": 258975, "cardmarketId": 729282, "tcgPlayerId": 508762}},
    {"id": 1433, "name": "Moana", "fullName": "Moana - Adventurer of Land and Sea", "setCode": "7", "number": "26", "rarity": "Special", "type": "Character", "color": "Sapphire", "story": "Moana",
     "printings": [{"finish": "Cold Foil", "id": "1433_foil"}],
     "externalLinks": {"cardTraderId": 311906, "cardmarketId": 801862, "tcgPlayerId": 601112}},
    {"id": 1663, "name": "Vaiana", "fullName": "Vaiana - Adventurer of Land and Sea", "setCode": "7", "number": "26", "rarity": "Special", "type": "Character", "color": "Sapphire", "story": "Moana",
     "printings": [{"finish": "Cold Foil", "id": "1663_foil"}],
     "externalLinks": {"cardTraderId": 311906, "cardmarketId": 801862}}
  ]
}}`

// TestCardmarketIDs pins that a Cardmarket id is stamped on every card that
// carries one, and indexed only where it names one card alone.
func TestCardmarketIDs(t *testing.T) {
	b, err := Load(strings.NewReader(cardmarketIDData))
	if err != nil {
		t.Fatal(err)
	}

	if got := b.ConvertID(mtgmatcher.IDSpaceCardmarket, "729282"); got != "101" {
		t.Errorf("ConvertID(729282) = %q, want the nonfoil printing 101", got)
	}
	if got := b.ConvertID(mtgmatcher.IDSpaceCardmarket, "801862"); got != "" {
		t.Errorf("ConvertID(801862) = %q, want nothing for an id two cards claim", got)
	}
	for _, uuid := range []string{"101", "1433_foil", "1663_foil"} {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Identifiers["mcmId"] == "" {
			t.Errorf("%s carries no mcmId", uuid)
		}
	}
}
