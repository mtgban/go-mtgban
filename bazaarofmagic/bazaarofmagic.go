// Package bazaarofmagic scrapes Bazaar of Magic, a Dutch store pricing in
// euro.
package bazaarofmagic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// A full crawl at two takes about five minutes and draws no 429.
const defaultConcurrency = 2

// buyGrades is the store's published buy-in schedule: what each grade below
// NM pays as a share of the NM price.
var buyGrades = []struct {
	cond   mtgban.Condition
	factor float64
}{
	{mtgban.NM, 1},
	{mtgban.SP, 0.8},
	{mtgban.MP, 0.6},
	{mtgban.HP, 0.4},
}

// BazaarOfMagic prices Bazaar of Magic's singles, both what it sells and
// what it buys.
type BazaarOfMagic struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	backend *matcher.Backend
	client  *http.Client
	rate    float64

	disableRetail  bool
	disableBuylist bool

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord
}

// NewScraper returns a scraper matching against b.
func NewScraper(b *matcher.Backend) *BazaarOfMagic {
	bom := BazaarOfMagic{backend: b}
	bom.inventory = mtgban.InventoryRecord{}
	bom.buylist = mtgban.BuylistRecord{}
	bom.maxConcurrency = defaultConcurrency
	bom.client = newClient()
	return &bom
}

func (bom *BazaarOfMagic) printf(format string, a ...any) {
	if bom.logCallback != nil {
		bom.logCallback("[BoM] "+format, a...)
	}
}

type entry struct {
	cardID string
	inv    *mtgban.InventoryEntry
	buy    *mtgban.BuylistEntry
	nmBuy  float64
}

// match reads one row and resolves it, logging what it cannot resolve.
func (bom *BazaarOfMagic) match(set Set, row Listing) (string, *listing, bool) {
	l, err := preprocess(bom.backend, set, row.Title, row.Finish)
	if err != nil {
		return "", nil, false
	}
	cardID, err := bom.backend.Match(l.card)
	if errors.Is(err, matcher.ErrUnsupported) {
		return "", nil, false
	}
	if err != nil {
		// A foreign copy is priced only where the datastore holds that
		// language's printing.
		if l.card.Language != "" {
			return "", nil, false
		}
		bom.printf("%v", err)
		bom.printf("%q", l.card)
		bom.printf("%s ~ %s (%s)", row.Title, set.Name, set.Code)
		var alias *matcher.AliasingError
		if errors.As(err, &alias) {
			for _, probe := range alias.Probe() {
				co, _ := bom.backend.GetUUID(probe)
				bom.printf("- %s", co)
			}
		}
		return "", nil, false
	}
	return cardID, l, true
}

func (bom *BazaarOfMagic) scrapeRetail(ctx context.Context, sets []Set) error {
	var failed atomic.Int32
	mtgban.WorkerPool(ctx, bom.maxConcurrency, sets,
		func(ctx context.Context, set Set, results chan<- entry) error {
			rows, err := GetListings(ctx, bom.client, singlesURL, SinglesQuery(set.Code))
			for _, row := range rows {
				if row.Price <= 0 {
					continue
				}
				cardID, l, ok := bom.match(set, row)
				if !ok {
					continue
				}
				results <- entry{cardID: cardID, inv: &mtgban.InventoryEntry{
					Conditions: l.cond,
					Price:      row.Price * bom.rate,
					URL:        row.URL,
					OriginalID: row.ID,
				}}
			}
			if err != nil {
				failed.Add(1)
			}
			return err
		},
		func(e entry) {
			err := bom.inventory.Add(e.cardID, e.inv)
			if err != nil {
				bom.printf("%v", err)
			}
		},
		bom.printf,
	)
	bom.inventoryDate = time.Now()
	if failed.Load() > 0 {
		return fmt.Errorf("%d of %d sets failed", failed.Load(), len(sets))
	}
	return nil
}

func (bom *BazaarOfMagic) scrapeBuylist(ctx context.Context, sets []Set) error {
	var failed atomic.Int32
	mtgban.WorkerPool(ctx, bom.maxConcurrency, sets,
		func(ctx context.Context, set Set, results chan<- entry) error {
			rows, err := GetListings(ctx, bom.client, buylistURL, BuylistQuery(set.Code))
			for _, row := range rows {
				if row.Price <= 0 {
					continue
				}
				cardID, l, ok := bom.match(set, row)
				if !ok || l.cond != mtgban.NM {
					continue
				}
				for _, grade := range buyGrades {
					results <- entry{cardID: cardID, nmBuy: row.Price * bom.rate, buy: &mtgban.BuylistEntry{
						Conditions: grade.cond,
						BuyPrice:   row.Price * grade.factor * bom.rate,
						URL:        row.URL,
						OriginalID: row.ID,
					}}
				}
			}
			if err != nil {
				failed.Add(1)
			}
			return err
		},
		func(e entry) {
			var sellPrice float64
			for _, inv := range bom.inventory[e.cardID] {
				if inv.Conditions == mtgban.NM {
					sellPrice = inv.Price
					break
				}
			}
			// Every grade carries the NM ratio, as the schedule is fixed.
			if sellPrice > 0 {
				e.buy.PriceRatio = e.nmBuy / sellPrice * 100
			}
			err := bom.buylist.Add(e.cardID, e.buy)
			if err != nil {
				bom.printf("%v", err)
			}
		},
		bom.printf,
	)
	bom.buylistDate = time.Now()
	if failed.Load() > 0 {
		return fmt.Errorf("%d of %d sets failed", failed.Load(), len(sets))
	}
	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (bom *BazaarOfMagic) SetConfig(opt mtgban.ScraperOptions) {
	bom.disableRetail = opt.DisableRetail
	bom.disableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (bom *BazaarOfMagic) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	bom.rate = rate

	sets, err := GetSets(ctx, bom.client)
	if err != nil {
		return err
	}
	bom.printf("Found %d sets", len(sets))

	var errs []error
	if !bom.disableRetail {
		err := bom.scrapeRetail(ctx, sets)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err))
		}
	}
	if !bom.disableBuylist {
		err := bom.scrapeBuylist(ctx, sets)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, err))
		}
	}
	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (bom *BazaarOfMagic) Inventory() mtgban.InventoryRecord {
	return bom.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (bom *BazaarOfMagic) Buylist() mtgban.BuylistRecord {
	return bom.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (bom *BazaarOfMagic) Info() (info mtgban.ScraperInfo) {
	info.Name = "Bazaar of Magic"
	info.Shorthand = "BoM"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &bom.inventoryDate
	info.BuylistTimestamp = &bom.buylistDate
	// Store credit pays 1.3 times the cash price, by the store's own
	// example of 5.00 euro or 6.50 in credit.
	info.CreditMultiplier = 1.3
	// The listings show a price but no stock count.
	info.NoQuantityInventory = true
	info.Game = bom.backend.Game
	return
}
