package gamenerdz

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// untitledVariantDatastore is the published One Piece datastore cut down to
// the two numbers this test turns on, every row at either of them copied
// verbatim from it. The whole number is here rather than the one row each
// product wants, so the wording the storefront brackets still has to pick
// between the printings that share it.
const untitledVariantDatastore = `{"data": {
 "game": "onepiece",
 "sets": {
  "OP01": {"name": "Romance Dawn", "releaseDate": "2022-12-02"},
  "OP04": {"name": "Kingdoms of Intrigue", "releaseDate": "2023-09-22"},
  "OP10": {"name": "Royal Blood", "releaseDate": "2025-03-21"},
  "OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-09-30", "type": "promo"},
  "PRB-01": {"name": "Premium Booster -The Best-", "releaseDate": "2024-11-08"},
  "ST-36": {"name": "Starter Deck 36: YELLOW Eustass\"Captain\"Kid", "releaseDate": "2026-07-31"}
 },
 "cards": [
  {"bandaiId": "OP01-078", "color": "Blue", "externalLinks": {"bandaiId": "OP01-078", "tcgPlayerId": 454608}, "finish": "Foil", "id": "op01-078_454608_foil", "name": "Boa Hancock", "number": "OP01-078", "rarity": "SR", "setCode": "OP01", "type": "Character"},
  {"bandaiId": "OP01-078_p1", "color": "Blue", "externalLinks": {"bandaiId": "OP01-078_p1", "tcgPlayerId": 454609}, "finish": "Foil", "id": "op01-078_454609_foil", "name": "Boa Hancock", "number": "OP01-078", "promoTypes": ["parallel"], "rarity": "SR", "setCode": "OP01", "type": "Character", "variant": "Parallel"},
  {"bandaiId": "OP01-078_p2", "color": "Blue", "externalLinks": {"bandaiId": "OP01-078_p2", "tcgPlayerId": 516555}, "finish": "Foil", "id": "op01-078_516555_foil", "name": "Boa Hancock", "number": "OP01-078", "promoTypes": ["sp"], "rarity": "SR", "setCode": "OP04", "type": "Character", "variant": "SP"},
  {"bandaiId": "OP01-078_p4", "color": "Blue", "externalLinks": {"bandaiId": "OP01-078_p4", "tcgPlayerId": 586548}, "finish": "Foil", "id": "op01-078_586548_foil", "name": "Boa Hancock", "number": "OP01-078", "promoTypes": ["alternateart"], "rarity": "SR", "setCode": "PRB-01", "type": "Character", "variant": "Alternate Art"},
  {"bandaiId": "OP01-078_r1", "color": "Blue", "externalLinks": {"bandaiId": "OP01-078_r1", "tcgPlayerId": 594315}, "finish": "Foil", "id": "op01-078_594315_foil", "name": "Boa Hancock", "number": "OP01-078", "promoTypes": ["reprint"], "rarity": "SR", "setCode": "PRB-01", "type": "Character", "variant": "Reprint"},
  {"bandaiId": "OP10-111", "color": "Yellow", "externalLinks": {"bandaiId": "OP10-111", "tcgPlayerId": 617160}, "finish": "Foil", "id": "op10-111_617160_foil", "name": "Monkey.D.Luffy", "number": "OP10-111", "rarity": "R", "setCode": "OP10", "type": "Character"},
  {"bandaiId": "OP10-111_p1", "color": "Yellow", "externalLinks": {"bandaiId": "OP10-111_p1", "tcgPlayerId": 617161}, "finish": "Foil", "id": "op10-111_617161_foil", "name": "Monkey.D.Luffy", "number": "OP10-111", "promoTypes": ["parallel"], "rarity": "R", "setCode": "OP10", "type": "Character", "variant": "Parallel"},
  {"color": "Yellow", "externalLinks": {"tcgPlayerId": 649706}, "finish": "Foil", "id": "op10-111_649706_foil", "name": "Monkey.D.Luffy", "number": "OP10-111", "promoTypes": ["premiumcardcollection"], "rarity": "R", "setCode": "OP-PR", "type": "Character", "variant": "Premium Card Collection 6 assort vol. 1", "watermark": "6 assort vol. 1"},
  {"color": "Yellow", "externalLinks": {"tcgPlayerId": 656608}, "finish": "Normal", "id": "op10-111_656608", "name": "Monkey.D.Luffy", "number": "OP10-111", "promoTypes": ["setsailevent"], "rarity": "R", "setCode": "OP-PR", "type": "Character", "variant": "Learn Together Deck Set - Set Sail Event", "watermark": "learn together deck set"},
  {"color": "Yellow", "externalLinks": {"tcgPlayerId": 693122}, "finish": "Normal", "id": "op10-111_693122", "name": "Monkey.D.Luffy", "number": "OP10-111", "originalReleaseDate": "2026-01-01", "promoTypes": ["welcomepack"], "rarity": "R", "setCode": "OP-PR", "type": "Character", "variant": "Welcome Pack 2026 Vol. 1", "watermark": "vol. 1"},
  {"color": "Yellow", "externalLinks": {"tcgPlayerId": 706359}, "finish": "Normal", "id": "op10-111_706359", "name": "Monkey.D.Luffy", "number": "OP10-111", "rarity": "R", "setCode": "ST-36", "type": "Character"}
 ]
}}`

