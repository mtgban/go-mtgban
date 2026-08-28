package vegassingles

import "testing"

// entombFoil is the product as the storefront answers for it, trimmed to the
// fields the retail and buylist sides read. The store holds none of it in any
// condition, buys the top three, and prices the two it will not buy at a
// placeholder.
func entombFoil() VSProduct {
	return VSProduct{
		ID:          "fCMnFs2X9W",
		ProductID:   9907557794098,
		DisplayName: "Entomb (ODY-132) - Odyssey Foil",
		Price:       599.99,
		OfferPrice:  299.99,
		VariantInfo: []VSVariant{
			{ID: 1, Title: "Near Mint", OfferPrice: 299.99},
			{ID: 2, Title: "Lightly Played", OfferPrice: 254.99},
			{ID: 3, Title: "Moderately Played", OfferPrice: 209.99},
			{ID: 4, Title: "Heavily Played", OfferPrice: 0},
			{ID: 5, Title: "Damaged", OfferPrice: 0},
		},
		RetailVariantInfo: []VSRetailVariant{
			{ID: 1, Title: "Near Mint", Price: 599.99, SKU: "F-1"},
			{ID: 2, Title: "Lightly Played", Price: 509.99, SKU: "F-2"},
			{ID: 3, Title: "Moderately Played", Price: 419.99, SKU: "F-3"},
			{ID: 4, Title: "Heavily Played", Price: 0.5, SKU: "F-4"},
			{ID: 5, Title: "Damaged", Price: 0.3, SKU: "F-5"},
		},
	}
}

// TestOutOfStockIsNotForSale pins that a retail row the store holds none of
// is never published. Every variant here has no stock, which is the ordinary
// case for this storefront: it is a buylist that keeps a priced catalog.
func TestOutOfStockIsNotForSale(t *testing.T) {
	vs := NewScraper(GameMagic)
	if err := vs.processProduct(entombFoil()); err != nil {
		t.Fatal(err)
	}

	for uuid, entries := range vs.Inventory() {
		for _, entry := range entries {
			t.Errorf("%s: %s at $%.2f published for a card the store has none of",
				uuid, entry.Conditions, entry.Price)
		}
	}

	// The buylist is what this storefront is for, and it is unaffected: the
	// three conditions the store bids on are all there, and the two it
	// passes on were already dropped for offering nothing.
	var got int
	for _, entries := range vs.Buylist() {
		got += len(entries)
	}
	if got != 3 {
		t.Errorf("buylist holds %d entries, want the 3 conditions with an offer", got)
	}
}

// TestStockedRowsSurvive pins the other half: stock is the whole gate, and a
// grade the store holds is published at whatever the store asks for it. The
// placeholder prices sit on grades nothing is ever stocked in - across every
// game the scraper reads, not one Heavily Played or Damaged row has stock -
// so they are unreachable rather than filtered.
func TestStockedRowsSurvive(t *testing.T) {
	product := entombFoil()
	product.RetailVariantInfo[0].InventoryQuantity = 2 // Near Mint
	product.RetailVariantInfo[3].InventoryQuantity = 1 // Heavily Played

	vs := NewScraper(GameMagic)
	if err := vs.processProduct(product); err != nil {
		t.Fatal(err)
	}

	got := map[string]float64{}
	for _, entries := range vs.Inventory() {
		for _, entry := range entries {
			got[entry.Conditions] = entry.Price
			if entry.Conditions == "NM" && entry.Quantity != 2 {
				t.Errorf("NM published x%d, want x2", entry.Quantity)
			}
		}
	}
	want := map[string]float64{"NM": 599.99, "HP": 0.5}
	if len(got) != len(want) {
		t.Fatalf("published %v, want exactly the two stocked grades", got)
	}
	for cond, price := range want {
		if got[cond] != price {
			t.Errorf("%s published at $%.2f, want $%.2f", cond, got[cond], price)
		}
	}
}

// TestBuylistKeepsTheLowerConditions pins that refusing to sell a condition
// does not stop us reading what the store pays for it. Most cards carry a real
// bid on both - 255 of 264 Heavily Played rows and 228 of 264 Damaged ones,
// measured against the live feed - and only the ones bidding nothing drop out.
func TestBuylistKeepsTheLowerConditions(t *testing.T) {
	product := entombFoil()
	product.VariantInfo[3].OfferPrice = 149.99 // Heavily Played, bid on
	product.VariantInfo[4].OfferPrice = 99.99  // Damaged, bid on

	vs := NewScraper(GameMagic)
	if err := vs.processProduct(product); err != nil {
		t.Fatal(err)
	}

	want := map[string]float64{"NM": 299.99, "SP": 254.99, "MP": 209.99, "HP": 149.99, "PO": 99.99}
	got := map[string]float64{}
	for _, entries := range vs.Buylist() {
		for _, entry := range entries {
			got[entry.Conditions] = entry.BuyPrice
		}
	}
	if len(got) != len(want) {
		t.Fatalf("buylist holds %v, want all five conditions", got)
	}
	for cond, price := range want {
		if got[cond] != price {
			t.Errorf("%s bid is $%.2f, want $%.2f", cond, got[cond], price)
		}
	}
}
