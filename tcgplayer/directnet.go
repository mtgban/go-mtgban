package tcgplayer

import (
	"errors"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
)

type TCGDirectNet struct {
	buylistDate     time.Time
	buylist         mtgban.BuylistRecord
	DirectInventory mtgban.InventoryRecord
}

func NewTCGDirectNet() *TCGDirectNet {
	tcg := TCGDirectNet{}
	tcg.buylist = mtgban.BuylistRecord{}
	return &tcg
}

func (tcg *TCGDirectNet) Buylist() (mtgban.BuylistRecord, error) {
	if len(tcg.DirectInventory) == 0 {
		return nil, errors.New("missing inventory")
	}

	if len(tcg.buylist) > 0 {
		return tcg.buylist, nil
	}

	for cardId, entries := range tcg.DirectInventory {
		for _, entry := range entries {
			directCost := 0.3 + entry.Price*(0.0895+0.025)

			var replacementFees float64
			if entry.Price < 3 {
				replacementFees = entry.Price / 2
				directCost = 0
			} else if entry.Price < 20 {
				replacementFees = 1.12
			} else if entry.Price < 250 {
				replacementFees = 3.97
			} else {
				replacementFees = 6.85
			}

			link := entry.URL
			if link != "" {
				link += "&direct=true"
			}

			buylistEntry := mtgban.BuylistEntry{
				Conditions: entry.Conditions,
				BuyPrice:   entry.Price - directCost - replacementFees,
				URL:        link,
			}

			tcg.buylist[cardId] = append(tcg.buylist[cardId], buylistEntry)
		}
	}

	tcg.buylistDate = time.Now()

	return tcg.buylist, nil
}

func (tcg *TCGDirectNet) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Direct (net)"
	info.Shorthand = "TCGDirectNet"
	info.BuylistTimestamp = &tcg.buylistDate
	return
}
