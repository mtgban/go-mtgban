package mtgban

import "fmt"

func InventoryAdd(inventory map[string][]InventoryEntry, card InventoryEntry) error {
	entries, found := inventory[card.Id]
	if found {
		for _, entry := range entries {
			if entry.Conditions == card.Conditions && entry.Price == card.Price {
				return fmt.Errorf("Attempted to add a duplicate inventory card:\n-new: %v\n-old: %v", card, entry)
			}
		}
	}

	inventory[card.Id] = append(inventory[card.Id], card)
	return nil
}

func BuylistAdd(buylist map[string]BuylistEntry, card BuylistEntry) error {
	entry, found := buylist[card.Id]
	if found {
		return fmt.Errorf("Attempted to add a duplicate buylist card:\n-new: %v\n-old: %v", card, entry)
	}

	buylist[card.Id] = card
	return nil
}

type BaseInventory struct {
	inventory map[string][]InventoryEntry
}

func (inv *BaseInventory) Inventory() (map[string][]InventoryEntry, error) {
	return inv.inventory, nil
}

func (inv *BaseInventory) Info() (info ScraperInfo) {
	info.Name = "Base Type"
	info.Shorthand = "BT"
	return
}

type BaseBuylist struct {
	buylist map[string]BuylistEntry
	grade   map[string]float64
}

func (bl *BaseBuylist) Buylist() (map[string]BuylistEntry, error) {
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
