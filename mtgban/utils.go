package mtgban

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type LogCallbackFunc func(format string, a ...interface{})

const exchangeRateURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.json"

// Retrieve the USD exchange rate (you need to multiply this value to obtain USD)
func GetExchangeRate(currency string) (float64, error) {
	resp, err := cleanhttp.DefaultClient().Get(exchangeRateURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response struct {
		USD map[string]float64 `json:"usd"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}

	rate, found := response.USD[strings.ToLower(currency)]
	if !found {
		return 0, fmt.Errorf("%s not found in response", strings.ToLower(currency))
	}

	return 1 / rate, nil
}

func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
