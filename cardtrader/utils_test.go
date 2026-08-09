package cardtrader

import "testing"

func TestPriceToUSD(t *testing.T) {
	rates := map[string]float64{"EUR": 1.10, "GBP": 1.25}

	tests := []struct {
		name     string
		cents    int
		currency string
		want     float64
		wantErr  bool
	}{
		{"dollars pass through", 1234, "USD", 12.34, false},
		{"dollars need no rate at all", 500, "USD", 5, false},
		{"euros convert", 1000, "EUR", 11, false},
		{"pounds convert", 1000, "GBP", 12.5, false},
		{"a currency with no rate is refused", 1000, "CHF", 0, true},
		{"an empty currency is refused", 1000, "", 0, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			price, err := priceToUSD(test.cents, test.currency, rates)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %f", price)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if price != test.want {
				t.Errorf("expected %f, got %f", test.want, price)
			}
		})
	}
}

// A nil map is what a scraper holds before Load runs; dollars still have to
// convert, since that path never consults a rate.
func TestPriceToUSDWithoutRates(t *testing.T) {
	price, err := priceToUSD(1234, "USD", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if price != 12.34 {
		t.Errorf("expected 12.34, got %f", price)
	}
	if _, err = priceToUSD(1234, "EUR", nil); err == nil {
		t.Error("expected an error converting euros with no rates")
	}
}
