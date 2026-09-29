package main

import (
	"math"
	"slices"
	"sort"
	"time"
)

// snapshot is one day of a CK product in the newspaper's history.
type snapshot struct {
	Day    int32   // days after the first day read
	Buy    float64 // CK's buy price, listed even while it is not buying
	Buying int32   // copies CK buys
	Stock  int32   // copies CK sells, across conditions
}

// product is one CK product's history and the category it is measured in.
type product struct {
	ID       string
	Foil     bool
	Category string // baseCategory's
	Released string
	Days     []snapshot // by day
}

// The measurement, as ADR-0004 of mtgban-website has it.
const (
	minBuy        = 1.0  // the rules apply where CK buys at $1 or more
	p90Window     = 90   // days before the day, which the P90 also counts
	p90MinDays    = 30   // buying days the P90 needs
	outcomeDays   = 14   // CK's price this many days on
	move          = 0.05 // up or down by this much
	buyoutFrom    = 3    // a buyout halves a stock of at least this
	cutRatio      = 0.8  // a cut is a price this share of last week's or less
	reopenHorizon = 30   // a paused day needs this many days after it
	minProducts   = 300  // a cell on fewer products is left out
	minCardDays   = 3000 // and on fewer card-days
)

// pauseBuckets are the pause lengths the site shows, the longest first.
var pauseBuckets = []int{30, 14, 7, 3, 0}

type oddsKey struct{ Category, Finish, Rule string }

type pauseKey struct {
	Category string
	MinDays  int
}

// cell counts the card-days of one table cell, the ones that moved each
// way (for pauses, came back within 7 and 30 days), and its products.
type cell struct {
	Days, Up, Down, Products int
}

// tally collects every product's card-days into the tables' cells.
type tally struct {
	start  time.Time // day 0
	last   int32     // the newest day read
	odds   map[oddsKey]*cell
	pauses map[pauseKey]*cell
}

func newTally(start time.Time, last int32) *tally {
	return &tally{start: start, last: last, odds: map[oddsKey]*cell{}, pauses: map[pauseKey]*cell{}}
}

// p90 is the P90 of sorted as Postgres' percentile_disc has it, which is how
// the site computes CK's P90: the first value at or past the 90th percentile.
func p90(sorted []float64) float64 {
	return sorted[int(math.Ceil(0.9*float64(len(sorted))))-1]
}

// add measures one product's card-days and pauses.
func (t *tally) add(p *product) {
	if len(p.Days) == 0 {
		return
	}
	finish := "nonfoil"
	if p.Foil {
		finish = "foil"
	}

	// A dense panel from the first day seen: a missing day carries the last
	// snapshot, as the backtest did.
	first := p.Days[0].Day
	n := int(t.last-first) + 1
	observed := make([]bool, n)
	buy := make([]float64, n)
	buying := make([]int32, n)
	stock := make([]int32, n)
	for _, s := range p.Days {
		observed[s.Day-first] = true
	}
	next := 0
	for i := range n {
		if next < len(p.Days) && int(p.Days[next].Day-first) == i {
			buy[i], buying[i], stock[i] = p.Days[next].Buy, p.Days[next].Buying, p.Days[next].Stock
			next++
		} else if i > 0 {
			buy[i], buying[i], stock[i] = buy[i-1], buying[i-1], stock[i-1]
		}
	}

	touched := map[oddsKey]bool{}
	count := func(key oddsKey, up, down bool) {
		c := t.odds[key]
		if c == nil {
			c = &cell{}
			t.odds[key] = c
		}
		c.Days++
		if up {
			c.Up++
		}
		if down {
			c.Down++
		}
		touched[key] = true
	}

	// The buying days of the last 91, sorted, for the P90 and the high.
	var window []float64
	inWindow := make([]bool, n)
	for i := range n {
		if i > p90Window && inWindow[i-p90Window-1] {
			window = remove(window, buy[i-p90Window-1])
		}
		prevHigh := math.NaN()
		if len(window) > 0 {
			prevHigh = window[len(window)-1]
		}
		if observed[i] && buying[i] > 0 && buy[i] > 0 {
			window = insert(window, buy[i])
			inWindow[i] = true
		}

		if !observed[i] || buying[i] == 0 || buy[i] < minBuy || len(window) < p90MinDays || i+outcomeDays >= n {
			continue
		}
		yesterday, weekAgo := int32(-1), 0.0
		if i >= 1 {
			yesterday = stock[i-1]
		}
		if i >= 7 {
			weekAgo = buy[i-7]
		}
		rule := ruleOf(buy[i], p90(window), stock[i], yesterday, weekAgo)
		later := i + outcomeDays
		up := buying[later] > 0 && buy[later] >= buy[i]*(1+move)
		down := buying[later] == 0 || buy[later] <= buy[i]*(1-move)

		day := t.start.AddDate(0, 0, int(first)+i)
		for _, category := range []string{"all", categoryOn(p.Category, p.Released, day)} {
			count(oddsKey{category, "", "typical"}, up, down)
			count(oddsKey{category, finish, "typical"}, up, down)
			if rule != "" {
				count(oddsKey{category, "", rule}, up, down)
			}
			if rule == "outofstock" {
				count(oddsKey{category, finish, rule}, up, down)
			}
			if !math.IsNaN(prevHigh) && buy[i] > prevHigh {
				count(oddsKey{category, "", "newhigh"}, up, down)
			}
		}
	}
	for key := range touched {
		t.odds[key].Products++
	}

	t.addPauses(p)
}

