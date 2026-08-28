// Package vegassingles scrapes Vegas Singles.
package vegassingles

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

// The games this scraper covers, as the storefront names its product lines.
// Lorcana, Flesh And Blood and YuGiOh are lines the store knows but holds no
// buylist for today, so they are not wired up.
const (
	GameMagic     = "Magic: the Gathering"
	GameRiftbound = "Riftbound"
	GameOnePiece  = "One Piece"
	GamePokemon   = "Pokemon"
)

var conditionMap = map[string]string{
	"Near Mint":         "NM",
	"Lightly Played":    "SP",
	"Moderately Played": "MP",
	"Heavily Played":    "HP",
	"Damaged":           "PO",
}

func buildProductSlug(displayName string) string {
	slug := strings.ToLower(displayName)
	slug = strings.ReplaceAll(slug, "(", "")
	slug = strings.ReplaceAll(slug, ")", "")
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, " - ", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

// Vegassingles prices Vegas Singles' stock of one game.
type Vegassingles struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	client *VSClient
	game   string

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord
}

// NewScraper returns a scraper for one game.
func NewScraper(game string) *Vegassingles {
	vs := Vegassingles{}
	vs.inventory = mtgban.InventoryRecord{}
	vs.buylist = mtgban.BuylistRecord{}
	vs.client = NewVSClient(game)
	vs.game = game
	vs.MaxConcurrency = defaultConcurrency
	return &vs
}

func (vs *Vegassingles) printf(format string, a ...any) {
	if vs.LogCallback != nil {
		vs.LogCallback("[VS] "+format, a...)
	}
}

func (vs *Vegassingles) processProduct(product VSProduct) error {
	theCard, err := preprocess(product, vs.game)
	if err != nil {
		// Name the product, the way the failure below already does. A
		// reason alone says a listing was dropped without saying which,
		// and a bucket nobody can read is a bucket nobody empties
		return fmt.Errorf("%s %q: %w", product.ID, product.DisplayName, err)
	}

	cardID, err := mtgmatcher.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		vs.printf("%v", err)
		vs.printf("%s: %q", product.ID, product.DisplayName)
		return nil
	}

	// Build buylist URL
	u, _ := url.Parse("https://buylist.vegas.singles/retailer/buylist")
	q := u.Query()
	q.Set("product_line", vs.game)
	q.Set("q", product.DisplayName)
	q.Set("sort", "Relevance")
	u.RawQuery = q.Encode()
	buylistLink := u.String()

	// Build retail product URL
	retailLink := "https://vegas.singles/products/" + buildProductSlug(product.DisplayName)

	// vegas.singles lists a card under more than one product - the same
	// display name, the same prices, the same stock - so the second
	// product's variants arrive as exact duplicates of the first's. The
	// record keeps the one it has and drops the new one, which is the right
	// answer; reporting each as an error only buried the run's real ones
	// under a thousand lines of it.
	for _, variant := range product.VariantInfo {
		if variant.OfferPrice == 0 {
			continue
		}

		cond, found := conditionMap[variant.Title]
		if !found {
			vs.printf("unknown condition: %s", variant.Title)
			continue
		}

		var priceRatio float64
		if product.Price > 0 {
			priceRatio = variant.OfferPrice / product.Price * 100
		}

		err = vs.buylist.Add(cardID, &mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   variant.OfferPrice,
			PriceRatio: priceRatio,
			URL:        buylistLink,
			OriginalID: strconv.FormatInt(product.ProductID, 10),
			InstanceID: strconv.FormatInt(variant.ID, 10),
		})
		if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
			vs.printf("%d: %s", product.ProductID, err.Error())
		}
	}

	// Process retail variants (from variant_info)
	for _, variant := range product.RetailVariantInfo {
		// A condition the store holds none of is not on sale whatever number
		// hangs off it: Add() treats a zero quantity as unsaid and writes 1,
		// so the row would be published as a copy in stock.
		if variant.Price == 0 || variant.InventoryQuantity < 1 {
			continue
		}

		cond, found := conditionMap[variant.Title]
		if !found {
			continue
		}

		err = vs.inventory.Add(cardID, &mtgban.InventoryEntry{
			Conditions: cond,
			Price:      variant.Price,
			Quantity:   variant.InventoryQuantity,
			URL:        retailLink,
			OriginalID: strconv.FormatInt(product.ProductID, 10),
			InstanceID: variant.SKU,
		})
		if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
			vs.printf("%d: %s", product.ProductID, err.Error())
		}
	}

	return nil
}

