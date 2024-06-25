package mtgban

import (
	"fmt"

	"golang.org/x/exp/maps"
)

// Separate a Market into multiple Seller objects
func Seller2Sellers(market Market) ([]Seller, error) {
	// Make sure inventory is loaded
	_, err := market.Inventory()
	if err != nil {
		return nil, err
	}

	// Retrieve the list of unique sellers, and create a single seller
	var sellers []Seller
	for _, sellerName := range market.MarketNames() {
		inventory, err := InventoryForSeller(market, sellerName)
		if err != nil {
			return nil, err
		}
		seller := &BaseSeller{}
		seller.inventory = inventory
		seller.info = market.Info()
		seller.info.Name = sellerName
		seller.info.Shorthand = sellerName
		seller.info.CustomFields = maps.Clone(market.Info().CustomFields)
		sellers = append(sellers, seller)
	}
	return sellers, nil
}

// Return the inventory for any given seller present in the market.
// If possible, it will use the Inventory() call to populate data.
func InventoryForSeller(seller Seller, sellerName string) (InventoryRecord, error) {
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

// Base structure for the conversion of a Market to a standard Seller
// This will hold the original Market scraper and retrieve the loaded
// subseller from its ScraperInfo
type BaseMarket struct {
	inventory InventoryRecord
	info      ScraperInfo
	scraper   Seller
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
