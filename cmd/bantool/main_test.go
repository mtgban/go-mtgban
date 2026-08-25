package main

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// An empty scrape must count as no data even when the scraper reported a
// timestamp and unfolded into named sellers and vendors: those come back
// with empty records, and uploading them would blank the bucket.
func TestCountResults(t *testing.T) {
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
		name        string
		sellers     []mtgban.Seller
		vendors     []mtgban.Vendor
		wantRetail  int
		wantBuylist int
	}{
		{"nothing unfolded at all", nil, nil, 0, 0},
		{"a seller with an empty record", []mtgban.Seller{emptySeller}, nil, 0, 0},
		{"empty on both sides", []mtgban.Seller{emptySeller}, []mtgban.Vendor{emptyVendor}, 0, 0},
		{"one priced card counts", []mtgban.Seller{fullSeller}, []mtgban.Vendor{emptyVendor}, 1, 0},
		{"a buylist card counts apart", []mtgban.Seller{emptySeller}, []mtgban.Vendor{fullVendor}, 0, 1},
		{"two sellers of the same card both count", []mtgban.Seller{fullSeller, fullSeller}, []mtgban.Vendor{fullVendor}, 2, 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			retail, buylist := countResults(test.sellers, test.vendors)
			if retail != test.wantRetail || buylist != test.wantBuylist {
				t.Errorf("countResults() = (%d, %d), want (%d, %d)",
					retail, buylist, test.wantRetail, test.wantBuylist)
			}
		})
	}
}
