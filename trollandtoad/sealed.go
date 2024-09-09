package trollandtoad

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	"github.com/hashicorp/go-cleanhttp"
	http "github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TrollandtoadSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	productMap map[string]string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	httpclient *http.Client
}

func NewScraperSealed() *TrollandtoadSealed {
	tnt := TrollandtoadSealed{}
	tnt.inventory = mtgban.InventoryRecord{}
	tnt.buylist = mtgban.BuylistRecord{}
	tnt.httpclient = http.NewClient()
	tnt.httpclient.Logger = nil
	tnt.MaxConcurrency = defaultConcurrency

	tnt.productMap = map[string]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		id, found := co.Identifiers["tntId"]
		if !found {
			continue
		}
		tnt.productMap[id] = co.UUID
	}
	return &tnt
}

func (tnt *TrollandtoadSealed) printf(format string, a ...interface{}) {
	if tnt.LogCallback != nil {
		tnt.LogCallback("[TNTSealed] "+format, a...)
	}
}

func (tnt *TrollandtoadSealed) parsePages(link string, lastPage int) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.trollandtoad.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 2 * time.Second,
		Parallelism: tnt.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		tnt.printf("Visiting page %s", r.URL.Query().Get("page-no"))
	})

	c.OnHTML(`div[class="product-col col-12 p-0 my-1 mx-sm-1 mw-100"]`, func(e *colly.HTMLElement) {
		link := e.ChildAttr(`a[class='card-text']`, "href")

		oos := e.ChildText(`div[class='row mb-2 '] div[class='col-12'] div[class='font-weight-bold font-smaller text-muted']`)
		if oos == "Out of Stock" {
			return
		}

		links := strings.Split(link, "/")
		tntId := links[len(links)-1]

		uuid, found := tnt.productMap[tntId]
		if !found {
			return
		}

		e.ForEach(`div[class="row position-relative align-center py-2 m-auto"]`, func(_ int, el *colly.HTMLElement) {
			qtys := el.ChildTexts(`option`)
			if len(qtys) == 0 {
				return
			}
			qtyStr := qtys[len(qtys)-1]
			qty, _ := strconv.Atoi(qtyStr)
			if qty == 0 {
				return
			}

			priceStr := el.ChildText(`div[class='col-2 text-center p-1']`)
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				tnt.printf("%s", err.Error())
				return
			}
			if price == 0 {
				return
			}

			out := responseChan{
				cardId: uuid,
				invEntry: &mtgban.InventoryEntry{
					Price:    price,
					Quantity: qty,
					URL:      e.Request.AbsoluteURL(link),
				},
			}
			channel <- out
		})
	})

	q, _ := queue.New(
		tnt.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for i := 0; i < lastPage; i++ {
		opts := fmt.Sprintf(tntOptions, i+1)
		q.AddURL(link + opts)
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	for res := range channel {
		err := tnt.inventory.Add(res.cardId, res.invEntry)
		if err != nil {
			// Too many false positives
			//tnt.printf("%v", err)
		}
	}

	tnt.inventoryDate = time.Now()

	return nil
}

const (
	categorySealedPage = "https://www.trollandtoad.com/magic-the-gathering/magic-the-gathering-sealed-product/909"
)

func (tnt *TrollandtoadSealed) scrape() error {
	resp, err := cleanhttp.DefaultClient().Get(categorySealedPage + tntOptions)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	lastPage := 0
	doc.Find(`div[class="lastPage pageLink d-flex font-weight-bold"]`).Each(func(_ int, s *goquery.Selection) {
		page, _ := s.Attr("data-page")
		lastPage, err = strconv.Atoi(page)
	})
	if err != nil {
		return err
	}

	if lastPage == 0 {
		lastPage = 1
	}
	tnt.printf("Parsing %d pages from sealed", lastPage)
	return tnt.parsePages(categorySealedPage, lastPage)
}

func (tnt *TrollandtoadSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(tnt.inventory) > 0 {
		return tnt.inventory, nil
	}

	err := tnt.scrape()
	if err != nil {
		return nil, err
	}

	return tnt.inventory, nil
}

func (tnt *TrollandtoadSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "TrollandToad"
	info.Shorthand = "TNTSealed"
	info.InventoryTimestamp = &tnt.inventoryDate
	info.SealedMode = true
	return
}
