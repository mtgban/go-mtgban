package mtgban

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"golang.org/x/exp/slices"
)

func ComputeSKU(cardId, condition string) (string, error) {
	co, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return "", err
	}

	scryfallNamespace, err := uuid.Parse(co.Identifiers["scryfallId"])
	if err != nil {
		return "", fmt.Errorf("invalid scryfall ID: %v", err)
	}
	condition = strings.ToLower(condition)

	conditionMap := map[string]string{
		"nm": "nm", "near mint": "nm",
		"lp": "sp", "lightly played": "sp", "slightly played": "sp",
		"mp": "mp", "moderately played": "mp",
		"hp": "hp", "heavily played": "hp",
		"po": "po", "damaged": "po", "poor": "po",
	}
	conditionCode, ok := conditionMap[condition]
	if !ok {
		conditionCode = "nm"
	}

	language := strings.ToLower(co.Language)
	printing := "nonfoil"
	if co.Etched {
		printing = "etched"
	} else if co.Foil {
		printing = "foil"
	}

	data := fmt.Sprintf("%s_%s_%s", conditionCode, language, printing)

	sku := uuid.NewSHA1(scryfallNamespace, []byte(data))
	return sku.String(), nil
}

func (inv InventoryRecord) add(cardId string, entry *InventoryEntry, strict int) error {
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}
	if entry.SKUID == "" {
		entry.SKUID, _ = ComputeSKU(cardId, entry.Conditions)
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
	if entry.SKUID == "" {
		entry.SKUID, _ = ComputeSKU(cardId, entry.Conditions)
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

type BaseScraper struct {
	inventory InventoryRecord
	buylist   BuylistRecord
	info      ScraperInfo
}

func (scraper *BaseScraper) Inventory() (InventoryRecord, error) {
	return scraper.inventory, nil
}

func (scraper *BaseScraper) Buylist() (BuylistRecord, error) {
	return scraper.buylist, nil
}

func (scraper *BaseScraper) Info() (info ScraperInfo) {
	return scraper.info
}

func NewScraperFromData(inventory InventoryRecord, buylist BuylistRecord, info ScraperInfo) Scraper {
	scraper := BaseScraper{}
	scraper.inventory = inventory
	scraper.buylist = buylist
	scraper.info = info
	return &scraper
}
