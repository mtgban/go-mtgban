package manapool

import (
	"context"
	"encoding/json"
	"fmt"
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

func GetPriceList(ctx context.Context) ([]Card, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manapoolURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pricelist struct {
		Meta struct {
			AsOf time.Time `json:"as_of"`
		} `json:"meta"`
		Data []Card `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %w", err)
	}

	return pricelist.Data, nil
}

func GetSealedList(ctx context.Context) ([]Product, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sealedURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pricelist struct {
		Meta struct {
			AsOf time.Time `json:"as_of"`
		} `json:"meta"`
		Data []Product `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %w", err)
	}

	return pricelist.Data, nil
}
