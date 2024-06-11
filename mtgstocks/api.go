package mtgstocks

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

type StocksInterest struct {
	InterestType string  `json:"interest_type"`
	Foil         bool    `json:"foil"`
	Percentage   float64 `json:"percentage"`
	PastPrice    float64 `json:"past_price"`
	PresentPrice float64 `json:"present_price"`
	Date         int64   `json:"date"`
	Print        struct {
		Id        int         `json:"id"`
		Slug      interface{} `json:"slug"` // string & int
		Name      string      `json:"name"`
		Rarity    string      `json:"rarity"`
		SetId     int         `json:"set_id"`
		SetName   string      `json:"set_name"`
		IconClass string      `json:"icon_class"`
		Reserved  bool        `json:"reserved"`
		SetType   string      `json:"set_type"`
		Legal     struct {
			Frontier  string `json:"frontier"`
			Pauper    string `json:"pauper"`
			Pioneer   string `json:"pioneer"`
			Modern    string `json:"modern"`
			Standard  string `json:"standard"`
			Commander string `json:"commander"`
			Vintage   string `json:"vintage"`
			Legacy    string `json:"legacy"`
		}
		IncludeDefault bool   `json:"include_default"`
		Image          string `json:"image"`
	} `json:"print"`
}

type MTGStocksInterests struct {
	Error     string           `json:"error"`
	Date      string           `json:"date"`
	Interests []StocksInterest `json:"interests"`
}

const (
	stksAverageURL = "https://api.mtgstocks.com/interests/average"
	stksMarketURL  = "https://api.mtgstocks.com/interests/market"
	stksSetsURL    = "https://api.mtgstocks.com/card_sets"

	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Windows 1.0; rv:100.0)"
)

func AverageInterests(foil bool) ([]StocksInterest, error) {
	out, err := query(stksAverageURL, foil)
	if err != nil {
		return nil, err
	}
	return out.Interests, nil
}

func MarketInterests(foil bool) ([]StocksInterest, error) {
	out, err := query(stksMarketURL, foil)
	if err != nil {
		return nil, err
	}
	return out.Interests, nil
}

func query(link string, foil bool) (*MTGStocksInterests, error) {
	extra := "/regular"
	if foil {
		extra = "/foil"
	}
	link += extra

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var interests MTGStocksInterests
	err = json.Unmarshal(data, &interests)
	if err != nil {
		return nil, err
	}

	if interests.Error != "" {
		return nil, errors.New(interests.Error)
	}

	return &interests, nil
}
