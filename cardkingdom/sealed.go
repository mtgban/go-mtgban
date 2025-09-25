package cardkingdom

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	ckBuylistLink = "https://www.cardkingdom.com/purchasing/mtg_sealed"
)

type CardkingdomSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string
	PreserveOOS bool

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

func (ck *CardkingdomSealed) Load(ctx context.Context) error {
	pricelist, err := cardkingdom.SealedPricelist(ctx, nil)
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
				if ckId != sealed.ID {
					continue
				}

				foundProduct++

				u, _ := url.Parse("https://www.cardkingdom.com/")

				// Rebuild the URL to have the same format as non-sealed
				basename := strings.TrimPrefix(sealed.URL, "mtg-sealed/")

				// Slugify the edition
				edition := strings.ToLower(sealed.Edition)
				edition = strings.Replace(edition, "'", "", -1)
				edition = strings.Replace(edition, ":", "", -1)
				edition = strings.Replace(edition, ".", "", -1)
				edition = strings.Replace(edition, ",", "", -1)
				edition = strings.Replace(edition, " -", "", -1)
				edition = strings.Replace(edition, "&", "and", -1)
				edition = strings.Replace(edition, " ", "-", -1)

				u.Path = "mtg" + "/" + edition + "/" + basename

				if ck.Partner != "" {
					q := u.Query()
					q.Set("partner", ck.Partner)
					q.Set("utm_source", ck.Partner)
					q.Set("utm_medium", "affiliate")
					q.Set("utm_campaign", ck.Partner)
					u.RawQuery = q.Encode()
				}
				link := u.String()

				if sealed.QtyRetail > 0 && sealed.PriceRetail > 0 {
					out := &mtgban.InventoryEntry{
						Conditions: "NM",
						Price:      sealed.PriceRetail,
						Quantity:   sealed.QtyRetail,
						URL:        link,
					}
					err = ck.inventory.Add(product.UUID, out)
					if err != nil {
						ck.printf("%v", err)
					}
				} else if ck.PreserveOOS {
					// Only save URL information
					out := &mtgban.InventoryEntry{
						URL: link,
					}
					err = ck.inventory.AddUnique(product.UUID, out)
				}

				if sealed.QtyBuying > 0 && sealed.PriceBuy > 0 {
					var priceRatio float64

					if sealed.PriceRetail > 0 {
						priceRatio = sealed.PriceBuy / sealed.PriceRetail * 100
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
						BuyPrice:   sealed.PriceBuy,
						Quantity:   sealed.QtyBuying,
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
	return ck.inventory, nil
}

func (ck *CardkingdomSealed) Buylist() (mtgban.BuylistRecord, error) {
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
