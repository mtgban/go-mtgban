package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// Sealed prices Magic sealed product from the partner API.
type Sealed struct {
	logCallback    mtgban.LogCallbackFunc
	affiliate      string
	maxConcurrency int
	skusData       SKUMap

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time

	backend *mtgmatcher.Backend
	client  *tcgplayer.Client
}

func (tcg *Sealed) printf(format string, a ...any) {
	if tcg.logCallback != nil {
		tcg.logCallback("[TCGSealed] "+format, a...)
	}
}

// NewScraperSealed returns a sealed scraper authenticated with a partner API
// key pair.
func NewScraperSealed(b *mtgmatcher.Backend, publicID, privateID string) (*Sealed, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Sealed{}
	tcg.backend = b
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.maxConcurrency = defaultConcurrency
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

		link := GenerateProductURL(productID, "", tcg.affiliate, "", "", false)

		out := responseChan{
			cardID: uuid,
			entry: mtgban.InventoryEntry{
				Conditions: mtgban.NM,
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
	skusMap := tcg.skusData
	if skusMap == nil {
		return errors.New("sku map not loaded")
	}
	tcg.printf("Found skus for %d entries", len(skusMap))

	// Every sku to price once, gathered before any is asked for
	requests := func() []marketChan {
		var reqs []marketChan
		idsFound := map[int]struct{}{}
		sets := tcg.backend.GetAllSets()
		for _, code := range sets {
			set, _ := tcg.backend.GetSet(code)

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
					// Skip dupes
					_, found := idsFound[sku.SkuID]
					if found {
						continue
					}
					idsFound[sku.SkuID] = struct{}{}

					reqs = append(reqs, marketChan{
						UUID:      uuid,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductID: sku.ProductID,
						SkuID:     sku.SkuID,
						Language:  sku.Language,
					})
				}
			}
		}
		return reqs
	}()

	consume := func(result responseChan) {
		// Relaxed because sometimes we get duplicates due to how the ids
		// get buffered, but there is really no harm
		err := tcg.inventory.AddRelaxed(result.cardID, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
		}
	}
	mtgban.WorkerPool(ctx, tcg.maxConcurrency, slices.Collect(slices.Chunk(requests, tcgplayer.MaxIDsInRequest)),
		func(ctx context.Context, reqs []marketChan, channel chan<- responseChan) error {
			return tcg.processEntries(ctx, channel, reqs)
		},
		consume,
		tcg.printf,
	)

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