// addPauses measures the product's pauses: runs of snapshots with CK not
// buying after one where it bought at $1 or more, and on every paused day
// with a month after it, whether CK bought again within 7 and 30 days.
func (t *tally) addPauses(p *product) {
	touched := map[pauseKey]bool{}
	for k := 1; k < len(p.Days); k++ {
		prev, s := p.Days[k-1], p.Days[k]
		if s.Buying > 0 || prev.Buying == 0 || prev.Buy < minBuy {
			continue
		}
		start := s.Day
		end := k
		for end < len(p.Days) && p.Days[end].Buying == 0 {
			end++
		}
		resumed := int32(-1)
		if end < len(p.Days) {
			resumed = p.Days[end].Day
		}
		for _, d := range p.Days[k:end] {
			if d.Day+reopenHorizon > t.last {
				break
			}
			paused := int(d.Day - start)
			bucket := 0
			for _, b := range pauseBuckets {
				if paused >= b {
					bucket = b
					break
				}
			}
			week := resumed >= 0 && resumed <= d.Day+7
			month := resumed >= 0 && resumed <= d.Day+reopenHorizon
			day := t.start.AddDate(0, 0, int(d.Day))
			for _, category := range []string{"all", categoryOn(p.Category, p.Released, day)} {
				key := pauseKey{category, bucket}
				c := t.pauses[key]
				if c == nil {
					c = &cell{}
					t.pauses[key] = c
				}
				c.Days++
				if week {
					c.Up++
				}
				if month {
					c.Down++
				}
				touched[key] = true
			}
		}
		k = end
	}
	for key := range touched {
		t.pauses[key].Products++
	}
}

// ruleOf is the rule of ADR-0004 a card-day meets, in its order, or "":
// CK's price and P90, its stock today and yesterday (-1 unknown), and its
// price a week ago (0 unknown).
func ruleOf(buy, good float64, stock, yesterday int32, weekAgo float64) string {
	switch {
	case yesterday >= buyoutFrom && stock*2 <= yesterday:
		return "buyout"
	case stock == 0 && buy <= good:
		return "outofstock"
	case weekAgo > 0 && buy <= weekAgo*cutRatio:
		return "cut"
	case stock > 0 && buy > good:
		return "sell"
	}
	return ""
}

// insert and remove keep a sorted slice sorted.
func insert(sorted []float64, v float64) []float64 {
	i := sort.SearchFloat64s(sorted, v)
	return slices.Insert(sorted, i, v)
}

func remove(sorted []float64, v float64) []float64 {
	i := sort.SearchFloat64s(sorted, v)
	return slices.Delete(sorted, i, i+1)
}

// percent is part of whole in whole points, rounded.
func percent(part, whole int) int {
	return int(math.Round(100 * float64(part) / float64(whole)))
}
