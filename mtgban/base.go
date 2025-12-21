package mtgban

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var ErrInvalidCondition = errors.New("invalid condition")

func (inv InventoryRecord) add(cardId string, entry *InventoryEntry, strict int) error {
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}

	if !slices.Contains(FullGradeTags, entry.Conditions) {
		return ErrInvalidCondition
	}

	entries, found := inv[cardId]
	if found {
		for i := range entries {
			if strict > 2 && entry.Conditions == entries[i].Conditions && entry.SellerName == entries[i].SellerName {
				card, _ := mtgmatcher.GetUUID(cardId)
				return fmt.Errorf("duplicate inventory key, same conditions:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
			}

			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price && entry.SellerName == entries[i].SellerName {
				if strict > 1 {
					card, _ := mtgmatcher.GetUUID(cardId)
					return fmt.Errorf("duplicate inventory key, same conditions and price:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
				}

				if strict > 0 && entry.URL == entries[i].URL && entry.Quantity == entries[i].Quantity && entry.Bundle == entries[i].Bundle {
					card, _ := mtgmatcher.GetUUID(cardId)
					return fmt.Errorf("duplicate inventory key, same url, and qty:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
				}

				inv[cardId][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	inv[cardId] = append(inv[cardId], *entry)

	// Keep array sorted
	sort.Slice(inv[cardId], func(i, j int) bool {
		iIdx := slices.Index(FullGradeTags, inv[cardId][i].Conditions)
		jIdx := slices.Index(FullGradeTags, inv[cardId][j].Conditions)

		if iIdx == jIdx {
			if inv[cardId][i].Price == inv[cardId][j].Price {
				// Prioritize higher quantity for same price and same condition
				return inv[cardId][i].Quantity > inv[cardId][j].Quantity
			}
			// Prioritize lower prices first for the same condition
			return inv[cardId][i].Price < inv[cardId][j].Price
		}

		return iIdx < jIdx
	})

	return nil
}

// Add a new record to the inventory, existing entries are always merged
func (inv InventoryRecord) AddRelaxed(cardId string, entry *InventoryEntry) error {
	return inv.add(cardId, entry, 0)
}

// Add a new record to the inventory, similar existing entries are merged
func (inv InventoryRecord) Add(cardId string, entry *InventoryEntry) error {
	return inv.add(cardId, entry, 1)
}

// Add new record to the inventory, similar existing entries are not merged
func (inv InventoryRecord) AddStrict(cardId string, entry *InventoryEntry) error {
	return inv.add(cardId, entry, 2)
}

// Add new record to the inventory, if same card and condition exist, error out
func (inv InventoryRecord) AddUnique(cardId string, entry *InventoryEntry) error {
	return inv.add(cardId, entry, 3)
}

func (bl BuylistRecord) AddRelaxed(cardId string, entry *BuylistEntry) error {
	return bl.add(cardId, entry, false)
}

func (bl BuylistRecord) Add(cardId string, entry *BuylistEntry) error {
	return bl.add(cardId, entry, true)
}

func (bl BuylistRecord) add(cardId string, entry *BuylistEntry, strict bool) error {
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}

	if !slices.Contains(FullGradeTags, entry.Conditions) {
		return ErrInvalidCondition
	}

	entries, found := bl[cardId]
	if found {
		for i := range entries {
			if entry.Quantity == entries[i].Quantity && entry.Conditions == entries[i].Conditions && entry.BuyPrice == entries[i].BuyPrice && entry.VendorName == entries[i].VendorName {
				if strict {
					card, _ := mtgmatcher.GetUUID(cardId)
					return fmt.Errorf("attempted to add a duplicate buylist card:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, bl[cardId])
				}
				bl[cardId][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	bl[cardId] = append(bl[cardId], *entry)

	sort.Slice(bl[cardId], func(i, j int) bool {
		iIdx := slices.Index(FullGradeTags, bl[cardId][i].Conditions)
		jIdx := slices.Index(FullGradeTags, bl[cardId][j].Conditions)

		if iIdx == jIdx {
			if bl[cardId][i].BuyPrice == bl[cardId][j].BuyPrice {
				// Prioritize higher quantity for same price and same condition
				return bl[cardId][i].Quantity > bl[cardId][j].Quantity
			}
			// Prioritize higher prices first for the same condition
			return bl[cardId][i].BuyPrice > bl[cardId][j].BuyPrice
		}

		return iIdx < jIdx
	})

	return nil
}

type BaseSeller struct {
	inventory InventoryRecord
	info      ScraperInfo
}

func (seller *BaseSeller) Load(ctx context.Context) error {
	return nil
}

func (seller *BaseSeller) Inventory() InventoryRecord {
	return seller.inventory
}

func (seller *BaseSeller) Info() ScraperInfo {
	return seller.info
}

func NewSellerFromInventory(inventory InventoryRecord, info ScraperInfo) Seller {
	seller := BaseSeller{}
	seller.inventory = inventory
	seller.info = info
	return &seller
}

type BaseVendor struct {
	buylist BuylistRecord
	info    ScraperInfo
}

func (vendor *BaseVendor) Load(ctx context.Context) error {
	return nil
}

func (vendor *BaseVendor) Buylist() BuylistRecord {
	return vendor.buylist
}

func (vendor *BaseVendor) Info() (info ScraperInfo) {
	return vendor.info
}

func NewVendorFromBuylist(buylist BuylistRecord, info ScraperInfo) Vendor {
	vendor := BaseVendor{}
	vendor.buylist = buylist
	vendor.info = info
	return &vendor
}

// Return how many independent components are present in the slice.
// This function can be safely called before Load().
func CountScrapers(scrapers []Scraper) (int, int) {
	var sellers, vendors int
	for _, scraper := range scrapers {
		market, isMarket := scraper.(Market)
		if isMarket {
			sellers += len(market.MarketNames())
		}
		trader, isTrader := scraper.(Trader)
		if isTrader {
			vendors += len(trader.TraderNames())
		}
		_, isSeller := scraper.(Seller)
		if isSeller && !isMarket {
			sellers++
		}
		_, isVendor := scraper.(Vendor)
		if isVendor && !isTrader {
			vendors++
		}
	}
	return sellers, vendors
}

// Commodity function to unfold a Scraper into their independent Seller and
// Vendor parts, unpacking Market and Trader into the various enabled sub-scrapers.
// Since it processes this kind of scrapers it needs to be called *after*
// the Load() call, otherwise the subscrapers will contain empty data.
func UnfoldScrapers(scrapers []Scraper) ([]Seller, []Vendor) {
	var sellers []Seller
	var vendors []Vendor

	for _, scraper := range scrapers {
		market, isMarket := scraper.(Market)
		if isMarket && scraper.Info().InventoryTimestamp != nil {
			for _, name := range market.MarketNames() {
				inv := InventoryForSeller(market, name)
				seller := NewSellerFromInventory(inv, market.InfoForScraper(name))
				sellers = append(sellers, seller)
			}
		}

		trader, isTrader := scraper.(Trader)
		if isTrader && scraper.Info().BuylistTimestamp != nil {
			for _, name := range trader.TraderNames() {
				bl := BuylistForVendor(trader, name)
				vendor := NewVendorFromBuylist(bl, trader.InfoForScraper(name))
				vendors = append(vendors, vendor)
			}
		}

		seller, isSeller := scraper.(Seller)
		if isSeller && !isMarket && scraper.Info().InventoryTimestamp != nil {
			inv := seller.Inventory()
			seller := NewSellerFromInventory(inv, seller.Info())
			sellers = append(sellers, seller)
		}

		vendor, isVendor := scraper.(Vendor)
		if isVendor && !isTrader && scraper.Info().BuylistTimestamp != nil {
			bl := vendor.Buylist()
			vendor := NewVendorFromBuylist(bl, vendor.Info())
			vendors = append(vendors, vendor)
		}
	}

	return sellers, vendors
}

// Return the inventory for any given seller present in the market.
// If possible, it will use the Inventory() call to populate data.
func InventoryForSeller(seller Market, sellerName string) InventoryRecord {
	inventory := seller.Inventory()

	marketplace := InventoryRecord{}
	for uuid := range inventory {
		for i := range inventory[uuid] {
			if inventory[uuid][i].SellerName == sellerName {
				marketplace[uuid] = append(marketplace[uuid], inventory[uuid][i])
			}
		}
	}

	return marketplace
}

// Return the buylsit for any given vendor present in the Trader.
// If possible, it will use the Buylist() call to populate data.
func BuylistForVendor(vendor Trader, vendorName string) BuylistRecord {
	buylist := vendor.Buylist()

	traderpost := BuylistRecord{}
	for uuid := range buylist {
		for i := range buylist[uuid] {
			if buylist[uuid][i].VendorName == vendorName {
				traderpost[uuid] = append(traderpost[uuid], buylist[uuid][i])
			}
		}
	}

	return traderpost
}
