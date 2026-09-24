package riftbound

import (
	"strings"
	"testing"
)

// numberFixture pins the three shapes a Riftbound collector number takes:
// a padded number with a variant letter, a starred printing, and a promo
// track whose numbers carry a letter prefix and no numeric identity of
// their own.
const numberFixture = `{"data": {
	"pageProps": {"page": {"blades": [{
		"type": "riftboundCardGallery",
		"sets": {"items": [
			{"id": "OGN", "name": "Origins", "baseSetSize": 298, "releaseDate": "2025-10-31"},
			{"id": "SFD", "name": "Spiritforged", "baseSetSize": 221, "releaseDate": "2026-03-27"},
			{"id": "OPP", "name": "Riftbound Organized Play Promotional Cards", "type": "promo", "releaseDate": "2025-10-31"}
		]},
		"cards": {"items": [
			{
				"id": "ogn-66a",
				"name": "Fixture Blade",
				"number": "066a",
				"setCode": "OGN",
				"rarity": {"value": {"id": "rare"}},
				"printings": [{"finish": "Foil", "id": "ogn-66a_foil"}]
			},
			{
				"id": "sfd-227",
				"name": "Fixture Cannon",
				"number": "227*",
				"setCode": "SFD",
				"rarity": {"value": {"id": "epic"}},
				"printings": [{"finish": "Foil", "id": "sfd-227_foil"}]
			},
			{
				"id": "opp-r2b",
				"name": "Fixture Rune",
				"number": "R2b",
				"setCode": "OPP",
				"rarity": {"value": {"id": "common"}},
				"printings": [{"finish": "Foil", "id": "opp-r2b_foil"}]
			}
		]}
	}]}}
}}`

// TestCollectorNumbers pins that a printing keeps the number it is sold
// under. PlainNumber is what a plain-number search matches, so it may
// drop the star and nothing else: filling it from the numeric collector
// number instead left every lettered printing unreachable by number.
func TestCollectorNumbers(t *testing.T) {
	b, err := Load(strings.NewReader(numberFixture))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		uuid         string
		number       string
		originalWant string
	}{
		{"ogn-66a_foil", "66a", "66a"},
		{"sfd-227_foil", "227*", "227"},
		{"opp-r2b_foil", "R2b", "R2b"},
	} {
		co, found := b.UUIDs[tt.uuid]
		if !found {
			t.Errorf("%s is not a card", tt.uuid)
			continue
		}
		if co.Number != tt.number {
			t.Errorf("%s: Number is %q, want %q", tt.uuid, co.Number, tt.number)
		}
		if co.PlainNumber != tt.originalWant {
			t.Errorf("%s: PlainNumber is %q, want %q", tt.uuid, co.PlainNumber, tt.originalWant)
		}
	}
}
