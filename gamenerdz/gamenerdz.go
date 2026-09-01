// Package gamenerdz scrapes Game Nerdz.
package gamenerdz

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

// conditionMap spells the condition a buylist variant's title names. The
// storefront's offers hang off one untitled variant per product today, and
// an untitled variant names none: the grade is written into the display
// name instead, and the empty entry is what says to read it there. A titled
// variant spells its own, the way the platform's other stores already do.
var conditionMap = map[string]string{
	"Default Title":     "",
	"Near Mint":         "NM",
	"Lightly Played":    "SP",
	"Moderately Played": "MP",
	"Heavily Played":    "HP",
	"Damaged":           "PO",
}

// gradeTag is the grade this storefront writes at the end of a display name
// when it lists a copy that is not near mint: "Legions Foil(MP)", "Revised
// Edition (MP)".
var gradeTag = regexp.MustCompile(`\(([A-Z]{1,2})\)$`)

// gradeMap spells those grades as the conditions mtgban keeps. It is a
// closed list on purpose: a name ending in some other bracketed capitals is
// a name, not a grade.
var gradeMap = map[string]string{
	"NM": "NM",
	"LP": "SP",
	"MP": "MP",
	"HP": "HP",
	"D":  "PO",
}

// grade reads the grade a display name ends in. A name ending in none, or
// in bracketed capitals that are not one, is the near mint this storefront
// leaves unwritten.
func grade(displayName string) string {
	match := gradeTag.FindStringSubmatch(displayName)
	if match == nil {
		return "NM"
	}
	cond, found := gradeMap[match[1]]
	if !found {
		return "NM"
	}
	return cond
}

// The games this scraper covers, as the storefront names its product lines.
// YuGiOh and Riftbound are lines the store knows but holds nothing of today,
// so they are not wired up.
const (
	GameMagic         = "Magic: the Gathering"
	GameLorcana       = "Lorcana"
	GamePokemon       = "Pokemon"
	GameOnePiece      = "One Piece"
	GameFleshAndBlood = "Flesh And Blood"
)

// Gamenerdz prices Game Nerdz's stock of one game. The storefront's two
// faces answer from one search endpoint in two modes, and neither list is a
// subset of the other - the store buys cards it does not retail and retails
// cards it does not buy - so retail and buylist are each their own crawl.
type Gamenerdz struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	DisableRetail  bool
	DisableBuylist bool

	client *GNClient
	game   string

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord
}

// NewScraper returns a scraper for one game.
func NewScraper(game string) *Gamenerdz {
	gn := Gamenerdz{}
	gn.inventory = mtgban.InventoryRecord{}
	gn.buylist = mtgban.BuylistRecord{}
	gn.client = NewGNClient(game)
	gn.game = game
	gn.MaxConcurrency = defaultConcurrency
	return &gn
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (gn *Gamenerdz) SetConfig(opt mtgban.ScraperOptions) {
	gn.DisableRetail = opt.DisableRetail
	gn.DisableBuylist = opt.DisableBuylist
}

func (gn *Gamenerdz) printf(format string, a ...any) {
	if gn.LogCallback != nil {
		gn.LogCallback("[GN] "+format, a...)
	}
}

func (gn *Gamenerdz) processProduct(mode string, product GNProduct) error {
	theCard, err := preprocess(product, gn.game)
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
		gn.printf("%v", err)
		gn.printf("%s: %q", product.ID, product.DisplayName)
		return nil
	}

	if mode == modeBuylist {
		u, _ := url.Parse("https://buylist.gamenerdz.com/retailer/buylist")
		q := u.Query()
		q.Set("product_line", gn.game)
		q.Set("q", product.DisplayName)
		q.Set("sort", "Relevance")
		u.RawQuery = q.Encode()
		buylistLink := u.String()

		for _, variant := range product.BuyVariants {
			if variant.OfferPrice == 0 {
				continue
			}

			cond, found := conditionMap[variant.Title]
			if !found {
				gn.printf("unknown condition: %s", variant.Title)
				continue
			}
			if cond == "" {
				cond = grade(product.DisplayName)
			}

			var priceRatio float64
			if product.Price > 0 {
				priceRatio = variant.OfferPrice / product.Price * 100
			}

			err = gn.buylist.Add(cardID, &mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   variant.OfferPrice,
				PriceRatio: priceRatio,
				URL:        buylistLink,
				OriginalID: strconv.FormatInt(product.ProductID, 10),
				InstanceID: strconv.FormatInt(variant.ID, 10),
			})
			if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
				gn.printf("%d: %s", product.ProductID, err.Error())
			}
		}
		return nil
	}

	retailLink := "https://www.gamenerdz.com/search.php?search_query=" +
		url.QueryEscape(product.DisplayName)

	for _, variant := range product.RetailVariants {
		if variant.Price == 0 || variant.InventoryLevel < 1 || variant.PurchasingDisabled {
			continue
		}

		err = gn.inventory.Add(cardID, &mtgban.InventoryEntry{
			Conditions: grade(product.DisplayName),
			Price:      variant.Price,
			Quantity:   variant.InventoryLevel,
			URL:        retailLink,
			OriginalID: strconv.FormatInt(product.ProductID, 10),
			InstanceID: variant.SKU,
		})
		if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
			gn.printf("%d: %s", product.ProductID, err.Error())
		}
	}

	return nil
}

