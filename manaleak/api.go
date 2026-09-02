package manaleak

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	// The retail singles all file under the Wizards of the Coast brand page;
	// the buylist mirrors the category tree, where 416 is the singles.
	inventoryURL = "https://www.manaleak.com/wizards-of-the-coast"
	buylistURL   = "https://www.manaleak.com/index.php?route=buylist/category&path=416"

	// The widest page the storefront serves. The sort is pinned so the
	// pages keep their order while the crawl fans out over them.
	pageLimit = 100

	// The server answers 406 to a bare client, so every request wears a
	// browser's User-Agent and Accept.
	staticUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"
	staticAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

// MLProduct is one listing row: a product in one finish, priced in pounds.
type MLProduct struct {
	Name       string
	SetName    string
	URL        string
	Price      float64
	OutOfStock bool

	// The card behind the row, read from its image file: the newer sets
	// name theirs by TCGplayer product id, the older by multiverse id, and
	// the file name says which era it is.
	TCGProductID string
	MultiverseID string
}

// cardImage is the path card images carry, "/mtg/<set>/<id>_200w-..." for the
// TCGplayer-id era and "/mtg/<set>/<id>.full-..." for the multiverse-id one.
// A row whose image matches neither is not a single at all - the brand page
// also lists the sealed product and the repacks.
var cardImage = regexp.MustCompile(`/mtg/[^/]+/(\d+)(_200w|\.full)`)

// showingTotal is the listing's own count of everything it paginates.
var showingTotal = regexp.MustCompile(`Showing \d+ to \d+ of (\d+)`)

// MLClient reads the Manaleak storefront.
type MLClient struct {
	client *http.Client
}

// NewMLClient returns a client for the storefront.
func NewMLClient() *MLClient {
	ml := MLClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	ml.client = client.StandardClient()
	return &ml
}

func pageURL(base string, page int) string {
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("limit", strconv.Itoa(pageLimit))
	q.Set("page", strconv.Itoa(page))
	q.Set("sort", "pd.name")
	q.Set("order", "ASC")
	u.RawQuery = q.Encode()
	return u.String()
}

// GetPage fetches one listing page and answers its rows and the listing's
// own total, off which the caller sizes the fan-out.
func (ml *MLClient) GetPage(ctx context.Context, base string, page int) ([]MLProduct, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL(base, page), http.NoBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", staticUA)
	req.Header.Set("Accept", staticAccept)

	resp, err := ml.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	total := 0
	if m := showingTotal.FindStringSubmatch(doc.Text()); m != nil {
		total, _ = strconv.Atoi(m[1])
	}

	return parseListing(doc), total, nil
}

func parseListing(doc *goquery.Document) []MLProduct {
	var products []MLProduct
	doc.Find("div.product-list-item").Each(func(_ int, row *goquery.Selection) {
		var product MLProduct

		name := row.Find(".caption .name a").First()
		product.Name = strings.TrimSpace(name.Text())
		product.URL, _ = name.Attr("href")

		// The set rides in the row's description as a link on the retail
		// side; the buylist fills the description with rules text instead,
		// and its rows lean on the image id alone.
		product.SetName = strings.TrimSpace(row.Find(".caption .description a").First().Text())

		price := strings.TrimSpace(row.Find(".caption .price").First().Text())
		price = strings.TrimPrefix(price, "£")
		product.Price, _ = strconv.ParseFloat(strings.ReplaceAll(price, ",", ""), 64)

		product.OutOfStock = row.HasClass("outofstock")

		src, _ := row.Find(".image img").First().Attr("data-src")
		if m := cardImage.FindStringSubmatch(src); m != nil {
			if m[2] == "_200w" {
				product.TCGProductID = m[1]
			} else {
				product.MultiverseID = m[1]
			}
		}

		products = append(products, product)
	})
	return products
}
