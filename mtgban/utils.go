package mtgban

import (
	"encoding/json"
	"net/http"
)

type LogCallbackFunc func(format string, a ...interface{})

func GetExchangeRate(currency string) (float64, error) {
	resp, err := http.Get("https://api.exchangeratesapi.io/latest?base=" + currency)
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