// crawlState is what one mode's passes accumulate together: the products any
// pass already processed, and the rarity and finish vocabularies harvested
// from the rows themselves, so a narrower slice never has to be known ahead
// of time.
type crawlState struct {
	seen     map[string]bool
	rarities map[string]bool
	finishes map[string]bool
}

func sorted(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sliceAxis is one way a slice too deep for the crawl windows can be split
// further: the query parameter that narrows it, and where its values come
// from once the wider passes have run.
type sliceAxis struct {
	param  string
	values func(ctx context.Context, state *crawlState) ([]string, error)
}

func (gn *Gamenerdz) axes() []sliceAxis {
	fromState := func(pick func(*crawlState) map[string]bool) func(context.Context, *crawlState) ([]string, error) {
		return func(_ context.Context, state *crawlState) ([]string, error) {
			return sorted(pick(state)), nil
		}
	}
	return []sliceAxis{
		{"rarity", fromState(func(s *crawlState) map[string]bool { return s.rarities })},
		{"finish", fromState(func(s *crawlState) map[string]bool { return s.finishes })},
		// The set list comes from the storefront rather than the harvest:
		// it is the one vocabulary wide enough that a slice of it can hold
		// products every wider pass ran out of window before reaching.
		{"set_name", func(ctx context.Context, _ *crawlState) ([]string, error) {
			return gn.client.getSets(ctx)
		}},
	}
}

func (gn *Gamenerdz) scrape(ctx context.Context, mode string) error {
	// One ordering cannot see past the storefront's result window, so
	// coverage widens as far as each slice proves it needs: the alphabet
	// forward, then backward only when the forward pass was cut off, and a
	// slice whose own two windows both ran out split along the next axis -
	// rarity, then finish, then set, each vocabulary read from the crawl
	// itself or the storefront's own filters. The store's Magic buylist
	// runs deep enough to need all three. Every decision reads what the
	// crawl observed rather than where the window sat last measured, so
	// the storefront resizing it resizes the crawl.
	state := &crawlState{
		seen:     map[string]bool{},
		rarities: map[string]bool{},
		finishes: map[string]bool{},
	}
	err := gn.widen(ctx, mode, map[string]string{}, gn.axes(), state)
	if err != nil {
		return err
	}
	gn.printf("%s processed %d products", mode, len(state.seen))

	return nil
}

// widen crawls one slice of the feed both ways, and when both windows ran
// out with products still unserved, splits the slice along the next axis and
// widens each piece the same way. A slice no axis is left to split reports
// its unreachable middle instead of guessing at one.
//
// A last page can be flush by chance - a feed holding an exact multiple of
// the page size ends looking cut - so looking cut is never enough on its
// own: only a backward pass that reached products the forward one could not
// proves something sat past the window. A backward pass that found nothing
// new proves the opposite, whatever the page edges looked like, and the
// slice is done.
func (gn *Gamenerdz) widen(ctx context.Context, mode string, filters map[string]string, axes []sliceAxis, state *crawlState) error {
	hint, err := gn.client.getCount(ctx, mode, filters)
	if err != nil {
		return err
	}

	if !gn.crawl(ctx, mode, sortForward, filters, hint, state) {
		return nil
	}
	before := len(state.seen)
	if !gn.crawl(ctx, mode, sortBackward, filters, hint, state) {
		return nil
	}
	if len(state.seen) == before {
		return nil
	}
	if len(axes) == 0 {
		gn.printf("%s slice %v exceeds both crawl windows, its middle is unreachable", mode, filters)
		return nil
	}

	gn.printf("%s slice %v cut both ways, splitting by %s", mode, filters, axes[0].param)
	values, err := axes[0].values(ctx, state)
	if err != nil {
		return err
	}
	for _, value := range values {
		sub := maps.Clone(filters)
		sub[axes[0].param] = value
		err := gn.widen(ctx, mode, sub, axes[1:], state)
		if err != nil {
			return err
		}
	}
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
// count, so the widest one seen is the size the storefront is serving - and
// this storefront serves differently sized pages per mode.
//
// The page count the storefront reports is only a fan-out hint: both modes
// understate their feeds, by as much as half, so the walk keeps fanning out
// rounds of pages until one of them comes back empty, which is the only way
// this feed ends. The bound is not a page the crawl expects to reach; it
// only keeps a misbehaving feed finite.
//
// A page that will not load never fails the crawl: the pool logs it and
// carries on, and the walk still ends at the first empty page behind it. So a
// run's inventory is never thrown away over one page, and there is nothing
// here for a caller to handle.
func (gn *Gamenerdz) crawl(ctx context.Context, mode, sortDir string, filters map[string]string, hint int, state *crawlState) bool {
	if hint < 1 {
		hint = 1
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
			if state.seen[product.ID] {
				continue
			}
			state.seen[product.ID] = true
			if product.ProductData.Rarity != "" {
				state.rarities[product.ProductData.Rarity] = true
			}
			if product.SelectedFinish != "" {
				state.finishes[product.SelectedFinish] = true
			}
			err := gn.processProduct(mode, product)
			if err != nil {
				gn.printf("process error: %s", err.Error())
			}
		}
	}

	for start := 1; !sawEmpty && start <= maxPages; start += hint {
		pageNums := make([]int, 0, hint)
		for page := start; page < start+hint && page <= maxPages; page++ {
			pageNums = append(pageNums, page)
		}

		mtgban.WorkerPool(ctx, gn.MaxConcurrency, pageNums,
			func(ctx context.Context, page int, results chan<- pageResult) error {
				products, err := gn.client.getPage(ctx, mode, page, sortDir, filters)
				if err != nil {
					return fmt.Errorf("page %d: %s", page, err.Error())
				}
				results <- pageResult{page: page, products: products}
				return nil
			},
			consume,
			gn.printf,
		)
	}

	// One page of its own says nothing: it is both the widest and the last,
	// and a result window running out inside a single page is not what the
	// storefront does.
	return pages > 1 && lastPageLen == fullPageLen
}

type pageResult struct {
	page     int
	products []GNProduct
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (gn *Gamenerdz) Load(ctx context.Context) error {
	var errs []error

	if !gn.DisableRetail {
		err := gn.scrape(ctx, modeRetail)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		} else {
			gn.inventoryDate = time.Now()
		}
	}

	if !gn.DisableBuylist {
		err := gn.scrape(ctx, modeBuylist)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		} else {
			gn.buylistDate = time.Now()
		}
	}

	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (gn *Gamenerdz) Inventory() mtgban.InventoryRecord {
	return gn.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (gn *Gamenerdz) Buylist() mtgban.BuylistRecord {
	return gn.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (gn *Gamenerdz) Info() (info mtgban.ScraperInfo) {
	info.Name = "Game Nerdz"
	info.Shorthand = "GN"
	info.InventoryTimestamp = &gn.inventoryDate
	info.BuylistTimestamp = &gn.buylistDate
	// The storefront quotes its buylist in cash and pays 25% over it in
	// store credit, a ratio its own feed restates on every offer.
	info.CreditMultiplier = 1.25
	switch gn.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	}
	return
}
