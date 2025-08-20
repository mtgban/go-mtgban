package tcgplayer

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	tcgplayer "github.com/mtgban/go-tcgplayer"
)

type TCGPlayerSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	Affiliate      string
	MaxConcurrency int

	SKUsData map[string][]TCGSku

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
	client        *tcgplayer.Client
}

func (tcg *TCGPlayerSealed) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSealed] "+format, a...)
	}
}

func NewScraperSealed(publicId, privateId string) (*TCGPlayerSealed, error) {
	if publicId == "" || privateId == "" {
		return nil, fmt.Errorf("missing authentication data")
	}

	tcg := TCGPlayerSealed{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = tcgplayer.NewClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *TCGPlayerSealed) processEntries(channel chan<- responseChan, reqs []marketChan) error {
	ids := make([]int, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].SkuId
	}

	results, err := tcg.client.GetMarketPricesBySKUs(ids)
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

		link := GenerateProductURL(productId, "", tcg.Affiliate, "", "", false)

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
			buffer := make([]marketChan, 0, tcgplayer.MaxIdsInRequest)

			for page := range pages {
				// Skip dupes
				if slices.Contains(idsFound, page.SkuId) {
					continue
				}
				idsFound = append(idsFound, page.SkuId)

				// Add our pair to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == cap(buffer) {
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
