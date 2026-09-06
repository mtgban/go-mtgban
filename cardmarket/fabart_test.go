package cardmarket

import (
	"context"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// fabArtDatastore is the published Flesh and Blood datastore cut down to the
// printings these tests turn on, every row copied verbatim from it: Display
// Loyalty (Red), the GEM Pack promo TCGplayer sells as one product in two
// treatments where Cardmarket sells two products; and Twinning Blade, whose
// extended art the datastore keeps in a row of its own with a TCGplayer id
// of its own.
const fabArtDatastore = `{
 "game": "fleshandblood",
 "sets": {"GEM": {"name": "GEM Pack 1", "releaseDate": "2025-02-01"}, "CRU": {"name": "Crucible of War", "releaseDate": "2020-08-28"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 616347}, "fabId": "GEM010", "finish": "Normal", "id": "gem010_616347", "name": "Display Loyalty (Red)", "number": "GEM010", "rarity": "Promo", "setCode": "GEM"},
  {"externalLinks": {"tcgPlayerId": 616347}, "fabId": "GEM010", "finish": "Rainbow Foil", "id": "gem010_616347_rainbow", "name": "Display Loyalty (Red)", "number": "GEM010", "rarity": "Promo", "setCode": "GEM"},
  {"externalLinks": {"tcgPlayerId": 225982}, "fabId": "CRU082", "finish": "1st Edition Normal", "id": "cru082_225982_1e", "name": "Twinning Blade", "number": "CRU082", "rarity": "Majestic", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 225982}, "fabId": "CRU082", "finish": "1st Edition Rainbow Foil", "id": "cru082_225982_1erainbow", "name": "Twinning Blade", "number": "CRU082", "rarity": "Majestic", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 225982}, "fabId": "CRU082", "finish": "Unlimited Edition Normal", "id": "cru082_225982_unl", "name": "Twinning Blade", "number": "CRU082", "rarity": "Majestic", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 225982}, "fabId": "CRU082", "finish": "Unlimited Edition Rainbow Foil", "id": "cru082_225982_unlrainbow", "name": "Twinning Blade", "number": "CRU082", "rarity": "Majestic", "setCode": "CRU"},
  {"externalLinks": {"tcgPlayerId": 225983}, "fabId": "CRU082", "finish": "1st Edition Rainbow Foil", "id": "cru082_225983_1erainbow", "name": "Twinning Blade", "number": "CRU082", "promoTypes": ["extended art"], "rarity": "Majestic", "setCode": "CRU", "variant": "Extended Art"}
 ]
}`

// TestGemPackTreatments pins the two printings a GEM Pack number is sold in
// against the two products Cardmarket sells them as.
//
// TCGplayer keeps the pack's promo as one product in two treatments, so the
// bridge - which speaks through cardtrader's blueprints, and cardtrader
// lumps both Cardmarket products under one - answers both products with the
// same id. Only the product's own name says which treatment it is, and it
// spells the art ahead of it ("Extended Art Rainbow Foil"). Left unread,
// both products file on the printing the id defaults to and one of the two
// prices is thrown away: 269 rejections over 134 printings in the night of
// 2026-08-23, 226 of them this pack's.
func TestGemPackTreatments(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabArtDatastore))
	if err != nil {
		t.Fatal(err)
	}

	products := []MKMProduct{
		{
			IDProduct:     810664,
			Name:          "Display Loyalty (Regular)",
			Number:        "010",
			ExpansionName: "GEM Pack Promos",
		},
		{
			IDProduct:     810195,
			Name:          "Display Loyalty (Extended Art Rainbow Foil)",
			Number:        "010",
			ExpansionName: "GEM Pack Promos",
		},
	}

	mkm := &Index{
		gameID:         GameFleshAndBlood,
		exchangeRate:   1,
		MaxConcurrency: 1,
		inventory:      mtgban.InventoryRecord{},
		TCGBridge:      map[int]int{810664: 616347, 810195: 616347},
		priceGuide: map[int]PriceGuide{
			810664: {IDProduct: 810664, LowPrice: 1, TrendPrice: 2},
			810195: {IDProduct: 810195, LowPrice: 9, TrendPrice: 10},
		},
	}

	mkm.collectPrices(context.Background(), []MKMExpansion{{Name: "GEM Pack Promos"}},
		func(_ context.Context, _ MKMExpansion, channel chan<- responseChan) error {
			for i := range products {
				err := mkm.processProduct(channel, &products[i])
				if err != nil {
					return err
				}
			}
			return nil
		})

	for uuid, want := range map[string]string{
		"gem010_616347":         "810664",
		"gem010_616347_rainbow": "810195",
	} {
		entries := mkm.inventory[uuid]
		if len(entries) != len(availableIndexNames) {
			t.Errorf("got %d entries for %s, want %d", len(entries), uuid, len(availableIndexNames))
			continue
		}
		for _, entry := range entries {
			if entry.OriginalID != want {
				t.Errorf("%s kept product %s for %s, want %s", entry.SellerName, entry.OriginalID, uuid, want)
			}
		}
	}
}

// TestMatchProductExtendedArt pins that reading the treatment off a name
// does not take the art with it. The datastore keeps an extended art in a
// printing of its own, reachable only by naming it, so a name stripped down
// to the card's would answer with the ordinary printing instead - and take
// the price the ordinary product had.
func TestMatchProductExtendedArt(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(fabArtDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GameFleshAndBlood}
	for _, tt := range []struct{ name, want string }{
		{"Twinning Blade (Extended Art Rainbow Foil)", "cru082_225983_1erainbow"},
		{"Twinning Blade (Rainbow Foil)", "cru082_225982_1erainbow"},
		{"Twinning Blade (Regular)", "cru082_225982_1e"},
	} {
		got := mkm.matchProduct(&MKMProduct{
			Name:          tt.name,
			Number:        "CRU082",
			ExpansionName: "Crucible of War - First",
		})
		if got != tt.want {
			t.Errorf("matchProduct(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
