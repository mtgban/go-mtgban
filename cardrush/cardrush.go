// Package cardrush scrapes Card Rush's store and buylist.
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

// CardRush prices what Card Rush sells and buys.
type CardRush struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	disableRetail  bool
	disableBuylist bool

	backend *matcher.Backend

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord

	buylistDate time.Time
	buylist     mtgban.BuylistRecord

	client *http.Client
}

// NewScraper returns a scraper matching against b.
func NewScraper(b *matcher.Backend) *CardRush {
	cr := CardRush{backend: b}
	cr.inventory = mtgban.InventoryRecord{}
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

// crCondition reads the buylist's bracketed grade; a row with none is NM.
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

// storeCondition reads the grade a store title opens with; a title with none
// is NM. EX+ is sold as SP, the grade below it, so a copy is never claimed
// better than it is.
func storeCondition(s string) (mtgban.Condition, error) {
	switch s {
	case "":
		return mtgban.NM, nil
	case "EX+", "EX":
		return mtgban.SP, nil
	case "PLD":
		return mtgban.MP, nil
	case "HPLD":
		return mtgban.HP, nil
	case "POOR":
		return mtgban.PO, nil
	}
	return "", fmt.Errorf("%w: %q", mtgban.ErrInvalidCondition, s)
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (cr *CardRush) SetConfig(opt mtgban.ScraperOptions) {
	cr.disableRetail = opt.DisableRetail
	cr.disableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (cr *CardRush) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "JPY")
	if err != nil {
		return err
	}

	var errs []error
	if !cr.disableRetail {
		err := cr.loadStore(ctx, rate)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err))
		}
	}
	if !cr.disableBuylist {
		err := cr.loadBuylist(ctx, rate)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, err))
		}
	}
	return errors.Join(errs...)
}

// walk reads page 1, then every page after it that page names.
func walk[T any](ctx context.Context, cr *CardRush, get func(context.Context, int) ([]T, int, error)) ([]T, int, error) {
	items, lastPage, err := get(ctx, 1)
	if err != nil {
		return nil, 0, err
	}
	var pages []int
	for page := 2; page <= lastPage; page++ {
		pages = append(pages, page)
	}
	mtgban.WorkerPool(ctx, cr.maxConcurrency, pages,
		func(ctx context.Context, page int, results chan<- []T) error {
			items, _, err := get(ctx, page)
			if err != nil {
				return err
			}
			results <- items
			return nil
		},
		func(page []T) {
			items = append(items, page...)
		},
		cr.printf)
	return items, lastPage, ctx.Err()
}

func (cr *CardRush) loadStore(ctx context.Context, rate float64) error {
	products, lastPage, err := walk(ctx, cr, func(ctx context.Context, page int) ([]Product, int, error) {
		return GetStorePage(ctx, cr.client, page)
	})
	if err != nil {
		return err
	}
	cr.printf("Found %d store listings over %d pages", len(products), lastPage)

	// A listing that moves while the pages are read shows up twice.
	seen := map[string]bool{}
	for _, product := range products {
		if seen[product.ID] {
			continue
		}
		seen[product.ID] = true
		cr.addProduct(product, rate)
	}
	cr.inventoryDate = time.Now()
	return nil
}

func (cr *CardRush) loadBuylist(ctx context.Context, rate float64) error {
	rows, lastPage, err := walk(ctx, cr, func(ctx context.Context, page int) ([]BuyingPrice, int, error) {
		return GetBuylistPage(ctx, cr.client, page)
	})
	if err != nil {
		return err
	}
	cr.printf("Found %d buylist entries over %d pages", len(rows), lastPage)

	for _, row := range rows {
		cr.add(row, rate)
	}
	cr.buylistDate = time.Now()
	return nil
}

// match finds the card the store's naming stands for, logging what it cannot
// place, and reports whether there is one. The name it returns is the one
// read off the title, which the store's own search finds.
func (cr *CardRush) match(title, pack, lang string) (cardID, name string, found bool) {
	theCard, err := preprocess(cr.backend, title, pack, lang)
	if errors.Is(err, matcher.ErrUnsupported) {
		return "", "", false
	}
	if err != nil {
		cr.printf("%v", err)
		return "", "", false
	}
	name = theCard.Name

	cardID, err = cr.backend.Match(theCard)
	if errors.Is(err, matcher.ErrUnsupported) {
		return "", "", false
	}
	if err != nil {
		cr.printf("%v", err)
		cr.printf("%q: %s", title, theCard)
		var alias *matcher.AliasingError
		if errors.As(err, &alias) {
			for _, probe := range alias.Probe() {
				co, _ := cr.backend.GetUUID(probe)
				cr.printf("- %s", co)
			}
		}
		return "", "", false
	}

	// A copy in another language than its printing's is another product.
	co, err := cr.backend.GetUUID(cardID)
	if err != nil || theCard.Language != "" && co.Language != theCard.Language {
		return "", "", false
	}
	return cardID, name, true
}

func (cr *CardRush) addProduct(product Product, rate float64) {
	m := reTitle.FindStringSubmatch(product.Title)
	// Sealed product, supplies, the store's own test listings, and a listing
	// with no price. A copy in a grading service's slab is skipped too: the
	// stores selling slabs list them under a graded seller of their own,
	// which the two here do not warrant.
	if m == nil || m[2] != "" || m[4] == "-" || product.Price <= 0 {
		return
	}
	grade, title, lang, pack := m[1], m[3], m[4], m[5]

	cardID, _, found := cr.match(title, pack, lang)
	if !found {
		return
	}

	conditions, err := storeCondition(grade)
	if err != nil {
		cr.printf("unsupported %s condition", grade)
		return
	}

	err = cr.inventory.Add(cardID, &mtgban.InventoryEntry{
		Conditions: conditions,
		Price:      float64(product.Price) * rate,
		Quantity:   product.Stock,
		URL:        productURL + product.ID,
		OriginalID: product.ID,
		InstanceID: product.ID,
	})
	if err != nil {
		cr.printf("%v", err)
	}
}

func (cr *CardRush) add(row BuyingPrice, rate float64) {
	cardID, name, found := cr.match(row.Name, row.PackCode, row.Language)
	if !found {
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
		URL:        buylistURL + "?" + url.Values{"name": {name}}.Encode(),
		OriginalID: fmt.Sprint(row.ID),
	})
	if err != nil {
		cr.printf("%v", err)
	}
}

// Inventory returns what Load collected. See mtgban.Seller.
func (cr *CardRush) Inventory() mtgban.InventoryRecord {
	return cr.inventory
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
	info.InventoryTimestamp = &cr.inventoryDate
	info.BuylistTimestamp = &cr.buylistDate
	info.Game = cr.backend.Game
	return
}
