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
)

const (
	baseURL  = "https://buylist.vegas.singles/saas/search"
	storeID  = "d4lDsS3ZNf"
	staticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0"
)

type VSResponse struct {
	Count    int         `json:"count"`
	Pages    int         `json:"pages"`
	Products []VSProduct `json:"products"`
}

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

type VSProductData struct {
	Set                       string `json:"set"`
	SetName                   string `json:"setName"`
	Rarity                    string `json:"rarity"`
	CollectorNumberNormalized int    `json:"collector_number_normalized"`
}

type VSVariant struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	SelectedFinish string  `json:"selected_finish"`
	OfferPrice     float64 `json:"offer_price"`
}

type VSRetailVariant struct {
	ID                int64   `json:"id"`
	Title             string  `json:"title"`
	Price             float64 `json:"price"`
	SKU               string  `json:"sku"`
	InventoryQuantity int     `json:"inventory_quantity"`
}

type VSClient struct {
	client *http.Client
}

func NewVSClient() *VSClient {
	vs := VSClient{}
	client := retryablehttp.NewClient()
	client.Logger = nil
	vs.client = client.StandardClient()
	return &vs
}

func (vs *VSClient) buildURL(params map[string]string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()

	// Required parameters
	q.Set("store_id", storeID)
	q.Set("product_line", "Magic: the Gathering")
	q.Set("mongo", "true")
	q.Set("buylist_products", "true")
	q.Set("ignore_is_hot_order", "true")
	q.Set("sort", "Relevance")

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

func (vs *VSClient) getCount(ctx context.Context) (int, error) {
	reqURL := vs.buildURL(map[string]string{
		"with_count": "true",
		"no_track":   "true",
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

func (vs *VSClient) getPage(ctx context.Context, page int) ([]VSProduct, error) {
	reqURL := vs.buildURL(map[string]string{
		"page": strconv.Itoa(page),
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
