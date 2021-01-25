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

type MMProduct struct {
	UUID     string `json:"uniqueId"`
	CardName string `json:"mtg_cardname"`
	Edition  string `json:"mtg_set"`
	Title    string `json:"title"`
	URL      string `json:"productUrl"`
	Variants []struct {
		Title    string  `json:"vTitle"`
		Price    float64 `json:"vPrice"`
		Quantity int     `json:"vQty"`
	} `json:"variants"`
}

type MMSearchResponse struct {
	Response struct {
		NumberOfProducts int         `json:"numberOfProducts"`
		Products         []MMProduct `json:"products"`
	}
}

type MMClient struct {
	client *http.Client
}

const (
	mmBuyBackURL       = "https://www.miniaturemarket.com/buyback/data/products/"
	mmBuyBackSearchURL = "https://www.miniaturemarket.com/buyback/data/productsearch/"

	MMCategoryMtgSingles    = "1466"
	MMDefaultResultsPerPage = 32

	mmSearchURL = "https://search.unbxd.io/fb500edbf5c28edfa74cc90561fe33c3/prod-miniaturemarket-com811741582229555/category"
)

func NewMMClient() *MMClient {
	mm := MMClient{}
	mm.client = http.NewClient()
	mm.client.Logger = nil
	return &mm
}

func (mm *MMClient) NumberOfProducts() (int, error) {
	resp, err := mm.query(0, 0)
	if err != nil {
		return 0, err
	}
	return resp.Response.NumberOfProducts, nil
}

func (mm *MMClient) GetInventory(start int) (*MMSearchResponse, error) {
	return mm.query(start, MMDefaultResultsPerPage)
}

func (mm *MMClient) query(start, maxResults int) (*MMSearchResponse, error) {
	u, err := url.Parse(mmSearchURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("format", "json")
	q.Set("version", "V2")
	q.Set("start", fmt.Sprint(start))
	q.Set("rows", fmt.Sprint(maxResults))
	q.Set("variants", "true")
	q.Set("variants.count", "10")
	q.Set("fields", "*")
	q.Set("facet.multiselect", "true")
	q.Set("selectedfacet", "true")
	q.Set("pagetype", "boolean")
	q.Set("p", `categoryPath:"Card Games>Magic the Gathering>MTG Singles"`)
	q.Set("filter", `categoryPath1_fq:"Card Games"`)
	q.Set("filter", `categoryPath2_fq:"Card Games>Magic the Gathering"`)
	q.Set("filter", `categoryPath3_fq:"Card Games>Magic the Gathering>MTG Singles"`)
	q.Set("filter", `categoryPath4_fq:"Card Games>Magic the Gathering>MTG Singles>All Sets"`)
	u.RawQuery = q.Encode()

	resp, err := mm.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search MMSearchResponse
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
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
