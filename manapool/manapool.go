// Package manapool scrapes Mana Pool, for singles and sealed product.
package manapool

import (
	"cmp"
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
	"github.com/mtgban/go-mtgban/mtgban"
)

// Manapool prices Mana Pool's catalog.
type Manapool struct {
	logCallback mtgban.LogCallbackFunc
	partner     string

	backend *matcher.Backend

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraper returns a scraper pricing Mana Pool's singles, both what they
// sell and what they buy, matching against b.
func NewScraper(b *matcher.Backend) *Manapool {
	mp := Manapool{backend: b}
	mp.inventory = mtgban.InventoryRecord{}
	return &mp
}

func (mp *Manapool) printf(format string, a ...any) {
	if mp.logCallback != nil {
		mp.logCallback("[MP] "+format, a...)
	}
}

// isUnindexed reports whether the backend was never meant to know this card,
// so that failing to match its id is expected and not worth reporting. Whole
// editions are dropped when the datastore is built - oversize, minigames,
// front cards, playtest - and a sheet of tokens the datastore carries no set
// for is dropped the same way, which is what the edition answers for.
func isUnindexed(b *matcher.Backend, card Product) bool {
	_, err := b.GetSet(card.SetCode)
	return err != nil
}

// isTokenSheet reports whether the printing a listing landed on sits in a set
// of tokens, the only place a two-sided listing is a pairing of two faces.
func isTokenSheet(b *matcher.Backend, co *matcher.CardObject) bool {
	set, err := b.GetSet(co.SetCode)
	return err == nil && set.Type == "token"
}

// matchSheet resolves a two-sided token sheet by both faces' scryfall ids to
// the pairing they were printed in, or "" when the datastore pairs them in no
// product or not in this finish.
func matchSheet(b *matcher.Backend, card Product, foil bool) string {
	if len(card.ScryfallIDs) != 2 {
		return ""
	}
	uuidA := b.ConvertID(matcher.IDSpaceScryfall, card.ScryfallIDs[0])
	uuidB := b.ConvertID(matcher.IDSpaceScryfall, card.ScryfallIDs[1])
	return magic.MatchTokenPairingByUUIDs(b, uuidA, uuidB, foil)
}

// cardKey names the scryfall ids a row is matched by, for the log: a sheet's
// faces where it has none of its own.
func cardKey(card Product) string {
	return cmp.Or(card.ScryfallID, strings.Join(card.ScryfallIDs, "+"))
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
		foil := card.FinishID == "FO"
		etched := card.FinishID == "EF"

		inputID := cmp.Or(card.ScryfallID, matchSheet(mp.backend, card, foil))
		cardID, err := mp.backend.MatchID(inputID, foil, etched)
		if err != nil {
			if !isUnindexed(mp.backend, card) {
				mp.printf("%v %s for %s [%s]", err, cardKey(card), card.Name, card.SetCode)
			}
			continue
		}
		// A row with a scryfall_id of its own, unlike a sheet matched by its
		// faces above, can name two faces and land on one: price it on the
		// pairing, or not at all.
		co, err := mp.backend.GetUUID(cardID)
		if err != nil {
			continue
		}
		if strings.Contains(card.Name, " // ") && !strings.Contains(co.Name, " // ") &&
			isTokenSheet(mp.backend, co) {
			cardID = ""
			// The pairing is a derived uuid or a TCGplayer product id, and
			// MatchID reads both
			if id := magic.MatchTokenPairing(mp.backend, card.ScryfallID, card.Name, foil); id != "" {
				cardID, _ = mp.backend.MatchID(id, foil, etched)
			}
			if cardID == "" {
				continue
			}
		}
		mp.record(card, cardID)
	}
}

// record validates and prices one already-resolved row, the tail end of
// price() shared by both the plain scryfall-id path and the token-pairing
// one above.
func (mp *Manapool) record(card Product, cardID string) {
	// Validate language
	co, err := mp.backend.GetUUID(cardID)
	if err != nil {
		return
	}
	// The table names English by the empty tag, and Portuguese without the
	// "(Brazil)" the datastore adds, so the printing's language holds it.
	lang, found := matcher.LanguageCode2LanguageTag[strings.ToLower(card.LanguageID)]
	if !found {
		return
	}
	if !strings.Contains(co.Language, cmp.Or(lang, "English")) {
		// A foreign listing is keyed by the English printing's scryfall id, so
		// the datastore's copy in that language, when it loads one, is priced.
		copyID, err := mp.backend.MatchIDLanguage(cardID, lang)
		if err != nil {
			return
		}
		copied, err := mp.backend.GetUUID(copyID)
		if err != nil || copied.Foil != co.Foil || copied.Etched != co.Etched {
			return
		}
		cardID = copyID
	}

	// Build URL
	u, err := url.Parse(card.URL)
	if err != nil {
		mp.printf("%v", err)
		return
	}
	v := url.Values{}
	if mp.partner != "" {
		v.Set("ref", mp.partner)
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
	grade, err := mtgban.ParseCondition(card.ConditionID)
	if err != nil {
		mp.printf("unsupported %s condition", card.ConditionID)
		return
	}

	// Convert price to float and add the 4.2% fee
	price := float64(card.LowPrice) / 100.0 * 1.042

	// Got there! The product id is what Mana Pool's cart optimizer takes
	out := &mtgban.InventoryEntry{
		Conditions: grade,
		Price:      price,
		Available:  card.AvailableQuantity,
		URL:        link,
		InstanceID: card.ProductID,
	}
	mp.addCheapest(cardID, out)
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
	info.Game = mp.backend.Game
	return
}

// addCheapest records one price per printing and grade: the lowest one, with
// its product's link and id, and the copies every product holds as its
// Available. The list carries a row per product, and a printing the store
// files under more than one product - the same token from several decks, a
// card in two of its own product lines - arrives once per product, each with
// its own low price. The site shows one row for the grade, and a buyer pays
// the lower of them. Available can so count more copies than the product
// InstanceID names holds.
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
		if entries[i].Conditions != entry.Conditions {
			continue
		}
		entries[i].Available += entry.Available
		if entry.Price < entries[i].Price {
			entries[i].Price = entry.Price
			entries[i].URL = entry.URL
			entries[i].InstanceID = entry.InstanceID
		}
	}
}
