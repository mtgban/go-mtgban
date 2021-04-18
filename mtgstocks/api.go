package mtgstocks

import (
	"encoding/json"
	"io/ioutil"

	"github.com/hashicorp/go-cleanhttp"
)

type Interest struct {
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

type StocksInterest struct {
	Foil   []Interest `json:"foil"`
	Normal []Interest `json:"normal"`
}

type MTGStocksInterests struct {
	Date    int64           `json:"date"`
	Average *StocksInterest `json:"average"`
	Market  *StocksInterest `json:"market"`
}

const (
	stksAverageURL = "https://api.mtgstocks.com/interests/average"
	stksMarketURL  = "https://api.mtgstocks.com/interests/market"
)

func AverageInterests() (*StocksInterest, error) {
	out, err := query(stksAverageURL)
	if err != nil {
		return nil, err
	}
	return out.Average, nil
}

func MarketInterests() (*StocksInterest, error) {
	out, err := query(stksMarketURL)
	if err != nil {
		return nil, err
	}
	return out.Market, nil
}

func query(link string) (*MTGStocksInterests, error) {
	resp, err := cleanhttp.DefaultClient().Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var interests MTGStocksInterests
	err = json.Unmarshal(data, &interests)
	if err != nil {
		return nil, err
	}

	return &interests, nil
}
