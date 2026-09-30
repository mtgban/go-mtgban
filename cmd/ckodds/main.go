// Command ckodds measures what Card Kingdom's buylist did after the states
// mtgban-website marks on it (ADR-0004 there): the chances CK paid more, or
// less or nothing, a week and a month later, by group (the Reserved List and
// sets through 1994 apart), finish, CK's retail band and state; and, on a
// paused card, the chances CK bought it again within 30 days and the price it
// came back at, by how long the pause had lasted. It reads CK's daily history
// and TCG Market from the newspaper, ties each CK product to its card through
// CK's own scraper, and writes the tables, with what the rules read about
// every product's card, for the site to load.
//
//	NEWSPAPER_SQL_LINE=postgres://... ckodds -datastore allprintings5.json \
//	    -output b2://mtgban-datastore/magic/ck-odds-v2.json.xz
//
// A b2:// output reads its key pair from B2_APPLICATION_KEY_ID_DATASTORE and
// B2_APPLICATION_KEY_DATASTORE, as a b2:// datastore does.
package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/mtgban/simplecloud"

	_ "github.com/mtgban/go-mtgban/cardkingdom"
	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// Tables is the file the site loads.
type Tables struct {
	Generated time.Time `json:"generated"`
	// The history measured, its first and last day read.
	From   string  `json:"from"`
	To     string  `json:"to"`
	Cells  []Cell  `json:"cells"`
	Pauses []Pause `json:"pauses"`
	// What the rules read about every CK product's card, by CK's product id.
	Products map[string]Product `json:"products"`
}

// Cell is the chances, in percent, of CK paying more (More) or less or
// nothing (Less) a week and a month after a card-day of the cell, with the
// card-days and printings measured.
type Cell struct {
	Group     string `json:"group"`   // "cohort", or "exceptions": the Reserved List and sets through 1994
	Finish    string `json:"finish"`  // "nonfoil" or "foil"
	Bucket    string `json:"bucket"`  // CK's retail band, or "all"
	Verdict   string `json:"verdict"` // "typical", "wait", "sell" or "newhigh"
	WeekMore  int    `json:"week_more"`
	WeekLess  int    `json:"week_less"`
	MonthMore int    `json:"month_more"`
	MonthLess int    `json:"month_less"`
	Days      int    `json:"days"`
	Printings int    `json:"printings"`
}

// Pause is, for a paused card of the finish whose pause has lasted MinDays
// or more (up to the next length), the chance in percent of CK buying it
// again within 30 days, and the deciles of the price it came back at over the
// price it listed.
type Pause struct {
	Finish         string    `json:"finish"`
	MinDays        int       `json:"min_days"`
	BackMonth      int       `json:"back_month"`
	BackOverListed []float64 `json:"back_over_listed"`
	Days           int       `json:"days"`
	Pauses         int       `json:"pauses"`
}

// Product is what the rules read about a CK product's card, beyond its
// prices.
type Product struct {
	Foil        bool   `json:"foil,omitempty"` // etched included, as CK files it
	Exception   bool   `json:"exception,omitempty"`
	SetReleased string `json:"set_released,omitempty"`
	// The latest Modern Horizons-type, Commander or Masters set that
	// reprinted the card, released on or before the run.
	Reprinted string `json:"reprinted,omitempty"`
	// TCG Market of the printing and finish on the history's last day, and a
	// week before; 0 when the newspaper has none.
	Market        float64 `json:"market,omitempty"`
	MarketWeekAgo float64 `json:"market_week_ago,omitempty"`
}

// The order the tables list their cells in.
var (
	groupOrder   = []string{"cohort", "exceptions"}
	finishOrder  = []string{"nonfoil", "foil"}
	verdictOrder = []string{"typical", "wait", "sell", "newhigh"}
)

func main() {
	datastoreOpt := flag.String("datastore", "", "Path to the Magic datastore, a file or b2://")
	outputOpt := flag.String("output", "", "Where to write the tables, a file or b2://bucket/path; .xz compresses")
	daysOpt := flag.Int("days", 365, "Days of CK history to measure")
	flag.Parse()
	if *datastoreOpt == "" || *outputOpt == "" {
		log.Fatalln("-datastore and -output are required")
	}
	line := os.Getenv("NEWSPAPER_SQL_LINE")
	if line == "" {
		log.Fatalln("NEWSPAPER_SQL_LINE is not set")
	}
	ctx := context.Background()

	b, err := datastore.Read(mtgmatcher.GameMagic, *datastoreOpt)
	if err != nil {
		log.Fatalln("datastore:", err)
	}
	cards, err := ckCards(ctx, b)
	if err != nil {
		log.Fatalln("card kingdom:", err)
	}
	log.Println(len(cards), "CK products tied to cards")

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -*daysOpt)
	began := time.Now()
	products, first, last, err := readHistory(ctx, line, start)
	if err != nil {
		log.Fatalln("newspaper:", err)
	}
	log.Println(len(products), "products' history read in", time.Since(began).Round(time.Second))

	t := newTally(start, last)
	tables := &Tables{Products: map[string]Product{}}
	measured := 0
	for id := range cards {
		co, err := cardOf(b, cards, id)
		if err != nil {
			continue
		}
		printings, _ := b.Printings4Card(co.Name)
		a := attrsOf(b, co, printings)
		out := Product{Foil: co.Foil || co.Etched, Exception: a.Exception}
		if !a.SetReleased.IsZero() {
			out.SetReleased = a.SetReleased.Format(time.DateOnly)
		}
		if r := a.latestReprint(today); !r.IsZero() {
			out.Reprinted = r.Format(time.DateOnly)
		}
		p := products[id]
		if p != nil {
			p.Card = a
			t.add(p)
			measured++
			d := newDays(p, last)
			if s := d.on(last); s != nil {
				out.Market = s.Market
			}
			if s := d.on(last - 7); s != nil {
				out.MarketWeekAgo = s.Market
			}
		}
		tables.Products[id] = out
	}
	log.Println(len(products)-measured, "products with history but no card, left out")

	t.fill(tables)
	tables.Generated = time.Now().UTC()
	tables.From, tables.To = start.AddDate(0, 0, int(first)).Format(time.DateOnly), start.AddDate(0, 0, int(last)).Format(time.DateOnly)

	err = write(ctx, *outputOpt, tables)
	if err != nil {
		log.Fatalln("output:", err)
	}
	log.Println(len(tables.Cells), "cells,", len(tables.Pauses), "pause cells and", len(tables.Products), "products written to", *outputOpt)
}

