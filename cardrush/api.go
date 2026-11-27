package cardrush

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	buylistURL = "https://cardrush.media/mtg/buying_prices"

	storeURL      = "https://www.cardrush-mtg.jp/product-list"
	storePageSize = 100
	productURL    = "https://www.cardrush-mtg.jp/product/"

	// pageSize is the most rows the buylist page serves at once; asking for
	// more still returns 1000.
	pageSize = 1000
)

// BuyingPrice is one row of the published buylist.
type BuyingPrice struct {
	ID int `json:"id"`

	// OchaProductID is the product's id on the store's retail site.
	OchaProductID int `json:"mtg_ocha_product_id"`

	// Name carries the treatment notes in parentheses ahead of the card's
	// Japanese and English names: "(FOIL)(フルアート)稲妻/Lightning Bolt".
	Name string `json:"name"`

	// PackCode is a set code, or a set's name where the store has no code.
	PackCode string `json:"pack_code"`

	DisplayCategory string `json:"display_category"`

	// Amount is the price in yen.
	Amount int `json:"amount"`

	// Language is "日本語", "英語" or "イタリア語".
	Language string `json:"language"`

	// Condition is a bracketed grade, "〔EX+〕", or empty for NM.
	Condition string `json:"condition"`

	UpdatedAt string `json:"updated_at"`
}

type nextData struct {
	Props struct {
		PageProps struct {
			BuyingPrices []BuyingPrice `json:"buyingPrices"`
			LastPage     int           `json:"lastPage"`
		} `json:"pageProps"`
	} `json:"props"`
}

// GetBuylistPage returns one page of the buylist and the number of pages it
// spans. The rows come embedded in the page's Next.js data, since the site's
// own /api is closed to crawlers.
func GetBuylistPage(ctx context.Context, client *http.Client, page int) ([]BuyingPrice, int, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(pageSize))
	v.Set("page", strconv.Itoa(page))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buylistURL+"?"+v.Encode(), http.NoBody)
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("page %d: %s", page, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return parseBuylistPage(body)
}

var reStock = regexp.MustCompile(`在庫数\s*(\d+)`)

var (
	nextDataOpen  = []byte(`<script id="__NEXT_DATA__" type="application/json">`)
	nextDataClose = []byte(`</script>`)
)

func parseBuylistPage(body []byte) ([]BuyingPrice, int, error) {
	start := bytes.Index(body, nextDataOpen)
	if start < 0 {
		return nil, 0, errors.New("page carries no __NEXT_DATA__")
	}
	body = body[start+len(nextDataOpen):]
	end := bytes.Index(body, nextDataClose)
	if end < 0 {
		return nil, 0, errors.New("__NEXT_DATA__ is not closed")
	}

	var data nextData
	err := json.Unmarshal(body[:end], &data)
	if err != nil {
		return nil, 0, err
	}
	props := data.Props.PageProps
	return props.BuyingPrices, props.LastPage, nil
}

// Product is one listing of the store, in stock.
type Product struct {
	ID string

	// Title carries the grade, the notes, the names, the language and the
	// set: "[EX+](FOIL)稲妻/Lightning Bolt《英語》【M10】".
	Title string

	// Price is in yen, tax included.
	Price int

	Stock int
}

// GetStorePage returns one page of the store's in-stock listings and the
// number of pages they span. A search stops at its 100th page, so the walk
// reads the whole catalog unfiltered.
func GetStorePage(ctx context.Context, client *http.Client, page int) ([]Product, int, error) {
	v := url.Values{}
	v.Set("num", strconv.Itoa(storePageSize))
	v.Set("available", "1")
	v.Set("page", strconv.Itoa(page))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, storeURL+"?"+v.Encode(), http.NoBody)
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("store page %d: %s", page, resp.Status)
	}
	return parseStorePage(resp.Body)
}

func parseStorePage(r io.Reader) ([]Product, int, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, 0, err
	}

	lastPage := 1
	doc.Find(".pager_btn").Each(func(_ int, s *goquery.Selection) {
		n, err := strconv.Atoi(strings.TrimSpace(s.Text()))
		if err == nil {
			lastPage = max(lastPage, n)
		}
	})

	var products []Product
	doc.Find(".item_data[data-product-id]").Each(func(_ int, s *goquery.Selection) {
		product := Product{
			ID:    s.AttrOr("data-product-id", ""),
			Title: strings.TrimSpace(s.Find(".goods_name").Text()),
		}
		price := strings.ReplaceAll(s.Find(".selling_price .figure").Text(), ",", "")
		product.Price, _ = strconv.Atoi(strings.TrimSuffix(price, "円"))
		// "在庫数 3枚"; a listing reading only "在庫あり" gives no count.
		stock := reStock.FindStringSubmatch(s.Find(".stock").Text())
		if stock != nil {
			product.Stock, _ = strconv.Atoi(stock[1])
		}
		products = append(products, product)
	})

	// A page whose markup moved reads as listings with no price, or as one
	// page; either would publish a store with nothing in it.
	priced := slices.ContainsFunc(products, func(product Product) bool {
		return product.Price > 0
	})
	if len(products) > 0 && !priced {
		return nil, 0, errors.New("store page lists no prices")
	}
	if lastPage == 1 && len(products) == storePageSize {
		return nil, 0, errors.New("store page has a full page and no pager")
	}
	return products, lastPage, nil
}
