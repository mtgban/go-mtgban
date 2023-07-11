package mtgstocks

import (
	"encoding/json"
	"errors"
	"fmt"
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
	req, err := http.NewRequest("GET", stksSetsURL, nil)
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

	var sets []MTGStocksSet
	err = json.Unmarshal(data, &sets)
	if err != nil {
		return nil, err
	}

	return sets, nil
}

func GetPrints(id int) ([]MTGStockPrint, error) {
	link := fmt.Sprintf("%s/%d", stksSetsURL, id)
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

	var set MTGStocksSet
	err = json.Unmarshal(data, &set)
	if err != nil {
		return nil, err
	}

	return set.Prints, nil
}
