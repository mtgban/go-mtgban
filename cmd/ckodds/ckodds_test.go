package main

import (
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestP90 takes the P90 the way Postgres' percentile_disc does, as the site's
// own P90 is taken.
func TestP90(t *testing.T) {
	for _, tc := range []struct {
		values []float64
		want   float64
	}{
		{[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 9},
		{[]float64{1, 2, 3}, 3},
		{[]float64{10, 10, 10, 12}, 12},
	} {
		got := p90(tc.values)
		if got != tc.want {
			t.Errorf("p90(%v) = %v, want %v", tc.values, got, tc.want)
		}
	}
}

// TestRuleOf pins the rules of ADR-0004 and their order.
func TestRuleOf(t *testing.T) {
	for _, tc := range []struct {
		name             string
		buy, good        float64
		stock, yesterday int32
		weekAgo          float64
		want             string
	}{
		{"above P90 in stock", 10, 9, 5, 5, 10, "sell"},
		{"at P90", 9, 9, 5, 5, 9, ""},
		{"stock halved from 6", 10, 9, 3, 6, 10, "buyout"},
		{"halved from under 3", 10, 9, 1, 2, 10, "sell"},
		{"sold out since yesterday wins over out of stock", 9, 9, 0, 4, 9, "buyout"},
		{"out of stock at P90", 9, 9, 0, 0, 9, "outofstock"},
		{"out of stock above P90", 10, 9, 0, 0, 10, ""},
		{"cut 20% wins over sell", 10, 9, 5, 5, 13, "cut"},
		{"cut under 20%", 10, 9, 5, 5, 12, "sell"},
		{"no yesterday, no week ago", 10, 9, 5, -1, 0, "sell"},
	} {
		got := ruleOf(tc.buy, tc.good, tc.stock, tc.yesterday, tc.weekAgo)
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestTallyOdds counts a card's days in its cells: one day above a flat P90
// is a sell now and a new high, and CK paid less two weeks on.
func TestTallyOdds(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := &product{ID: "1", Category: "masters", Released: "2020-08-07"}
	for day := int32(0); day <= 50; day++ {
		buy := 10.0
		if day == 32 {
			buy = 12
		}
		p.Days = append(p.Days, snapshot{Day: day, Buy: buy, Buying: 4, Stock: 2})
	}
	tally := newTally(start, 50)
	tally.add(p)

	// Days 29 to 36: a P90 from day 29, an outcome inside the history to day 36.
	for key, want := range map[oddsKey]cell{
		{"all", "", "typical"}:            {Days: 8, Up: 0, Down: 1, Products: 1},
		{"masters", "", "typical"}:        {Days: 8, Up: 0, Down: 1, Products: 1},
		{"masters", "nonfoil", "typical"}: {Days: 8, Up: 0, Down: 1, Products: 1},
		{"masters", "", "sell"}:           {Days: 1, Up: 0, Down: 1, Products: 1},
		{"masters", "", "newhigh"}:        {Days: 1, Up: 0, Down: 1, Products: 1},
	} {
		got := tally.odds[key]
		if got == nil || *got != want {
			t.Errorf("%v: got %+v, want %+v", key, got, want)
		}
	}

	tables := tally.tables(1, 1)
	found := false
	for _, o := range tables.Odds {
		if o == (Odds{Category: "masters", Rule: "sell", Up: 0, Down: 100, CardDays: 1, Products: 1}) {
			found = true
		}
	}
	if !found {
		t.Errorf("no masters sell odds in %+v", tables.Odds)
	}
}

// TestTallyPauses counts a pause's days by how long it had lasted, and
// whether CK bought again within 7 and 30 days.
func TestTallyPauses(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := &product{ID: "1", Category: "regular", Released: "2015-01-01"}
	for day := int32(0); day <= 60; day++ {
		buying := int32(4)
		if day >= 10 && day < 20 {
			buying = 0
		}
		p.Days = append(p.Days, snapshot{Day: day, Buy: 5, Buying: buying, Stock: 2})
	}
	tally := newTally(start, 60)
	tally.add(p)

	// Paused days 10 to 19, CK buying again on day 20.
	for key, want := range map[pauseKey]cell{
		{"all", 0}:   {Days: 3, Up: 0, Down: 3, Products: 1},
		{"all", 3}:   {Days: 4, Up: 4, Down: 4, Products: 1},
		{"all", 7}:   {Days: 3, Up: 3, Down: 3, Products: 1},
		{"older", 3}: {Days: 4, Up: 4, Down: 4, Products: 1},
	} {
		got := tally.pauses[key]
		if got == nil || *got != want {
			t.Errorf("%v: got %+v, want %+v", key, got, want)
		}
	}
	if tally.pauses[pauseKey{"all", 14}] != nil {
		t.Error("a 10-day pause reached the 14-day bucket")
	}
}

// TestCategories sorts cards the way the backtest did, the first category
// that applies winning.
func TestCategories(t *testing.T) {
	today := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	b := &mtgmatcher.Backend{Sets: map[string]*mtgmatcher.Set{
		"LEA":  {Code: "LEA", Name: "Limited Edition Alpha", ReleaseDate: "1993-08-05", Type: "core"},
		"SLD":  {Code: "SLD", Name: "Secret Lair Drop", ReleaseDate: "2019-12-02", Type: "box"},
		"SLX":  {Code: "SLX", Name: "Secret Lair x Something", ReleaseDate: "2022-01-01", Type: "box"},
		"NEO":  {Code: "NEO", Name: "Kamigawa: Neon Dynasty", ReleaseDate: "2022-02-18", Type: "expansion"},
		"M19":  {Code: "M19", Name: "Core Set 2019", ReleaseDate: "2018-07-13", Type: "core"},
		"PM19": {Code: "PM19", Name: "Core Set 2019 Promos", ReleaseDate: "2018-07-13", Type: "promo"},
		"C21":  {Code: "C21", Name: "Commander 2021", ReleaseDate: "2021-04-23", Type: "commander"},
		"2XM":  {Code: "2XM", Name: "Double Masters", ReleaseDate: "2020-08-07", Type: "masters"},
		"FDN":  {Code: "FDN", Name: "Foundations", ReleaseDate: "2024-11-15", Type: "core"},
		"MH3":  {Code: "MH3", Name: "Modern Horizons 3", ReleaseDate: "2024-06-14", Type: "draft_innovation"},
	}}
	card := func(set string, edit func(c *mtgmatcher.Card)) *mtgmatcher.CardObject {
		co := &mtgmatcher.CardObject{Card: mtgmatcher.Card{SetCode: set}}
		if edit != nil {
			edit(&co.Card)
		}
		return co
	}
	for _, tc := range []struct {
		name string
		co   *mtgmatcher.CardObject
		want string
	}{
		{"Reserved List before vintage", card("LEA", func(c *mtgmatcher.Card) { c.IsReserved = true }), "reserved"},
		{"Alpha", card("LEA", nil), "vintage"},
		{"Secret Lair, borderless", card("SLD", func(c *mtgmatcher.Card) { c.BorderColor = "borderless" }), "secret lair"},
		{"a Secret Lair set by name", card("SLX", nil), "secret lair"},
		{"borderless since Eldraine", card("NEO", func(c *mtgmatcher.Card) { c.BorderColor = "borderless" }), "booster fun"},
		{"showcase", card("NEO", func(c *mtgmatcher.Card) { c.FrameEffects = []string{"legendary", "showcase"} }), "booster fun"},
		{"borderless before Eldraine", card("M19", func(c *mtgmatcher.Card) { c.BorderColor = "borderless" }), "older"},
		{"a promo set", card("PM19", nil), "promo"},
		{"a promo printing", card("NEO", func(c *mtgmatcher.Card) { c.IsPromo = true }), "promo"},
		{"Commander", card("C21", nil), "commander"},
		{"Masters", card("2XM", nil), "masters"},
		{"two years old or less", card("FDN", nil), "recent"},
		{"over two years old", card("MH3", nil), "older"},
		{"an unknown set", card("XXX", nil), "older"},
	} {
		category, released := baseCategory(b, tc.co)
		got := categoryOn(category, released, today)
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
