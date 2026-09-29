// Command ckodds measures what Card Kingdom's buylist did after each state
// mtgban-website marks on it (ADR-0004 there): the chances CK paid 5% more,
// or 5% less or nothing, two weeks on, by card category, finish and rule;
// and the chances CK bought a paused card again within 7 and 30 days, by
// how long the pause had lasted. It reads CK's daily history from the
// newspaper, ties each CK product to its card through CK's own scraper, and
// writes the tables with every product's category for the site to load.
//
//	NEWSPAPER_SQL_LINE=postgres://... ckodds -datastore allprintings5.json \
//	    -output b2://mtgban-datastore/magic/ck-odds.json.xz
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
	From string `json:"from"`
	To   string `json:"to"`
	// Every CK product's category today, by CK's product id.
	Categories map[string]string `json:"categories"`
	Odds       []Odds            `json:"odds"`
	Pauses     []Pause           `json:"pauses"`
}

// Odds are the chances, in percent, of CK paying 5% more (Up) and 5% less or
// nothing (Down) two weeks after a card-day of the category that met the rule;
// "typical" is every card-day of it. Finish is set for out of stock and
// typical only, and "all" is every category.
type Odds struct {
	Category string `json:"category"`
	Finish   string `json:"finish,omitempty"`
	Rule     string `json:"rule"`
	Up       int    `json:"up"`
	Down     int    `json:"down"`
	CardDays int    `json:"card_days"`
	Products int    `json:"products"`
}

// Pause is the chance, in percent, of CK buying a paused card of the
// category again within 7 days (Week) and 30 days (Month), once the pause has
// lasted MinDays or more (up to the next bucket).
type Pause struct {
	Category string `json:"category"`
	MinDays  int    `json:"min_days"`
	Week     int    `json:"week"`
	Month    int    `json:"month"`
	CardDays int    `json:"card_days"`
	Products int    `json:"products"`
}

// The order the tables list categories and rules in.
var (
	categoryOrder = []string{"all", "reserved", "vintage", "secret lair", "booster fun", "promo", "commander", "masters", "recent", "older"}
	ruleOrder     = []string{"typical", "sell", "buyout", "outofstock", "cut", "newhigh"}
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
	unmapped := 0
	for id, p := range products {
		co, err := cardOf(b, cards, id)
		if err != nil {
			unmapped++
			continue
		}
		p.Category, p.Released = baseCategory(b, co)
		t.add(p)
	}
	log.Println(unmapped, "products with history but no card, left out")

	tables := t.tables(minProducts, minCardDays)
	tables.Generated = time.Now().UTC()
	tables.From, tables.To = start.AddDate(0, 0, int(first)).Format(time.DateOnly), start.AddDate(0, 0, int(last)).Format(time.DateOnly)
	tables.Categories = map[string]string{}
	for id := range cards {
		co, err := cardOf(b, cards, id)
		if err != nil {
			continue
		}
		category, released := baseCategory(b, co)
		tables.Categories[id] = categoryOn(category, released, today)
	}

	err = write(ctx, *outputOpt, tables)
	if err != nil {
		log.Fatalln("output:", err)
	}
	log.Println(len(tables.Odds), "odds and", len(tables.Pauses), "pause cells written to", *outputOpt)
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

// tables turns the tally into the file's tables, leaving out the cells
// measured on fewer than products products or days card-days, and any
// category's pauses with a bucket that thin.
func (t *tally) tables(products, days int) *Tables {
	tables := &Tables{}
	for key, c := range t.odds {
		if c.Products < products || c.Days < days {
			continue
		}
		tables.Odds = append(tables.Odds, Odds{
			Category: key.Category, Finish: key.Finish, Rule: key.Rule,
			Up: percent(c.Up, c.Days), Down: percent(c.Down, c.Days),
			CardDays: c.Days, Products: c.Products,
		})
	}
	slices.SortFunc(tables.Odds, func(a, b Odds) int {
		return cmp.Or(
			cmp.Compare(slices.Index(categoryOrder, a.Category), slices.Index(categoryOrder, b.Category)),
			cmp.Compare(a.Finish, b.Finish),
			cmp.Compare(slices.Index(ruleOrder, a.Rule), slices.Index(ruleOrder, b.Rule)))
	})

	for _, category := range categoryOrder {
		var pauses []Pause
		for _, bucket := range pauseBuckets {
			c := t.pauses[pauseKey{category, bucket}]
			if c == nil || c.Products < products || c.Days < days {
				pauses = nil
				break
			}
			pauses = append(pauses, Pause{
				Category: category, MinDays: bucket,
				Week: percent(c.Up, c.Days), Month: percent(c.Down, c.Days),
				CardDays: c.Days, Products: c.Products,
			})
		}
		tables.Pauses = append(tables.Pauses, pauses...)
	}
	return tables
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
