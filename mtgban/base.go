package mtgban

import "fmt"

func (inv InventoryRecord) Add(card InventoryEntry) error {
	entries, found := inv[card.Id]
	if found {
		for _, entry := range entries {
			if entry.Conditions == card.Conditions && entry.Price == card.Price {
				return fmt.Errorf("Attempted to add a duplicate inventory card:\n-new: %v\n-old: %v", card, entry)
			}
		}
	}

	inv[card.Id] = append(inv[card.Id], card)
	return nil
}

func (bl BuylistRecord) Add(card BuylistEntry) error {
	entry, found := bl[card.Id]
	if found {
		return fmt.Errorf("Attempted to add a duplicate buylist card:\n-new: %v\n-old: %v", card, entry)
	}

	bl[card.Id] = card
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

func (bl *BaseBuylist) Grading(entry BuylistEntry) map[string]float64 {
	return bl.grade
}

func (inv *BaseBuylist) Info() (info ScraperInfo) {
	info.Name = "Base Type"
	info.Shorthand = "BT"
	return
}
