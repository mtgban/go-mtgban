package vegassingles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mtgban/go-mtgban/internal/jsonflex"
)

const (
	baseURL  = "https://buylist.vegas.singles/saas/search"
	storeID  = "d4lDsS3ZNf"
	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"

	// The two stable orderings one crawl pass runs under. Relevance reshuffles
	// while the pages are being fetched, which loses whatever moves between
	// two of them, so the crawl only ever asks for the alphabet.
	sortForward  = "Alphabetical: A-Z"
	sortBackward = "Alphabetical: Z-A"

	// A bound on how deep any one crawl walks, far past the ~417 pages the
	// storefront serves today before its result window runs out, purely so
	// a feed that never answers empty keeps the crawl finite.
	maxPages = 5000
)

// VSResponse is what the storefront's product endpoint answers with.
type VSResponse struct {
	Count    int         `json:"count"`
	Pages    int         `json:"pages"`
	Products []VSProduct `json:"products"`
}

// VSProduct is one catalog entry.
type VSProduct struct {
	ID                string            `json:"id"`
	ProductID         int64             `json:"product_id"`
	DisplayName       string            `json:"display_name"`
	Price             float64           `json:"price"`
	OfferPrice        float64           `json:"offer_price"`
	SelectedFinish    string            `json:"selectedFinish"`
	ProductData       VSProductData     `json:"product_data"`
	VariantInfo       []VSVariant       `json:"store_pass_variant_info"`
	RetailVariantInfo []VSRetailVariant `json:"variant_info"`
}

// VSProductData is the body of a product, apart from the envelope it arrives
// in.
type VSProductData struct {
	Set                       jsonflex.String `json:"set"`
	SetName                   string          `json:"setName"`
	Rarity                    string          `json:"rarity"`
	CollectorNumberNormalized int             `json:"collector_number_normalized"`
}

// VSVariant is one sellable version of a product, a printing in a condition.
type VSVariant struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	SelectedFinish string  `json:"selected_finish"`
	OfferPrice     float64 `json:"offer_price"`
}

// VSRetailVariant is a variant with the retail price attached.
type VSRetailVariant struct {
	ID                int64   `json:"id"`
	Title             string  `json:"title"`
	Price             float64 `json:"price"`
	SKU               string  `json:"sku"`
	InventoryQuantity int     `json:"inventory_quantity"`
}

// VSClient reads the Vegas Singles storefront, one product line at a time.
type VSClient struct {
	client      *http.Client
	productLine string
	// The storefront's search endpoint, held rather than referenced so a
	// test can point the client at a server it controls.
	baseURL string
}

// NewVSClient returns a client for one product line, in the storefront's own
// spelling (the Game constants).
func NewVSClient(productLine string) *VSClient {
	vs := VSClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	vs.client = client.StandardClient()
	vs.productLine = productLine
	vs.baseURL = baseURL
	return &vs
}

func (vs *VSClient) buildURL(params map[string]string) string {
	u, _ := url.Parse(vs.baseURL)
	q := u.Query()

	// Required parameters
	q.Set("store_id", storeID)
	q.Set("product_line", vs.productLine)
	q.Set("mongo", "true")
	q.Set("buylist_products", "true")
	q.Set("ignore_is_hot_order", "true")
	q.Set("sort", sortForward)

	// Ask only for what the store holds. Nearly three products in four the
	// endpoint answers with are stocked in no condition at all, and none of
	// them is priced either way: the retail side skips a condition the
	// store has none of, and a bid on a card it does not carry is not a
	// bid it will honour. Asking the endpoint is what keeps the crawl off
	// those pages rather than walking them to drop them.
	q.Set("in_stock", "true")

	// Empty filter parameters (required by API)
	for _, param := range []string{
		"set_name", "rarity", "import_list_text", "name", "is_hot",
		"type_line", "color", "finish", "players", "playtime",
		"min_year", "max_year", "publisher", "vendor", "designer",
		"mechanic", "category", "tags",
	} {
		q.Set(param, "")
	}

	for k, v := range params {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (vs *VSClient) getCount(ctx context.Context, rarity string) (int, error) {
	reqURL := vs.buildURL(map[string]string{
		"with_count": "true",
		"no_track":   "true",
		"rarity":     rarity,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", staticUA)

	resp, err := vs.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response VSResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return 0, err
	}

	return response.Pages, nil
}

func (vs *VSClient) getPage(ctx context.Context, page int, sort, rarity string) ([]VSProduct, error) {
	reqURL := vs.buildURL(map[string]string{
		"page":   strconv.Itoa(page),
		"sort":   sort,
		"rarity": rarity,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", staticUA)

	resp, err := vs.client.Do(req)
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

	var response VSResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return response.Products, nil
}
