package mtgban

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

type LogCallbackFunc func(format string, a ...interface{})

func GetExchangeRate(currency string) (float64, error) {
	resp, err := cleanhttp.DefaultClient().Get("https://api.exchangeratesapi.io/latest?base=" + currency)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var reply struct {
		Rates struct {
			USD float64 `json:"USD"`
		} `json:"rates"`
	}
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		return 0, err
	}

	return reply.Rates.USD, nil
}

func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func sliceStringHas(slice []string, probe string) bool {
	for i := range slice {
		if slice[i] == probe {
			return true
		}
	}
	return false
}
