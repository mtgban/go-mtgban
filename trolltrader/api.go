package trolltrader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
)

const (
	baseURL = "https://www.trolltradercards.com"

	// Every collection of the storefront, and the products of one.
	collectionsURL = baseURL + "/collections.json"
	collectionURL  = baseURL + "/collections/%s/products.json"

	// The widest page Shopify serves, and the deepest it walks one listing.
	productsPerPage = 250
	walkLimit       = 25000
)

// ttClient reads the storefront.
type ttClient struct {
	client *http.Client
}

func newClient() *ttClient {
	// A throttled storefront can hold a request open, which a retry gets
	// past
	return &ttClient{client: mtgban.NewHTTPClient(
		mtgban.WithHTTPHeaderTimeout(30*time.Second),
		mtgban.WithHTTPTimeout(time.Minute),
	)}
}

func (tc *ttClient) getJSON(ctx context.Context, link string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// shopifyCollection is one collection of the storefront: a set's singles,
// or a shelf such as sleeves or new arrivals.
type shopifyCollection struct {
	Handle        string `json:"handle"`
	ProductsCount int    `json:"products_count"`
}

// shopifyProduct is one product of the storefront's products.json.
type shopifyProduct struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	Handle      string           `json:"handle"`
	Vendor      string           `json:"vendor"`
	ProductType string           `json:"product_type"`
	Variants    []shopifyVariant `json:"variants"`
	Options     []shopifyOption  `json:"options"`
}

type shopifyVariant struct {
	ID        int64  `json:"id"`
	SKU       string `json:"sku"`
	Available bool   `json:"available"`
	Price     string `json:"price"`
	Option1   string `json:"option1"`
	Option2   string `json:"option2"`
	Option3   string `json:"option3"`
}

type shopifyOption struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

func pageQuery(page int) string {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(productsPerPage))
	v.Set("page", strconv.Itoa(page))
	return v.Encode()
}

// collections returns one page of the storefront's collections, the first
// being 1.
func (tc *ttClient) collections(ctx context.Context, page int) ([]shopifyCollection, error) {
	var out struct {
		Collections []shopifyCollection `json:"collections"`
	}
	err := tc.getJSON(ctx, collectionsURL+"?"+pageQuery(page), &out)
	if err != nil {
		return nil, err
	}
	return out.Collections, nil
}

// products returns one page of a collection, the first being 1.
func (tc *ttClient) products(ctx context.Context, handle string, page int) ([]shopifyProduct, error) {
	var out struct {
		Products []shopifyProduct `json:"products"`
	}
	link := fmt.Sprintf(collectionURL, url.PathEscape(handle)) + "?" + pageQuery(page)
	err := tc.getJSON(ctx, link, &out)
	if err != nil {
		return nil, err
	}
	return out.Products, nil
}
