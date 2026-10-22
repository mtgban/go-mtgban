package mtgban

import (
	"errors"
	"math"
	"testing"
)

// TestAddRefusesBadPrices pins that add() refuses a price no store quotes - a
// NaN, an infinity, a negative - and a negative quantity, on both sides, and
// leaves the record as it was, while a price of zero (Card Kingdom's
// link-only out-of-stock entries) is still taken.
func TestAddRefusesBadPrices(t *testing.T) {
	tests := []struct {
		price float64
		qty   int
		want  error
	}{
		{math.NaN(), 1, ErrInvalidPrice},
		{math.Inf(1), 1, ErrInvalidPrice},
		{math.Inf(-1), 1, ErrInvalidPrice},
		{-0.01, 1, ErrInvalidPrice},
		{1, -2, ErrInvalidQuantity},
		{0, 1, nil},
		{2.5, 1, nil},
	}
	for _, test := range tests {
		inv := InventoryRecord{}
		err := inv.AddRelaxed("id", &InventoryEntry{Conditions: NM, Price: test.price, Quantity: test.qty})
		if !errors.Is(err, test.want) || (test.want != nil && len(inv) != 0) {
			t.Errorf("inventory price %v qty %d = %v (kept %d), want %v", test.price, test.qty, err, len(inv), test.want)
		}

		bl := BuylistRecord{}
		err = bl.AddRelaxed("id", &BuylistEntry{Conditions: NM, BuyPrice: test.price, Quantity: test.qty})
		if !errors.Is(err, test.want) || (test.want != nil && len(bl) != 0) {
			t.Errorf("buylist price %v qty %d = %v (kept %d), want %v", test.price, test.qty, err, len(bl), test.want)
		}
	}
}
