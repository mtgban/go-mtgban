package manapool

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type Manapool struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

func NewScraper() *Manapool {
	mp := Manapool{}
	mp.inventory = mtgban.InventoryRecord{}
	return &mp
}

func (mp *Manapool) printf(format string, a ...interface{}) {
	if mp.LogCallback != nil {
		mp.LogCallback("[MP] "+format, a...)
	}
}

func (mp *Manapool) Load(ctx context.Context) error {
	pricelist, err := GetPriceList(ctx)
	if err != nil {
		return err
	}

	mp.printf("Found %d prices", len(pricelist))

	for _, card := range pricelist {
		cardId, err := mtgmatcher.MatchId(card.ScryfallID, card.FinishID == "FO", card.FinishID == "EF")
		if err != nil {
			// Skip errors for unsupported cards (tokens, art cards, front cards)
			if !mtgmatcher.IsToken(card.Name) &&
				!strings.HasPrefix(card.SetCode, "T") &&
				!strings.HasPrefix(card.SetCode, "A") &&
				!strings.HasPrefix(card.SetCode, "F") {
				mp.printf("%v %s for %s [%s]", err, card.ScryfallID, card.Name, card.SetCode)
			}
			continue
		}

		// Validate language
		co, err := mtgmatcher.GetUUID(cardId)
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
		if card.FinishID != "NF" {
			v.Set("finish", "foil")
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
			Quantity:   card.AvailableQuantity,
			URL:        link,
		}
		err = mp.inventory.AddUnique(cardId, out)
	}

	mp.inventoryDate = time.Now()

	return nil
}

func (mp *Manapool) Inventory() mtgban.InventoryRecord {
	return mp.inventory
}

func (mp *Manapool) Info() (info mtgban.ScraperInfo) {
	info.Name = "Manapool"
	info.Shorthand = "MP"
	info.InventoryTimestamp = &mp.inventoryDate
	return
}
