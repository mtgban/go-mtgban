package mtgban

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

// LogCallbackFunc receives a scraper's progress messages. Scrapers log
// nothing until one is set, so a quiet run means no callback rather than no
// activity.
type LogCallbackFunc func(format string, a ...interface{})

const exchangeRateURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.json"

// GetExchangeRate returns the rate that converts the given currency to USD:
// multiply a price by it to get dollars.
func GetExchangeRate(ctx context.Context, currency string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeRateURL, http.NoBody)
	if err != nil {
		return 0, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
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

// DateEqual reports whether two times fall on the same calendar day, in
// whatever location each carries. Prices are dated by the day they were
// collected, so the clock time is noise.
func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
