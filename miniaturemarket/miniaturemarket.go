package miniaturemarket

import (
	"context"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	client        *MMClient
	inventory     mtgban.InventoryRecord
	productMap    map[string]string
}

func NewScraperSealed() *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.client = NewMMClient()
	mm.inventory = mtgban.InventoryRecord{}
	mm.MaxConcurrency = defaultConcurrency
	mm.productMap = map[string]string{}
	return &mm
}

const (
	defaultConcurrency = 6
)

type respChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (mm *Miniaturemarket) printf(format string, a ...interface{}) {
	if mm.LogCallback != nil {
		mm.LogCallback("[MMSealed] "+format, a...)
	}
}

func (mm *Miniaturemarket) processPage(ctx context.Context, channel chan<- respChan, start int) error {
	resp, err := mm.client.GetInventory(ctx, start)
	if err != nil {
		return nil
	}
	resp = resp

	for _, product := range resp.Response.Products {
		if product.Quantity <= 0 {
			continue
		}

		uuid, found := mm.productMap[product.EntityId]
		if !found {
			continue
		}

		link := product.URL
		if mm.Affiliate != "" {
			link += "?utm_source=" + mm.Affiliate + "&utm_medium=feed&utm_campaign=mtg_singles"
		}

		channel <- respChan{
			cardId: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price:    product.Price,
				Quantity: product.Quantity,
				URL:      link,
			},
		}
	}

	return nil
}

func (mm *Miniaturemarket) Load(ctx context.Context) error {
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || co.Identifiers["miniaturemarketId"] == "" {
			continue
		}
		mm.productMap[co.Identifiers["miniaturemarketId"]] = uuid
	}
	mm.printf("Loaded %d sealed products", len(mm.productMap))

	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	totalProducts, err := mm.client.NumberOfProducts(ctx)
	if err != nil {
		return err
	}
	mm.printf("Parsing %d items", totalProducts)

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for start := range pages {
				err = mm.processPage(ctx, channel, start)
				if err != nil {
					mm.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < totalProducts; i += MMDefaultResultsPerPage {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := mm.inventory.Add(record.cardId, record.invEntry)
		if err != nil {
			mm.printf("%v", err)
			continue
		}
	}

	mm.inventoryDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) Inventory() mtgban.InventoryRecord {
	return mm.inventory
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MMSealed"
	info.InventoryTimestamp = &mm.inventoryDate
	info.SealedMode = true
	return
}
