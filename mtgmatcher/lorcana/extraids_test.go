package lorcana

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// extraIDsData is a two-card cut of the datastore shape cmd/lorcanadatastore
// emits. Card 100 is sold by TCGplayer as two products, the plain art under
// the id upstream publishes and the Panorama foil under its own, whose image
// upstream publishes as fullFoil; card 200 carries no extra ids, as every card
// in the upstream file does.
const extraIDsData = `{"data": {
  "metadata": {"formatVersion": "2.3.5", "language": "en"},
  "sets": {"1": {"name": "The First Chapter", "type": "expansion", "releaseDate": "2023-09-01"}},
  "cards": [
    {
      "id": 100, "name": "Louie", "fullName": "Louie - One Cool Duck",
      "setCode": "1", "number": "1", "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "DuckTales", "printings": [{"finish": "Cold Foil", "id": "100_silver"}, {"finish": "Normal", "id": "100"}],
      "images": {"full": "framed.jpg", "fullFoil": "panorama.jpg", "thumbnail": "framed_thumb.jpg"},
      "externalLinks": {"tcgPlayerId": 631349, "tcgPlayerExtraIds": [633427]}
    },
    {
      "id": 200, "name": "Dewey", "fullName": "Dewey - Lovable Showoff",
      "setCode": "1", "number": "2", "rarity": "Common", "type": "Character",
      "color": "Amber", "story": "DuckTales", "printings": [{"finish": "Cold Foil", "id": "200_silver"}, {"finish": "Normal", "id": "200"}],
      "externalLinks": {"tcgPlayerId": 631350}
    }
  ]
}}`

func TestLorcanaExtraProductIds(t *testing.T) {
	b, err := Load(strings.NewReader(extraIDsData))
	if err != nil {
		t.Fatal(err)
	}

	// The extra id is the ★ twin's own product and reaches its foil alone;
	// the main product sells no foil, so a foil flag on it reaches the plain.
	for _, tc := range []struct {
		id   string
		foil bool
		want string
	}{
		{"631349", false, "100"},
		{"631349", true, "100"},
		{"633427", false, "100_silver"},
		{"633427", true, "100_silver"},
		{"631350", true, "200_silver"},
	} {
		got, err := b.MatchID(tc.id, tc.foil)
		if err != nil {
			t.Errorf("MatchID(%q, %v) = error %v", tc.id, tc.foil, err)
			continue
		}
		if got != tc.want {
			t.Errorf("MatchID(%q, %v) = %q, want %q", tc.id, tc.foil, got, tc.want)
		}
	}

	// The twin keeps the foil's uuid, so the uuid space is exactly what the
	// upstream file produces.
	if n := len(b.GetUUIDs()); n != 4 {
		t.Errorf("got %d uuids, want 4", n)
	}
	if _, err := b.GetUUID(""); err == nil {
		t.Error(`GetUUID("") resolved to a card`)
	}

	// The twin shows the Panorama's own art; the plain card keeps upstream's
	// images untouched.
	for _, tc := range []struct {
		uuid, full, thumbnail string
	}{
		{"100", "framed.jpg", "framed_thumb.jpg"},
		{"100_silver", "panorama.jpg", "panorama.jpg"},
	} {
		co, err := b.GetUUID(tc.uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Images["full"] != tc.full || co.Images["thumbnail"] != tc.thumbnail {
			t.Errorf("%s images = %v, want full %q and thumbnail %q", tc.uuid, co.Images, tc.full, tc.thumbnail)
		}
	}

	// An id absent from both maps must still be unknown.
	if _, err := b.MatchID("999999"); err == nil {
		t.Error("an unknown product id resolved")
	}
}

// TestLorcanaExtraProductIdsAbsent pins the backward-compatible half: the
// upstream file has no tcgPlayerExtraIds at all, and must load exactly as it
// does today.
func TestLorcanaExtraProductIdsAbsent(t *testing.T) {
	upstream := strings.Replace(extraIDsData, `, "tcgPlayerExtraIds": [633427]`, "", 1)
	b, err := Load(strings.NewReader(upstream))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]) != 2 {
		t.Errorf("got %d external ids, want 2", len(b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]))
	}
	if _, err := b.MatchID("633427", true); err != mtgmatcher.ErrCardUnknownID {
		t.Errorf("MatchID on the split-foil product = %v, want %v", err, mtgmatcher.ErrCardUnknownID)
	}
}
