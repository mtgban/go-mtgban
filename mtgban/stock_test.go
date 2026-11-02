package mtgban

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

func TestLoadHalves(t *testing.T) {
	failure := errors.New("site down")
	ok := func(context.Context) error { return nil }
	fail := func(context.Context) error { return failure }

	tests := []struct {
		name            string
		config          ScraperOptions
		retail, buylist func(context.Context) error
		wantRan         []string
		inventoryErr    bool
		buylistErr      bool
	}{
		{name: "both", retail: ok, buylist: ok, wantRan: []string{"retail", "buylist"}},
		{name: "retail off", config: ScraperOptions{DisableRetail: true},
			retail: ok, buylist: ok, wantRan: []string{"buylist"}},
		{name: "buylist off", config: ScraperOptions{DisableBuylist: true},
			retail: ok, buylist: ok, wantRan: []string{"retail"}},
		{name: "no buylist", retail: ok, wantRan: []string{"retail"}},
		{name: "both fail", retail: fail, buylist: fail,
			wantRan: []string{"retail", "buylist"}, inventoryErr: true, buylistErr: true},
		{name: "buylist fails", retail: ok, buylist: fail,
			wantRan: []string{"retail", "buylist"}, buylistErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			track := func(half string, fn func(context.Context) error) func(context.Context) error {
				if fn == nil {
					return nil
				}
				return func(ctx context.Context) error {
					ran = append(ran, half)
					return fn(ctx)
				}
			}

			s := NewStock()
			s.SetConfig(tt.config)
			err := s.LoadHalves(context.Background(), track("retail", tt.retail), track("buylist", tt.buylist))
			if !slices.Equal(ran, tt.wantRan) {
				t.Errorf("ran %v, want %v", ran, tt.wantRan)
			}
			if errors.Is(err, ErrInventoryLoad) != tt.inventoryErr {
				t.Errorf("error %v, want ErrInventoryLoad %v", err, tt.inventoryErr)
			}
			if errors.Is(err, ErrBuylistLoad) != tt.buylistErr {
				t.Errorf("error %v, want ErrBuylistLoad %v", err, tt.buylistErr)
			}
			if err != nil && !errors.Is(err, failure) {
				t.Errorf("error %v does not wrap the half's own", err)
			}
			if s.InventoryDate.IsZero() != (tt.inventoryErr || !slices.Contains(ran, "retail")) {
				t.Errorf("inventory date %v after ran %v, error %v", s.InventoryDate, ran, err)
			}
			if s.BuylistDate.IsZero() != (tt.buylistErr || !slices.Contains(ran, "buylist")) {
				t.Errorf("buylist date %v after ran %v, error %v", s.BuylistDate, ran, err)
			}
		})
	}
}

func TestCollect(t *testing.T) {
	var logged []string
	printf := func(format string, a ...any) {
		logged = append(logged, fmt.Sprintf(format, a...))
	}

	s := NewStock()
	collect := s.Collect(printf)
	collect(Result{CardID: "a", Inventory: &InventoryEntry{Price: 1, Conditions: NM}})
	collect(Result{CardID: "a", Buylist: &BuylistEntry{BuyPrice: 1, Conditions: NM}})
	collect(Result{CardID: "a", Inventory: &InventoryEntry{Price: -1, Conditions: NM}})

	if len(s.Inventory()["a"]) != 1 || len(s.Buylist()["a"]) != 1 {
		t.Errorf("collected %v and %v, want one entry each", s.Inventory(), s.Buylist())
	}
	if len(logged) != 1 {
		t.Errorf("logged %q, want the one refused entry", logged)
	}
}

func TestTaggedLog(t *testing.T) {
	TaggedLog("XX", nil)("dropped %d", 1)

	var got string
	TaggedLog("XX", func(format string, a ...any) {
		got = fmt.Sprintf(format, a...)
	})("line %d", 2)
	if got != "[XX] line 2" {
		t.Errorf("logged %q", got)
	}
}

func TestForGame(t *testing.T) {
	table := map[matcher.Game]string{matcher.GameMagic: "mtg"}

	got, err := ForGame(table, matcher.GameMagic)
	if err != nil || got != "mtg" {
		t.Errorf("ForGame(magic) = %q, %v", got, err)
	}
	_, err = ForGame(table, matcher.GameLorcana)
	if err == nil || err.Error() != `unsupported game "lorcana"` {
		t.Errorf("ForGame(lorcana) error = %v", err)
	}
}
