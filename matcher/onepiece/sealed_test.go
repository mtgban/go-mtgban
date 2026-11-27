package onepiece

import (
	"maps"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// sealedIDsFixture is one card and two sealed products: one carrying an
// identifiers object beside its TCGplayer id, one carrying only the
// TCGplayer id, as every datastore publishes today.
const sealedIDsFixture = `{"data": {
	"game": "onepiece",
	"sets": {
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02", "type": "promo"}
	},
	"cards": [
		{"id": "p-041_531486", "name": "Monkey.D.Luffy", "number": "P-041", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 531486}}
	],
	"sealed": [
		{"id": "box-600001", "name": "Fixture Booster Box", "setCode": "OP-PR", "image": "x", "externalLinks": {"tcgPlayerId": 600001}, "identifiers": {"mcmId": "7", "scgId": "S1"}},
		{"id": "pack-600002", "name": "Fixture Booster Pack", "setCode": "OP-PR", "image": "x", "externalLinks": {"tcgPlayerId": 600002}}
	]
}}`

func loadSealedIDsFixture(t *testing.T) *matcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(sealedIDsFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestSealedIdentifiers pins that a sealed entry's identifiers object lands
// on the product beside the TCGplayer id its externalLinks carry.
func TestSealedIdentifiers(t *testing.T) {
	b := loadSealedIDsFixture(t)
	want := map[string]string{"tcgplayerProductId": "600001", "mcmId": "7", "scgId": "S1"}
	co, err := b.GetUUID("box-600001")
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(co.Identifiers, want) {
		t.Errorf("Identifiers = %v, want %v", co.Identifiers, want)
	}
}

// TestSealedWithoutIdentifiers pins that an entry with no identifiers object,
// as every datastore publishes today, still carries its TCGplayer id.
func TestSealedWithoutIdentifiers(t *testing.T) {
	b := loadSealedIDsFixture(t)
	want := map[string]string{"tcgplayerProductId": "600002"}
	co, err := b.GetUUID("pack-600002")
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(co.Identifiers, want) {
		t.Errorf("Identifiers = %v, want %v", co.Identifiers, want)
	}
}
