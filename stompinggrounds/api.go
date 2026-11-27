package stompinggrounds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
)

const (
	singlesURL = "https://singles.stompinggroundstcg.com"

	// The singles storefront's Storefront API, which answers without a token.
	graphqlURL = singlesURL + "/api/2025-10/graphql.json"

	// The most products Shopify walks one search through, and how far into
	// one a walk goes before starting a fresh search where it stopped.
	searchLimit  = 25000
	searchWindow = searchLimit - 5000
)

// sgClient reads the singles storefront.
type sgClient struct {
	client *http.Client
}

func newClient() *sgClient {
	return &sgClient{client: mtgban.NewHTTPClient(
		mtgban.WithHTTPHeaderTimeout(30*time.Second),
		mtgban.WithHTTPTimeout(time.Minute),
	)}
}

// searchQuery is one page of the products a search names, oldest first.
const searchQuery = `query($q: String!, $after: String) {
  products(first: 250, after: $after, query: $q, sortKey: CREATED_AT) {
    pageInfo { hasNextPage endCursor }
    nodes {
      id handle title createdAt
      featuredImage { url }
      variants(first: 100) {
        nodes { id title sku availableForSale price { amount } }
      }
    }
  }
}`

// sfProduct is one product of the Storefront API.
type sfProduct struct {
	ID            string `json:"id"`
	Handle        string `json:"handle"`
	Title         string `json:"title"`
	CreatedAt     string `json:"createdAt"`
	FeaturedImage *struct {
		URL string `json:"url"`
	} `json:"featuredImage"`
	Variants struct {
		Nodes []sfVariant `json:"nodes"`
	} `json:"variants"`
}

type sfVariant struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	SKU              string `json:"sku"`
	AvailableForSale bool   `json:"availableForSale"`
	Price            struct {
		Amount string `json:"amount"`
	} `json:"price"`
}

type sfPage struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []sfProduct `json:"nodes"`
}

// search returns the page of a search after a cursor, the first page for
// an empty one.
func (sc *sgClient) search(ctx context.Context, query, after string) (*sfPage, error) {
	variables := map[string]any{"q": query}
	if after != "" {
		variables["after"] = after
	}
	body, err := json.Marshal(map[string]any{"query": searchQuery, "variables": variables})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var out struct {
		Data struct {
			Products sfPage `json:"products"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, errors.New(out.Errors[0].Message)
	}
	return &out.Data.Products, nil
}
