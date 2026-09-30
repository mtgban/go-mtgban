package main

import (
	"math"
	"slices"
	"time"
)

// snapshot is one day of a CK product in the newspaper's history.
type snapshot struct {
	Day    int32   // days after the first day read
	Buy    float64 // CK's buy price, listed even while it is not buying
	Buying int32   // copies CK buys
	Stock  int32   // copies CK sells, across conditions
	Retail float64 // CK's NM retail
	Market float64 // TCG Market of the printing and finish; 0 when unknown
}

// product is one CK product's history and what the rules read about its card.
type product struct {
	ID    string
	Foil  bool
	TCGID int64
	Card  attrs
	Days  []snapshot // by day; only the days the newspaper has
}

// The measurement, as ADR-0004 of mtgban-website has it.
const (
	minBuy      = 3.0  // the rules apply where CK pays $3 or more
	halvedFrom  = 3    // a halving is of a stock of at least this
	marketRise  = 1.10 // TCG Market up this much in a week is a wait
	premiumSell = 2.0  // CK's retail at this many times TCG Market is a sell
	newSetFrom  = 28   // a set newSetFrom to newSetTo days past its
	newSetTo    = 55   // release is a sell
	reprintDays = 60   // so is a reprint released this many days ago or less
	highDays    = 90   // a new high beats every price CK paid in these days
	weekFrom    = 5    // "a week later" is the last snapshot 5 to 7 days on
	weekTo      = 7
	monthFrom   = 28 // "a month later", 28 to 30 days on
	monthTo     = 30
	backDays    = 30 // a pause's chance of CK buying again within these
	maxGap      = 2  // a pause reopening or starting after a longer gap is left out
)

// buckets are the bands of CK's retail the odds are quoted by, from the steps
// of its price ladder; a card CK pays $3 for retails at $5 or more.
var buckets = []band{{"5-10", 10}, {"10-20", 20}, {"20-50", 50}, {"50-100", 100}, {"100-200", 200}, {"200+", math.Inf(1)}}

// band is a bucket's name and the retail it holds up to, not included.
type band struct {
	Name  string
	Below float64
}

// pauseAges are the pause lengths the odds are quoted by, the longest first.
var pauseAges = []int{30, 14, 7, 3, 0}

func bucketOf(retail float64) string {
	for _, b := range buckets {
		if retail < b.Below {
			return b.Name
		}
	}
	return buckets[len(buckets)-1].Name
}

// signals are ADR-0004's inputs on one card-day.
type signals struct {
	Halved     bool // stock at half or less of yesterday's, from halvedFrom or more
	SoldOut    bool // out of stock today, in stock yesterday
	MarketRose bool // TCG Market up marketRise or more over the week
	NewSet     bool // the set newSetFrom to newSetTo days past its release
	Reprinted  bool // a reprint of reprintSetTypes released reprintDays ago or less
	Premium    bool // retail premiumSell times TCG Market or more
	NewHigh    bool // above every price CK paid in the last highDays days
}

// verdict is the card-day's state: wait wins over sell; "" is neutral.
func (s signals) verdict() string {
	switch {
	case s.Halved || s.SoldOut || s.MarketRose:
		return "wait"
	case s.NewSet || s.Reprinted || s.Premium || s.NewHigh:
		return "sell"
	}
	return ""
}

type cellKey struct{ Group, Finish, Bucket, Verdict string }

// cell counts the card-days of one table cell, and those with CK's price a
// week and a month later known, split by CK paying more or less (a pause is
// less); and its printings.
type cell struct {
	Days                            int
	WeekDays, WeekMore, WeekLess    int
	MonthDays, MonthMore, MonthLess int
	Printings                       int
}

type pauseKey struct {
	Finish  string
	MinDays int
}

// pauseCell counts the paused days of one cell and its pauses, those CK
// bought again within backDays after, and the price it came back at over the
// price it listed that day.
type pauseCell struct {
	Days, Back, Pauses int
	Ratios             []float64
}

// tally collects every product's card-days into the tables' cells.
type tally struct {
	start  time.Time // day 0
	last   int32     // the newest day read
	cells  map[cellKey]*cell
	pauses map[pauseKey]*pauseCell
}

func newTally(start time.Time, last int32) *tally {
	return &tally{start: start, last: last, cells: map[cellKey]*cell{}, pauses: map[pauseKey]*pauseCell{}}
}

// paid is what CK pays at a snapshot: nothing while it is not buying.
func paid(s *snapshot) float64 {
	if s.Buying > 0 {
		return s.Buy
	}
	return 0
}

// days is a product's snapshots by day, from its first, nil where missing.
type days struct {
	first int32
	at    []int32
	snaps []snapshot
}

func newDays(p *product, last int32) days {
	d := days{first: p.Days[0].Day, snaps: p.Days}
	d.at = make([]int32, int(last-d.first)+1)
	for i := range d.at {
		d.at[i] = -1
	}
	for i, s := range p.Days {
		d.at[s.Day-d.first] = int32(i)
	}
	return d
}

func (d days) on(day int32) *snapshot {
	if day < d.first || int(day-d.first) >= len(d.at) || d.at[day-d.first] < 0 {
		return nil
	}
	return &d.snaps[d.at[day-d.first]]
}

