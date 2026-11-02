package mtgban

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
)

// Stock is what a scraper collects: the inventory and buylist it publishes,
// when each was last loaded, and which of the two its caller asked for. A
// scraper whose retail and buylist load apart embeds it, built by NewStock,
// for its Inventory, Buylist and SetConfig, and loads through LoadHalves.
type Stock struct {
	// InventoryDate and BuylistDate are when each half last loaded in
	// full; a scraper's Info points its timestamps at them.
	InventoryDate time.Time
	BuylistDate   time.Time

	inventory InventoryRecord
	buylist   BuylistRecord
	config    ScraperOptions
}

// NewStock returns a Stock with both records empty.
func NewStock() Stock {
	return Stock{inventory: InventoryRecord{}, buylist: BuylistRecord{}}
}

// Inventory returns what Load collected. See Seller.
func (s *Stock) Inventory() InventoryRecord {
	return s.inventory
}

// Buylist returns what Load collected. See Vendor.
func (s *Stock) Buylist() BuylistRecord {
	return s.buylist
}

// SetConfig records which halves Load runs. See ScraperConfig.
func (s *Stock) SetConfig(opt ScraperOptions) {
	s.config = opt
}

// Config returns what SetConfig recorded.
func (s *Stock) Config() ScraperOptions {
	return s.config
}

// LoadHalves runs retail and then buylist, skipping a half that is nil or
// that SetConfig turned off. A half that fails is wrapped in
// ErrInventoryLoad or ErrBuylistLoad, and the failures are joined; a half
// that succeeds stamps its date.
func (s *Stock) LoadHalves(ctx context.Context, retail, buylist func(context.Context) error) error {
	var errs []error

	if retail != nil && !s.config.DisableRetail {
		err := retail(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInventoryLoad, err))
		} else {
			s.InventoryDate = time.Now()
		}
	}

	if buylist != nil && !s.config.DisableBuylist {
		err := buylist(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrBuylistLoad, err))
		} else {
			s.BuylistDate = time.Now()
		}
	}

	return errors.Join(errs...)
}

// Result is what a WorkerPool worker found for one printing: a retail
// listing, a buylist offer, or both.
type Result struct {
	CardID    string
	Inventory *InventoryEntry
	Buylist   *BuylistEntry
}

// Collect returns a WorkerPool consume func that adds each Result to the
// stock and logs every entry a record refuses through printf.
func (s *Stock) Collect(printf LogCallbackFunc) func(Result) {
	return func(r Result) {
		if r.Inventory != nil {
			err := s.inventory.Add(r.CardID, r.Inventory)
			if err != nil {
				printf("%v", err)
			}
		}
		if r.Buylist != nil {
			err := s.buylist.Add(r.CardID, r.Buylist)
			if err != nil {
				printf("%v", err)
			}
		}
	}
}

// TaggedLog returns a printf that prefixes each line with "[tag] " and hands
// it to fn, and drops it while fn is nil.
func TaggedLog(tag string, fn LogCallbackFunc) LogCallbackFunc {
	return func(format string, a ...any) {
		if fn != nil {
			fn("["+tag+"] "+format, a...)
		}
	}
}

// LogAliasing logs one "- card" line per printing an AliasingError names,
// and nothing for any other error.
func LogAliasing(printf LogCallbackFunc, b *matcher.Backend, err error) {
	var alias *matcher.AliasingError
	if !errors.As(err, &alias) {
		return
	}
	for _, probe := range alias.Probe() {
		card, _ := b.GetUUID(probe)
		printf("- %s", card)
	}
}

// ForGame returns a vendor's own value for game out of table, refusing a
// game the table does not hold.
func ForGame[V any](table map[matcher.Game]V, game matcher.Game) (V, error) {
	value, ok := table[game]
	if !ok {
		return value, fmt.Errorf("unsupported game %q", game)
	}
	return value, nil
}
