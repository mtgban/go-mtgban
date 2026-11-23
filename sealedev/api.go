package sealedev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

// v2Entry is one condition of a store's prices in the BAN API's version 2,
// best condition first; an index price and a sealed product carry none.
type v2Entry struct {
	Condition string  `json:"condition"`
	Price     float64 `json:"price"`
}

// v2Response is what the BAN API's version 2 answers with: card, then
// finish, then store, then that store's prices.
type v2Response struct {
	Error string `json:"error"`

	Retail  map[string]map[string]map[string][]v2Entry `json:"retail"`
	Buylist map[string]map[string]map[string][]v2Entry `json:"buylist"`
}

// priceSnapshot is the BAN price snapshot an EV reads: per side, a card's
// price at each store.
type priceSnapshot struct {
	Retail  map[string]map[string]float64
	Buylist map[string]map[string]float64
}

const (
	banAPIURL = "https://www.mtgban.com/api/v2/all%s.json?sig=%s"

	// tcgItemLowStore is TCGplayer's lowest listing per sku, its item price
	// without the shipping.
	tcgItemLowStore = "TCGItemLow"

	// BulkThreshold is the price under which a card counts as bulk and stops
	// counting toward a product that draws it at random. A product holding it
	// for certain, a deck or a fixed drop, still counts it.
	BulkThreshold = 0.5
	// MaxSinglePrice caps what one card may contribute, so a mispriced
	// outlier cannot carry a whole product
	MaxSinglePrice = 10000.0
)

// readPrice is the price a store's entries give a card: near mint, else
// lightly played, an index price counting as near mint. The API lists
// conditions best first, so the first of these is the one.
func readPrice(entries []v2Entry) float64 {
	for _, e := range entries {
		switch e.Condition {
		case "NM", "SP", "":
			return e.Price
		}
	}
	return 0
}

// cheaperGrade is the cheaper of a store's near mint and lightly played
// prices, how TCGplayer's own product Low reads them.
func cheaperGrade(entries []v2Entry) float64 {
	var price float64
	for _, e := range entries {
		if (e.Condition == "NM" || e.Condition == "SP") && e.Price > 0 && (price == 0 || e.Price < price) {
			price = e.Price
		}
	}
	return price
}

// readSide reads one side of the response, each card's prices from the
// finish it is sold in: TCGItemLow at the cheaper of its two best grades,
// every other store as readPrice reads it.
func readSide(b *matcher.Backend, side map[string]map[string]map[string][]v2Entry) map[string]map[string]float64 {
	out := make(map[string]map[string]float64, len(side))
	for uuid, finishes := range side {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		for store, entries := range finishes[co.Finish] {
			price := readPrice(entries)
			if store == tcgItemLowStore {
				price = cheaperGrade(entries)
			}
			if price == 0 {
				continue
			}
			if out[uuid] == nil {
				out[uuid] = map[string]float64{}
			}
			out[uuid][store] = price
		}
	}
	return out
}

// getPrice is a card's price at one store, a broken one counting as none.
func getPrice(b *matcher.Backend, uuid string, price float64) float64 {
	// Ignore broken prices, except for well known editions
	if price > MaxSinglePrice {
		co, err := b.GetUUID(uuid)
		if err != nil {
			return 0
		}
		switch co.SetCode {
		case "LEA", "LEB", "3ED", "ARN", "LEG":
		default:
			return 0
		}
	}
	return price
}

func (r *priceSnapshot) getRetail(b *matcher.Backend, uuid, source string) float64 {
	return getPrice(b, uuid, r.Retail[uuid][source])
}

func (r *priceSnapshot) getBuylist(b *matcher.Backend, uuid, source string) float64 {
	return getPrice(b, uuid, r.Buylist[uuid][source])
}

func (r *priceSnapshot) setRetail(b *matcher.Backend, uuid, store string, price float64) {
	setPrice(b, r.Retail, uuid, store, price)
}

func (r *priceSnapshot) setBuylist(b *matcher.Backend, uuid, store string, price float64) {
	setPrice(b, r.Buylist, uuid, store, price)
}

