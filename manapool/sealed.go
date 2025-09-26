package manapool

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type ManapoolSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

func NewScraperSealed() *ManapoolSealed {
	mp := ManapoolSealed{}
	mp.inventory = mtgban.InventoryRecord{}
	return &mp
}

func (mp *ManapoolSealed) printf(format string, a ...interface{}) {
	if mp.LogCallback != nil {
		mp.LogCallback("[MPSealed] "+format, a...)
	}
}

func (mp *ManapoolSealed) Load(ctx context.Context) error {
	pricelist, err := GetSealedList(ctx)
	if err != nil {
		return err
	}

	mp.printf("Found %d prices", len(pricelist))

	var foundProduct int

	sets := mtgmatcher.GetAllSets()
	for _, code := range sets {
		set, _ := mtgmatcher.GetSet(code)

		// Skip products without Sealed or Booster information
		switch set.Code {
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		}

		for _, product := range set.SealedProduct {
			tcgIdStr, found := product.Identifiers["tcgplayerProductId"]
			if !found {
				continue
			}

			tcgId, err := strconv.Atoi(tcgIdStr)
			if err != nil {
				continue
			}

			for _, sealed := range pricelist {
				if tcgId != sealed.TcgplayerProductID {
					continue
				}

				foundProduct++

				// Build URL
				u, err := url.Parse(sealed.URL)
				if err != nil {
					mp.printf("%v", err)
					continue
				}
				v := url.Values{}
				if mp.Partner != "" {
					v.Set("ref", mp.Partner)
				}
				u.RawQuery = v.Encode()

				out := &mtgban.InventoryEntry{
					Price:    float64(sealed.LowPrice) / 100.0,
					URL:      u.String(),
					Quantity: sealed.AvailableQuantity,
				}
				err = mp.inventory.AddUnique(product.UUID, out)
			}
		}
	}

	perc := float64(foundProduct) * 100 / float64(len(pricelist))
	mp.printf("Found %d products over %d items (%.02f%%)", foundProduct, len(pricelist), perc)

	mp.inventoryDate = time.Now()

	return nil
}

func (mp *ManapoolSealed) Inventory() mtgban.InventoryRecord {
	return mp.inventory
}

func (mp *ManapoolSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Manapool"
	info.Shorthand = "MPSealed"
	info.InventoryTimestamp = &mp.inventoryDate
	info.SealedMode = true
	return
}
