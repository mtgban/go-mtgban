package tcgplayer

import (
	"testing"
	"time"
)

func TestSalesPrice(t *testing.T) {
	now := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	sale := func(price float64, qty int, age time.Duration) LatestSalesData {
		return LatestSalesData{
			PurchasePrice: price,
			ShippingPrice: 4.99,
			Quantity:      qty,
			OrderDate:     now.Add(-age),
		}
	}

	for _, tt := range []struct {
		desc  string
		sales []LatestSalesData
		want  float64
	}{
		{"no sales say nothing", nil, 0},
		{"a single sale is its own median",
			[]LatestSalesData{sale(350, 1, 30*day)}, 350},
		{"an odd run takes the middle, not the outlier",
			[]LatestSalesData{sale(30, 1, 10*day), sale(35, 1, 20*day), sale(999, 1, 5*day)}, 35},
		{"an even run splits the middle pair",
			[]LatestSalesData{sale(30, 1, 10*day), sale(40, 1, 20*day)}, 35},
		{"a lot sale cannot say what one unit went for",
			[]LatestSalesData{sale(90, 3, 10*day)}, 0},
		{"a sale outside the window speaks for the market then",
			[]LatestSalesData{sale(350, 1, 400*day)}, 0},
		{"the stale and the lots fall away before the median",
			[]LatestSalesData{sale(30, 1, 10*day), sale(500, 4, 5*day), sale(700, 1, 500*day)}, 30},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := salesPrice(tt.sales, now); got != tt.want {
				t.Errorf("salesPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}
