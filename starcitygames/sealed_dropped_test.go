package starcitygames

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestSealedPricedCount covers the count a run signs off with. What it priced
// is what it came away with a price for on either side of the counter, and the
// two sides are far from the same list: Star City Games quotes a buy price on
// plenty of sealed product it holds no stock of, so counting the shelf alone
// would move the number with stock rather than with coverage - the reading the
// drop tally beside it is there to prevent.
func TestSealedPricedCount(t *testing.T) {
	scg := NewScraperSealed(GameLorcana, "")
	// One product on both sides, one it has stock of and will not buy, one it
	// will buy and has no stock of. Three products carry a price; neither side
	// on its own says three.
	for _, uuid := range []string{"both", "sell-only"} {
		if err := scg.inventory.Add(uuid, &mtgban.InventoryEntry{Price: 1, Quantity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	for _, uuid := range []string{"both", "buy-only"} {
		if err := scg.buylist.Add(uuid, &mtgban.BuylistEntry{BuyPrice: 1}); err != nil {
			t.Fatal(err)
		}
	}

	if got := scg.pricedProducts(); got != 3 {
		t.Errorf("pricedProducts() = %d, want 3 (inventory holds %d, buylist %d)",
			got, len(scg.inventory), len(scg.buylist))
	}
}

// TestSealedDropAccounting covers the products a sealed run turns down. For
// the games whose datastore carries no sku, every product goes through the
// name resolver, and a refusal used to leave no trace at all: a game losing
// coverage looked exactly like a game with nothing more to sell. Each refusal
// now names itself and is counted under its reason.
func TestSealedDropAccounting(t *testing.T) {
	withLorcana(t)

	scg := NewScraperSealed(GameLorcana, "")
	var logs []string
	scg.LogCallback = func(format string, a ...any) { logs = append(logs, fmt.Sprintf(format, a...)) }
	scg.productMap = map[string]string{}

	// A gift set the datastore does not carry, a Japanese box, and a product
	// of another game entirely, which is not this run's to account for.
	scg.processProduct(CatalogProduct{
		SKU: "SLD-LOR-BXS-011-EN", Name: "Lorcana: Winterspell - Scrooge McDuck Gift Set",
		Game: "Lorcana", ProductType: ProductTypeSealed, Language: "English",
	})
	scg.processProduct(CatalogProduct{
		SKU: "SLD-LOR-BBX-011-JP", Name: "Lorcana: Winterspell Booster Box",
		Game: "Lorcana", ProductType: ProductTypeSealed, Language: "Japanese",
	})
	scg.processProduct(CatalogProduct{
		SKU: "SLD-MTG-BBX-FDN-EN", Name: "Magic: The Gathering - Foundations Booster Box",
		Game: "Magic: The Gathering", ProductType: ProductTypeSealed, Language: "English",
	})

	if got := scg.dropped["not sold in English"]; got != 1 {
		t.Errorf("counted %d non-English drops, want 1", got)
	}
	total := 0
	for _, count := range scg.dropped {
		total += count
	}
	if total != 2 {
		t.Errorf("counted %d drops in %v, want 2", total, scg.dropped)
	}
	var named int
	for _, entry := range logs {
		if strings.Contains(entry, "Scrooge McDuck Gift Set") {
			named++
		}
	}
	if named != 1 {
		t.Errorf("the refused name was logged %d times, want 1", named)
	}
}
