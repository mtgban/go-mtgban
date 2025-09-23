package hareruya

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultConcurrency = 8

	modeInventory = "inventory"
	modeBuylist   = "buylist"

	inventoryURL = "https://www.hareruyamtg.com/en/products/search?language[0]=2&stock=1&page="
	buylistURL   = "https://www.hareruyamtg.com/ja/purchase/search?language[0]=2&stock=1&page="
)

type Hareruya struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	exchangeRate float64

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time

	buylist     mtgban.BuylistRecord
	buylistDate time.Time

	client *http.Client
}

func NewScraper() (*Hareruya, error) {
	ha := Hareruya{}
	ha.inventory = mtgban.InventoryRecord{}
	ha.buylist = mtgban.BuylistRecord{}
	ha.MaxConcurrency = defaultConcurrency
	client := retryablehttp.NewClient()
	client.Logger = nil
	ha.client = client.StandardClient()
	rate, err := mtgban.GetExchangeRate("JPY")
	if err != nil {
		return nil, err
	}
	ha.exchangeRate = rate
	return &ha, nil
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (ha *Hareruya) printf(format string, a ...interface{}) {
	if ha.LogCallback != nil {
		ha.LogCallback("[HA] "+format, a...)
	}
}

func (ha *Hareruya) processPage(channel chan<- responseChan, page int, mode string) error {
	link := inventoryURL
	if mode == modeBuylist {
		link = buylistURL
	}
	resp, err := ha.client.Get(link + fmt.Sprint(page))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(`div[class="autopagerize_page_element"] ul li`).Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find(`div[class="itemData"] a`).Attr("href")
		link = strings.TrimSpace(link)

		title := s.Find(`div[class="itemData"] a`).Text()
		title = strings.TrimSpace(title)

		priceStr := s.Find(`p[class="itemDetail__price"]`).Text()
		priceStr = strings.TrimPrefix(priceStr, "¥ ")
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			ha.printf("page %d entry %d: %s", page, i, err.Error())
			return
		}

		var cond string
		var qty int
		if mode == modeInventory {
			stock := s.Find(`p[class="itemDetail__stock"]`).Text()
			switch {
			case strings.Contains(stock, "Signed"),
				strings.Contains(stock, "Sighned"),
				strings.Contains(stock, "Inked"):
				return
			}

			fields := strings.Fields(stock)
			if len(fields) != 2 {
				ha.printf("page %d entry %d: unsupported %s", page, i, stock)
				return
			}

			cond = strings.TrimPrefix(fields[0], "【")
			qtyStr := strings.TrimPrefix(fields[1], "Stock:")
			qtyStr = strings.TrimSuffix(qtyStr, "】")
			qty, err = strconv.Atoi(qtyStr)
			if err != nil {
				ha.printf("page %d entry %d: %s", page, i, err.Error())
				return
			}
		} else if mode == modeBuylist {
			stock := s.Find(`span[class="itemUserAct__number__title"]`).Text()
			if stock != "個数" {
				return
			}
		}

		theCard, err := preprocess(title)
		if err != nil {
			return
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			if theCard.IsBasicLand() {
				return
			}
			ha.printf("%v at page %d", err, page)
			ha.printf("%q", theCard)
			ha.printf("%s", title)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ha.printf("- %s", card)
				}
			}
			return
		}

		if mode == modeInventory {
			if price != 0 && qty != 0 {
				out := responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Price:      price * ha.exchangeRate,
						Conditions: cond,
						Quantity:   qty,
						URL:        "https://www.hareruyamtg.com" + link,
					},
				}
				channel <- out
			}
		} else if mode == modeBuylist {
			var priceRatio, sellPrice float64

			invCards := ha.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			deductions := []float64{1, 0.8, 0.6, 0.4}
			for i, deduction := range deductions {
				out := responseChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						Conditions: mtgban.DefaultGradeTags[i],
						BuyPrice:   price * deduction * ha.exchangeRate,
						PriceRatio: priceRatio,
						URL:        "https://www.hareruyamtg.com" + link,
					},
				}
				channel <- out
			}

			return
		}

		s.Find(`div[class="tableHere product"] div[class="row not-first ng-star-inserted"]`).Each(func(i int, sub *goquery.Selection) {
			qtyStr := sub.Find(`div:nth-child(3)`).Text()
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				return
			}
			priceStr := sub.Find(`div:nth-child(2)`).Text()
			priceStr = strings.TrimPrefix(priceStr, "¥ ")
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				ha.printf("%s", err.Error())
				return
			}

			cond := sub.Find(`strong`).Text()

			if price != 0 && qty != 0 {
				out := responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Price:      price * ha.exchangeRate,
						Conditions: cond,
						Quantity:   qty,
						URL:        "https://www.hareruyamtg.com" + link,
					},
				}
				channel <- out
			}
		})
	})

	return nil
}

func (ha *Hareruya) totalPages(mode string) (int, error) {
	link := inventoryURL
	if mode == modeBuylist {
		link = buylistURL
	}
	resp, err := ha.client.Get(link)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	pagination := doc.Find(`div[class="result_pagenum"]`).Text()
	fields := strings.Fields(strings.TrimSpace(pagination))
	if len(fields) == 0 {
		ha.printf("unsupported structure %v", fields)
		return 0, errors.New("malformed pagination")
	}

	pagenum := strings.TrimSuffix(strings.TrimPrefix(fields[0], "Page1/"), "ページ中1ページ目")
	return strconv.Atoi(pagenum)
}

func (ha *Hareruya) scrape(mode string) error {
	total, err := ha.totalPages(mode)
	if err != nil {
		return err
	}

	ha.printf("Found %d pages", total)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < ha.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for i := range pages {
				err := ha.processPage(results, i, mode)
				if err != nil {
					ha.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= total; i++ {
			ha.printf("Processing page %d", i)
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		var err error
		if record.invEntry != nil {
			err = ha.inventory.Add(record.cardId, record.invEntry)
		} else if record.buyEntry != nil {
			err = ha.buylist.Add(record.cardId, record.buyEntry)
		}
		if err != nil {
			ha.printf("%s", err.Error())
		}
	}

	if mode == modeInventory {
		ha.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		ha.buylistDate = time.Now()
	}

	return nil
}

func (ha *Hareruya) Inventory() (mtgban.InventoryRecord, error) {
	if len(ha.inventory) > 0 {
		return ha.inventory, nil
	}

	err := ha.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return ha.inventory, nil
}

func (ha *Hareruya) Buylist() (mtgban.BuylistRecord, error) {
	if len(ha.buylist) > 0 {
		return ha.buylist, nil
	}

	err := ha.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return ha.buylist, nil
}

func (ha *Hareruya) Info() (info mtgban.ScraperInfo) {
	info.Name = "Hareruya"
	info.Shorthand = "HA"
	info.CountryFlag = "JP"
	info.InventoryTimestamp = &ha.inventoryDate
	info.BuylistTimestamp = &ha.buylistDate
	return
}
