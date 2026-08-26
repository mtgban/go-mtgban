package riftbound

import (
	"strings"
	"testing"
)

// numberFixture pins the three shapes a Riftbound collector number takes:
// a padded number with a variant letter, a starred printing, and a promo
// track whose numbers carry a letter prefix and no numeric identity of
// their own.
const numberFixture = `{
	"pageProps": {"page": {"blades": [{
		"type": "riftboundCardGallery",
		"sets": {"items": [
			{"id": "OGN", "name": "Origins", "collectorNumberMax": 298, "releaseDate": "2025-10-31"},
			{"id": "SFD", "name": "Spiritforged", "collectorNumberMax": 221, "releaseDate": "2026-03-27"},
			{"id": "OPP", "name": "Riftbound Organized Play Promotional Cards", "type": "promo", "releaseDate": "2025-10-31"}
		]},
		"cards": {"items": [
			{
				"id": "ogn-66a",
				"collectorNumber": 66,
				"name": "Fixture Blade",
				"publicCode": "OGN-066a/298",
				"set": {"value": {"id": "OGN", "label": "Origins"}},
				"rarity": {"value": {"id": "rare"}},
				"finishes": ["foil"]
			},
			{
				"id": "sfd-227",
				"collectorNumber": 227,
				"name": "Fixture Cannon",
				"publicCode": "SFD-227*/221",
				"set": {"value": {"id": "SFD", "label": "Spiritforged"}},
				"rarity": {"value": {"id": "epic"}},
				"finishes": ["foil"]
			},
			{
				"id": "opp-r2b",
				"collectorNumber": 0,
				"name": "Fixture Rune",
				"publicCode": "OPP-R2b/221",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "common"}},
				"finishes": ["foil"]
			}
		]}
	}]}}
}`

// TestCollectorNumbers pins that a printing keeps the number it is sold
// under. OriginalNumber is what a plain-number search matches, so it may
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
		if co.OriginalNumber != tt.originalWant {
			t.Errorf("%s: OriginalNumber is %q, want %q", tt.uuid, co.OriginalNumber, tt.originalWant)
		}
	}
}
