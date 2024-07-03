package tcgplayer

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

type TCGPlayerSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	Affiliate      string
	MaxConcurrency int

	SKUsData map[string][]mtgjson.TCGSku

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
	client        *TCGClient
}

func (tcg *TCGPlayerSealed) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSealed] "+format, a...)
	}
}

func NewScraperSealed(publicId, privateId string) *TCGPlayerSealed {
	tcg := TCGPlayerSealed{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerSealed) processEntries(channel chan<- responseChan, reqs []marketChan) error {
	ids := make([]string, len(reqs))
	for i := range reqs {
		ids[i] = fmt.Sprint(reqs[i].SkuId)
	}

	results, err := tcg.client.TCGPricesForSKUs(ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowestListingPrice == 0 {
			continue
		}

		uuid := ""
		productId := 0
		for _, req := range reqs {
			if result.SkuId == req.SkuId {
				uuid = req.UUID
				productId = req.ProductId
				break
			}
		}

		link := TCGPlayerProductURL(productId, "", tcg.Affiliate, "", "", false)

		out := responseChan{
			cardId: uuid,
			entry: mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      result.LowestListingPrice,
				Quantity:   1,
				URL:        link,
				OriginalId: fmt.Sprint(productId),
				InstanceId: fmt.Sprint(result.SkuId),
			},
		}

		channel <- out
	}

	return nil
}

func (tcg *TCGPlayerSealed) scrape() error {
	skusMap := tcg.SKUsData
	if skusMap == nil {
		var err error
		tcg.printf("Retrieving skus")
		skusMap, err = getAllSKUs()
		if err != nil {
			return err

		}
	}
	tcg.printf("Found skus for %d entries", len(skusMap))

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			var idsFound []int
			buffer := make([]marketChan, 0, maxIdsInRequest)

			for page := range pages {
				// Skip dupes
				if slices.Contains(idsFound, page.SkuId) {
					continue
				}
				idsFound = append(idsFound, page.SkuId)

				// Add our pair to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == maxIdsInRequest {
					err := tcg.processEntries(channel, buffer)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntries(channel, buffer)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetAllSets()
		for _, code := range sets {
			set, _ := mtgmatcher.GetSet(code)

			for _, product := range set.SealedProduct {
				uuid := product.UUID
				skus, found := skusMap[uuid]
				if !found {
					continue
				}
				for _, sku := range skus {
					// Only keep sealed products
					if sku.Condition != "UNOPENED" {
						continue
					}

					pages <- marketChan{
						UUID:      uuid,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductId: sku.ProductId,
						SkuId:     sku.SkuId,
						Language:  sku.Language,
					}
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		// Relaxed because sometimes we get duplicates due to how the ids
		// get buffered, but there is really no harm
		err := tcg.inventory.AddRelaxed(result.cardId, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGPlayerSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player"
	info.Shorthand = "TCGSealed"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.NoQuantityInventory = true
	info.SealedMode = true
	return
}
