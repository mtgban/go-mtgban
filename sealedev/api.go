package sealedev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-cleanhttp"
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
)

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
	for uuid := range response.Buylist {
		var basePrice float64
		basetcg, found := response.Retail[uuid]["TCG Low"]
		if found {
			basePrice = basetcg.Regular + basetcg.Foil + basetcg.Etched
		}

		var directPrice float64
		basedirect, found := response.Buylist[uuid]["TCGDirectNet"]
		if found {
			directPrice = basedirect.Regular + basedirect.Foil + basedirect.Etched
		}

		if basePrice != 0 && directPrice > 100*basePrice {
			delete(response.Buylist[uuid], "TCGDirectNet")
			delete(response.Retail[uuid], "TCG Direct")
		}
	}

	return &response, nil
}

func valueInBooster(uuids []string, prices map[string]map[string]*BanPrice, source string) float64 {
	var total float64
	for _, uuid := range uuids {
		price, found := prices[uuid][source]
		if !found {
			continue
		}
		// Only one of these will be non-zero
		total += price.Regular + price.Foil + price.Etched
	}
	return total
}
