package miniaturemarket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	http "github.com/hashicorp/go-retryablehttp"
)

type MMBuyBack struct {
	Id           string `json:"id"`
	Image        string `json:"image"`
	FullImage    string `json:"full_image"`
	Name         string `json:"name"`
	BuybackName  string `json:"buyback_name"`
	SKU          string `json:"sku"`
	Price        string `json:"price"`
	Note         string `json:"note"`
	IsFoil       bool   `json:"foil"`
	MtgSet       string `json:"mtg_set"`
	MtgRarity    string `json:"mtg_rarity"`
	MtgCondition string `json:"mtg_condition"`
}

type MMClient struct {
	client *http.Client
}

const (
	mmBuyBackURL       = "https://www.miniaturemarket.com/buyback/data/products/"
	mmBuyBackSearchURL = "https://www.miniaturemarket.com/buyback/data/productsearch/"

	MMCategoryMtgSingles = "1466"
)

func NewMMClient() *MMClient {
	mm := MMClient{}
	mm.client = http.NewClient()
	mm.client.Logger = nil
	return &mm
}

func (mm *MMClient) BuyBackPage(category string, page int) ([]MMBuyBack, error) {
	resp, err := mm.client.PostForm(mmBuyBackURL, url.Values{
		"category": {category},
		"page":     {fmt.Sprintf("%d", page)},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var buyback []MMBuyBack
	err = json.Unmarshal(data, &buyback)
	if err != nil {
		return nil, err
	}

	return buyback, nil
}

func (mm *MMClient) BuyBackSearch(search string) ([]MMBuyBack, error) {
	resp, err := http.PostForm(mmBuyBackSearchURL, url.Values{
		"search": {search},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var buyback []MMBuyBack
	err = json.Unmarshal(data, &buyback)
	if err != nil {
		return nil, err
	}

	return buyback, nil
}
