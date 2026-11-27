// Package cardrush scrapes Card Rush's buylist.
package cardrush

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

const defaultConcurrency = 2

// CardRush prices what Card Rush buys.
type CardRush struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	backend *matcher.Backend

	buylistDate time.Time
	buylist     mtgban.BuylistRecord

	client *http.Client
}

// NewScraper returns a buylist scraper matching against b.
func NewScraper(b *matcher.Backend) *CardRush {
	cr := CardRush{backend: b}
	cr.buylist = mtgban.BuylistRecord{}
	cr.maxConcurrency = defaultConcurrency
	cr.client = mtgban.NewHTTPClient()
	return &cr
}

func (cr *CardRush) printf(format string, a ...any) {
	if cr.logCallback != nil {
		cr.logCallback("[CR] "+format, a...)
	}
}

// crCondition reads the store's bracketed grade; a row with none is NM.
// NM- and EX+ sit between two of our grades and come back empty.
func crCondition(s string) (mtgban.Condition, error) {
	switch s {
	case "", "〔NM〕":
		return mtgban.NM, nil
	case "〔NM-〕", "〔EX+〕":
		return "", nil
	case "〔EX〕":
		return mtgban.SP, nil
	case "〔PLD〕":
		return mtgban.MP, nil
	case "〔HP〕":
		return mtgban.HP, nil
	case "〔PR〕":
		return mtgban.PO, nil
	}
	return "", fmt.Errorf("%w: %q", mtgban.ErrInvalidCondition, s)
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (cr *CardRush) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "JPY")
	if err != nil {
		return err
	}

	rows, lastPage, err := GetBuylistPage(ctx, cr.client, 1)
	if err != nil {
		return err
	}
	var pages []int
	for page := 2; page <= lastPage; page++ {
		pages = append(pages, page)
	}
	mtgban.WorkerPool(ctx, cr.maxConcurrency, pages,
		func(ctx context.Context, page int, results chan<- []BuyingPrice) error {
			rows, _, err := GetBuylistPage(ctx, cr.client, page)
			if err != nil {
				return err
			}
			results <- rows
			return nil
		},
		func(page []BuyingPrice) {
			rows = append(rows, page...)
		},
		cr.printf)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	cr.printf("Found %d buylist entries over %d pages", len(rows), lastPage)

	for _, row := range rows {
		cr.add(row, rate)
	}
	cr.buylistDate = time.Now()

	return nil
}

func (cr *CardRush) add(row BuyingPrice, rate float64) {
	theCard, err := preprocess(cr.backend, row)
	if errors.Is(err, matcher.ErrUnsupported) {
		return
	}
	if err != nil {
		cr.printf("%v", err)
		return
	}
	// The store's search reads its own spelling, before Match changes it.
	link := buylistURL + "?" + url.Values{"name": {theCard.Name}}.Encode()

	cardID, err := cr.backend.Match(theCard)
	if errors.Is(err, matcher.ErrUnsupported) {
		return
	}
	if err != nil {
		cr.printf("%v", err)
		cr.printf("%q: %s", row.Name, theCard)
		var alias *matcher.AliasingError
		if errors.As(err, &alias) {
			for _, probe := range alias.Probe() {
				co, _ := cr.backend.GetUUID(probe)
				cr.printf("- %s", co)
			}
		}
		return
	}

	// A copy in another language than its printing's is another product.
	co, err := cr.backend.GetUUID(cardID)
	if err != nil || theCard.Language != "" && co.Language != theCard.Language {
		return
	}

	grade, err := crCondition(row.Condition)
	if err != nil {
		cr.printf("unsupported %s condition", row.Condition)
		return
	}
	if grade == "" {
		return
	}

	err = cr.buylist.Add(cardID, &mtgban.BuylistEntry{
		Conditions: grade,
		BuyPrice:   float64(row.Amount) * rate,
		URL:        link,
		OriginalID: fmt.Sprint(row.ID),
	})
	if err != nil {
		cr.printf("%v", err)
	}
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (cr *CardRush) Buylist() mtgban.BuylistRecord {
	return cr.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (cr *CardRush) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Rush"
	info.Shorthand = "CR"
	info.CountryFlag = "JP"
	info.BuylistTimestamp = &cr.buylistDate
	info.Game = cr.backend.Game
	return
}
