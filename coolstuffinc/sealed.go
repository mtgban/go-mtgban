package coolstuffinc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type CoolstuffincSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	productMap map[string]string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	DisableRetail  bool
	DisableBuylist bool

	client *http.Client
	game   string
}

func NewScraperSealed() *CoolstuffincSealed {
	csi := CoolstuffincSealed{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	csi.client = client.StandardClient()
	csi.MaxConcurrency = defaultConcurrency

	csi.productMap = map[string]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		id, found := co.Identifiers["csiId"]
		if !found {
			continue
		}
		csi.productMap[id] = co.UUID
	}
	csi.game = GameMagic
	return &csi
}

func (csi *CoolstuffincSealed) printf(format string, a ...interface{}) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSISealed] "+format, a...)
	}
}

const sealedURL = "https://www.coolstuffinc.com/sq/2293832?page=1&sb=price|desc"

func (csi *CoolstuffincSealed) numOfPages(ctx context.Context) (int, error) {
	link := sealedURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := csi.client.Do(req)
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

func (csi *CoolstuffincSealed) processSealedPage(ctx context.Context, channel chan<- responseChan, page int) error {
	csi.printf("Processing page %d", page)

	u, err := url.Parse(sealedURL)
	if err != nil {
		return err
	}

	v := u.Query()
	v.Set("page", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	link := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := csi.client.Do(req)
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
		path, _ := s.Find(`a[class="productLink"]`).Attr("href")

		csiId := strings.TrimPrefix(path, "/p/")

		uuid, found := csi.productMap[csiId]
		if !found {
			return
		}

		qtyStr := s.Find(`span[class="card-qty"]`).Text()
		qtyStr = strings.TrimSuffix(qtyStr, "+")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			qty = 20
		}

		priceStr := s.Find(`b[itemprop="price"]`).First().Text()
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%s: %s", productName, err.Error())
			return
		}

		link := "https://coolstuffinc.com" + path
		if csi.Partner != "" {
			link += "?utm_referrer=" + csi.Partner
		}

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

func (csi *CoolstuffincSealed) scrape(ctx context.Context) error {
	totalPages, err := csi.numOfPages(ctx)
	if err != nil {
		return err
	}
	csi.printf("Found %d pages", totalPages)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := csi.processSealedPage(ctx, results, page)
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

func (csi *CoolstuffincSealed) parseBL(ctx context.Context) error {
	products, err := GetBuylist(ctx, csi.game)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName != "Box" {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", "mtg")
		v.Set("a", "1")
		v.Set("name", product.Name)
		u.RawQuery = v.Encode()
		link := u.String()

		uuid, found := csi.productMap[product.PID]
		if !found {
			continue
		}

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[uuid]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		for i, deduction := range deductions {
			buyEntry := mtgban.BuylistEntry{
				Conditions: mtgban.DefaultGradeTags[i],
				BuyPrice:   buyPrice * deduction,
				PriceRatio: priceRatio,
				URL:        link,
			}

			err := csi.buylist.Add(uuid, &buyEntry)
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

func (csi *CoolstuffincSealed) Load(ctx context.Context) error {
	var errs []error

	if !csi.DisableRetail {
		err := csi.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !csi.DisableBuylist {
		err := csi.parseBL(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (csi *CoolstuffincSealed) Inventory() (mtgban.InventoryRecord, error) {
	return csi.inventory, nil
}

func (csi *CoolstuffincSealed) Buylist() (mtgban.BuylistRecord, error) {
	return csi.buylist, nil
}

func (csi *CoolstuffincSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSISealed"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.25
	return
}
