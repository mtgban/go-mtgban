package lorcana

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// finishNamesData sells one card under two of TCGplayer's names for the
// standard foil, one under the plain finish alone and one foil alone, in
// the printings shape the builder publishes.
const finishNamesData = `{
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {
    "1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"},
    "P1": {"name": "Promo Cards Year 1", "type": "promo", "releaseDate": "2023-09-01"}
  },
  "cards": [
    {
      "id": 100, "name": "Louie", "fullName": "Louie - One Cool Duck",
      "setCode": "1", "number": 1, "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "DuckTales",
      "printings": [{"finish": "Foil", "id": "100_other"}, {"finish": "Cold Foil", "id": "100_foil"}, {"finish": "Normal", "id": "100"}],
      "externalLinks": {"tcgPlayerId": 631349}
    },
    {
      "id": 200, "name": "Dewey", "fullName": "Dewey - Lovable Showoff",
      "setCode": "1", "number": 2, "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "DuckTales",
      "printings": [{"finish": "Normal", "id": "200"}],
      "externalLinks": {"tcgPlayerId": 631350}
    },
    {
      "id": 300, "name": "Huey", "fullName": "Huey - Reliable Leader",
      "setCode": "P1", "number": 1, "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "DuckTales",
      "printings": [{"finish": "Cold Foil", "id": "300_foil"}],
      "externalLinks": {"tcgPlayerId": 631351}
    }
  ]
}`

// TestFinishNames pins that two of TCGplayer's names for one finish place
// on it once, the first by name, whatever order the file lists them in.
func TestFinishNames(t *testing.T) {
	b, err := Load(strings.NewReader(finishNamesData))
	if err != nil {
		t.Fatal(err)
	}
	co, err := b.GetUUID("100")
	if err != nil {
		t.Fatal(err)
	}
	if got := co.FoilUUIDs[mtgmatcher.FinishFoil]; got != "100_foil" {
		t.Errorf("the standard foil is %q, want the first name's uuid 100_foil", got)
	}
	if _, err := b.GetUUID("100_other"); err == nil {
		t.Error("the second name's uuid was stored beside the first's")
	}
	got, err := b.Match(&mtgmatcher.InputCard{Name: "Louie - One Cool Duck", Edition: "The First Chapter", Variation: "1"})
	if err != nil || got != "100" {
		t.Errorf("Match(Louie 1) = %q, %v; want 100", got, err)
	}
}
