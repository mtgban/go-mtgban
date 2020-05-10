package mtgstocks

import (
	"encoding/json"
	"io/ioutil"

	http "github.com/hashicorp/go-retryablehttp"
)

type Interest struct {
	InterestType string  `json:"interest_type"`
	Foil         bool    `json:"foil"`
	Percentage   float64 `json:"percentage"`
	PastPrice    float64 `json:"past_price"`
	PresentPrice float64 `json:"present_price"`
	Date         int64   `json:"date"`
	Print        struct {
		Id        int    `json:"id"`
		Name      string `json:"name"`
		Rarity    string `json:"rarity"`
		SetId     int    `json:"set_id"`
		SetName   string `json:"set_name"`
		IconClass string `json:"icon_class"`
		Reserved  bool   `json:"reserved"`
		SetType   string `json:"set_type"`
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
	Date    int64 `json:"date"`
	Average struct {
		Foil   []Interest `json:"foil"`
		Normal []Interest `json:"normal"`
	} `json:"average"`
	Market struct {
		Foil   []Interest `json:"foil"`
		Normal []Interest `json:"normal"`
	} `json:"market"`
}

const stksBaseURL = "https://api.mtgstocks.com/interests"

func GetInterests() (*MTGStocksInterests, error) {
	resp, err := http.Get(stksBaseURL)
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
