package sealedev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	// No omitempty: always emit the keys so decoders never see a nil map.
	Retail  map[string]map[string]*BanPrice `json:"retail"`
	Buylist map[string]map[string]*BanPrice `json:"buylist"`
}

const (
	banAPIURL = "https://www.mtgban.com/api/mtgban/all%s.json?tag=tags&conds=true&sig=%s"

	BulkThreshold  = 0.5
	MaxSinglePrice = 10000.0
)

func getPrice(uuid string, price *BanPrice) float64 {
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

func (r *BANPriceResponse) getRetail(uuid, source string) float64 {
	return getPrice(uuid, r.Retail[uuid][source])
}

func (r *BANPriceResponse) getBuylist(uuid, source string) float64 {
	return getPrice(uuid, r.Buylist[uuid][source])
}

func (r *BANPriceResponse) setRetail(uuid, store string, price float64) {
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
	r.Retail[uuid][store] = &BanPrice{
		Conditions: map[string]float64{
			"NM" + tag: price,
		},
	}
}

func (r *BANPriceResponse) setBuylist(uuid, store string, price float64) {
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
	r.Buylist[uuid][store] = &BanPrice{
		Conditions: map[string]float64{
			"NM" + tag: price,
		},
	}
}

func getCT0fees(price float64) float64 {
	if price <= 0.25 {
		return 0.09
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

func loadPrices(ctx context.Context, sig, selected string) (*BANPriceResponse, error) {
	link := fmt.Sprintf(banAPIURL, selected, sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response BANPriceResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	// Adjust Direct/CT0 estimates and prune bulk in a single pass over the catalog.
	uuids := mtgmatcher.GetUUIDs()
	for _, uuid := range uuids {
		tcgLow := response.getRetail(uuid, "TCGLow")
		tcgMarket := response.getRetail(uuid, "TCGMarket")
		directNet := response.getBuylist(uuid, "TCGDirectNet")

		if directNet == 0 {
			// TCG Direct (net) is missing: estimate it from Market, falling back
			// to Low. Skip entirely if neither is available.
			if tcgMarket != 0 || tcgLow != 0 {
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

				response.setBuylist(uuid, "TCGDirectNet", directNet)
			}
		} else if directNet/2 > tcgMarket {
			// Direct exists but looks unreliable: cap it at twice Low, or drop it.
			// (else-if: an estimate from the branch above must not be re-judged here)
			if tcgLow == 0 || tcgLow*2 > directNet*0.9 {
				delete(response.Buylist[uuid], "TCGDirectNet")
			} else {
				directNet = tcgplayer.DirectPriceAfterFees(tcgLow * 2)
				response.setBuylist(uuid, "TCGDirectNet", directNet)
			}
		}

		// CardTrader Zero: subtract its flat fee.
		ct0 := response.getRetail(uuid, "CT0")
		ct0 -= getCT0fees(ct0)
		if ct0 > 0 {
			response.setRetail(uuid, "CT0", ct0)
		}

		// Prune prices too low to matter, after the adjustments above.
		for _, category := range []map[string]map[string]*BanPrice{response.Retail, response.Buylist} {
			for store := range category[uuid] {
				if getPrice(uuid, category[uuid][store]) < BulkThreshold {
					delete(category[uuid], store)
				}
			}
		}
	}

	return &response, nil
}

// maxStorePrice returns the highest available price for a card across the given
// source stores (0 if none are present).
func maxStorePrice(uuid string, prices map[string]map[string]*BanPrice, stores []string) float64 {
	var price float64
	for _, source := range stores {
		sourcePrice := getPrice(uuid, prices[uuid][source])
		if sourcePrice > price {
			price = sourcePrice
		}
	}
	return price
}
