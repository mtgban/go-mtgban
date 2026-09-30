package main

import (
	"slices"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestVerdict pins the states of ADR-0004 and their order: wait wins.
func TestVerdict(t *testing.T) {
	for _, tc := range []struct {
		name string
		sig  signals
		want string
	}{
		{"nothing", signals{}, ""},
		{"halved", signals{Halved: true}, "wait"},
		{"sold out", signals{SoldOut: true}, "wait"},
		{"TCG Market rose", signals{MarketRose: true}, "wait"},
		{"new set", signals{NewSet: true}, "sell"},
		{"reprinted", signals{Reprinted: true}, "sell"},
		{"twice TCG Market", signals{Premium: true}, "sell"},
		{"new high", signals{NewHigh: true}, "sell"},
		{"wait wins over sell", signals{SoldOut: true, NewHigh: true}, "wait"},
	} {
		got := tc.sig.verdict()
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBucketOf(t *testing.T) {
	for retail, want := range map[float64]string{5.99: "5-10", 9.99: "5-10", 10: "10-20", 19.99: "10-20", 49.99: "20-50", 99.99: "50-100", 199.99: "100-200", 1999.99: "200+"} {
		got := bucketOf(retail)
		if got != want {
			t.Errorf("bucketOf(%v) = %q, want %q", retail, got, want)
		}
	}
}

// TestTallyCells counts a card's days: its stock halves on day 30 (a wait),
// CK raises it on day 36 (a new high, a sell), and a day's week and month
// outcomes read the snapshots 5-7 and 28-30 days on.
func TestTallyCells(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := &product{ID: "1"}
	for day := int32(0); day <= 60; day++ {
		s := snapshot{Day: day, Buy: 10, Buying: 4, Stock: 6, Retail: 20}
		if day >= 30 {
			s.Stock = 3
		}
		if day >= 36 {
			s.Buy = 11
		}
		p.Days = append(p.Days, s)
	}
	tally := newTally(start, 60)
	tally.add(p)

	// Days 0 to 53 have a week after them; 0 to 30 a month.
	typical := cell{Days: 54, WeekDays: 54, WeekMore: 7, MonthDays: 31, MonthMore: 25, Printings: 1}
	for key, want := range map[cellKey]cell{
		{"cohort", "nonfoil", "20-50", "typical"}: typical,
		{"cohort", "nonfoil", "all", "typical"}:   typical,
		{"cohort", "nonfoil", "20-50", "wait"}:    {Days: 1, WeekDays: 1, WeekMore: 1, MonthDays: 1, MonthMore: 1, Printings: 1},
		{"cohort", "nonfoil", "20-50", "sell"}:    {Days: 1, WeekDays: 1, Printings: 1},
		{"cohort", "nonfoil", "20-50", "newhigh"}: {Days: 1, WeekDays: 1, Printings: 1},
	} {
		got := tally.cells[key]
		if got == nil || *got != want {
			t.Errorf("%v: got %+v, want %+v", key, got, want)
		}
	}

	tables := &Tables{}
	tally.fill(tables)
	want := Cell{Group: "cohort", Finish: "nonfoil", Bucket: "20-50", Verdict: "wait", WeekMore: 100, MonthMore: 100, Days: 1, Printings: 1}
	if !slices.Contains(tables.Cells, want) {
		t.Errorf("no %+v in %+v", want, tables.Cells)
	}
}

// TestTallyMarket reads TCG Market: at twice CK's retail it is a sell, and up
// 10% on the week a wait; a day with no snapshot a week before has no rise.
func TestTallyMarket(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := &product{ID: "1", Foil: true}
	for day := int32(0); day <= 30; day++ {
		market := 10.0
		if day >= 7 {
			market = 11
		}
		p.Days = append(p.Days, snapshot{Day: day, Buy: 8, Buying: 4, Stock: 6, Retail: 20, Market: market})
	}
	tally := newTally(start, 30)
	tally.add(p)
	for verdict, days := range map[string]int{"sell": 7, "wait": 7} {
		got := tally.cells[cellKey{"cohort", "foil", "20-50", verdict}]
		if got == nil || got.Days != days {
			t.Errorf("%s: got %+v, want %d days", verdict, got, days)
		}
	}
}

// TestAttrs finds the reprints that count, and the exceptions group.
func TestAttrs(t *testing.T) {
	b := &mtgmatcher.Backend{Sets: map[string]*mtgmatcher.Set{
		"LEA":  {Code: "LEA", ReleaseDate: "1993-08-05", Type: "core"},
		"M19":  {Code: "M19", ReleaseDate: "2018-07-13", Type: "core"},
		"MSSN": {Code: "MSSN", ReleaseDate: "2018-07-20", Type: "masters"},
		"2XM":  {Code: "2XM", ReleaseDate: "2020-08-07", Type: "masters"},
		"C21":  {Code: "C21", ReleaseDate: "2021-04-23", Type: "commander"},
		"NEO":  {Code: "NEO", ReleaseDate: "2022-02-18", Type: "expansion"},
		"SLD":  {Code: "SLD", ReleaseDate: "2019-12-02", Type: "box"},
		"MH3":  {Code: "MH3", ReleaseDate: "2024-06-14", Type: "draft_innovation"},
	}}
	printings := []string{"LEA", "M19", "MSSN", "2XM", "C21", "NEO", "SLD", "MH3"}
	date := func(s string) time.Time {
		d, _ := time.Parse(time.DateOnly, s)
		return d
	}

	a := attrsOf(b, &mtgmatcher.CardObject{Card: mtgmatcher.Card{SetCode: "M19"}}, printings)
	want := []time.Time{date("2020-08-07"), date("2021-04-23"), date("2024-06-14")}
	if a.Exception || !a.SetReleased.Equal(date("2018-07-13")) || !slices.EqualFunc(a.Reprints, want, time.Time.Equal) {
		t.Errorf("M19: got %+v, want reprints %v", a, want)
	}
	if got := a.latestReprint(date("2022-01-01")); !got.Equal(date("2021-04-23")) {
		t.Errorf("latest reprint by 2022: got %v", got)
	}
	if got := a.latestReprint(date("2019-01-01")); !got.IsZero() {
		t.Errorf("latest reprint by 2019: got %v", got)
	}

	for name, co := range map[string]*mtgmatcher.CardObject{
		"Alpha":         {Card: mtgmatcher.Card{SetCode: "LEA"}},
		"Reserved List": {Card: mtgmatcher.Card{SetCode: "M19", IsReserved: true}},
	} {
		if !attrsOf(b, co, nil).Exception {
			t.Errorf("%s is not an exception", name)
		}
	}
}

// TestTallyPauses counts a pause's days by how long it had lasted, whether
// CK bought again within 30 days and the price it came back at over the one it
// listed; a pause reopening after a hole is left out, one with a hole inside
// is not.
func TestTallyPauses(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := &product{ID: "1"}
	for day := int32(0); day <= 60; day++ {
		s := snapshot{Day: day, Buy: 5, Buying: 4, Stock: 2, Retail: 10}
		if day >= 10 && day < 20 {
			s.Buying, s.Buy = 0, 4
		}
		p.Days = append(p.Days, s)
	}
	tally := newTally(start, 60)
	tally.add(p)

	// Paused days 10 to 19, CK buying again at 5 on day 20.
	for key, want := range map[pauseKey]pauseCell{
		{"nonfoil", 0}: {Days: 3, Back: 3, Pauses: 1, Ratios: []float64{1.25, 1.25, 1.25}},
		{"nonfoil", 3}: {Days: 4, Back: 4, Pauses: 1, Ratios: []float64{1.25, 1.25, 1.25, 1.25}},
		{"nonfoil", 7}: {Days: 3, Back: 3, Pauses: 1, Ratios: []float64{1.25, 1.25, 1.25}},
	} {
		got := tally.pauses[key]
		if got == nil || got.Days != want.Days || got.Back != want.Back || got.Pauses != want.Pauses || !slices.Equal(got.Ratios, want.Ratios) {
			t.Errorf("%v: got %+v, want %+v", key, got, want)
		}
	}
	if tally.pauses[pauseKey{"nonfoil", 14}] != nil {
		t.Error("a 10-day pause reached the 14-day bucket")
	}

	// Paused from day 3; days 4 to 6 missing; CK buying again on day 9.
	holed := &product{ID: "2"}
	for _, day := range []int32{0, 1, 2, 3, 7, 8, 9, 40} {
		s := snapshot{Day: day, Buy: 5, Buying: 4, Retail: 10}
		if day >= 3 && day < 9 {
			s.Buying = 0
		}
		holed.Days = append(holed.Days, s)
	}
	tally = newTally(start, 60)
	tally.add(holed)
	if got := tally.pauses[pauseKey{"nonfoil", 3}]; got == nil || got.Days != 2 || got.Back != 2 {
		t.Errorf("a pause with a hole inside: got %+v, want its days 7 and 8", got)
	}

	// Paused days 3 and 4; days 5 to 8 missing; CK buying again on day 9.
	late := &product{ID: "3"}
	for _, day := range []int32{0, 1, 2, 3, 4, 9, 40} {
		s := snapshot{Day: day, Buy: 5, Buying: 4, Retail: 10}
		if day >= 3 && day < 9 {
			s.Buying = 0
		}
		late.Days = append(late.Days, s)
	}
	tally = newTally(start, 60)
	tally.add(late)
	if len(tally.pauses) != 0 {
		t.Errorf("a pause reopening after a 5-day hole was counted: %+v", tally.pauses)
	}
}

func TestDeciles(t *testing.T) {
	got := deciles([]float64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	if !slices.Equal(got, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Errorf("got %v", got)
	}
	if deciles(nil) != nil {
		t.Error("deciles of nothing are not nil")
	}
}
