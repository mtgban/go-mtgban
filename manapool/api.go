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
	URL                string `json:"url"`
	ProductType        string `json:"product_type"`
	ProductID          string `json:"product_id"`
	SetCode            string `json:"set_code"`
	Number             string `json:"number"`
	Name               string `json:"name"`
	ScryfallID         string `json:"scryfall_id"`
	TcgplayerProductID int    `json:"tcgplayer_product_id"`
	LanguageID         string `json:"language_id"`
	ConditionID        string `json:"condition_id"`
	FinishID           string `json:"finish_id"`
	LowPrice           int    `json:"low_price"`
	AvailableQuantity  int    `json:"available_quantity"`
}

type Product struct {
	URL                string `json:"url"`
	ProductType        string `json:"product_type"`
	ProductID          string `json:"product_id"`
	SetCode            string `json:"set_code"`
	Name               string `json:"name"`
	TcgplayerProductID int    `json:"tcgplayer_product_id"`
	LanguageID         string `json:"language_id"`
	LowPrice           int    `json:"low_price"`
	AvailableQuantity  int    `json:"available_quantity"`
}

const (
	manapoolURL = "https://manapool.com/api/v1/prices/variants"
	sealedURL   = "https://manapool.com/api/v1/prices/sealed"
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

func (mp *Client) GetSealedList() ([]Product, error) {
	req, err := http.NewRequest("GET", sealedURL, nil)
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
		Data []Product `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %s", string(data))
	}

	return pricelist.Data, nil
}
