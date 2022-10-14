package mtgban

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func (inv InventoryRecord) add(cardId string, entry *InventoryEntry, strict int) error {
	entries, found := inv[cardId]
	if found {
		for i := range entries {
			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price && entry.SellerName == entries[i].SellerName {
				if strict > 1 {
					card, _ := mtgmatcher.GetUUID(cardId)
					return fmt.Errorf("duplicate inventory key, same conditions and price:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
				}

				if strict > 0 && entry.URL == entries[i].URL && entry.Quantity == entries[i].Quantity {
					card, _ := mtgmatcher.GetUUID(cardId)
					return fmt.Errorf("duplicate inventory key, same url, and qty:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
				}

				inv[cardId][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	inv[cardId] = append(inv[cardId], *entry)
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

	return nil
}

type BaseSeller struct {
	inventory InventoryRecord
	info      ScraperInfo
}

func (seller *BaseSeller) Inventory() (InventoryRecord, error) {
	return seller.inventory, nil
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

func (vendor *BaseVendor) Buylist() (BuylistRecord, error) {
	return vendor.buylist, nil
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
