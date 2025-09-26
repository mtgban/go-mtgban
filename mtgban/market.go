package mtgban

import (
	"fmt"
	"slices"
)

// Return the inventory for any given seller present in the market.
// If possible, it will use the Inventory() call to populate data.
func InventoryForSeller(seller Market, sellerName string) (InventoryRecord, error) {
	inventory, err := seller.Inventory()
	if err != nil {
		return nil, err
	}

	if !slices.Contains(seller.MarketNames(), sellerName) {
		return nil, fmt.Errorf("%s is not present in %s", sellerName, seller.Info().Name)
	}

	marketplace := InventoryRecord{}
	for uuid := range inventory {
		for i := range inventory[uuid] {
			if inventory[uuid][i].SellerName == sellerName {
				marketplace[uuid] = append(marketplace[uuid], inventory[uuid][i])
			}
		}
	}

	return marketplace, nil
}

// Return the buylsit for any given vendor present in the Trader.
// If possible, it will use the Buylist() call to populate data.
func BuylistForVendor(vendor Trader, vendorName string) (BuylistRecord, error) {
	buylist, err := vendor.Buylist()
	if err != nil {
		return nil, err
	}

	if !slices.Contains(vendor.TraderNames(), vendorName) {
		return nil, fmt.Errorf("%s is not present in %s", vendorName, vendor.Info().Name)
	}

	traderpost := BuylistRecord{}
	for uuid := range buylist {
		for i := range buylist[uuid] {
			if buylist[uuid][i].VendorName == vendorName {
				traderpost[uuid] = append(traderpost[uuid], buylist[uuid][i])
			}
		}
	}

	return traderpost, nil
}
