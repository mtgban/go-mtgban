package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// onePieceLinlinDatastore is the published datastore's two ST20-005 rows,
// images left out: the starter deck's leader and its 4th Anniversary
// Treasure Campaign Pack reprint.
const onePieceLinlinDatastore = `{"data": {
 "game": "onepiece",
 "sets": {
  "ST-20": {"name": "Starter Deck 20: YELLOW Charlotte Katakuri", "releaseDate": "2024-10-25"},
  "OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-09-30", "type": "promo"}
 },
 "cards": [
  {"color": "Yellow", "externalLinks": {"bandaiId": "ST20-005", "tcgPlayerId": 581065}, "finish": "Foil", "id": "st20-005_581065_foil", "name": "Charlotte Linlin", "number": "ST20-005", "rarity": "SR", "setCode": "ST-20", "type": "Character"},
  {"color": "Yellow", "externalLinks": {"tcgPlayerId": 714355}, "finish": "Foil", "id": "st20-005_714355_foil", "name": "Charlotte Linlin", "number": "ST20-005", "promoTypes": ["anniversarytreasure", "campaignpack"], "rarity": "SR", "setCode": "OP-PR", "type": "Character", "variant": "4th Anniversary Treasure Campaign Pack", "watermark": "4th"}
 ]
}}`

// TestOnePieceFrenchPrintIsForeign pins that the Première Édition
// alternate art Cardmarket shelves on Unnumbered Promos stays off the
// English campaign reprint its wording reaches, which the V.3 beside it
// still prices. The products are the catalog's own.
func TestOnePieceFrenchPrintIsForeign(t *testing.T) {
	b := datastoreBackend(t, "onepiece", onePieceLinlinDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	exp := cm.Expansion{IDExpansion: 5303, Name: "Unnumbered Promos"}

	french := mkm.resolver.resolveMapped(818093, cm.CatalogProduct{Name: "Charlotte Linlin (ST20-005) (V.2)", Number: "ST20-005", Rarity: "Super Rare", Version: 2}, exp)
	if !errors.Is(french.err, errForeign) {
		t.Errorf("818093: resolveMapped = (%q, %v), want errForeign", french.cardID, french.err)
	}
	campaign := mkm.resolver.resolveMapped(902487, cm.CatalogProduct{Name: "Charlotte Linlin (ST20-005) (V.3)", Number: "ST20-005", Rarity: "Super Rare", Version: 3}, exp)
	if campaign.err != nil || campaign.cardID != "st20-005_714355_foil" {
		t.Errorf("902487: resolveMapped = (%q, %v), want %q", campaign.cardID, campaign.err, "st20-005_714355_foil")
	}
}
