package starcitygames

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

type SCGEntry struct {
	Price              float64 `json:"price"`
	InventoryLevel     int     `json:"inventory_level"`
	PurchasingDisabled bool    `json:"purchasing_disabled"`
	OptionValues       []struct {
		Label             string `json:"label"`
		OptionDisplayName string `json:"option_display_name"`
	} `json:"option_values"`
}

type SCGResponse struct {
	Response struct {
		Data []SCGEntry `json:"data"`
	} `json:"response"`
}

type SCGCard struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Subtitle  string `json:"subtitle"`
	Condition string `json:"condition"`
	Foil      bool   `json:"foil"`
	Language  string `json:"language"`
	Price     string `json:"price"`
	Rarity    string `json:"rarity"`
	Image     string `json:"image"`
}

type SCGSearch struct {
	Ok      bool        `json:"ok"`
	Msg     string      `json:"msg"`
	Edition string      `json:"search"`
	Results [][]SCGCard `json:"results"`
}

const (
	scgInventoryURL = "https://newstarcityconnector.herokuapp.com/eyApi/products/%s/variants"
	scgBuylistURL   = "https://old.starcitygames.com/buylist/search?search-type=category&id="
)

type SCGClient struct {
	client *retryablehttp.Client
}

func NewSCGClient() *SCGClient {
	scg := SCGClient{}
	scg.client = retryablehttp.NewClient()
	scg.client.Logger = nil
	// The inventory side is sensitive to multiple concurrent requests,
	// This backoff strategy lets the system chill out a bit before retrying
	scg.client.Backoff = retryablehttp.LinearJitterBackoff
	scg.client.RetryWaitMin = 5 * time.Second
	scg.client.RetryWaitMax = 60 * time.Second
	scg.client.RetryMax = 10
	return &scg
}

func (scg *SCGClient) List(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	retryableRequest, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}

	return scg.client.Do(retryableRequest)
}

func (scg *SCGClient) SearchData(dataId string) ([]SCGEntry, error) {
	apiURL := fmt.Sprintf(scgInventoryURL, dataId)
	resp, err := scg.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var variants SCGResponse
	err = json.Unmarshal(data, &variants)
	if err != nil {
		return nil, err
	}

	return variants.Response.Data, nil
}

func (scg *SCGClient) SearchProduct(product string) (*SCGSearch, error) {
	resp, err := scg.client.Get(scgBuylistURL + product)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search SCGSearch
	err = json.Unmarshal(data, &search)
	if err != nil {
		return nil, err
	}
	if !search.Ok {
		return nil, fmt.Errorf("%s", search.Msg)
	}
	if search.Results == nil {
		return nil, fmt.Errorf("product %s not found", product)
	}

	return &search, nil
}
