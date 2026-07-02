package hareruya

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	unisearchURL = "https://www.hareruyamtg.com/en/products/search/unisearch_api?fq.cardset=%s&fq.price=1~*&fq.language=2&fq.stock=1~*&rows=60&page=%d"

	// Sealed products are grouped under the "Product Category > Sealed Product"
	// node, which the site addresses with the category_id 177:505 (parent:child).
	// The stock filter keeps only the items currently in stock.
	sealedURL = "https://www.hareruyamtg.com/en/products/search/unisearch_api?fq.category_id=177:505&fq.price=1~*&fq.stock=1~*&rows=60&page=%d"
)

type Response struct {
	ResponseHeader struct {
		Status int    `json:"status"`
		QTime  string `json:"QTime"`
		ReqID  string `json:"reqID"`
	} `json:"responseHeader"`
	Response struct {
		NumFound int       `json:"numFound"`
		Docs     []Product `json:"docs"`
		Page     int       `json:"page"`
	} `json:"response"`
}

type Product struct {
	Product       string `json:"product"`
	ProductName   string `json:"product_name"`
	ProductNameEN string `json:"product_name_en"`
	CardName      string `json:"card_name"`
	Language      string `json:"language"`
	Price         string `json:"price"`
	ImageURL      string `json:"image_url"`
	FoilFlag      string `json:"foil_flg"`
	Stock         string `json:"stock"` // Stock of the card across printings
	WeeklySales   string `json:"weekly_sales"`
	ProductClass  string `json:"product_class"`
	CardCondition string `json:"card_condition"`
	SaleFlag      string `json:"sale_flg"`
	HighPriceCode string `json:"high_price_code"`
}

func search(ctx context.Context, client *http.Client, link string) ([]Product, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Response.Docs, nil
}

func SearchCardSet(ctx context.Context, client *http.Client, cardSet string, page int) ([]Product, error) {
	return search(ctx, client, fmt.Sprintf(unisearchURL, cardSet, page))
}

func SearchSealed(ctx context.Context, client *http.Client, page int) ([]Product, error) {
	return search(ctx, client, fmt.Sprintf(sealedURL, page))
}
