package pokemon

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// sealedIDsFixture is one card and three sealed products: one carrying every
// store's ids beside its TCGplayer id, one carrying only the TCGplayer id, as
// every datastore publishes today, and one whose store lists are mistyped.
const sealedIDsFixture = `{"meta": {"version": "1"}, "data": {
	"game": "pokemon",
	"sets": {
		"SVP": {"name": "SV: Scarlet & Violet Promo Cards", "type": "promo"}
	},
	"cards": [
		{"id": "169_653353_holofoil", "name": "Jolteon", "setCode": "SVP", "number": "169", "rarity": "Promo", "finish": "Holofoil", "externalLinks": {"tcgPlayerId": 653353}}
	],
	"sealed": [
		{"id": "box-600001", "name": "Fixture Booster Box", "setCode": "SVP", "image": "x", "externalLinks": {"tcgPlayerId": 600001, "cardmarketIds": [7, 9], "cardTraderIds": [501], "csiIds": ["C1"], "scgIds": ["S1"], "miniaturemarketIds": ["M1"]}},
		{"id": "pack-600002", "name": "Fixture Booster Pack", "setCode": "SVP", "image": "x", "externalLinks": {"tcgPlayerId": 600002}},
		{"id": "tin-600003", "name": "Fixture Tin", "setCode": "SVP", "image": "x", "externalLinks": {"tcgPlayerId": 600003, "cardmarketIds": "7", "cardTraderIds": [7.5], "csiIds": [123456]}}
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
		"cardtraderId":       {"501"},
		"csiId":              {"C1"},
		"scgId":              {"S1"},
		"miniaturemarketId":  {"M1"},
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

// TestSealedMistypedStoreIDs pins that a store list of the wrong shape on one
// sealed entry costs that list alone: the load succeeds and the entry keeps
// its TCGplayer id and every id it spelled readably.
func TestSealedMistypedStoreIDs(t *testing.T) {
	b, err := Load(strings.NewReader(sealedIDsFixture))
	if err != nil {
		t.Fatal(err)
	}

	for idName, want := range map[string][]string{
		"tcgplayerProductId": {"600003"},
		"mcmId":              nil,
		"cardtraderId":       nil,
		"csiId":              {"123456"},
	} {
		got := b.SealedIdentifiers("tin-600003", idName)
		if !slices.Equal(got, want) {
			t.Errorf("SealedIdentifiers(%s) = %v, want %v", idName, got, want)
		}
	}
}
