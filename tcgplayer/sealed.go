package tcgplayer

import (
	"fmt"
	"sync"

	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type TCGPlayerSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	Affiliate      string
	MaxConcurrency int

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

func (tcg *TCGPlayerSealed) processEntries(channel chan<- responseChan, reqs []indexChan) error {
	ids := make([]string, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].TCGProductId
	}

	results, err := tcg.client.TCGPricesForIds(ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowPrice == 0 {
			continue
		}

		productId := fmt.Sprint(result.ProductId)

		uuid := ""
		for _, req := range reqs {
			if req.TCGProductId == productId {
				uuid = req.UUID
				break
			}
		}

		link := TCGPlayerProductURL(result.ProductId, "", tcg.Affiliate, "")
		out := responseChan{
			cardId: uuid,
			entry: mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      result.LowPrice,
				Quantity:   1,
				URL:        link,
			},
		}

		channel <- out
	}

	return nil
}

func (tcg *TCGPlayerSealed) scrape() error {
	pages := make(chan indexChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			idFound := map[string]string{}
			buffer := make([]indexChan, 0, maxIdsInRequest)

			for page := range pages {
				// Skip dupes
				_, found := idFound[page.TCGProductId]
				if found {
					continue
				}
				idFound[page.TCGProductId] = ""

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
		sets := mtgmatcher.GetSets()
		i := 1
		for _, set := range sets {
			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			for _, product := range set.SealedProduct {
				tcgId, found := product.Identifiers["tcgplayerProductId"]
				if !found {
					continue
				}

				pages <- indexChan{
					TCGProductId: tcgId,
					UUID:         product.UUID,
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