// The two products below are the storefront's own buylist records, less the
// keys the scraper does not read. Neither offer carries a variant title:
// the first sends null and the second sends no title key at all, and both
// decode to the empty string an untitled variant has always meant.
const nullTitleProduct = `{
 "id": "zG1IqsjrZk",
 "product_id": 183400,
 "display_name": "Boa Hancock (SP) (OP01-078) Kingdoms of Intrigue Foil",
 "price": 1070.23,
 "selectedFinish": "Foil",
 "product_data": {"rarity": "Super Rare", "set": "Kingdoms of Intrigue", "setName": "Kingdoms of Intrigue"},
 "store_pass_variant_info": [
  {"id": 181038, "title": null, "offer_price": 668.9, "offer_price_credit": 836.125, "selected_finish": "Foil"}
 ]
}`

const noTitleProduct = `{
 "id": "U7Qhtune4C",
 "product_id": 358457,
 "display_name": "Monkey.D.Luffy (Welcome Pack 2026 Vol.1) (OP10-111) One Piece Promotion Cards",
 "price": 32.47,
 "selectedFinish": "Normal",
 "product_data": {"rarity": "Rare", "set": "One Piece Promotion Cards", "setName": "One Piece Promotion Cards"},
 "store_pass_variant_info": [
  {"id": 314671, "offer_price": 13.62, "offer_price_credit": 17.025, "selected_finish": "Normal"}
 ]
}`

// A title this storefront has never sent, to hold the vocabulary closed: an
// offer naming a condition nobody can spell is still refused rather than
// recorded at whatever the display name happens to end in.
const unknownTitleProduct = `{
 "id": "zG1IqsjrZk",
 "product_id": 183400,
 "display_name": "Boa Hancock (SP) (OP01-078) Kingdoms of Intrigue Foil",
 "price": 1070.23,
 "selectedFinish": "Foil",
 "product_data": {"rarity": "Super Rare", "set": "Kingdoms of Intrigue", "setName": "Kingdoms of Intrigue"},
 "store_pass_variant_info": [
  {"id": 181038, "title": "Signed", "offer_price": 668.9, "selected_finish": "Foil"}
 ]
}`

// TestProcessUntitledBuyVariant pins that an offer hanging off an untitled
// variant is recorded. The store buys these three at $668.90, $13.62 and
// $541.13, and every one of them went unpriced because the platform sends
// the untitled variant's title as null or leaves it out rather than filling
// in the placeholder string the rest of the feed carries.
func TestProcessUntitledBuyVariant(t *testing.T) {
	b, err := mtgmatcher.Open("onepiece", strings.NewReader(untitledVariantDatastore))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		raw   string
		uuid  string
		cond  mtgban.Condition
		price float64
	}{
		{nullTitleProduct, "op01-078_516555_foil", mtgban.NM, 668.9},
		{noTitleProduct, "op10-111_693122", mtgban.NM, 13.62},
		{unknownTitleProduct, "", "", 0},
	}
	for _, tt := range tests {
		var product GNProduct
		if err := json.Unmarshal([]byte(tt.raw), &product); err != nil {
			t.Fatal(err)
		}

		gn, err := NewScraper(b)
		if err != nil {
			t.Fatal(err)
		}
		var logs []string
		gn.logCallback = func(format string, a ...any) {
			logs = append(logs, format)
		}
		if err := gn.processProduct(modeBuylist, product); err != nil {
			t.Errorf("%s: unexpected error %v", product.ID, err)
			continue
		}

		if tt.uuid == "" {
			if len(gn.Buylist()) != 0 {
				t.Errorf("%s: recorded %v; want nothing", product.ID, gn.Buylist())
			}
			if len(logs) != 1 || !strings.Contains(logs[0], "unknown condition") {
				t.Errorf("%s: logged %v; want an unknown condition", product.ID, logs)
			}
			continue
		}

		if len(logs) != 0 {
			t.Errorf("%s: logged %v; want nothing", product.ID, logs)
		}
		entries := gn.Buylist()[tt.uuid]
		if len(entries) != 1 {
			t.Errorf("%s: %q has %d entries; want 1", product.ID, tt.uuid, len(entries))
			continue
		}
		if entries[0].Conditions != tt.cond {
			t.Errorf("%s: condition %q; want %q", product.ID, entries[0].Conditions, tt.cond)
		}
		if entries[0].BuyPrice != tt.price {
			t.Errorf("%s: price %v; want %v", product.ID, entries[0].BuyPrice, tt.price)
		}
	}
}