// setPrice files a price for a card the datastore knows.
func setPrice(b *matcher.Backend, side map[string]map[string]float64, uuid, store string, price float64) {
	_, err := b.GetUUID(uuid)
	if err != nil {
		return
	}
	if side[uuid] == nil {
		side[uuid] = map[string]float64{}
	}
	side[uuid][store] = price
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

func loadPrices(ctx context.Context, b *matcher.Backend, sig, selected string) (*priceSnapshot, error) {
	link := fmt.Sprintf(banAPIURL, selected, sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := mtgban.NewHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// An authentication or server failure still carries a JSON body, which
	// decodes into an empty response without error and leaves every estimate
	// at zero. Without this the run reports missing data instead of the
	// reason for it.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BAN API returned HTTP %d", resp.StatusCode)
	}

	var raw v2Response
	err = json.NewDecoder(resp.Body).Decode(&raw)
	if err != nil {
		return nil, err
	}

	if raw.Error != "" {
		return nil, errors.New(raw.Error)
	}

	response := priceSnapshot{
		Retail:  readSide(b, raw.Retail),
		Buylist: readSide(b, raw.Buylist),
	}

	// Measured over the whole snapshot before any of it is written back, so
	// that an estimate filed under Cardmarket's own name never becomes what
	// the next card is measured against. Only the writing folds into the
	// pass below.
	mkm := fitMKMCalibration(b, &response)

	// Adjust Direct/CT0 estimates and drop what they leave at nothing in a
	// single pass over the catalog. Bulk stays; simulate applies the floor.
	for _, uuid := range b.GetUUIDs() {
		response.adjust(b, mkm, uuid)
	}

	return &response, nil
}

// adjust settles one card's prices after the fetch: the estimates and the
// fees taken off, then the drop of whatever they leave at nothing.
func (r *priceSnapshot) adjust(b *matcher.Backend, mkm *mkmCalibration, uuid string) {
	// Price what Cardmarket never polled.
	mkm.fill(b, r, uuid)

	tcgLow := r.getRetail(b, uuid, "TCGLow")
	tcgMarket := r.getRetail(b, uuid, "TCGMarket")
	directNet := r.getBuylist(b, uuid, "TCGDirectNet")

	if directNet == 0 {
		// TCG Direct (net) is missing: estimate it from Market, falling back
		// to Low. Skip entirely if neither is available.
		if tcgMarket != 0 || tcgLow != 0 {
			// Use Market as base estimate, or Low as fallback
			directNet = tcgMarket
			if directNet == 0 {
				directNet = tcgLow
			}

			// Adjust estimate for fees
			directNet = tcgplayer.DirectPriceAfterFees(directNet)

			r.setBuylist(b, uuid, "TCGDirectNet", directNet)
		}
	} else if directNet/2 > tcgMarket {
		// Direct exists but looks unreliable: cap it at twice Low, or drop it.
		// (else-if: an estimate from the branch above must not be re-judged here)
		if tcgLow == 0 || tcgLow*2 > directNet*0.9 {
			delete(r.Buylist[uuid], "TCGDirectNet")
		} else {
			directNet = tcgplayer.DirectPriceAfterFees(tcgLow * 2)
			r.setBuylist(b, uuid, "TCGDirectNet", directNet)
		}
	}

	// TCG Low EV reads the lower of the product Low and the item low.
	low := r.getRetail(b, uuid, tcgItemLowStore)
	if low == 0 || (tcgLow != 0 && tcgLow < low) {
		low = tcgLow
	}
	if low != 0 {
		r.setRetail(b, uuid, tcgItemLowStore, low)
	}

	// Create a custom price
	direct := r.getRetail(b, uuid, "TCGDirect")
	directSyp := tcgplayer.DirectSYPPriceAfterFees(direct)
	r.setBuylist(b, uuid, "TCGDirectSYPNet", directSyp)

	// CardTrader Zero: subtract its flat fee.
	ct0 := r.getRetail(b, uuid, "CT0")
	if ct0 > 0 {
		r.setRetail(b, uuid, "CT0", ct0-getCT0fees(ct0))
	}

	// Drop the prices left at nothing, and the broken ones.
	for _, category := range []map[string]map[string]float64{r.Retail, r.Buylist} {
		for store := range category[uuid] {
			if getPrice(b, uuid, category[uuid][store]) <= 0 {
				delete(category[uuid], store)
			}
		}
	}
}

// maxStorePrice returns the highest available price for a card across the given
// source stores (0 if none are present).
func maxStorePrice(b *matcher.Backend, uuid string, prices map[string]map[string]float64, stores []string) float64 {
	var price float64
	for _, source := range stores {
		sourcePrice := getPrice(b, uuid, prices[uuid][source])
		if sourcePrice > price {
			price = sourcePrice
		}
	}
	return price
}
