// Package manaleak scrapes Manaleak.
package manaleak

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

// Manaleak prices Manaleak's Magic singles, both what they sell and what
// they buy. The storefront quotes pounds; every price is converted to
// dollars at the day's rate.
type Manaleak struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	DisableRetail  bool
	DisableBuylist bool

	client *MLClient
	rate   float64

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord
}

// NewScraper returns a scraper for the storefront.
func NewScraper() *Manaleak {
	ml := Manaleak{}
	ml.inventory = mtgban.InventoryRecord{}
	ml.buylist = mtgban.BuylistRecord{}
	ml.client = NewMLClient()
	ml.MaxConcurrency = defaultConcurrency
	return &ml
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (ml *Manaleak) SetConfig(opt mtgban.ScraperOptions) {
	ml.DisableRetail = opt.DisableRetail
	ml.DisableBuylist = opt.DisableBuylist
}

func (ml *Manaleak) printf(format string, a ...any) {
	if ml.LogCallback != nil {
		ml.LogCallback("[ML] "+format, a...)
	}
}

// match resolves the card a row lists. The newer sets carry their TCGplayer
// product id and the older ones their multiverse id, each converted through
// its own id space; either answers by itself. The name and set only speak
// for a row whose id the datastore does not know.
func (ml *Manaleak) match(product MLProduct) (string, error) {
	cardName := product.Name
	foil := false
	etched := false
	switch {
	case strings.HasSuffix(cardName, " - Etched Foil"), strings.HasSuffix(cardName, " - Foil Etched"):
		cardName = strings.TrimSuffix(cardName, " - Etched Foil")
		cardName = strings.TrimSuffix(cardName, " - Foil Etched")
		etched = true
	case strings.HasSuffix(cardName, " - Foil"):
		cardName = strings.TrimSuffix(cardName, " - Foil")
		foil = true
	}

	space, inputID := mtgmatcher.IDSpaceTCGplayer, product.TCGProductID
	if inputID == "" {
		space, inputID = mtgmatcher.IDSpaceMultiverse, product.MultiverseID
	}
	cardID, err := mtgmatcher.MatchID(mtgmatcher.ConvertID(space, inputID), foil, etched)
	if err == nil {
		return cardID, nil
	}

	return mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:    cardName,
		Edition: product.SetName,
		Foil:    foil,
	})
}

func (ml *Manaleak) processProduct(mode string, product MLProduct) {
	// The brand page lists the sealed product and the repacks beside the
	// singles; only a row wearing a card image is one.
	if product.TCGProductID == "" && product.MultiverseID == "" {
		return
	}
	if product.Price == 0 {
		return
	}
	if mode == modeRetail && product.OutOfStock {
		return
	}

	cardID, err := ml.match(product)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return
	} else if err != nil {
		ml.printf("%v", err)
		ml.printf("%s: %q [%s]", mode, product.Name, product.SetName)
		return
	}

	if mode == modeRetail {
		err = ml.inventory.Add(cardID, &mtgban.InventoryEntry{
			Conditions: "NM",
			Price:      product.Price * ml.rate,
			Quantity:   1,
			URL:        product.URL,
		})
	} else {
		var sellPrice, priceRatio float64
		invCards := ml.inventory[cardID]
		for _, invCard := range invCards {
			if invCard.Conditions == "NM" {
				sellPrice = invCard.Price
				break
			}
		}
		if sellPrice > 0 {
			priceRatio = product.Price * ml.rate / sellPrice * 100
		}

		err = ml.buylist.Add(cardID, &mtgban.BuylistEntry{
			Conditions: "NM",
			BuyPrice:   product.Price * ml.rate,
			PriceRatio: priceRatio,
			URL:        product.URL,
		})
	}
	if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
		ml.printf("%s", err.Error())
	}
}

const (
	modeRetail  = "retail"
	modeBuylist = "buylist"
)

func (ml *Manaleak) scrape(ctx context.Context, mode string) error {
	base := inventoryURL
	if mode == modeBuylist {
		base = buylistURL
	}

	// The first page carries the listing's own total, which sizes the
	// fan-out over the rest.
	products, total, err := ml.client.GetPage(ctx, base, 1)
	if err != nil {
		return err
	}
	for _, product := range products {
		ml.processProduct(mode, product)
	}

	pages := (total + pageLimit - 1) / pageLimit
	ml.printf("%s: %d products over %d pages", mode, total, pages)

	pageNums := make([]int, 0, pages)
	for page := 2; page <= pages; page++ {
		pageNums = append(pageNums, page)
	}

	mtgban.WorkerPool(ctx, ml.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- []MLProduct) error {
			products, _, err := ml.client.GetPage(ctx, base, page)
			if err != nil {
				return fmt.Errorf("page %d: %s", page, err.Error())
			}
			results <- products
			return nil
		},
		func(products []MLProduct) {
			for _, product := range products {
				ml.processProduct(mode, product)
			}
		},
		ml.printf,
	)

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (ml *Manaleak) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "GBP")
	if err != nil {
		return err
	}
	ml.rate = rate

	var errs []error

	if !ml.DisableRetail {
		err := ml.scrape(ctx, modeRetail)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		} else {
			ml.inventoryDate = time.Now()
		}
	}

	if !ml.DisableBuylist {
		err := ml.scrape(ctx, modeBuylist)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		} else {
			ml.buylistDate = time.Now()
		}
	}

	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (ml *Manaleak) Inventory() mtgban.InventoryRecord {
	return ml.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (ml *Manaleak) Buylist() mtgban.BuylistRecord {
	return ml.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (ml *Manaleak) Info() (info mtgban.ScraperInfo) {
	info.Name = "Manaleak"
	info.Shorthand = "ML"
	info.Game = mtgban.GameMagic
	info.CountryFlag = "GB"
	info.InventoryTimestamp = &ml.inventoryDate
	info.BuylistTimestamp = &ml.buylistDate
	return
}
