package vegassingles

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// bantool type-asserts this interface before it can turn a half off, and the
// assertion is silent when it fails: a scraper that does not answer it is
// asked for one half and publishes both.
var _ mtgban.ScraperConfig = (*Vegassingles)(nil)

// TestMagicPublishesOnlyTheBuylist pins what the Magic target is for. The
// store keeps no Magic singles shelf, so it is registered asking for the
// buylist alone; the retail row below would publish if the option were
// dropped, since a grade in stock is the one shape that reaches it.
func TestMagicPublishesOnlyTheBuylist(t *testing.T) {
	withMagic(t)

	product := entombFoil()
	product.RetailVariantInfo[0].InventoryQuantity = 2

	vs := NewScraper(GameMagic)
	vs.SetConfig(mtgban.ScraperOptions{DisableRetail: true})
	if err := vs.processProduct(product); err != nil {
		t.Fatal(err)
	}

	for uuid, entries := range vs.Inventory() {
		for _, entry := range entries {
			t.Errorf("%s: %s at $%.2f published for a shelf that was not asked for",
				uuid, entry.Conditions, entry.Price)
		}
	}

	var bids int
	for _, entries := range vs.Buylist() {
		bids += len(entries)
	}
	if bids != 3 {
		t.Errorf("buylist holds %d entries, want the 3 conditions with an offer", bids)
	}
}

// TestDisablingTheBuylistLeavesTheShelf runs the option the other way, so the
// two are not passing for each other.
func TestDisablingTheBuylistLeavesTheShelf(t *testing.T) {
	withMagic(t)

	product := entombFoil()
	product.RetailVariantInfo[0].InventoryQuantity = 2

	vs := NewScraper(GameMagic)
	vs.SetConfig(mtgban.ScraperOptions{DisableBuylist: true})
	if err := vs.processProduct(product); err != nil {
		t.Fatal(err)
	}

	for uuid, entries := range vs.Buylist() {
		for _, entry := range entries {
			t.Errorf("%s: %s bid $%.2f published for a buylist that was not asked for",
				uuid, entry.Conditions, entry.BuyPrice)
		}
	}

	var rows int
	for _, entries := range vs.Inventory() {
		rows += len(entries)
	}
	if rows != 1 {
		t.Errorf("inventory holds %d rows, want the one grade in stock", rows)
	}
}
