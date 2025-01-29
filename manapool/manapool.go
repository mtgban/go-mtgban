package manapool

import (
	"net/url"
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

		u, _ := url.Parse("https://www.manapool.com")
		u.Path = card.URL
		v := url.Values{}
		if mp.Partner != "" {
			v.Set("ref", mp.Partner)
		}

		conds := []string{"NM", "SP", "NM", "SP"}
		ids := []string{cardId, cardId, cardIdFoil, cardIdFoil}
		prices := []int{card.PriceCentsNm, card.PriceCentsLpPlus, card.PriceCentsNmFoil, card.PriceCentsLpPlusFoil}
		linkConds := []string{"NM", "LP", "NM", "LP"}
		linkFinishes := []string{"nonfoil", "nonfoil", "foil", "foil"}

		for i, price := range prices {
			if price == 0 || ids[i] == "" {
				continue
			}
			// Sometimes LP+ is the same as NM, but there is no real difference,
			// so just skip those prices
			if (i == 1 || i == 3) && prices[i] == prices[i-1] {
				continue
			}
			v.Set("conditions", linkConds[i])
			v.Set("finish", linkFinishes[i])
			u.RawQuery = v.Encode()
			link := u.String()

			out := &mtgban.InventoryEntry{
				Conditions: conds[i],
				Price:      float64(price) / 100.0,
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
