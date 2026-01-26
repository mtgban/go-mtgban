package miniaturemarket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	productMap    map[string]string
}

func NewScraperSealed() *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.inventory = mtgban.InventoryRecord{}
	mm.MaxConcurrency = defaultConcurrency
	mm.productMap = map[string]string{}
	return &mm
}

const (
	defaultConcurrency = 6

	mainURL = "https://www.miniaturemarket.com/widgets/cms/navigation/be53d253d6bc3258a8160556dda3e9b2?filter-inStock=1&no-aggregations=1&order=name-asc&p=1"
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

func (mm *Miniaturemarket) processPage(ctx context.Context, channel chan<- respChan, page int) error {
	u, err := url.Parse(mainURL)
	if err != nil {
		return err
	}
	v := u.Query()
	v.Set("p", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return err
	}
	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		mm.printf("newDoc - %s", err.Error())
		return err
	}

	doc.Find(`div[class="product-info"]`).Each(func(i int, s *goquery.Selection) {
		id, _ := s.Find(`input[name="product-id"]`).Attr("value")
		uuid, found := mm.productMap[id]
		if !found {
			return
		}

		link, _ := s.Find(`a.product-name`).Attr("href")
		if mm.Affiliate != "" {
			link += "?utm_source=" + mm.Affiliate + "&utm_medium=feed&utm_campaign=mtg_singles"
		}

		priceStr := s.Find(`.product-price`).Text()
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			mm.printf("uuid %s - %s", uuid, err.Error())
			return
		}

		channel <- respChan{
			cardId: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price: price,
				URL:   link,
			},
		}
	})

	return nil
}

func (mm *Miniaturemarket) NumberOfProducts(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mainURL, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		mm.printf("newDoc - %s", err.Error())
		return 0, err
	}

	num, _ := doc.Find(`input[id="p-last-bottom"]`).Attr("value")
	return strconv.Atoi(num)
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

	totalProducts, err := mm.NumberOfProducts(ctx)
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
		for i := 0; i < totalProducts; i++ {
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
	info.NoQuantityInventory = true
	return
}
