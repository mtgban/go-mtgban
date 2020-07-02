package mtgban

import (
	"errors"
	"fmt"

	"github.com/kodabb/go-mtgban/mtgdb"
)

func (inv InventoryRecord) add(card *mtgdb.Card, entry *InventoryEntry, strict bool) error {
	entries, found := inv[*card]
	if found {
		for i := range entries {
			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price {
				if strict {
					return fmt.Errorf("Attempted to add a duplicate inventory card:\n-key: %v\n-new: %v\n-old: %v", card, *entry, entries[i])
				}

				check := entry.URL == entries[i].URL
				if entry.SellerName != "" {
					check = check && entry.SellerName == entries[i].SellerName
				}
				if check && entry.Quantity == entries[i].Quantity {
					return fmt.Errorf("Attempted to add a duplicate inventory card:\n-key: %v\n-new: %v\n-old: %v", card, *entry, entries[i])
				}

				inv[*card][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	inv[*card] = append(inv[*card], *entry)
	return nil
}

// Add a new record to the inventory, similar existing entries are merged
func (inv InventoryRecord) Add(card *mtgdb.Card, entry *InventoryEntry) error {
	return inv.add(card, entry, false)
}

// Add new record to the inventory, similar existing entries are not merged
func (inv InventoryRecord) AddStrict(card *mtgdb.Card, entry *InventoryEntry) error {
	return inv.add(card, entry, true)
}

func (bl BuylistRecord) Add(card *mtgdb.Card, entry *BuylistEntry) error {
	_, found := bl[*card]
	if found {
		return fmt.Errorf("Attempted to add a duplicate buylist card:\n-key: %v\n-new: %v\n-old: %v", card, *entry, bl[*card])
	}

	bl[*card] = *entry
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
