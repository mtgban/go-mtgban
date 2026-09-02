package lorcana

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// englishData is a four-card cut of the upstream shape. Cards 100 and 101 are
// the two arts of one printing, told apart only by the variant letter, and
// share every storefront id: cardmarket and cardtrader routinely sell one
// product per printing regardless of art. Cards 200 and 201 are the regionally
// renamed repeat of a single card, which is what the fold exists for.
const englishData = `{
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {"1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"}},
  "cards": [
    {
      "id": 100, "name": "Dalmatian Puppy", "fullName": "Dalmatian Puppy - Tail Wagger",
      "setCode": "1", "number": 4, "variant": "a", "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "101 Dalmatians", "foilTypes": ["None", "Silver"],
      "externalLinks": {"tcgPlayerId": 500, "cardmarketId": 600, "cardTraderId": 700}
    },
    {
      "id": 101, "name": "Dalmatian Puppy", "fullName": "Dalmatian Puppy - Tail Wagger",
      "setCode": "1", "number": 4, "variant": "b", "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "101 Dalmatians", "foilTypes": ["None", "Silver"],
      "externalLinks": {"tcgPlayerId": 500, "cardmarketId": 600, "cardTraderId": 700}
    },
    {
      "id": 200, "name": "Moana", "fullName": "Moana - Adventurer of Land and Sea",
      "setCode": "1", "number": 26, "rarity": "Rare", "type": "Character",
      "color": "Amber", "story": "Moana", "foilTypes": ["None", "Silver"],
      "externalLinks": {"tcgPlayerId": 900, "cardmarketId": 901, "cardTraderId": 902}
    },
    {
      "id": 201, "name": "Vaiana", "fullName": "Vaiana - Adventurer of Land and Sea",
      "setCode": "1", "number": 26, "rarity": "Rare", "type": "Character",
      "color": "Amber", "story": "Moana", "foilTypes": ["None", "Silver"],
      "externalLinks": {"tcgPlayerId": 900, "cardmarketId": 901, "cardTraderId": 902}
    }
  ]
}`

// TestEnglishCardsKeepsArtSiblings pins that the identity behind the fold is
// the printed collector number, letter included: two arts of one number are
// two cards even when every storefront sells them under one product.
func TestEnglishCardsKeepsArtSiblings(t *testing.T) {
	b, err := Load(strings.NewReader(englishData))
	if err != nil {
		t.Fatal(err)
	}

	for uuid, number := range map[string]string{"100": "4a", "101": "4b"} {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Errorf("art sibling %s did not load: %v", uuid, err)
			continue
		}
		if co.Number != number {
			t.Errorf("%s: number %q, want %q", uuid, co.Number, number)
		}
	}

	// The regionally renamed repeat still folds, and its English name wins.
	if _, err := b.GetUUID("201"); err == nil {
		t.Error("the renamed repeat was kept as a card of its own")
	}
	if _, err := b.GetUUID("200"); err != nil {
		t.Errorf("the English printing did not load: %v", err)
	}

	// The id the two arts share names neither of them, so it goes
	// unregistered; the id the fold freed resolves to the survivor.
	if uuid, found := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]["500"]; found {
		t.Errorf("the product id both arts claim resolved to %q", uuid)
	}
	if uuid := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]["900"]; uuid != "200" {
		t.Errorf("the freed product id resolved to %q, want %q", uuid, "200")
	}
}
