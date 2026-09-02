package manapool

import (
	"context"
	"net/url"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Index prices singles from Mana Pool's market valuation, what a card is
// reckoned to be worth rather than the cheapest copy anyone has listed.
type Index struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraperIndex returns an index scraper.
func NewScraperIndex() *Index {
	mp := Index{}
	mp.inventory = mtgban.InventoryRecord{}
	return &mp
}

func (mp *Index) printf(format string, a ...any) {
	if mp.LogCallback != nil {
		mp.LogCallback("[MPIndex] "+format, a...)
	}
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mp *Index) Load(ctx context.Context) error {
	singles, err := GetSinglesList(ctx)
	if err != nil {
		return err
	}

	mp.printf("Found %d singles", len(singles))

	for _, card := range singles {
		// The finish is named by the field the price sits in rather than by
		// a column, so each one resolves to its own printing.
		for _, finish := range []struct {
			foil  bool
			cents int
		}{
			{false, card.PriceMarket},
			{true, card.PriceMarketFoil},
		} {
			// A finish the card is not sold in reports a null, and a market
			// price of zero is not a quote worth publishing either way.
			if finish.cents == 0 {
				continue
			}

			uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceScryfall, card.ScryfallID)
			cardID, err := mtgmatcher.MatchID(uuid, finish.foil)
			if err != nil {
				if !isUnindexed(card) {
					mp.printf("%v %s for %s [%s]", err, card.ScryfallID, card.Name, card.SetCode)
				}
				continue
			}

			link := card.URL
			if mp.Partner != "" {
				u, err := url.Parse(card.URL)
				if err != nil {
					mp.printf("%v", err)
					continue
				}
				v := url.Values{}
				v.Set("ref", mp.Partner)
				u.RawQuery = v.Encode()
				link = u.String()
			}

			// A market price is what the card is reckoned to be worth rather
			// than an offer to sell one, so it carries neither the buyer fee
			// the listings do nor a quantity.
			out := &mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      float64(finish.cents) / 100.0,
				URL:        link,
			}
			err = mp.inventory.AddRelaxed(cardID, out)
			if err != nil {
				mp.printf("%v", err)
			}
		}
	}

	mp.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mp *Index) Inventory() mtgban.InventoryRecord {
	return mp.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (mp *Index) Info() (info mtgban.ScraperInfo) {
	info.Name = "Mana Pool Market"
	info.Shorthand = "MPIndex"
	info.InventoryTimestamp = &mp.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.Family = "MP"
	return
}
