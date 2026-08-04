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

	// Total quantity across listings
	AvailableQuantity int `json:"available_quantity"`
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

	// Total quantity across listings
	AvailableQuantity int `json:"available_quantity"`
}

const (
	manapoolURL = "https://manapool.com/api/v1/prices/variants"
	sealedURL   = "https://manapool.com/api/v1/prices/sealed"
)

func GetPriceList(ctx context.Context) ([]Card, error) {
	return getList[Card](ctx, manapoolURL)
}

func GetSealedList(ctx context.Context) ([]Product, error) {
	return getList[Product](ctx, sealedURL)
}

// getList fetches and decodes one of the price lists. The lists are large
// downloads whose connection occasionally resets mid-stream, so retry the
// whole fetch a few times.
func getList[T any](ctx context.Context, link string) ([]T, error) {
	const attempts = 3
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		var data []T
		data, err = fetchList[T](ctx, link)
		if err == nil {
			return data, nil
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, err
}

func fetchList[T any](ctx context.Context, link string) ([]T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
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
		Data []T `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %w", err)
	}

	return pricelist.Data, nil
}
