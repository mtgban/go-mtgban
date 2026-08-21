package mtgban

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

// LogCallbackFunc receives a scraper's progress messages. Scrapers log
// nothing until one is set, so a quiet run means no callback rather than no
// activity.
type LogCallbackFunc func(format string, a ...any)

const exchangeRateURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.json"

// GetExchangeRates returns the rate that converts each currency the feed
// quotes to USD: multiply a price by its entry to get dollars. Keys are lower
// case, as the feed writes them.
//
// The feed answers every currency in one response, so a caller facing more
// than one asks once rather than once per currency - and a caller that cannot
// know in advance which it will be handed can look the answer up instead of
// having to have named it.
//
// A currency quoted at zero is left out rather than kept as an infinity: it
// converts nothing, and a caller reading a missing entry refuses the price
// where one read as a number would invent it.
func GetExchangeRates(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeRateURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		USD map[string]float64 `json:"usd"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	if len(response.USD) == 0 {
		return nil, errors.New("no exchange rates in response")
	}

	rates := make(map[string]float64, len(response.USD))
	for currency, rate := range response.USD {
		if rate == 0 {
			continue
		}
		rates[currency] = 1 / rate
	}
	return rates, nil
}

// GetExchangeRate returns the rate that converts the given currency to USD:
// multiply a price by it to get dollars.
func GetExchangeRate(ctx context.Context, currency string) (float64, error) {
	rates, err := GetExchangeRates(ctx)
	if err != nil {
		return 0, err
	}

	rate, found := rates[strings.ToLower(currency)]
	if !found {
		return 0, fmt.Errorf("%s not found in response", strings.ToLower(currency))
	}

	return rate, nil
}

// DateEqual reports whether two times fall on the same calendar day, in
// whatever location each carries. Prices are dated by the day they were
// collected, so the clock time is noise.
func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
