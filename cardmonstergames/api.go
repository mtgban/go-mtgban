package cardmonstergames

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
	baseURL     = "https://cardmonstergames.com"
	productsURL = baseURL + "/products.json"

	// The widest page Shopify serves, and the deepest it walks one listing.
	productsPerPage = 250
	walkLimit       = 25000
)

type cmgClient struct {
	client *http.Client
}

func newClient() *cmgClient {
	return &cmgClient{client: mtgban.NewHTTPClient(
		mtgban.WithHTTPHeaderTimeout(30*time.Second),
		mtgban.WithHTTPTimeout(time.Minute),
	)}
}

// shopifyProduct is one product of the storefront's products.json.
type shopifyProduct struct {
	ID       int64            `json:"id"`
	Title    string           `json:"title"`
	Handle   string           `json:"handle"`
	Variants []shopifyVariant `json:"variants"`
}

// shopifyVariant is one purchasable copy of a product. The store sets a
// single's SKU to the TCGplayer sku id of the printing, grade and language
// it sells.
type shopifyVariant struct {
	ID        int64  `json:"id"`
	SKU       string `json:"sku"`
	Available bool   `json:"available"`
	Price     string `json:"price"`
}

// products returns one page of every product the storefront lists, the
// first being 1.
func (cc *cmgClient) products(ctx context.Context, page int) ([]shopifyProduct, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(productsPerPage))
	v.Set("page", strconv.Itoa(page))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, productsURL+"?"+v.Encode(), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := cc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	var out struct {
		Products []shopifyProduct `json:"products"`
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return out.Products, nil
}
