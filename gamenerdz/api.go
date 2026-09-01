package gamenerdz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mtgban/go-mtgban/internal/jsonflex"
)

const (
	// The buylist subdomain fronts the StorePass platform, which also renders
	// the retail storefront's card pages: one search endpoint serves both
	// sides, told apart by a mode flag.
	baseURL  = "https://buylist.gamenerdz.com/saas/search"
	setsURL  = "https://buylist.gamenerdz.com/saas/filters/sets"
	storeID  = "OvpVz0pNlL"
	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"

	// The two stable orderings one crawl pass runs under. Relevance reshuffles
	// while the pages are being fetched, which loses whatever moves between
	// two of them, so the crawl only ever asks for the alphabet.
	sortForward  = "Alphabetical: A-Z"
	sortBackward = "Alphabetical: Z-A"

	// A bound far past the deepest feed the storefront serves today, purely
	// so a feed that never answers empty keeps the crawl finite.
	maxPages = 5000

	modeRetail  = "retail_products"
	modeBuylist = "buylist_products"
)

// GNResponse is what the storefront's product endpoint answers with.
type GNResponse struct {
	Count    int         `json:"count"`
	Pages    int         `json:"pages"`
	Products []GNProduct `json:"products"`
}

// GNProduct is one catalog entry.
type GNProduct struct {
	ID             string            `json:"id"`
	ProductID      int64             `json:"product_id"`
	DisplayName    string            `json:"display_name"`
	Price          float64           `json:"price"`
	SelectedFinish string            `json:"selectedFinish"`
	ProductData    GNProductData     `json:"product_data"`
	BuyVariants    []GNBuyVariant    `json:"store_pass_variant_info"`
	RetailVariants []GNRetailVariant `json:"variant_info"`
}

// GNProductData is the body of a product, apart from the envelope it arrives
// in.
type GNProductData struct {
	Set     jsonflex.String `json:"set"`
	SetName string          `json:"setName"`
	Rarity  string          `json:"rarity"`
}

// GNBuyVariant is one buylist offer on a product. The credit price arrives
// beside the cash one, already carrying the store's 25% bump, so the ratio is
// theirs to keep - the scraper reads the cash price and states the bump once,
// as the vendor's credit multiplier.
type GNBuyVariant struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	OfferPrice       float64 `json:"offer_price"`
	OfferPriceCredit float64 `json:"offer_price_credit"`
	SelectedFinish   string  `json:"selected_finish"`
}

// GNRetailVariant is the BigCommerce variant behind a product, which is where
// the live stock lives: the buylist-mode feed prices retail too but reports no
// inventory, so only the retail mode can say what is actually on sale.
type GNRetailVariant struct {
	ID                 int64   `json:"id"`
	SKU                string  `json:"sku"`
	Price              float64 `json:"price"`
	InventoryLevel     int     `json:"inventory_level"`
	PurchasingDisabled bool    `json:"purchasing_disabled"`
}

// GNClient reads the Game Nerdz storefront, one product line at a time.
type GNClient struct {
	client      *http.Client
	productLine string
	// The storefront's search endpoint, held rather than referenced so a
	// test can point the client at a server it controls.
	baseURL string
}

// NewGNClient returns a client for one product line, in the storefront's own
// spelling (the Game constants).
func NewGNClient(productLine string) *GNClient {
	gn := GNClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	gn.client = client.StandardClient()
	gn.productLine = productLine
	gn.baseURL = baseURL
	return &gn
}

func (gn *GNClient) buildURL(mode string, params map[string]string) string {
	u, _ := url.Parse(gn.baseURL)
	q := u.Query()

	q.Set("store_id", storeID)
	q.Set("product_line", gn.productLine)
	q.Set("mongo", "true")
	q.Set(mode, "true")
	q.Set("ignore_is_hot_order", "true")
	q.Set("sort", sortForward)

	// Retail pages arrive with the whole BigCommerce record per product,
	// near a megabyte a page over a feed that is mostly out of stock, and
	// the server takes seconds to render each. Only what is on sale is
	// worth those seconds; the buylist reads everything, because an offer
	// stands whether or not the store also retails the card.
	if mode == modeRetail {
		q.Set("in_stock", "true")
	}

	for k, v := range params {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (gn *GNClient) query(ctx context.Context, reqURL string) (*GNResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", staticUA)

	resp, err := gn.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response GNResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (gn *GNClient) getCount(ctx context.Context, mode string, filters map[string]string) (int, error) {
	params := map[string]string{
		"with_count": "true",
		"no_track":   "true",
	}
	maps.Copy(params, filters)
	response, err := gn.query(ctx, gn.buildURL(mode, params))
	if err != nil {
		return 0, err
	}
	return response.Pages, nil
}

func (gn *GNClient) getPage(ctx context.Context, mode string, page int, sort string, filters map[string]string) ([]GNProduct, error) {
	params := map[string]string{
		"page": strconv.Itoa(page),
		"sort": sort,
	}
	maps.Copy(params, filters)
	response, err := gn.query(ctx, gn.buildURL(mode, params))
	if err != nil {
		return nil, err
	}
	return response.Products, nil
}

// getSets lists the sets the storefront files a product line under, from the
// same filter endpoint its own set dropdown reads.
func (gn *GNClient) getSets(ctx context.Context) ([]string, error) {
	u, _ := url.Parse(setsURL)
	q := u.Query()
	q.Set("store_id", storeID)
	q.Set("product_line", gn.productLine)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", staticUA)

	resp, err := gn.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Sets []struct {
			Name string `json:"name"`
		} `json:"sets"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(response.Sets))
	for _, set := range response.Sets {
		names = append(names, set.Name)
	}
	sort.Strings(names)
	return names, nil
}
