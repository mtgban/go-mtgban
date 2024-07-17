package sealedev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type BanPrice struct {
	Regular float64 `json:"regular,omitempty"`
	Foil    float64 `json:"foil,omitempty"`
	Etched  float64 `json:"etched,omitempty"`
}

type BANPriceResponse struct {
	Error string `json:"error,omitempty"`
	Meta  struct {
		Date    time.Time `json:"date"`
		Version string    `json:"version"`
		BaseURL string    `json:"base_url"`
	} `json:"meta"`

	// uuid > store > price {regular/foil/etched}
	Retail  map[string]map[string]*BanPrice `json:"retail,omitempty"`
	Buylist map[string]map[string]*BanPrice `json:"buylist,omitempty"`
}

const (
	BANAPIURL = "https://www.mtgban.com/api/mtgban/all.json?sig="

	BulkThreshold = 0.3
)

func getRetail(response BANPriceResponse, source, uuid string) float64 {
	price, found := response.Retail[uuid][source]
	if !found {
		return 0
	}
	return price.Regular + price.Foil + price.Etched
}

func getBuylist(response BANPriceResponse, source, uuid string) float64 {
	price, found := response.Buylist[uuid][source]
	if !found {
		return 0
	}
	return price.Regular + price.Foil + price.Etched
}

func loadPrices(sig string) (*BANPriceResponse, error) {
	resp, err := cleanhttp.DefaultClient().Get(BANAPIURL + sig)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s\n%s", err.Error(), string(data))
	}

	var response BANPriceResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("%s\n%s", err.Error(), string(data))
	}

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	// Remove outliers from Direct
	uuids := mtgmatcher.GetUUIDs()
	for _, uuid := range uuids {
		tcgMarket := getRetail(response, "TCG Market", uuid)
		directNet := getBuylist(response, "TCGDirectNet", uuid)

		// If TCG Direct (net) is fully missing, try assigning Market and fallback to Low
		if directNet == 0 {
			// If both fallbacks are missing, then just skip the entry entirely
			if response.Retail[uuid]["TCG Market"] == nil && response.Retail[uuid]["TCG Low"] == nil {
				continue
			}
			// Allocate memory
			if response.Buylist[uuid] == nil {
				response.Buylist[uuid] = map[string]*BanPrice{}
			}
			// Assign Market price
			response.Buylist[uuid]["TCGDirectNet"] = response.Retail[uuid]["TCG Market"]

			// If Market is absent, use the Low
			if tcgMarket == 0 {
				tcgMarket = getRetail(response, "TCG Low", uuid)
				response.Buylist[uuid]["TCGDirectNet"] = response.Retail[uuid]["TCG Low"]
			}

			// Refresh the price
			directNet = getBuylist(response, "TCGDirectNet", uuid)
		}

		// If Direct looks unreliable, cap maximum price to twice TCG_LOW (estimate)
		if directNet/2 > tcgMarket {
			response.Buylist[uuid]["TCGDirectNet"].Regular = response.Retail[uuid]["TCG Low"].Regular * 2
			response.Buylist[uuid]["TCGDirectNet"].Foil = response.Retail[uuid]["TCG Low"].Foil * 2
			response.Buylist[uuid]["TCGDirectNet"].Etched = response.Retail[uuid]["TCG Low"].Etched * 2
		}
	}

	// Remove prices that are too low
	for _, uuid := range uuids {
		for _, category := range []map[string]map[string]*BanPrice{response.Retail, response.Buylist} {
			for store, price := range category[uuid] {
				if price.Regular+price.Foil+price.Etched < BulkThreshold {
					delete(category[uuid], store)
				}
			}
		}
	}

	return &response, nil
}

func valueInBooster(uuids []string, prices map[string]map[string]*BanPrice, source string, probabilities []float64) float64 {
	var total float64
	for i, uuid := range uuids {
		priceEntry, found := prices[uuid][source]
		if !found {
			continue
		}

		// Only one of these will be non-zero
		price := priceEntry.Regular + priceEntry.Foil + priceEntry.Etched

		// Adjust price by its probability
		probability := 1.0
		if probabilities != nil {
			probability = probabilities[i]
		}

		// Add to the final value
		total += price * probability
	}
	return total
}
