package coolstuffinc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	http "github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
)

type CoolstuffincSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	httpclient *http.Client
}

func NewScraperSealed() *CoolstuffincSealed {
	csi := CoolstuffincSealed{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.httpclient = http.NewClient()
	csi.httpclient.Logger = nil
	csi.MaxConcurrency = defaultConcurrency
	return &csi
}

func (csi *CoolstuffincSealed) printf(format string, a ...interface{}) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSISealed] "+format, a...)
	}
}

const sealedURL = "https://www.coolstuffinc.com/sq/1555582?page=1&sb=price|desc"

func (csi *CoolstuffincSealed) numOfPages() (int, error) {
	resp, err := csi.httpclient.Get(sealedURL)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	text := doc.Find(".search-result-links").Text()
	text = strings.TrimPrefix(strings.Split(text, " Results")[0], "1 - ")

	fields := strings.Split(text, " of ")
	if len(fields) != 2 {
		return 0, errors.New("unknown page format")
	}

	resultsPerPage, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, err
	}

	resultsTotal, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, err
	}

	return resultsTotal/resultsPerPage + 1, nil
}

func (csi *CoolstuffincSealed) processSealedPage(channel chan<- responseChan, page int) error {
	u, err := url.Parse(sealedURL)
	if err != nil {
		return err
	}

	v := u.Query()
	v.Set("page", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	resp, err := csi.httpclient.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(".main-container").Each(func(i int, s *goquery.Selection) {
		productName := s.Find(`span[itemprop="name"]`).Text()

		edition := s.Find(`div[class="breadcrumb-trail"]`).Text()
		edition = strings.TrimPrefix(edition, "Magic: The Gathering » ")

		uuid, err := preprocessSealed(productName, edition)
		if (err != nil || uuid == "") && strings.Contains(productName, "Commander") && !strings.Contains(edition, "Commander") {
			uuid, err = preprocessSealed(productName, edition+" Commander")
		}
		if err != nil {
			if err.Error() != "unsupported" {
				csi.printf("%s in %s | %s", productName, edition, err.Error())
			}
			return
		}

		if uuid == "" {
			csi.printf("unable to parse %s in %s", productName, edition)
			return
		}

		qtyStr := s.Find(`span[class="card-qty"]`).Text()
		qtyStr = strings.TrimSuffix(qtyStr, "+")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			qty = 20
		}

		priceStr := s.Find(`b[itemprop="price"]`).Text()
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%s: %s", productName, err.Error())
			return
		}

		path, _ := s.Find(`a[class="productLink"]`).Attr("href")
		link := "https://coolstuffinc.com" + path

		out := responseChan{
			cardId: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price:    price,
				Quantity: qty,
				URL:      link,
			},
		}

		channel <- out
	})

	return nil
}

func (csi *CoolstuffincSealed) scrape() error {
	totalPages, err := csi.numOfPages()
	if err != nil {
		return err
	}
	csi.printf("Processing %d pages", totalPages)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := csi.processSealedPage(results, page)
				if err != nil {
					csi.printf("page %d: %s", page, err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= totalPages; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := csi.inventory.Add(record.cardId, record.invEntry)
		if err != nil {
			csi.printf("%s", err.Error())
			continue
		}
	}

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *CoolstuffincSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(csi.inventory) > 0 {
		return csi.inventory, nil
	}

	err := csi.scrape()
	if err != nil {
		return nil, err
	}

	return csi.inventory, nil
}

func (csi *CoolstuffincSealed) processPage(channel chan<- responseChan, edition string) error {
	resp, err := csi.httpclient.PostForm(csiBuylistURL, url.Values{
		"ajaxtype": {"selectProductSetName2"},
		"ajaxdata": {edition},
		"gamename": {"mtg"},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var blob struct {
		HTML string `json:"html"`
	}
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(blob.HTML))
	if err != nil {
		return err
	}

	doc.Find(".main-container").Each(func(i int, s *goquery.Selection) {
		extra := s.Find(".search-info-cell").Find(".mini-print").Not(".breadcrumb-trail").Text()
		extra = strings.Replace(extra, "\n", " ", -1)
		info := s.Find(".search-info-cell").Find(".breadcrumb-trail").Text()
		if !strings.Contains(info, "Sealed") {
			return
		}

		productName, _ := s.Attr("data-name")

		uuid, err := preprocessSealed(productName, edition)
		if err != nil {
			if err.Error() != "unsupported" {
				csi.printf("%s: %s for %s", err.Error(), edition, productName)
			}
			return
		}

		if uuid == "" {
			csi.printf("unable to parse %s in", productName)
			return
		}

		priceStr, _ := s.Attr("data-price")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%v", err)
			return
		}

		channel <- responseChan{
			cardId: uuid,
			buyEntry: &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: price * 1.3,
				URL:        defaultBuylistPage,
			},
		}
	})
	return nil
}

func (csi *CoolstuffincSealed) parseBL() error {
	resp, err := csi.httpclient.PostForm(csiBuylistURL, url.Values{
		"ajaxtype": {"selectsearchgamename2"},
		"ajaxdata": {"mtg"},
	})
	if err != nil {
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var blob struct {
		HTML string `json:"html"`
	}
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(blob.HTML))
	if err != nil {
		return err
	}

	editions := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := csi.processPage(results, edition)
				if err != nil {
					csi.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		doc.Find("option").Each(func(_ int, s *goquery.Selection) {
			edition := s.Text()
			if edition == "Bulk Magic" {
				return
			}
			editions <- edition
		})
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := csi.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			csi.printf("%s", err.Error())
			continue
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

func (csi *CoolstuffincSealed) Buylist() (mtgban.BuylistRecord, error) {
	if len(csi.buylist) > 0 {
		return csi.buylist, nil
	}

	err := csi.parseBL()
	if err != nil {
		return nil, err
	}

	return csi.buylist, nil
}

func (csi *CoolstuffincSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSISealed"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.SealedMode = true
	return
}
