package mtgstocks

import (
	"encoding/json"
	"fmt"
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
	stksSetsURL    = "https://api.mtgstocks.com/card_sets"
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

type MTGStockPrint struct {
	ID          int         `json:"id"`
	Slug        interface{} `json:"slug"` // string & int
	Foil        bool        `json:"foil"`
	Image       string      `json:"image"`
	Name        string      `json:"name"`
	Rarity      string      `json:"rarity"`
	LatestPrice struct {
		Avg        float64 `json:"avg"`
		Foil       float64 `json:"foil"`
		Market     float64 `json:"market"`
		MarketFoil float64 `json:"market_foil"`
	} `json:"latest_price"`
	LastWeekPrice float64 `json:"last_week_price,string"`
	PreviousPrice float64 `json:"previous_price,string"`
}

type MTGStocksSet struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Abbreviation string `json:"abbreviation"`
	IconClass    string `json:"icon_class"`
	SetType      string `json:"set_type"`
	Date         int64  `json:"date"`

	Ev     bool            `json:"ev,omitempty"`
	EvDesc string          `json:"ev_desc,omitempty"`
	Prints []MTGStockPrint `json:"prints,omitempty"`
}

func GetSets() ([]MTGStocksSet, error) {
	resp, err := cleanhttp.DefaultClient().Get(stksSetsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sets []MTGStocksSet
	err = json.Unmarshal(data, &sets)
	if err != nil {
		return nil, err
	}

	return sets, nil
}

func GetPrints(id int) ([]MTGStockPrint, error) {
	resp, err := cleanhttp.DefaultClient().Get(fmt.Sprintf("%s/%d", stksSetsURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var set MTGStocksSet
	err = json.Unmarshal(data, &set)
	if err != nil {
		return nil, err
	}

	return set.Prints, nil
}
