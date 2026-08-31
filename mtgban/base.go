package mtgban

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// ErrInvalidCondition is returned when an entry carries a grade that is not
// one of FullGradeTags. An empty grade is not invalid: it is filled in as NM
// before the check.
var ErrInvalidCondition = errors.New("invalid condition")

// ErrDuplicateEntry reports an entry the record already holds, identical in
// everything it is keyed on. The record keeps the one it has and drops the
// new one, so a scraper whose storefront lists a card under more than one
// product can tell this apart from a failure and stay quiet about it.
var ErrDuplicateEntry = errors.New("duplicate entry")

func (inv InventoryRecord) add(cardID string, entry *InventoryEntry, strict int) error {
	// Safe defaults
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}
	if entry.Quantity == 0 {
		entry.Quantity = 1
	}

	if !slices.Contains(FullGradeTags, entry.Conditions) {
		return ErrInvalidCondition
	}

	entries, found := inv[cardID]
	if found {
		for i := range entries {
			if strict > 2 && entry.Conditions == entries[i].Conditions && entry.SellerName == entries[i].SellerName {
				card, _ := mtgmatcher.GetUUID(cardID)
				return fmt.Errorf("%w: duplicate inventory key, same conditions:\n-key: %s %s\n-new: %v\n-old: %v", ErrDuplicateEntry, cardID, card, *entry, entries[i])
			}

			if entry.Conditions == entries[i].Conditions && entry.Price == entries[i].Price && entry.SellerName == entries[i].SellerName {
				if strict > 1 {
					card, _ := mtgmatcher.GetUUID(cardID)
					return fmt.Errorf("%w: duplicate inventory key, same conditions and price:\n-key: %s %s\n-new: %v\n-old: %v", ErrDuplicateEntry, cardID, card, *entry, entries[i])
				}

				if strict > 0 && entry.URL == entries[i].URL && entry.Quantity == entries[i].Quantity && entry.Bundle == entries[i].Bundle {
					card, _ := mtgmatcher.GetUUID(cardID)
					return fmt.Errorf("%w: duplicate inventory key, same url, and qty:\n-key: %s %s\n-new: %v\n-old: %v", ErrDuplicateEntry, cardID, card, *entry, entries[i])
				}

				inv[cardID][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	inv[cardID] = append(inv[cardID], *entry)

	// Keep array sorted
	sort.Slice(inv[cardID], func(i, j int) bool {
		iIdx := slices.Index(FullGradeTags, inv[cardID][i].Conditions)
		jIdx := slices.Index(FullGradeTags, inv[cardID][j].Conditions)

		if iIdx == jIdx {
			if inv[cardID][i].Price == inv[cardID][j].Price {
				// Prioritize higher quantity for same price and same condition
				return inv[cardID][i].Quantity > inv[cardID][j].Quantity
			}
			// Prioritize lower prices first for the same condition
			return inv[cardID][i].Price < inv[cardID][j].Price
		}

		return iIdx < jIdx
	})

	return nil
}

// AddRelaxed adds a record to the inventory, always merging into an existing
// entry rather than reporting one.
func (inv InventoryRecord) AddRelaxed(cardID string, entry *InventoryEntry) error {
	return inv.add(cardID, entry, 0)
}

// Add a new record to the inventory, similar existing entries are merged
func (inv InventoryRecord) Add(cardID string, entry *InventoryEntry) error {
	return inv.add(cardID, entry, 1)
}

// AddStrict adds a record to the inventory and keeps similar existing entries
// apart instead of merging them.
func (inv InventoryRecord) AddStrict(cardID string, entry *InventoryEntry) error {
	return inv.add(cardID, entry, 2)
}

// AddUnique adds a record to the inventory and reports an error if the same
// card and condition are already present.
func (inv InventoryRecord) AddUnique(cardID string, entry *InventoryEntry) error {
	return inv.add(cardID, entry, 3)
}

// AddRelaxed adds an entry to the buylist, folding a duplicate into the
// quantity of the one already there rather than reporting it.
func (bl BuylistRecord) AddRelaxed(cardID string, entry *BuylistEntry) error {
	return bl.add(cardID, entry, 0)
}

// Add adds an entry to the buylist and reports a duplicate as an error. Two
// entries count as duplicates when quantity, grade, price and vendor all
// match, which is what a scraper reading the same listing twice produces.
func (bl BuylistRecord) Add(cardID string, entry *BuylistEntry) error {
	return bl.add(cardID, entry, 1)
}

// AddUnique adds an entry to the buylist and reports a second one for the
// same card and grade, whatever it is priced at.
//
// A shop pays one price for one card at one grade, so a second is the feed
// naming two of its products and the match folding them onto one id - the
// textured foil bought at the plain card's id, the manga art at the base
// common's. Add cannot see that: the two entries differ in price, which is
// the whole symptom, so they are not duplicates by its reading and both are
// kept, the higher one sorting to the front where it prices the wrong card.
//
// It is for the scrapers whose feed states one price per printing. Where a
// shop publishes a quantity tier or folds a credit multiplier into the same
// vendor name, the second entry is honest and Add is the one to call.
func (bl BuylistRecord) AddUnique(cardID string, entry *BuylistEntry) error {
	return bl.add(cardID, entry, 2)
}

func (bl BuylistRecord) add(cardID string, entry *BuylistEntry, strict int) error {
	if entry.Conditions == "" {
		entry.Conditions = "NM"
	}

	if !slices.Contains(FullGradeTags, entry.Conditions) {
		return ErrInvalidCondition
	}

	entries, found := bl[cardID]
	if found {
		for i := range entries {
			if strict > 1 && entry.Conditions == entries[i].Conditions && entry.VendorName == entries[i].VendorName {
				card, _ := mtgmatcher.GetUUID(cardID)
				return fmt.Errorf("%w: attempted to add a second buylist price at one grade:\n-key: %s %s\n-new: %v\n-old: %v", ErrDuplicateEntry, cardID, card, *entry, entries[i])
			}

			if entry.Quantity == entries[i].Quantity && entry.Conditions == entries[i].Conditions && entry.BuyPrice == entries[i].BuyPrice && entry.VendorName == entries[i].VendorName {
				if strict > 0 {
					card, _ := mtgmatcher.GetUUID(cardID)
					return fmt.Errorf("%w: attempted to add a duplicate buylist card:\n-key: %s %s\n-new: %v\n-old: %v", ErrDuplicateEntry, cardID, card, *entry, entries[i])
				}
				bl[cardID][i].Quantity += entry.Quantity
				return nil
			}
		}
	}

	bl[cardID] = append(bl[cardID], *entry)

	sort.Slice(bl[cardID], func(i, j int) bool {
		iIdx := slices.Index(FullGradeTags, bl[cardID][i].Conditions)
		jIdx := slices.Index(FullGradeTags, bl[cardID][j].Conditions)

		if iIdx == jIdx {
			if bl[cardID][i].BuyPrice == bl[cardID][j].BuyPrice {
				// Prioritize higher quantity for same price and same condition
				return bl[cardID][i].Quantity > bl[cardID][j].Quantity
			}
			// Prioritize higher prices first for the same condition
			return bl[cardID][i].BuyPrice > bl[cardID][j].BuyPrice
		}

		return iIdx < jIdx
	})

	return nil
}

// BaseSeller holds an inventory that has already been collected. It is what
// a seller read back from JSON or CSV becomes: prices without the scraper
// that fetched them.
type BaseSeller struct {
	inventory InventoryRecord
	info      ScraperInfo
}

// Load does nothing. The inventory arrived with the value, so there is
// nothing to fetch, and callers driving a Seller generically can call it
// without checking what they hold.
func (seller *BaseSeller) Load(ctx context.Context) error {
	return nil
}

// Inventory returns the record this seller was built from.
func (seller *BaseSeller) Inventory() InventoryRecord {
	return seller.inventory
}

// Info returns the scraper info this seller was built from.
func (seller *BaseSeller) Info() ScraperInfo {
	return seller.info
}

// NewSellerFromInventory wraps an already-collected inventory as a Seller,
// for prices restored from storage rather than fetched.
func NewSellerFromInventory(inventory InventoryRecord, info ScraperInfo) Seller {
	seller := BaseSeller{}
	seller.inventory = inventory
	seller.info = info
	return &seller
}

// BaseVendor is the buylist counterpart of BaseSeller: a buylist that has
// already been collected, with no scraper behind it.
type BaseVendor struct {
	buylist BuylistRecord
	info    ScraperInfo
}

// Load does nothing, for the same reason BaseSeller.Load does not.
func (vendor *BaseVendor) Load(ctx context.Context) error {
	return nil
}

// Buylist returns the record this vendor was built from.
func (vendor *BaseVendor) Buylist() BuylistRecord {
	return vendor.buylist
}

// Info returns the scraper info this vendor was built from.
func (vendor *BaseVendor) Info() (info ScraperInfo) {
	return vendor.info
}

// NewVendorFromBuylist wraps an already-collected buylist as a Vendor, for
// prices restored from storage rather than fetched.
func NewVendorFromBuylist(buylist BuylistRecord, info ScraperInfo) Vendor {
	vendor := BaseVendor{}
	vendor.buylist = buylist
	vendor.info = info
	return &vendor
}

// CountScrapers returns how many independent components the slice holds. It
// is safe to call before Load.
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

// UnfoldScrapers splits scrapers into their independent Seller and Vendor
// parts, unpacking a Market or Trader into its enabled sub-scrapers. Call it
// after Load, or the sub-scrapers come back empty.
//
// A side holding nothing is not one of the parts. What a scraper implements
// says what it is able to do, not what this store does today: Cool Stuff Inc
// sells sealed for every game it stocks and buys it only for two of them, so
// four of its games came back as vendors with an empty buylist, which the
// caller then had to refuse. Reading the records rather than the interface
// keeps the answer true as a store's own answer changes, with nothing to
// edit here on the day it starts buying something it did not.
//
// An empty record and a broken load are not the same thing and are not told
// apart here: a load that fails says so by returning an error, and the caller
// reports that.
func UnfoldScrapers(scrapers []Scraper) ([]Seller, []Vendor) {
	var sellers []Seller
	var vendors []Vendor

	for _, scraper := range scrapers {
		market, isMarket := scraper.(Market)
		if isMarket {
			for _, name := range market.MarketNames() {
				inv := InventoryForSeller(market, name)
				if len(inv) == 0 {
					continue
				}
				seller := NewSellerFromInventory(inv, market.InfoForScraper(name))
				sellers = append(sellers, seller)
			}
		}

		trader, isTrader := scraper.(Trader)
		if isTrader {
			for _, name := range trader.TraderNames() {
				bl := BuylistForVendor(trader, name)
				if len(bl) == 0 {
					continue
				}
				vendor := NewVendorFromBuylist(bl, trader.InfoForScraper(name))
				vendors = append(vendors, vendor)
			}
		}

		seller, isSeller := scraper.(Seller)
		if isSeller && !isMarket {
			inv := seller.Inventory()
			if len(inv) > 0 {
				sellers = append(sellers, NewSellerFromInventory(inv, seller.Info()))
			}
		}

		vendor, isVendor := scraper.(Vendor)
		if isVendor && !isTrader {
			bl := vendor.Buylist()
			if len(bl) > 0 {
				vendors = append(vendors, NewVendorFromBuylist(bl, vendor.Info()))
			}
		}
	}

	return sellers, vendors
}

// InventoryForSeller returns one seller's inventory out of a market, using the
// market's own Inventory call where it can.
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

// BuylistForVendor returns one vendor's buylist out of a trader, using the
// trader's own Buylist call where it can.
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
