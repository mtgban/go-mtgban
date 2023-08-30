package mtgban

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type LogCallbackFunc func(format string, a ...interface{})

const exchangeRateURL = "https://tassidicambio.bancaditalia.it/terzevalute-wf-web/rest/v1.0/latestRates?lang=en"

func GetExchangeRate(currency string) (float64, error) {
	req, err := http.NewRequest("GET", exchangeRateURL, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var reply struct {
		LatestRates []struct {
			ISOCode string `json:"isoCode"`
			USDRate string `json:"usdRate"`
		} `json:"latestRates"`
	}
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		return 0, err
	}

	for _, rate := range reply.LatestRates {
		if rate.ISOCode != currency {
			continue
		}

		rate, err := strconv.ParseFloat(rate.USDRate, 64)
		if err != nil {
			return 0, err
		}
		if rate == 0 {
			return 0, errors.New("no exchange rate obtained")
		}

		return 1 / rate, nil
	}

	return 0, errors.New("input currency not found")
}

func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
