// Package manapool scrapes Mana Pool, for singles and sealed product.
package manapool

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Manapool prices Mana Pool's catalog.
type Manapool struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraper returns a scraper pricing Mana Pool's singles, both what they
// sell and what they buy.
func NewScraper() *Manapool {
	mp := Manapool{}
	mp.inventory = mtgban.InventoryRecord{}
	return &mp
}

func (mp *Manapool) printf(format string, a ...any) {
	if mp.LogCallback != nil {
		mp.LogCallback("[MP] "+format, a...)
	}
}

// isUnindexed reports whether the backend was never meant to know this card,
// so that failing to match its id is expected and not worth reporting. Whole
// editions are dropped when the datastore is built - oversize, minigames,
// front cards, playtest - and a sheet of tokens the datastore carries no set
// for is dropped the same way, which is what the edition answers for.
func isUnindexed(card Product) bool {
	_, err := mtgmatcher.GetSet(card.SetCode)
	return err != nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mp *Manapool) Load(ctx context.Context) error {
	pricelist, err := GetPriceList(ctx)
	if err != nil {
		return err
	}

	mp.printf("Found %d prices", len(pricelist))
	mp.price(pricelist)
	mp.inventoryDate = time.Now()
	return nil
}

// price records every row of the list the store answers with.
func (mp *Manapool) price(pricelist []Product) {
	for _, card := range pricelist {
		cardID, err := mtgmatcher.MatchID(card.ScryfallID, card.FinishID == "FO", card.FinishID == "EF")
		if err != nil {
			if !isUnindexed(card) {
				mp.printf("%v %s for %s [%s]", err, card.ScryfallID, card.Name, card.SetCode)
			}
			continue
		}

		// Validate language
		co, err := mtgmatcher.GetUUID(cardID)
		if err != nil {
			continue
		}
		if mtgmatcher.LanguageTag2LanguageCode[co.Language] != strings.ToLower(card.LanguageID) {
			continue
		}

		// Build URL
		u, err := url.Parse(card.URL)
		if err != nil {
			mp.printf("%v", err)
			continue
		}
		v := url.Values{}
		if mp.Partner != "" {
			v.Set("ref", mp.Partner)
		}
		v.Set("conditions", card.ConditionID)
		switch card.FinishID {
		case "EF":
			v.Set("finish", "etched")
		case "FO":
			v.Set("finish", "foil")
		case "NF":
			v.Set("finish", "nonfoil")
		}
		u.RawQuery = v.Encode()
		link := u.String()

		// Match conditions
		conds := card.ConditionID
		switch card.ConditionID {
		case "NM", "MP", "HP":
		case "LP":
			conds = "SP"
		case "DMG":
			conds = "PO"
		default:
			mp.printf("Unknown %s condition for %s (%s)", conds, card.Name, card.SetCode)
			continue
		}

		// Convert price to float and add the 4.2% fee
		price := float64(card.LowPrice) / 100.0 * 1.042

		// Got there!
		out := &mtgban.InventoryEntry{
			Conditions: conds,
			Price:      price,
			URL:        link,
		}
		mp.addCheapest(cardID, out)
	}
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mp *Manapool) Inventory() mtgban.InventoryRecord {
	return mp.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (mp *Manapool) Info() (info mtgban.ScraperInfo) {
	info.Name = "Mana Pool"
	info.Shorthand = "MP"
	info.InventoryTimestamp = &mp.inventoryDate
	info.NoQuantityInventory = true
	return
}

// addCheapest records one price per printing and grade: the lowest one. The
// list carries a row per product, and a printing the store files under more
// than one product - the same token from several decks, a card in two of its
// own product lines - arrives once per product, each with its own low price.
// The site shows one row for the grade, and a buyer pays the lower of them.
func (mp *Manapool) addCheapest(cardID string, entry *mtgban.InventoryEntry) {
	err := mp.inventory.AddUnique(cardID, entry)
	if !errors.Is(err, mtgban.ErrDuplicateEntry) {
		if err != nil {
			mp.printf("%v", err)
		}
		return
	}
	entries := mp.inventory[cardID]
	for i := range entries {
		if entries[i].Conditions == entry.Conditions && entry.Price < entries[i].Price {
			entries[i].Price = entry.Price
			entries[i].URL = entry.URL
		}
	}
}