func (vs *Vegassingles) scrape(ctx context.Context) error {
	totalPages, err := vs.client.getCount(ctx, "")
	if err != nil {
		return err
	}
	vs.printf("Total pages: %d", totalPages)

	// One ordering cannot see past the storefront's result window, so
	// coverage widens as far as each line proves it needs: the alphabet
	// forward, then backward, then rarity by rarity when the backward pass
	// still found products the forward one could not reach, each slice
	// walked backward too only when its own feed was cut off. Every
	// decision reads what the crawl observed rather than where the window
	// sat last measured, so the storefront resizing it resizes the crawl.
	// The seen filter keeps the overlap from being processed twice, and
	// the rarity vocabulary is whatever the wider passes came across, so a
	// slice never has to be known ahead of time.
	seen := map[string]bool{}
	rarities := map[string]bool{}
	vs.crawl(ctx, sortForward, "", totalPages, seen, rarities)
	before := len(seen)
	vs.crawl(ctx, sortBackward, "", totalPages, seen, rarities)
	if len(seen) > before {
		names := make([]string, 0, len(rarities))
		for rarity := range rarities {
			names = append(names, rarity)
		}
		sort.Strings(names)
		for _, rarity := range names {
			// The estimate is as capped for a slice as for the whole line,
			// but it still fans the bulk of the fetch out; the tail-walk
			// covers whatever it undersells.
			hint, err := vs.client.getCount(ctx, rarity)
			if err != nil {
				return err
			}
			cut := vs.crawl(ctx, sortForward, rarity, hint, seen, rarities)
			if cut {
				cut = vs.crawl(ctx, sortBackward, rarity, hint, seen, rarities)
				if cut {
					vs.printf("rarity %q exceeds both crawl windows, its middle is unreachable", rarity)
				}
			}
		}
	}
	vs.printf("Processed %d products", len(seen))

	vs.inventoryDate = time.Now()
	vs.buylistDate = time.Now()

	return nil
}

// crawl fetches every page of one ordering, optionally narrowed to a rarity,
// processing what it has not seen yet and noting the rarities it passes. It
// reports whether the feed was cut off rather than finished: a catalog that
// really ends leaves a ragged last page, so ending flush on a full one means
// the storefront's result window ran out with products still unserved. That
// reads the cut wherever the window happens to sit, at the cost of a false
// positive when a catalog is an exact multiple of the page size, which only
// spends a redundant pass.
//
// How full a full page is comes from the pages this crawl was served rather
// than from a number written here: every page but the last carries the same
// count, so the widest one seen is the size the storefront is serving. A
// storefront that resizes its page is then read as it is, where a constant
// would make every last page look ragged, report no cut ever, and quietly stop
// the passes that widen the crawl.
//
// A page that will not load never fails the crawl: inside the fanned-out range
// the pool logs it and carries on, and past it the walk stops and reads the
// feed as unfinished. So a run's inventory is never thrown away over one page,
// and there is nothing here for a caller to handle.
func (vs *Vegassingles) crawl(ctx context.Context, sortDir, rarity string, hint int, seen, rarities map[string]bool) bool {
	pageNums := make([]int, hint)
	for i := range pageNums {
		pageNums[i] = i + 1
	}

	lastPage := 0
	lastPageLen := 0
	fullPageLen := 0
	pages := 0
	sawEmpty := false
	consume := func(result pageResult) {
		if len(result.products) == 0 {
			sawEmpty = true
			return
		}
		pages++
		if len(result.products) > fullPageLen {
			fullPageLen = len(result.products)
		}
		if result.page > lastPage {
			lastPage = result.page
			lastPageLen = len(result.products)
		}
		for _, product := range result.products {
			if seen[product.ID] {
				continue
			}
			seen[product.ID] = true
			if product.ProductData.Rarity != "" {
				rarities[product.ProductData.Rarity] = true
			}
			err := vs.processProduct(product)
			if err != nil {
				vs.printf("process error: %s", err.Error())
			}
		}
	}

	mtgban.WorkerPool(ctx, vs.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- pageResult) error {
			products, err := vs.client.getPage(ctx, page, sortDir, rarity)
			if err != nil {
				return fmt.Errorf("page %d: %s", page, err.Error())
			}
			results <- pageResult{page: page, products: products}
			return nil
		},
		consume,
		vs.printf,
	)

	// The page count the storefront reports is a capped estimate, so the
	// feed does not end until a page comes back empty: keep walking past
	// the fanned-out range until one does. The bound is not a page the
	// crawl expects to reach; it only keeps a misbehaving feed finite.
	for page := hint + 1; !sawEmpty && page <= maxPages; page++ {
		products, err := vs.client.getPage(ctx, page, sortDir, rarity)
		if err != nil {
			// The same failure inside the fanned-out range is logged and
			// the page skipped, and this one used to throw away the whole
			// run's inventory and buylist instead. It cannot be skipped
			// and stay finite - only an empty page ends this walk - so the
			// walk stops here and the feed is reported as unfinished,
			// which is what a page nobody could read leaves it.
			vs.printf("page %d: %s", page, err.Error())
			return true
		}
		consume(pageResult{page: page, products: products})
	}

	// One page of its own says nothing: it is both the widest and the last,
	// and a result window running out inside a single page is not what the
	// storefront does.
	return pages > 1 && lastPageLen == fullPageLen
}

type pageResult struct {
	page     int
	products []VSProduct
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (vs *Vegassingles) Load(ctx context.Context) error {
	return vs.scrape(ctx)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (vs *Vegassingles) Inventory() mtgban.InventoryRecord {
	return vs.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (vs *Vegassingles) Buylist() mtgban.BuylistRecord {
	return vs.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (vs *Vegassingles) Info() (info mtgban.ScraperInfo) {
	info.Name = "Vegas Singles"
	info.Shorthand = "VS"
	info.InventoryTimestamp = &vs.inventoryDate
	info.BuylistTimestamp = &vs.buylistDate
	switch vs.game {
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	}
	return
}
