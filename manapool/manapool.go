package manapool

import (
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

func (mp *Manapool) scrape() error {
	mpClient := NewClient()
	pricelist, err := mpClient.GetPriceList()
	if err != nil {
		return err
	}

	mp.printf("Found %d prices", len(pricelist))

	for _, card := range pricelist {
		cardId, err := mtgmatcher.MatchId(card.ScryfallID)
		if err != nil {
			mp.printf("%v", err)
			continue
		}
		cardIdFoil, _ := mtgmatcher.MatchId(card.ScryfallID, true)

		link := "https://www.manapool.com" + card.URL
		if mp.Partner != "" {
			link += "?ref=" + mp.Partner
		}

		conds := []string{"NM", "SP", "NM", "SP"}
		ids := []string{cardId, cardId, cardIdFoil, cardIdFoil}
		prices := []float64{
			float64(card.PriceCentsNm) / float64(100),
			float64(card.PriceCentsLpPlus) / float64(100),
			float64(card.PriceCentsNmFoil) / float64(100),
			float64(card.PriceCentsLpPlusFoil) / float64(100),
		}

		for i, price := range prices {
			if price == 0 || ids[i] == "" {
				continue
			}

			out := &mtgban.InventoryEntry{
				Conditions: conds[i],
				Price:      price,
				URL:        link,
			}
			err = mp.inventory.AddUnique(ids[i], out)
		}
	}

	mp.inventoryDate = time.Now()

	return nil
}

func (mp *Manapool) Inventory() (mtgban.InventoryRecord, error) {
	if len(mp.inventory) > 0 {
		return mp.inventory, nil
	}

	err := mp.scrape()
	if err != nil {
		return nil, err
	}

	return mp.inventory, nil

}

func (mp *Manapool) Info() (info mtgban.ScraperInfo) {
	info.Name = "Manapool"
	info.Shorthand = "MP"
	info.InventoryTimestamp = &mp.inventoryDate
	info.NoQuantityInventory = true
	return
}
