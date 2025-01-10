package manapool

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type Card struct {
	URL                  string `json:"url"`
	Name                 string `json:"name"`
	SetCode              string `json:"set_code"`
	Number               string `json:"number"`
	MultiverseID         string `json:"multiverse_id"`
	ScryfallID           string `json:"scryfall_id"`
	AvailableQuantity    int    `json:"available_quantity"`
	PriceCents           int    `json:"price_cents"`
	PriceCentsFoil       int    `json:"price_cents_foil"`
	PriceCentsLpPlus     int    `json:"price_cents_lp_plus"`
	PriceCentsLpPlusFoil int    `json:"price_cents_lp_plus_foil"`
	PriceCentsNm         int    `json:"price_cents_nm"`
	PriceCentsNmFoil     int    `json:"price_cents_nm_foil"`
}

const (
	manapoolURL = "https://manapool.com/api/beta/pricelists/cards.json"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	mp := Client{}
	mp.client = cleanhttp.DefaultClient()
	return &mp
}

func (mp *Client) GetPriceList() ([]Card, error) {
	req, err := http.NewRequest("GET", manapoolURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := mp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Meta struct {
			AsOf time.Time `json:"as_of"`
		} `json:"meta"`
		Data []Card `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %s", string(data))
	}

	return pricelist.Data, nil
}
