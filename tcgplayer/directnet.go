package tcgplayer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgban"
)

type TCGDirectNet struct {
	buylistDate time.Time
	buylist     mtgban.BuylistRecord
	signature   string
}

func NewTCGDirectNet(signature string) *TCGDirectNet {
	tcg := TCGDirectNet{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.signature = signature
	return &tcg
}

func (tcg *TCGDirectNet) Buylist() (mtgban.BuylistRecord, error) {
	response, err := loadPrices(tcg.signature)
	if err != nil {
		return nil, err
	}

	for cardId, stores := range response.Retail {
		direct, found := stores["TCG Direct"]
		if !found {
			continue
		}

		for cond, price := range direct.Conditions {
			cond = strings.Split(cond, "_")[0]

			price = DirectPriceAfterFees(price)
			if price <= 0 {
				continue
			}

			buylistEntry := mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   price,
				URL:        response.Meta.BaseURL + "r/TCG Direct/" + cardId,
			}

			tcg.buylist.Add(cardId, &buylistEntry)
		}
	}

	tcg.buylistDate = response.Meta.Date

	return tcg.buylist, nil
}

func (tcg *TCGDirectNet) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Direct (net)"
	info.Shorthand = "TCGDirectNet"
	info.BuylistTimestamp = &tcg.buylistDate
	info.MetadataOnly = true
	return
}

type BanPrice struct {
	Conditions map[string]float64 `json:"conditions"`
}

type BANPriceResponse struct {
	Error string `json:"error,omitempty"`
	Meta  struct {
		Date    time.Time `json:"date"`
		Version string    `json:"version"`
		BaseURL string    `json:"base_url"`
	} `json:"meta"`

	// uuid > store > price {regular/foil/etched}
	Retail map[string]map[string]BanPrice `json:"retail,omitempty"`
}

const (
	BANAPIURL = "https://www.mtgban.com/api/mtgban/all.json?cond=true&vendor=TCG+Direct&sig="
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

	return &response, nil
}
