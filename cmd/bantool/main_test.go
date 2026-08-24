package main

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// An empty scrape must read as no data even when the scraper reported a
// timestamp and unfolded into named sellers and vendors: those come back
// with empty records, and uploading them would blank the bucket.
func TestHasAnyData(t *testing.T) {
	var info mtgban.ScraperInfo
	emptySeller := mtgban.NewSellerFromInventory(nil, info)
	emptyVendor := mtgban.NewVendorFromBuylist(nil, info)
	fullSeller := mtgban.NewSellerFromInventory(mtgban.InventoryRecord{
		"uuid": {{Price: 1}},
	}, info)
	fullVendor := mtgban.NewVendorFromBuylist(mtgban.BuylistRecord{
		"uuid": {{BuyPrice: 1}},
	}, info)

	tests := []struct {
		name    string
		sellers []mtgban.Seller
		vendors []mtgban.Vendor
		want    bool
	}{
		{"nothing unfolded at all", nil, nil, false},
		{"a seller with an empty record", []mtgban.Seller{emptySeller}, nil, false},
		{"empty on both sides", []mtgban.Seller{emptySeller}, []mtgban.Vendor{emptyVendor}, false},
		{"one priced card is data", []mtgban.Seller{fullSeller}, []mtgban.Vendor{emptyVendor}, true},
		{"a buylist alone is data too", []mtgban.Seller{emptySeller}, []mtgban.Vendor{fullVendor}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasAnyData(test.sellers, test.vendors); got != test.want {
				t.Errorf("hasAnyData() = %v, want %v", got, test.want)
			}
		})
	}
}
