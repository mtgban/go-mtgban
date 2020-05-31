package mtgban

import (
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
	inventory InventoryRecord
	name      string
	shorthand string
	metaonly  bool
}

func (seller *BaseSeller) Inventory() (InventoryRecord, error) {
	return seller.inventory, nil
}

func (seller *BaseSeller) Info() (info ScraperInfo) {
	info.Name = seller.name
	info.Shorthand = seller.shorthand
	return
}

type BaseVendor struct {
	buylist   BuylistRecord
	grade     map[string]float64
	name      string
	shorthand string
}

func (vendor *BaseVendor) Buylist() (BuylistRecord, error) {
	return vendor.buylist, nil
}

func (vendor *BaseVendor) Grading(card mtgdb.Card, entry BuylistEntry) map[string]float64 {
	return vendor.grade
}

func (vendor *BaseVendor) Info() (info ScraperInfo) {
	info.Name = vendor.name
	info.Shorthand = vendor.shorthand
	return
}
