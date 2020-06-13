package cklite

import (
	"encoding/json"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

type CKPriceList struct {
	Meta struct {
		CreatedAt string `json:"created_at"`
		BaseURL   string `json:"base_url"`
	} `json:"meta"`
	Data []CKCard `json:"data"`
}

type CKCard struct {
	Id           int    `json:"id"`
	SKU          string `json:"sku"`
	URL          string `json:"url"`
	Name         string `json:"name"`
	Variation    string `json:"variation"`
	Edition      string `json:"edition"`
	IsFoil       string `json:"is_foil"`
	SellPrice    string `json:"price_retail"`
	SellQuantity int    `json:"qty_retail"`
	BuyPrice     string `json:"price_buy"`
	BuyQuantity  int    `json:"qty_buying"`
}

const (
	ckPricelistURL = "https://api.cardkingdom.com/api/pricelist"
)

func GetPriceList() (*CKPriceList, error) {
	resp, err := cleanhttp.DefaultClient().Get(ckPricelistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	var pricelist CKPriceList
	err = dec.Decode(&pricelist)
	if err != nil {
		return nil, err
	}

	return &pricelist, nil
}