// latest is the last snapshot from day+from to day+to, or nil.
func (d days) latest(day, from, to int32) *snapshot {
	for x := day + to; x >= day+from; x-- {
		s := d.on(x)
		if s != nil {
			return s
		}
	}
	return nil
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
	group := "cohort"
	if p.Card.Exception {
		group = "exceptions"
	}
	d := newDays(p, t.last)

	touched := map[cellKey]bool{}
	count := func(key cellKey, buy float64, week, month *snapshot) {
		c := t.cells[key]
		if c == nil {
			c = &cell{}
			t.cells[key] = c
		}
		c.Days++
		if week != nil {
			c.WeekDays++
			if paid(week) > buy {
				c.WeekMore++
			} else if paid(week) < buy {
				c.WeekLess++
			}
		}
		if month != nil {
			c.MonthDays++
			if paid(month) > buy {
				c.MonthMore++
			} else if paid(month) < buy {
				c.MonthLess++
			}
		}
		touched[key] = true
	}

	// high holds the prices CK paid in the window before the day, as a
	// deque of falling prices, for the new high.
	type paidOn struct {
		day int32
		buy float64
	}
	var high []paidOn
	for i := range p.Days {
		s := &p.Days[i]
		for len(high) > 0 && high[0].day < s.Day-highDays {
			high = high[1:]
		}
		priorHigh := 0.0
		if len(high) > 0 {
			priorHigh = high[0].buy
		}
		if s.Buying > 0 && s.Buy > 0 {
			for len(high) > 0 && high[len(high)-1].buy <= s.Buy {
				high = high[:len(high)-1]
			}
			high = append(high, paidOn{s.Day, s.Buy})
		}

		if s.Buying == 0 || s.Buy < minBuy || s.Retail <= 0 || s.Day+weekTo > t.last {
			continue
		}
		day := t.start.AddDate(0, 0, int(s.Day))
		var sig signals
		if prev := d.on(s.Day - 1); prev != nil {
			sig.Halved = prev.Stock >= halvedFrom && s.Stock*2 <= prev.Stock
			sig.SoldOut = s.Stock == 0 && prev.Stock > 0
		}
		if weekAgo := d.on(s.Day - 7); weekAgo != nil && weekAgo.Market > 0 && s.Market > 0 {
			sig.MarketRose = s.Market >= marketRise*weekAgo.Market
		}
		if !p.Card.SetReleased.IsZero() {
			age := int(day.Sub(p.Card.SetReleased).Hours() / 24)
			sig.NewSet = age >= newSetFrom && age <= newSetTo
		}
		if r := p.Card.latestReprint(day); !r.IsZero() {
			sig.Reprinted = day.Sub(r) <= reprintDays*24*time.Hour
		}
		sig.Premium = s.Market > 0 && s.Retail >= premiumSell*s.Market
		sig.NewHigh = priorHigh > 0 && s.Buy > priorHigh

		week := d.latest(s.Day, weekFrom, weekTo)
		var month *snapshot
		if s.Day+monthTo <= t.last {
			month = d.latest(s.Day, monthFrom, monthTo)
		}
		verdict := sig.verdict()
		for _, bucket := range []string{bucketOf(s.Retail), "all"} {
			count(cellKey{group, finish, bucket, "typical"}, s.Buy, week, month)
			if verdict != "" {
				count(cellKey{group, finish, bucket, verdict}, s.Buy, week, month)
			}
			if sig.NewHigh {
				count(cellKey{group, finish, bucket, "newhigh"}, s.Buy, week, month)
			}
		}
	}
	for key := range touched {
		t.cells[key].Printings++
	}

	t.addPauses(p, finish)
}

// addPauses measures the product's pauses: runs of snapshots with CK not
// buying after one where it paid minBuy or more. On each paused day with
// backDays after it: whether CK bought again within them, and the price it
// came back at over the one it listed that day. A pause that starts or ends
// more than maxGap days after the snapshot before is left out, its timing
// unknown; a gap inside it is not, or the pauses that last would be the ones
// left out.
func (t *tally) addPauses(p *product, finish string) {
	for k := 1; k < len(p.Days); k++ {
		prev, s := p.Days[k-1], p.Days[k]
		if s.Buying > 0 || prev.Buying == 0 || prev.Buy < minBuy {
			continue
		}
		gapped := s.Day-prev.Day > maxGap
		end := k + 1
		for end < len(p.Days) && p.Days[end].Buying == 0 {
			end++
		}
		var back *snapshot
		if end < len(p.Days) {
			back = &p.Days[end]
			gapped = gapped || back.Day-p.Days[end-1].Day > maxGap
		}
		if !gapped {
			keys := map[pauseKey]bool{}
			for _, d := range p.Days[k:end] {
				if d.Day+backDays > t.last {
					break
				}
				age := int(d.Day - s.Day)
				minDays := 0
				for _, a := range pauseAges {
					if age >= a {
						minDays = a
						break
					}
				}
				key := pauseKey{finish, minDays}
				c := t.pauses[key]
				if c == nil {
					c = &pauseCell{}
					t.pauses[key] = c
				}
				c.Days++
				if back != nil && back.Day <= d.Day+backDays {
					c.Back++
					if d.Buy > 0 {
						c.Ratios = append(c.Ratios, back.Buy/d.Buy)
					}
				}
				keys[key] = true
			}
			for key := range keys {
				t.pauses[key].Pauses++
			}
		}
		k = end
	}
}

// deciles are the 10th to the 90th percentiles of values, nearest rank,
// rounded to cents of a ratio.
func deciles(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	out := make([]float64, 9)
	for i := range out {
		rank := int(math.Ceil(float64(i+1)/10*float64(len(sorted)))) - 1
		out[i] = math.Round(sorted[rank]*100) / 100
	}
	return out
}

// percent is part of whole in whole points, rounded; 0 of nothing.
func percent(part, whole int) int {
	if whole == 0 {
		return 0
	}
	return int(math.Round(100 * float64(part) / float64(whole)))
}
