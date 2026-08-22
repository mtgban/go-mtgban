package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// Sealed prices Magic sealed product from the partner API.
type Sealed struct {
	LogCallback    mtgban.LogCallbackFunc
	Affiliate      string
	MaxConcurrency int
	SKUsData       SKUMap

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
	client        *tcgplayer.Client
}

func (tcg *Sealed) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSealed] "+format, a...)
	}
}

// NewScraperSealed returns a sealed scraper authenticated with a partner API
// key pair.
func NewScraperSealed(publicID, privateID string) (*Sealed, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Sealed{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *Sealed) processEntries(ctx context.Context, channel chan<- responseChan, reqs []marketChan) error {
	ids := make([]int, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].SkuID
	}

	results, err := tcg.client.GetMarketPricesBySKUs(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowestListingPrice == 0 {
			continue
		}

		uuid := ""
		productID := 0
		for _, req := range reqs {
			if result.SKUID == req.SkuID {
				uuid = req.UUID
				productID = req.ProductID
				break
			}
		}

		link := GenerateProductURL(productID, "", tcg.Affiliate, "", "", false)

		out := responseChan{
			cardID: uuid,
			entry: mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      result.LowestListingPrice,
				Quantity:   1,
				URL:        link,
				OriginalID: fmt.Sprint(productID),
				InstanceID: fmt.Sprint(result.SKUID),
			},
		}

		channel <- out
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Sealed) Load(ctx context.Context) error {
	skusMap := tcg.SKUsData
	if skusMap == nil {
		return errors.New("sku map not loaded")
	}
	tcg.printf("Found skus for %d entries", len(skusMap))

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			idsFound := map[int]struct{}{}
			buffer := make([]marketChan, 0, tcgplayer.MaxIDsInRequest)

			for page := range pages {
				// Skip dupes
				_, found := idsFound[page.SkuID]
				if found {
					continue
				}
				idsFound[page.SkuID] = struct{}{}

				// Add our pair to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == cap(buffer) {
					err := tcg.processEntries(ctx, channel, buffer)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntries(ctx, channel, buffer)
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
						ProductID: sku.ProductID,
						SkuID:     sku.SkuID,
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
		err := tcg.inventory.AddRelaxed(result.cardID, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *Sealed) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player"
	info.Shorthand = "TCGSealed"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.NoQuantityInventory = true
	info.SealedMode = true
	return
}
