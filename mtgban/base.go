package mtgban

import (
	"fmt"
	"time"

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
	timestamp time.Time
	metaonly  bool
}

func (seller *BaseSeller) Inventory() (InventoryRecord, error) {
	return seller.inventory, nil
}

func (seller *BaseSeller) Info() (info ScraperInfo) {
	info.Name = seller.name
	info.Shorthand = seller.shorthand
	info.InventoryTimestamp = seller.timestamp
	info.MetadataOnly = seller.metaonly
	return
}

func NewSellerFromInventory(inventory InventoryRecord, info ScraperInfo) Seller {
	seller := BaseSeller{}
	seller.inventory = inventory
	seller.name = info.Name
	seller.shorthand = info.Shorthand
	seller.timestamp = info.InventoryTimestamp
	seller.metaonly = info.MetadataOnly
	return &seller
}

type BaseVendor struct {
	buylist   BuylistRecord
	name      string
	shorthand string
	timestamp time.Time
	metaonly  bool
	grading   func(mtgdb.Card, BuylistEntry) map[string]float64
}

func (vendor *BaseVendor) Buylist() (BuylistRecord, error) {
	return vendor.buylist, nil
}

func (vendor *BaseVendor) Info() (info ScraperInfo) {
	info.Name = vendor.name
	info.Shorthand = vendor.shorthand
	info.BuylistTimestamp = vendor.timestamp
	info.Grading = vendor.grading
	if info.Grading == nil {
		info.Grading = DefaultGrading
	}
	return
}

func NewVendorFromBuylist(buylist BuylistRecord, info ScraperInfo) Vendor {
	vendor := BaseVendor{}
	vendor.buylist = buylist
	vendor.name = info.Name
	vendor.shorthand = info.Shorthand
	vendor.timestamp = info.BuylistTimestamp
	vendor.metaonly = info.MetadataOnly
	vendor.grading = info.Grading
	if vendor.grading == nil {
		vendor.grading = DefaultGrading
	}
	return &vendor
}
