package yugioh

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
	"game": "yugioh",
	"sets": {
		"LDK2": {"name": "Legendary Decks II", "releaseDate": "2016-10-07"}
	},
	"cards": [
		{"id": "ldk2-enj26_123593_unlimited", "name": "Polymerization", "number": "LDK2-ENJ26", "setCode": "LDK2", "rarity": "Common", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 123593}}
	],
	"sealed": [
		{"id": "box-600001", "name": "Fixture Booster Box", "setCode": "LDK2", "image": "x", "externalLinks": {"tcgPlayerId": 600001, "cardmarketIds": [7, 9], "scgIds": ["S1"]}},
		{"id": "pack-600002", "name": "Fixture Booster Pack", "setCode": "LDK2", "image": "x", "externalLinks": {"tcgPlayerId": 600002}}
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