// ckCards ties every CK product to its card through CK's scraper, as bantool
// runs it: every product sits on its buylist, the last known one included
// where CK is not buying, under the card it matched, with CK's product id.
func ckCards(ctx context.Context, b *mtgmatcher.Backend) (map[string]string, error) {
	scraper, err := mtgban.NewScraper(b, "cardkingdom")
	if err != nil {
		return nil, err
	}
	err = scraper.Load(ctx)
	if err != nil {
		return nil, err
	}
	vendor, ok := scraper.(mtgban.Vendor)
	if !ok {
		return nil, errors.New("the scraper has no buylist")
	}
	cards := map[string]string{}
	for cardID, entries := range vendor.Buylist() {
		for _, entry := range entries {
			if entry.OriginalID != "" {
				cards[entry.OriginalID] = cardID
			}
		}
	}
	return cards, nil
}

func cardOf(b *mtgmatcher.Backend, cards map[string]string, id string) (*mtgmatcher.CardObject, error) {
	cardID, found := cards[id]
	if !found {
		return nil, errors.New("no card")
	}
	return b.GetUUID(cardID)
}

// fill turns the tally into the file's cells, every one with its counts:
// the site decides which have enough printings to quote.
func (t *tally) fill(tables *Tables) {
	for key, c := range t.cells {
		tables.Cells = append(tables.Cells, Cell{
			Group: key.Group, Finish: key.Finish, Bucket: key.Bucket, Verdict: key.Verdict,
			WeekMore: percent(c.WeekMore, c.WeekDays), WeekLess: percent(c.WeekLess, c.WeekDays),
			MonthMore: percent(c.MonthMore, c.MonthDays), MonthLess: percent(c.MonthLess, c.MonthDays),
			Days: c.Days, Printings: c.Printings,
		})
	}
	bucketIndex := func(name string) int {
		return slices.IndexFunc(buckets, func(b band) bool { return b.Name == name })
	}
	slices.SortFunc(tables.Cells, func(a, b Cell) int {
		return cmp.Or(
			cmp.Compare(slices.Index(groupOrder, a.Group), slices.Index(groupOrder, b.Group)),
			cmp.Compare(slices.Index(finishOrder, a.Finish), slices.Index(finishOrder, b.Finish)),
			cmp.Compare(bucketIndex(a.Bucket), bucketIndex(b.Bucket)),
			cmp.Compare(slices.Index(verdictOrder, a.Verdict), slices.Index(verdictOrder, b.Verdict)))
	})

	for _, finish := range finishOrder {
		for i := len(pauseAges) - 1; i >= 0; i-- {
			c := t.pauses[pauseKey{finish, pauseAges[i]}]
			if c == nil {
				continue
			}
			tables.Pauses = append(tables.Pauses, Pause{
				Finish: finish, MinDays: pauseAges[i], BackMonth: percent(c.Back, c.Days),
				BackOverListed: deciles(c.Ratios), Days: c.Days, Pauses: c.Pauses,
			})
		}
	}
}

// write stores the tables at path, a file or a b2:// object.
func write(ctx context.Context, path string, tables *Tables) error {
	u, err := url.Parse(path)
	if err != nil {
		return err
	}
	var bucket simplecloud.Writer
	switch u.Scheme {
	case "":
		bucket = &simplecloud.FileBucket{}
	case "b2":
		bucket, err = simplecloud.NewB2Client(ctx,
			os.Getenv("B2_APPLICATION_KEY_ID_DATASTORE"), os.Getenv("B2_APPLICATION_KEY_DATASTORE"), u.Host)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output %s", path)
	}
	w, err := simplecloud.InitWriter(ctx, bucket, path)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(tables)
	if err != nil {
		aborter, ok := w.(simplecloud.Aborter)
		if ok {
			aborter.Abort()
		} else {
			w.Close()
		}
		return err
	}
	return w.Close()
}
