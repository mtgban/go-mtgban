package mtgban

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgdb"
)

func (inv InventoryRecord) Add(card *mtgdb.Card, entry *InventoryEntry) error {
	entries, found := inv[*card]
	if found {
		for i := range entries {
			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price {
				return fmt.Errorf("Attempted to add a duplicate inventory card:\n-key: %v\n-new: %v\n-old: %v", card, *entry, entries[i])
			}
		}
	}

	inv[*card] = append(inv[*card], *entry)
	return nil
}

func (bl BuylistRecord) Add(card *mtgdb.Card, entry *BuylistEntry) error {
	_, found := bl[*card]
	if found {
		return fmt.Errorf("Attempted to add a duplicate buylist card:\n-key: %v\n-new: %v\n-old: %v", card, *entry, bl[*card])
	}

	bl[*card] = *entry
	return nil
}

type BaseInventory struct {
	inventory InventoryRecord
}

func (inv *BaseInventory) Inventory() (InventoryRecord, error) {
	return inv.inventory, nil
}

func (inv *BaseInventory) Info() (info ScraperInfo) {
	info.Name = "Base Type"
	info.Shorthand = "BT"
	return
}

type BaseBuylist struct {
	buylist BuylistRecord
	grade   map[string]float64
}

func (bl *BaseBuylist) Buylist() (BuylistRecord, error) {
	return bl.buylist, nil
}

func (bl *BaseBuylist) Grading(card mtgdb.Card, entry BuylistEntry) map[string]float64 {
	return bl.grade
}

func (inv *BaseBuylist) Info() (info ScraperInfo) {
	info.Name = "Base Type"
	info.Shorthand = "BT"
	return
}
