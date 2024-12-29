package sealedev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

type BanPrice struct {
	Conditions map[string]float64 `json:"conditions,omitempty"`
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
	banAPIURL = "https://www.mtgban.com/api/mtgban/all%s.json?tag=tags&conds=true&sig=%s"

	BulkThreshold = 0.5
)

func getPrice(price *BanPrice, uuid string) float64 {
	if price == nil {
		return 0
	}

	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		return 0
	}

	var tag string
	if co.Etched {
		tag = "_etched"
	} else if co.Foil {
		tag = "_foil"
	}

	result := price.Conditions["NM"+tag]
	if result == 0 {
		result = price.Conditions["SP"+tag]
	}

	return result
}

func getRetail(response BANPriceResponse, source, uuid string) float64 {
	return getPrice(response.Retail[uuid][source], uuid)
}

func getBuylist(response BANPriceResponse, source, uuid string) float64 {
	return getPrice(response.Buylist[uuid][source], uuid)
}

func setBuylist(response BANPriceResponse, destination, uuid string, price float64) {
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		return
	}

	var tag string
	if co.Etched {
		tag = "_etched"
	} else if co.Foil {
		tag = "_foil"
	}

	// Rebuild the price entry
	response.Buylist[uuid][destination] = &BanPrice{
		Conditions: map[string]float64{
			"NM" + tag: price,
		},
	}
}

func loadPrices(sig, selected string) (*BANPriceResponse, error) {
	link := fmt.Sprintf(banAPIURL, selected, sig)
	resp, err := cleanhttp.DefaultClient().Get(link)
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
		tcgLow := getRetail(response, "TCGLow", uuid)
		tcgMarket := getRetail(response, "TCGMarket", uuid)
		directNet := getBuylist(response, "TCGDirectNet", uuid)

		// If TCG Direct (net) is fully missing, try assigning Market and fallback to Low
		if directNet == 0 {
			// If both fallbacks are missing, then just skip the entry entirely
			if tcgMarket == 0 && tcgLow == 0 {
				continue
			}

			// Allocate memory
			if response.Buylist[uuid] == nil {
				response.Buylist[uuid] = map[string]*BanPrice{}
			}

			// Use Market as base estimate, or Low as fallback
			directNet = tcgMarket
			if directNet == 0 {
				directNet = tcgLow
			}

			// Adjust estimate for fees
			directNet = tcgplayer.DirectPriceAfterFees(directNet)

			// Set the price
			setBuylist(response, "TCGDirectNet", uuid, directNet)
		}

		// If Direct looks unreliable, cap maximum price (estimate) or delete it
		if directNet/2 > tcgMarket {
			// If no low or twice as tcglow is within 10% of net, then delete this entry
			if tcgLow == 0 || tcgLow*2 > directNet*0.9 {
				delete(response.Buylist[uuid], "TCGDirectNet")
			} else {
				directNet = tcgLow * 2
				directNet = tcgplayer.DirectPriceAfterFees(directNet)

				setBuylist(response, "TCGDirectNet", uuid, directNet)
			}
		}
	}

	// Remove prices that are too low
	for _, uuid := range uuids {
		for _, category := range []map[string]map[string]*BanPrice{response.Retail, response.Buylist} {
			for store := range category[uuid] {
				if getPrice(category[uuid][store], uuid) < BulkThreshold {
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
		// Adjust price by its probability
		probability := 1.0
		if probabilities != nil {
			probability = probabilities[i]
		}

		// Add to the final value
		total += getPrice(prices[uuid][source], uuid) * probability
	}
	return total
}
