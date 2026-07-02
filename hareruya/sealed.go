package hareruya

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

type HareruyaSealed struct {
	LogCallback mtgban.LogCallbackFunc

	inventoryDate time.Time
	buylistDate   time.Time
	exchangeRate  float64

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *http.Client
}

func NewScraperSealed() *HareruyaSealed {
	ha := HareruyaSealed{}
	ha.inventory = mtgban.InventoryRecord{}
	ha.buylist = mtgban.BuylistRecord{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	ha.client = client.StandardClient()
	return &ha
}

func (ha *HareruyaSealed) printf(format string, a ...interface{}) {
	if ha.LogCallback != nil {
		ha.LogCallback("[HASealed] "+format, a...)
	}
}

// inventoryProducts returns the in-stock sealed products, indexed by Hareruya id
func (ha *HareruyaSealed) inventoryProducts(ctx context.Context) (map[string]Product, error) {
	products := map[string]Product{}
	for page := 1; ; page++ {
		results, err := SearchSealed(ctx, ha.client, page)
		if err != nil {
			return nil, err
		}

		// Exit loop condition: stop once a page returns no more products
		if len(results) == 0 {
			break
		}

		for _, product := range results {
			products[product.Product] = product
		}
	}
	return products, nil
}

// buylistPrices returns the buy price (in the store's native JPY) for every
// sealed product currently being purchased, indexed by Hareruya id. It mirrors
// the singles buylist scraping, only swapping the per-set query for the sealed
// product category.
func (ha *HareruyaSealed) buylistPrices(ctx context.Context) (map[string]float64, error) {
	prices := map[string]float64{}

	for page := 1; ; page++ {
		v := url.Values{}
		v.Set("sort", "price")
		v.Set("order", "DESC")
		v.Set("category", "505")
		v.Set("page", fmt.Sprint(page))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, buylistURL+v.Encode(), http.NoBody)
		if err != nil {
			return nil, err
		}
		resp, err := ha.client.Do(req)
		if err != nil {
			return nil, err
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var found int
		doc.Find(`.itemList`).Each(func(i int, s *goquery.Selection) {
			link, ok := s.Find(`div.itemData a`).Attr("href")
			if !ok {
				return
			}
			id := strings.Split(path.Base(strings.TrimSpace(link)), "?")[0]

			// Only consider items that are actively being bought
			stock := strings.TrimSpace(s.Find(`span.itemUserAct__number__title`).Text())
			if stock != "個数" {
				return
			}

			priceStr := strings.TrimSpace(s.Find(`p.itemDetail__price`).Text())
			price, err := mtgmatcher.ParsePrice(strings.TrimPrefix(priceStr, "¥ "))
			if err != nil {
				ha.printf("could not parse buylist price %q for %s: %v", priceStr, id, err)
				return
			}
			if price <= 0 {
				return
			}

			prices[id] = price
			found++
		})

		// Exit loop condition: stop once a page returns no buylist products
		if found == 0 {
			break
		}
	}

	return prices, nil
}

func (ha *HareruyaSealed) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "JPY")
	if err != nil {
		return err
	}
	ha.exchangeRate = rate
	ha.printf("Received JPY rate of %f", rate)

	inventory, err := ha.inventoryProducts(ctx)
	if err != nil {
		return err
	}
	ha.printf("Found %d in-stock sealed products", len(inventory))

	buylist, err := ha.buylistPrices(ctx)
	if err != nil {
		return err
	}
	ha.printf("Found %d sealed products being bought", len(buylist))

	var foundInventory, foundBuylist int

	sets := mtgmatcher.GetAllSets()
	for _, code := range sets {
		set, _ := mtgmatcher.GetSet(code)

		for _, sealedProduct := range set.SealedProduct {
			haId, found := sealedProduct.Identifiers["hareruyaId"]
			if !found {
				continue
			}

			// The retail price is also used as the buylist reference price
			var retailPrice float64

			if product, found := inventory[haId]; found {
				// The category query is already restricted to in-stock items,
				// but guard against a missing or zeroed price just in case.
				// The API stock field is an unreliable aggregate across
				// printings, so quantity is not reported (NoQuantityInventory).
				price, err := strconv.ParseFloat(product.Price, 64)
				if err != nil {
					ha.printf("skipping %s: could not parse price %q: %v", product.Product, product.Price, err)
					continue
				}
				if price <= 0 {
					continue
				}

				retailPrice = price
				foundInventory++

				link := "https://www.hareruyamtg.com/en/products/detail/" + product.Product + "?lang=EN&class=" + product.ProductClass

				out := &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price * ha.exchangeRate,
					URL:        link,
					OriginalId: product.Product,
					InstanceId: product.ProductClass,
				}
				err = ha.inventory.Add(sealedProduct.UUID, out)
				if err != nil {
					ha.printf("%v", err)
				}
			}

			if buyPrice, found := buylist[haId]; found {
				foundBuylist++

				var priceRatio float64
				if retailPrice > 0 {
					priceRatio = buyPrice / retailPrice * 100
				}

				out := &mtgban.BuylistEntry{
					Conditions: "NM",
					BuyPrice:   buyPrice * ha.exchangeRate,
					PriceRatio: priceRatio,
					URL:        "https://www.hareruyamtg.com/ja/purchase/detail/" + haId,
					OriginalId: haId,
				}
				err = ha.buylist.Add(sealedProduct.UUID, out)
				if err != nil {
					ha.printf("%v", err)
				}
			}
		}
	}

	ha.printf("Found %d sealed products in inventory, %d in buylist", foundInventory, foundBuylist)

	ha.inventoryDate = time.Now()
	ha.buylistDate = time.Now()

	return nil
}

func (ha *HareruyaSealed) Inventory() mtgban.InventoryRecord {
	return ha.inventory
}

func (ha *HareruyaSealed) Buylist() mtgban.BuylistRecord {
	return ha.buylist
}

func (ha *HareruyaSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Hareruya"
	info.Shorthand = "HASealed"
	info.CountryFlag = "JP"
	info.InventoryTimestamp = &ha.inventoryDate
	info.BuylistTimestamp = &ha.buylistDate
	info.SealedMode = true
	// The unisearch API only exposes an unreliable aggregate stock count, so
	// per-item quantity is not reported.
	info.NoQuantityInventory = true
	return
}
