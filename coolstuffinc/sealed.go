package coolstuffinc

import (
	"encoding/json"
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

	buylistDate    time.Time
	MaxConcurrency int

	buylist mtgban.BuylistRecord

	httpclient *http.Client
}

func NewScraperSealed() *CoolstuffincSealed {
	csi := CoolstuffincSealed{}
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
	info.BuylistTimestamp = &csi.buylistDate
	info.SealedMode = true
	return
}
