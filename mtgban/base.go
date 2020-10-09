package mtgban

import (
	"errors"
	"fmt"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func (inv InventoryRecord) add(cardId string, entry *InventoryEntry, strict int) error {
	entries, found := inv[cardId]
	if found {
		card, err := mtgmatcher.GetUUID(cardId)
		if err != nil {
			return err
		}
		for i := range entries {
			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price {
				if strict > 1 {
					return fmt.Errorf("Attempted to add a duplicate inventory card:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
				}

				check := entry.URL == entries[i].URL
				if entry.SellerName != "" {
					check = check && entry.SellerName == entries[i].SellerName
				}
				if strict > 0 && check && entry.Quantity == entries[i].Quantity {
					return fmt.Errorf("Attempted to add a duplicate inventory card:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, entries[i])
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

func (bl BuylistRecord) Add(cardId string, entry *BuylistEntry) error {
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}

	entries, found := bl[cardId]
	if found {
		for i := range entries {
			if entry.Conditions == entries[i].Conditions {
				card, err := mtgmatcher.GetUUID(cardId)
				if err != nil {
					return err
				}

				return fmt.Errorf("Attempted to add a duplicate buylist card:\n-key: %s %s\n-new: %v\n-old: %v", cardId, card, *entry, bl[cardId])
			}
		}
	}

	bl[cardId] = append(bl[cardId], *entry)

	return nil
}

type BaseSeller struct {
	inventory   InventoryRecord
	marketplace map[string]InventoryRecord
	info        ScraperInfo
}

func (seller *BaseSeller) Inventory() (InventoryRecord, error) {
	return seller.inventory, nil
}

func (seller *BaseSeller) Info() ScraperInfo {
	return seller.info
}

func (seller *BaseSeller) InventoryForSeller(sellerName string) (InventoryRecord, error) {
	if len(seller.inventory) == 0 {
		_, err := seller.Inventory()
		if err != nil {
			return nil, err
		}
	}

	if seller.marketplace == nil {
		seller.marketplace = map[string]InventoryRecord{}
	}

	for card := range seller.inventory {
		for i := range seller.inventory[card] {
			if seller.inventory[card][i].SellerName == sellerName {
				if seller.inventory[card][i].Price == 0 {
					continue
				}
				if seller.marketplace[sellerName] == nil {
					seller.marketplace[sellerName] = InventoryRecord{}
				}
				seller.marketplace[sellerName][card] = append(seller.marketplace[sellerName][card], seller.inventory[card][i])
			}
		}
	}

	if len(seller.marketplace[sellerName]) == 0 {
		return nil, errors.New("seller not found")
	}
	return seller.marketplace[sellerName], nil
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
