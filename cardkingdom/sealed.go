package cardkingdom

import (
	"net/url"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	ckBuylistLink = "https://www.cardkingdom.com/purchasing/mtg_sealed"
)

type CardkingdomSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraperSealed() *CardkingdomSealed {
	ck := CardkingdomSealed{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.buylist = mtgban.BuylistRecord{}
	return &ck
}

func (ck *CardkingdomSealed) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CKSealed] "+format, a...)
	}
}

func (ck *CardkingdomSealed) scrape() error {
	ckClient := NewCKClient()
	pricelist, err := ckClient.GetSealedList()
	if err != nil {
		return err
	}

	foundProduct := 0

	sets := mtgmatcher.GetAllSets()
	for _, code := range sets {
		set, _ := mtgmatcher.GetSet(code)

		// Skip products without Sealed or Booster information
		switch set.Code {
		case "FBB", "4BB", "DRKITA", "LEGITA", "RIN", "4EDALT", "BCHR":
			continue
		}

		for _, product := range set.SealedProduct {
			ckIdStr, found := product.Identifiers["cardKingdomId"]
			if !found {
				continue
			}

			ckId, err := strconv.Atoi(ckIdStr)
			if err != nil {
				continue
			}

			for _, sealed := range pricelist {
				if ckId != sealed.Id {
					continue
				}

				foundProduct++

				u, _ := url.Parse("https://www.cardkingdom.com/")
				sellPrice, err := strconv.ParseFloat(sealed.SellPrice, 64)
				if err != nil {
					ck.printf("%v", err)
				}
				if sealed.SellQuantity > 0 && sellPrice > 0 {
					u.Path = sealed.URL
					if ck.Partner != "" {
						q := u.Query()
						q.Set("partner", ck.Partner)
						q.Set("utm_source", ck.Partner)
						q.Set("utm_medium", "affiliate")
						q.Set("utm_campaign", ck.Partner)
						u.RawQuery = q.Encode()
					}

					out := &mtgban.InventoryEntry{
						Conditions: "NM",
						Price:      sellPrice,
						Quantity:   sealed.SellQuantity,
						URL:        u.String(),
					}
					err = ck.inventory.Add(product.UUID, out)
					if err != nil {
						ck.printf("%v", err)
					}
				}

				buyPrice, err := strconv.ParseFloat(sealed.BuyPrice, 64)
				if err != nil {
					ck.printf("%v", err)
				}
				if sealed.BuyQuantity > 0 && buyPrice > 0 {
					var priceRatio float64

					if sellPrice > 0 {
						priceRatio = buyPrice / sellPrice * 100
					}

					u, _ = url.Parse(ckBuylistLink)
					q := u.Query()
					q.Set("filter[sort]", "price_desc")
					q.Set("search", "mtg_advanced")
					q.Set("filter[name]", sealed.Name)
					q.Set("filter[edition]", "")
					q.Set("filter[subtype]", "all")
					if ck.Partner != "" {
						q.Set("partner", ck.Partner)
						q.Set("utm_source", ck.Partner)
						q.Set("utm_medium", "affiliate")
						q.Set("utm_campaign", ck.Partner)
					}
					u.RawQuery = q.Encode()

					out := &mtgban.BuylistEntry{
						BuyPrice:   buyPrice,
						Quantity:   sealed.BuyQuantity,
						PriceRatio: priceRatio,
						URL:        u.String(),
					}
					err = ck.buylist.Add(product.UUID, out)
					if err != nil {
						ck.printf("%v", err)
					}
				}
			}
		}
	}

	perc := float64(foundProduct) * 100 / float64(len(pricelist))
	ck.printf("Found %d products over %d items (%.02f%%)", foundProduct, len(pricelist), perc)

	ck.inventoryDate = time.Now()
	ck.buylistDate = time.Now()

	return nil
}

func (ck *CardkingdomSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(ck.inventory) > 0 {
		return ck.inventory, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.inventory, nil

}

func (ck *CardkingdomSealed) Buylist() (mtgban.BuylistRecord, error) {
	if len(ck.buylist) > 0 {
		return ck.buylist, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.buylist, nil
}

func (ck *CardkingdomSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom"
	info.Shorthand = "CKSealed"
	info.InventoryTimestamp = &ck.inventoryDate
	info.BuylistTimestamp = &ck.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.3
	return
}
