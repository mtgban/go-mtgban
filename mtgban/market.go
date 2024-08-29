package mtgban

import (
	"fmt"
)

// Return the inventory for any given seller present in the market.
// If possible, it will use the Inventory() call to populate data.
func InventoryForSeller(seller Market, sellerName string) (InventoryRecord, error) {
	inventory, err := seller.Inventory()
	if err != nil {
		return nil, err
	}

	marketplace := InventoryRecord{}
	for uuid := range inventory {
		for i := range inventory[uuid] {
			if inventory[uuid][i].SellerName == sellerName {
				marketplace[uuid] = append(marketplace[uuid], inventory[uuid][i])
			}
		}
	}

	if len(marketplace) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
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

	traderpost := BuylistRecord{}
	for uuid := range buylist {
		for i := range buylist[uuid] {
			if buylist[uuid][i].VendorName == vendorName {
				traderpost[uuid] = append(traderpost[uuid], buylist[uuid][i])
			}
		}
	}

	if len(traderpost) == 0 {
		return nil, fmt.Errorf("vendor %s not found", vendorName)
	}

	return traderpost, nil
}

// Base structure for the conversion of a Market to a standard Seller
// This will hold the original Market scraper and retrieve the loaded
// subseller from its ScraperInfo
type BaseMarket struct {
	inventory InventoryRecord
	info      ScraperInfo
	scraper   Market
}

func (m *BaseMarket) Inventory() (InventoryRecord, error) {
	if m.inventory == nil {
		// Retrieve inventory from the original scraper
		inventory, err := InventoryForSeller(m.scraper, m.info.Shorthand)
		if err != nil {
			return nil, err
		}
		m.inventory = inventory

		// Original scraper is not useful any more here
		m.scraper = nil
	}
	return m.inventory, nil
}

func (m *BaseMarket) Info() ScraperInfo {
	return m.info
}

// Base structure for the conversion of a Trader to a standard Vendor
// This will hold the original Trader scraper and retrieve the loaded
// subvendor from its ScraperInfo
type BaseTrader struct {
	buylist BuylistRecord
	info    ScraperInfo
	scraper Trader
}

func (m *BaseTrader) Buylist() (BuylistRecord, error) {
	if m.buylist == nil {
		// Retrieve inventory from the original scraper
		buylist, err := BuylistForVendor(m.scraper, m.info.Shorthand)
		if err != nil {
			return nil, err
		}
		m.buylist = buylist

		// Original scraper is not useful any more here
		m.scraper = nil
	}
	return m.buylist, nil
}

func (m *BaseTrader) Info() ScraperInfo {
	return m.info
}
