package mtgban

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type LogCallbackFunc func(format string, a ...interface{})

const exchangeRateURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/eur.json"

func GetExchangeRate(currency string) (float64, error) {
	resp, err := cleanhttp.DefaultClient().Get(exchangeRateURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response struct {
		EUR map[string]float64 `json:"eur"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}

	rate, found := response.EUR["usd"]
	if !found {
		return 0, errors.New("usd not found")
	}

	return rate, nil
}

func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
