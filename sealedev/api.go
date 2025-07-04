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

	BulkThreshold  = 0.5
	MaxSinglePrice = 10000.0
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

	// Ignore broken prices, except for well known editions
	if result > MaxSinglePrice {
		switch co.SetCode {
		case "LEA", "LEB", "3ED", "ARN", "LEG":
		default:
			result = 0
		}
	}

	return result
}

func (r *BANPriceResponse) getRetail(source, uuid string) float64 {
	return getPrice(r.Retail[uuid][source], uuid)
}

func (r *BANPriceResponse) getBuylist(source, uuid string) float64 {
	return getPrice(r.Buylist[uuid][source], uuid)
}

func (r *BANPriceResponse) setRetail(destination, uuid string, price float64) {
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
	r.Buylist[uuid][destination] = &BanPrice{
		Conditions: map[string]float64{
			"NM" + tag: price,
		},
	}
}

func (r *BANPriceResponse) setBuylist(destination, uuid string, price float64) {
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
	r.Buylist[uuid][destination] = &BanPrice{
		Conditions: map[string]float64{
			"NM" + tag: price,
		},
	}
}

func getCT0fees(price float64) float64 {
	if price <= 0.25 {
		return 0.9
	} else if price <= 3 {
		return 0.10
	} else if price <= 5 {
		return 0.11
	} else if price <= 7 {
		return 0.14
	} else if price <= 10 {
		return 0.15
	} else if price <= 15 {
		return 0.21
	} else if price <= 20 {
		return 0.27
	} else if price <= 30 {
		return 0.40
	} else if price <= 40 {
		return 0.52
	}
	return 0.64
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
		tcgLow := response.getRetail("TCGLow", uuid)
		tcgMarket := response.getRetail("TCGMarket", uuid)
		directNet := response.getBuylist("TCGDirectNet", uuid)

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
			response.setBuylist("TCGDirectNet", uuid, directNet)
		}

		// If Direct looks unreliable, cap maximum price (estimate) or delete it
		if directNet/2 > tcgMarket {
			// If no low or twice as tcglow is within 10% of net, then delete this entry
			if tcgLow == 0 || tcgLow*2 > directNet*0.9 {
				delete(response.Buylist[uuid], "TCGDirectNet")
			} else {
				directNet = tcgLow * 2
				directNet = tcgplayer.DirectPriceAfterFees(directNet)

				response.setBuylist("TCGDirectNet", uuid, directNet)
			}
		}

		ct0 := response.getRetail("CT0", uuid)
		ct0 -= getCT0fees(ct0)
		if ct0 > 0 {
			response.setRetail("CT0", uuid, ct0)
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
