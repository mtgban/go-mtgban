package miniaturemarket

import (
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	client        *MMClient
	inventory     mtgban.InventoryRecord
}

func NewScraperSealed() *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.client = NewMMClient()
	mm.inventory = mtgban.InventoryRecord{}
	mm.MaxConcurrency = defaultConcurrency
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

func (mm *Miniaturemarket) processPage(channel chan<- respChan, start int) error {
	resp, err := mm.client.GetInventory(start)
	if err != nil {
		return nil
	}
	resp = resp

	for _, product := range resp.Response.Products {
		if product.Quantity == 0 {
			continue
		}

		productName := strings.TrimPrefix(product.Title, "Magic the Gathering: ")
		productName = strings.TrimSuffix(productName, " (Preorder)")
		edition := product.Edition

		uuid, err := preprocessSealed(productName, edition)
		if (err != nil || uuid == "") && strings.Contains(productName, "Commander") && !strings.Contains(edition, "Commander") {
			uuid, err = preprocessSealed(productName, edition+" Commander")
		}
		if err != nil {
			if err.Error() != "unsupported" {
				mm.printf("%s in %s | %s", productName, edition, err.Error())
			}
			continue
		}

		if uuid == "" {
			if !strings.Contains(productName, "Prerelease Pack") &&
				!strings.Contains(productName, "Starter Kit") &&
				!strings.Contains(productName, "Case") {
				mm.printf("unable to parse %s in %s", productName, edition)
			}
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

func (mm *Miniaturemarket) scrape() error {
	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	totalProducts, err := mm.client.NumberOfProducts()
	if err != nil {
		return err
	}
	mm.printf("Parsing %d items", totalProducts)

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for start := range pages {
				err = mm.processPage(channel, start)
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

func (mm *Miniaturemarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(mm.inventory) > 0 {
		return mm.inventory, nil
	}

	err := mm.scrape()
	if err != nil {
		return nil, err
	}

	return mm.inventory, nil
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MMSealed"
	info.InventoryTimestamp = &mm.inventoryDate
	info.SealedMode = true
	return
}
