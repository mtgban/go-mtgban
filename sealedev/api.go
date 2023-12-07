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

	uuids := mtgmatcher.GetUUIDs()
	for _, uuid := range uuids {
		// Remove outliers from Direct
		var basePrice float64
		basetcg, found := response.Retail[uuid]["TCG Low"]
		if found {
			basePrice = basetcg.Regular + basetcg.Foil + basetcg.Etched
		}

		var directPrice float64
		basedirect, found := response.Buylist[uuid]["TCGDirectNet"]
		if found {
			directPrice = basedirect.Regular + basedirect.Foil + basedirect.Etched
		} else {
			// Use TCG Market or Low if Direct is fully missing
			replacement, found := response.Retail[uuid]["TCG Market"]
			if !found {
				replacement, found = response.Retail[uuid]["TCG Low"]
			}

			if found {
				if response.Buylist[uuid] == nil {
					response.Buylist[uuid] = map[string]*BanPrice{}
				}
				response.Buylist[uuid]["TCGDirectNet"] = replacement
				directPrice = replacement.Regular + replacement.Foil + replacement.Etched
			}
		}

		// Cap maximum price to twice as much tcg low
		if basePrice != 0 && directPrice > 2*basePrice {
			response.Buylist[uuid]["TCGDirectNet"].Regular = response.Retail[uuid]["TCG Low"].Regular * 2
			response.Buylist[uuid]["TCGDirectNet"].Foil = response.Retail[uuid]["TCG Low"].Foil * 2
			response.Buylist[uuid]["TCGDirectNet"].Etched = response.Retail[uuid]["TCG Low"].Etched * 2
			delete(response.Retail[uuid], "TCG Direct")
		}

		// Add bulk pricing
		_, found = response.Buylist[uuid]["CK"]
		if !found {
			co, _ := mtgmatcher.GetUUID(uuid)
			price := bulkBuylist(co)
			if response.Buylist[uuid] == nil {
				response.Buylist[uuid] = map[string]*BanPrice{}
			}
			if response.Buylist[uuid]["CK"] == nil {
				response.Buylist[uuid]["CK"] = &BanPrice{}
			}
			if co.Etched {
				response.Buylist[uuid]["CK"].Etched = price
			} else if co.Foil {
				response.Buylist[uuid]["CK"].Foil = price
			} else {
				response.Buylist[uuid]["CK"].Regular = price
			}
		}
	}

	return &response, nil
}

func bulkBuylist(co *mtgmatcher.CardObject) float64 {
	var price float64
	switch co.Rarity {
	case "mythic":
		price = 0.30
		if co.Foil {
			price = 0.25
		}
	case "rare":
		price = 0.08
		if co.Foil {
			price = 0.15
		}
	case "common", "uncommon":
		price = 5.0 / 1000
		if co.Foil {
			price = 0.02
		}
	default:
		if co.IsPromo {
			price = 0.05
		} else if mtgmatcher.IsBasicLand(co.Name) {
			price = 0.01
			if co.Foil {
				price = 0.10
			}
		}
	}
	return price
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
