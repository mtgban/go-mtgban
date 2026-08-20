package manapool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

// Product is a single priced item from a manapool price list. It covers every
// endpoint, and a field the producing feed does not quote is left zero: sealed
// products have no Number, ScryfallID, ConditionID or FinishID, and only the
// singles feed carries the market prices. ProductType distinguishes the feeds
// (e.g. "mtg_sealed").
type Product struct {
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

	// What manapool reckons a card is worth, quoted in cents like LowPrice
	// and only by the singles feed. That feed aggregates a card's listings
	// rather than describing one of them, so it names the finish by which
	// field the price sits in instead of through ConditionID and FinishID,
	// and reports a null that decodes to zero for a finish it does not sell.
	PriceMarket     int `json:"price_market"`
	PriceMarketFoil int `json:"price_market_foil"`
}

const (
	manapoolURL = "https://manapool.com/api/v1/prices/variants"
	sealedURL   = "https://manapool.com/api/v1/prices/sealed"
	singlesURL  = "https://manapool.com/api/v1/prices/singles"
)

// GetPriceList downloads the singles price list in one call.
func GetPriceList(ctx context.Context) ([]Product, error) {
	return getList(ctx, manapoolURL)
}

// GetSealedList downloads the sealed price list in one call.
func GetSealedList(ctx context.Context) ([]Product, error) {
	return getList(ctx, sealedURL)
}

// GetSinglesList downloads the singles price list in one call.
func GetSinglesList(ctx context.Context) ([]Product, error) {
	return getList(ctx, singlesURL)
}

func getList(ctx context.Context, link string) ([]Product, error) {
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
		Data []Product `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&pricelist)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for list, got: %w", err)
	}

	return pricelist.Data, nil
}
