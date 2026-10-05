package cardtrader

import (
	"maps"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestAddFirstOfferPerSeller pins that each storefront keeps its first offer
// per condition, holding the copies of every offer of its own, and that
// Zero and 1 Day Ready, both bundle listings, keep entries of their own.
func TestAddFirstOfferPerSeller(t *testing.T) {
	inventory := mtgban.InventoryRecord{}
	for _, entry := range []mtgban.InventoryEntry{
		{Price: 10, Quantity: 2, SellerName: availableMarketNames[0]},
		{Price: 11, Quantity: 3, SellerName: availableMarketNames[0]},
		{Price: 12, Quantity: 1, SellerName: availableMarketNames[0]},
		{Price: 13, Quantity: 4, SellerName: availableMarketNames[1], Bundle: true},
		{Price: 15, Quantity: 1, SellerName: availableMarketNames[2], Bundle: true},
		{Price: 14, Quantity: 2, SellerName: availableMarketNames[1], Bundle: true},
	} {
		entry.Conditions = mtgban.NM
		addFirstOffer(inventory, resultChan{cardID: "uuid", invEntry: &entry}, t.Logf)
	}

	prices := map[string]float64{}
	available := map[string]int{}
	for _, entry := range inventory["uuid"] {
		prices[entry.SellerName] = entry.Price
		available[entry.SellerName] = entry.Available
	}
	wantPrices := map[string]float64{
		availableMarketNames[0]: 10,
		availableMarketNames[1]: 13,
		availableMarketNames[2]: 15,
	}
	wantAvailable := map[string]int{
		availableMarketNames[0]: 6,
		availableMarketNames[1]: 6,
		availableMarketNames[2]: 1,
	}
	if len(inventory["uuid"]) != len(wantPrices) || !maps.Equal(prices, wantPrices) {
		t.Errorf("got %v, want %v", inventory["uuid"], wantPrices)
	}
	if !maps.Equal(available, wantAvailable) {
		t.Errorf("available %v, want %v", available, wantAvailable)
	}
}
