package onepiece

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// sealedIDsFixture is one card and two sealed products: one carrying store
// ids beside its TCGplayer id, one carrying only the TCGplayer id, as every
// datastore publishes today.
const sealedIDsFixture = `{"data": {
	"game": "onepiece",
	"sets": {
		"OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-12-02", "type": "promo"}
	},
	"cards": [
		{"id": "p-041_531486", "name": "Monkey.D.Luffy", "number": "P-041", "setCode": "OP-PR", "rarity": "PR", "finish": "Normal", "image": "x", "externalLinks": {"tcgPlayerId": 531486}}
	],
	"sealed": [
		{"id": "box-600001", "name": "Fixture Booster Box", "setCode": "OP-PR", "image": "x", "externalLinks": {"tcgPlayerId": 600001, "cardmarketIds": [7, 9], "scgIds": ["S1"]}},
		{"id": "pack-600002", "name": "Fixture Booster Pack", "setCode": "OP-PR", "image": "x", "externalLinks": {"tcgPlayerId": 600002}}
	]
}}`

func TestSealedStoreIDs(t *testing.T) {
	b, err := Load(strings.NewReader(sealedIDsFixture))
	if err != nil {
		t.Fatal(err)
	}

	for idName, want := range map[string][]string{
		"tcgplayerProductId": {"600001"},
		"mcmId":              {"7", "9"},
		"scgId":              {"S1"},
	} {
		got := b.SealedIdentifiers("box-600001", idName)
		if !slices.Equal(got, want) {
			t.Errorf("SealedIdentifiers(%s) = %v, want %v", idName, got, want)
		}
	}

	co, err := b.GetUUID("pack-600002")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"tcgplayerProductId": "600002"}
	if !maps.Equal(co.Identifiers, want) {
		t.Errorf("Identifiers = %v, want %v", co.Identifiers, want)
	}
	got := b.SealedIdentifiers("pack-600002", "mcmId")
	if got != nil {
		t.Errorf("SealedIdentifiers(mcmId) = %v, want nil", got)
	}
}
