package mtgban

import (
	"context"
	"testing"
)

// TestGetExchangeRates checks the feed answers in one response every currency
// a marketplace has quoted so far, which is what lets a caller look one up
// rather than having had to name it.
func TestGetExchangeRates(t *testing.T) {
	rates, err := GetExchangeRates(context.Background())
	if err != nil {
		t.Skipf("exchange rate feed unreachable: %v", err)
	}
	if len(rates) < 100 {
		t.Errorf("one response answered %d currencies, expected the whole table", len(rates))
	}
	t.Logf("one request answered %d currencies", len(rates))

	// The ones CardTrader has been seen quoting, plus the majors.
	for _, currency := range []string{"eur", "gbp", "aud", "cad", "chf", "sek", "pln", "jpy"} {
		if rates[currency] <= 0 {
			t.Errorf("no usable rate for %q", currency)
		}
	}
	// Keys are the feed's, so a caller folds rather than guessing.
	if _, found := rates["EUR"]; found {
		t.Error("rates are keyed in upper case, callers fold to lower")
	}
	// GetExchangeRate answers from the same table.
	rate, err := GetExchangeRate(context.Background(), "AUD")
	if err != nil {
		t.Fatalf("GetExchangeRate(AUD): %v", err)
	}
	if rate != rates["aud"] {
		t.Errorf("GetExchangeRate(AUD) = %v, table says %v", rate, rates["aud"])
	}
	if _, err := GetExchangeRate(context.Background(), "XYZ"); err == nil {
		t.Error("a currency the feed does not quote returned a rate")
	}
}
