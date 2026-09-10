package riftbound

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// thirdFinishFixture sells one printing in a finish that is neither plain
// nor foil, named the way TCGplayer would price it, beside a card sold both
// ways.
const thirdFinishFixture = `{
	"pageProps": {"page": {"blades": [{
		"type": "riftboundCardGallery",
		"sets": {"items": [
			{"id": "OGN", "name": "Origins", "collectorNumberMax": 298, "releaseDate": "2025-10-31"}
		]},
		"cards": {"items": [
			{
				"id": "ogn-001",
				"collectorNumber": 1,
				"name": "Fixture Blade",
				"publicCode": "OGN-001/298",
				"set": {"value": {"id": "OGN", "label": "Origins"}},
				"rarity": {"value": {"id": "rare"}},
				"tcgplayerProductId": 100,
				"printings": [{"finish": "Normal", "id": "ogn-001"}, {"finish": "Foil", "id": "ogn-001_f"}]
			},
			{
				"id": "ogn-002",
				"collectorNumber": 2,
				"name": "Fixture Cannon",
				"publicCode": "OGN-002/298",
				"set": {"value": {"id": "OGN", "label": "Origins"}},
				"rarity": {"value": {"id": "epic"}},
				"tcgplayerProductId": 200,
				"printings": [{"finish": "Prismatic Foil", "id": "ogn-002_prismatic"}]
			}
		]}
	}]}}
}`

// TestThirdFinish pins the loader half of CanonicalFinish placing a finish
// it was not taught: a printing sold in a treatment alone is a foil to the
// flag, its product id reaches it, and a card sold plain and foil is one
// candidate to Match however its uuids are spelled.
func TestThirdFinish(t *testing.T) {
	b, err := Load(strings.NewReader(thirdFinishFixture))
	if err != nil {
		t.Fatal(err)
	}
	co, err := b.GetUUID("ogn-002_prismatic")
	if err != nil {
		t.Fatal(err)
	}
	if !co.Foil {
		t.Error("a printing sold in a treatment alone was stored as plain")
	}
	if got := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]["200"]; got != "ogn-002_prismatic" {
		t.Errorf("product 200 points at %q, want the treatment's uuid", got)
	}
	got, err := b.MatchID("200", true)
	if err != nil || got != "ogn-002_prismatic" {
		t.Errorf("MatchID(200, foil) = %q, %v; want the treatment's uuid", got, err)
	}
	got, err = b.Match(&mtgmatcher.InputCard{Name: "Fixture Blade", Edition: "Origins", Variation: "001"})
	if err != nil || got != "ogn-001" {
		t.Errorf("Match(Fixture Blade 001) = %q, %v; want ogn-001", got, err)
	}
}
