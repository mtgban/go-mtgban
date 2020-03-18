package cardkingdom

import (
	"encoding/json"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
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

type CKClient struct {
	client *http.Client
}

func NewCKClient() *CKClient {
	ck := CKClient{}
	ck.client = http.NewClient()
	ck.client.Logger = nil
	return &ck
}

func (ck *CKClient) GetPriceList() (*CKPriceList, error) {
	resp, err := ck.client.Get(ckPricelistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist CKPriceList
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return &pricelist, nil
}
