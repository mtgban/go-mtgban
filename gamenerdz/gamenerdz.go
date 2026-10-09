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
	"strings"
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
// name instead, and the empty entry is what says to read it there. The
// platform writes that untitled variant two ways - the placeholder title it
// fills in, and a title of null or none at all - and both are the same
// offer. A titled variant spells its own, the way the platform's other
// stores already do.
var conditionMap = map[string]mtgban.Condition{
	"":                  "",
	"Default Title":     "",
	"Near Mint":         mtgban.NM,
	"Lightly Played":    mtgban.SP,
	"Moderately Played": mtgban.MP,
	"Heavily Played":    mtgban.HP,
	"Damaged":           mtgban.PO,
}

// gradeTag is the grade this storefront writes at the end of a display name
// when it lists a copy that is not near mint: "Legions Foil(MP)", "Revised
// Edition (MP)", "Base Holofoil (DMG)".
var gradeTag = regexp.MustCompile(`\(([A-Z]{1,3})\)$`)

// gradeMap spells those grades as the conditions mtgban keeps. It is a
// closed list on purpose: a name ending in some other bracketed capitals is
// a name, not a grade.
var gradeMap = map[string]mtgban.Condition{
	"NM":  mtgban.NM,
	"LP":  mtgban.SP,
	"MP":  mtgban.MP,
	"HP":  mtgban.HP,
	"D":   mtgban.PO,
	"DMG": mtgban.PO,
}

// grade reads the grade a display name ends in. A name ending in none, or
// in bracketed capitals that are not one, is the near mint this storefront
// leaves unwritten.
func grade(displayName string) mtgban.Condition {
	match := gradeTag.FindStringSubmatch(displayName)
	if match == nil {
		return mtgban.NM
	}
	cond, found := gradeMap[match[1]]
	if !found {
		return mtgban.NM
	}
	return cond
}

// The games this scraper covers, as the storefront names its product lines.
// YuGiOh and Riftbound are lines the store knows but holds nothing of today,
// so they are not wired up. Flesh and Blood is left out too: the store retired
// its buylist for it, and though the feed still answers and still looks like
// data, the links it hands back no longer work, which makes every listing
// inactionable, so the game is not kept for a retail
// side alone.
const (
	GameMagic    = "Magic: the Gathering"
	GameLorcana  = "Lorcana"
	GamePokemon  = "Pokemon"
	GameOnePiece = "One Piece"
)

// gnGames is what NewScraper is built through: it names the product line a
// game is sold under, and a game named nowhere here is not one Game Nerdz is
// read for. The lines themselves stay public, since NewGNClient takes one.
var gnGames = map[mtgmatcher.Game]string{
	mtgmatcher.GameMagic:    GameMagic,
	mtgmatcher.GameLorcana:  GameLorcana,
	mtgmatcher.GamePokemon:  GamePokemon,
	mtgmatcher.GameOnePiece: GameOnePiece,
}

// Gamenerdz prices Game Nerdz's stock of one game. The storefront's two
// faces answer from one search endpoint in two modes, and neither list is a
// subset of the other - the store buys cards it does not retail and retails
// cards it does not buy - so retail and buylist are each their own crawl.
type Gamenerdz struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	disableRetail  bool
	disableBuylist bool

	client  *GNClient
	backend *mtgmatcher.Backend
	line    string

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord

	// retailIDs holds the TCGplayer id each retail product carries, by the
	// storefront's own product id. The buylist record of the same product
	// shares that id and carries no TCGplayer id of its own.
	retailIDs map[string]int64
}

// NewScraper returns a scraper for the datastore's game.
func NewScraper(b *mtgmatcher.Backend) (*Gamenerdz, error) {
	game := b.Game
	line, ok := gnGames[game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	gn := Gamenerdz{}
	gn.inventory = mtgban.InventoryRecord{}
	gn.buylist = mtgban.BuylistRecord{}
	gn.client = NewGNClient(line)
	gn.backend = b
	gn.line = line
	gn.maxConcurrency = defaultConcurrency
	gn.retailIDs = map[string]int64{}
	return &gn, nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (gn *Gamenerdz) SetConfig(opt mtgban.ScraperOptions) {
	gn.disableRetail = opt.DisableRetail
	gn.disableBuylist = opt.DisableBuylist
}

func (gn *Gamenerdz) printf(format string, a ...any) {
	if gn.logCallback != nil {
		gn.logCallback("[GN] "+format, a...)
	}
}

func (gn *Gamenerdz) processProduct(mode string, product GNProduct) error {
	cardID, err := gn.resolveProduct(mode, product)
	if err != nil {
		return err
	}
	if cardID == "" {
		return nil
	}

	if mode == modeBuylist {
		u, _ := url.Parse("https://buylist.gamenerdz.com/retailer/buylist")
		q := u.Query()
		q.Set("product_line", gn.line)
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
				gn.printf("unsupported %s condition", variant.Title)
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

// staleTCGIDs pairs the TCGplayer ids the retail feed still carries after
// TCGplayer deleted the product with the live id of the same card: Torgal's
// MagicFest listing, whose twin under the Las Vegas 2025 name carries 638804.
var staleTCGIDs = map[int64]int64{
	638819: 638804,
}

// resolveProduct names the printing a product is. The retail feed carries
// the catalog's own TCGplayer id for nearly every Magic product, and it
// answers first: the display name is the storefront's own wording, and where
// the two disagree the id is right - a name copied from another card, a
// promo pack printing named as the plain one. The buylist feed carries no
// id, but its product shares the storefront's id with the retail product of
// the same card, so it answers with the id that product carried, where that
// id names the card the display does. Anything the id does not place is read
// by its wording, and the retail ids are what that reading is measured
// against. An empty id under a nil error is a product the catalog does not
// carry.
func (gn *Gamenerdz) resolveProduct(mode string, product GNProduct) (string, error) {
	etched := gn.backend.Game == mtgmatcher.GameMagic && saysEtched(product)
	tcgID := product.ProductData.TCGProductID
	if mode == modeRetail {
		gn.retailIDs[product.ID] = tcgID
	} else {
		tcgID = gn.retailIDs[product.ID]
	}
	live, found := staleTCGIDs[tcgID]
	if found {
		tcgID = live
	}
	if gn.backend.Game == mtgmatcher.GameMagic && tcgID != 0 {
		foil := strings.EqualFold(product.SelectedFinish, "foil") || nameSaysFoil(product.DisplayName)
		cardID, err := gn.backend.MatchID(strconv.FormatInt(tcgID, 10), foil, etched)
		if err == nil && (mode == modeRetail || gn.namesCard(cardID, product.DisplayName, magicName(gn.backend, product.DisplayName))) {
			return cardID, nil
		}
	}

	theCard, err := preprocess(gn.backend, product, gn.backend.Game)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return "", nil
	}
	if err != nil {
		// Name the product, the way the failure below already does. A
		// reason alone says a listing was dropped without saying which,
		// and a bucket nobody can read is a bucket nobody empties
		return "", fmt.Errorf("%s %q: %w", product.ID, product.DisplayName, err)
	}

	foil := theCard.Foil

	cardID, err := gn.backend.Match(theCard)
	if err != nil && gn.backend.Game == mtgmatcher.GamePokemon && !errors.Is(err, mtgmatcher.ErrUnsupported) {
		retried, found := retryPokemon(gn.backend, product, theCard)
		if found {
			cardID, err = retried, nil
		}
	}
	// The other games answer by wording, and where it turns a product down
	// the id may still place it, as long as it names the same card and does
	// not contradict the number the wording wrote.
	if err != nil && gn.backend.Game != mtgmatcher.GameMagic && !errors.Is(err, mtgmatcher.ErrUnsupported) && tcgID != 0 {
		idCard, idErr := gn.backend.MatchIDFinish(strconv.FormatInt(tcgID, 10), product.SelectedFinish)
		if idErr == nil && gn.namesCard(idCard, product.DisplayName, theCard.Name) && !gn.numberConflicts(idCard, theCard.Variation) {
			cardID, err = idCard, nil
		}
	}
	if err == nil && gn.backend.Game == mtgmatcher.GameOnePiece && onePieceCodeless(product.DisplayName) && !landedOnLeader(gn.backend, cardID) {
		err = errors.New("no card code in display name, and no promo leader answers it")
	}
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return "", nil
	} else if err != nil {
		gn.printf("%v", err)
		gn.printf("%s: %q", product.ID, product.DisplayName)
		return "", nil
	}

	if gn.backend.Game == mtgmatcher.GamePokemon && gn.leftShelf(cardID, product) {
		return "", nil
	}

	// The finish a listing names has to be one the printing was sold in.
	// This storefront mints a "-F-" sku beside the plain one whether or not
	// the set ever printed a foil, and where it did not both listings answer
	// with the single printing there is - the minted one carrying a price of
	// its own, which the buylist keeps whenever it is the higher of the two.
	// Nothing else was printed to move it to, so let it go.
	if gn.backend.Game == mtgmatcher.GameMagic && !finishPrinted(gn.backend, cardID, foil, etched) {
		return "", nil
	}
	// A "(N)" promo number names its printing without saying its finish, so
	// a plain listing can land on a printing that was only ever made foil.
	if gn.backend.Game == mtgmatcher.GameLorcana && !foil && !finishPrinted(gn.backend, cardID, false, false) {
		return "", nil
	}
	return cardID, nil
}

// namesCard reports whether a printing is the card a display name writes: its
// name is in the display, or the card name read off the display is the front
// of it. A flavor name before the dash ("Torgal, Clive's Companion -
// Yoshimaru, Ever Faithful") and a double-faced card's other face both pass.
func (gn *Gamenerdz) namesCard(cardID, displayName, cardName string) bool {
	co, err := gn.backend.GetUUID(cardID)
	if err != nil {
		return false
	}
	landed := mtgmatcher.Normalize(co.Name)
	name := mtgmatcher.Normalize(cardName)
	return strings.Contains(mtgmatcher.Normalize(displayName), landed) ||
		name != "" && strings.Contains(landed, name)
}

// numberConflicts reports whether the number a variation opens with is not the
// printing's own. Only the digits are compared, so "95" and "095a" agree and
// "98" and "105" do not; a variation or a printing with no digits says nothing.
func (gn *Gamenerdz) numberConflicts(cardID, variation string) bool {
	co, err := gn.backend.GetUUID(cardID)
	if err != nil {
		return false
	}
	fields := strings.Fields(variation)
	if len(fields) == 0 {
		return false
	}
	written, _, _ := strings.Cut(fields[0], "/")
	wrote := numberDigits(written)
	return wrote != "" && numberDigits(co.Number) != "" && wrote != numberDigits(co.Number)
}

// numberDigits is the digits of a collector number without their leading zeros.
func numberDigits(number string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, number)
	return strings.TrimLeft(digits, "0")
}

// pokemonMiscShelf is the shelf the storefront files its promos under, and
// the name of a set of the catalog's own.
const pokemonMiscShelf = "Miscellaneous Cards & Products"

// leftShelf reports whether a cosmos holo listing on the miscellaneous shelf
// was answered by a printing outside that set. The shelf's cosmos holo promos
// are not the cards they are cut from: a Galarian Zapdos the catalog does not
// carry would take the price of the Evolving Skies card.
func (gn *Gamenerdz) leftShelf(cardID string, product GNProduct) bool {
	if product.ProductData.SetName != pokemonMiscShelf ||
		!strings.Contains(pokemonLabels.Replace(product.DisplayName), "(Cosmos Holo)") {
		return false
	}
	shelf, err := gn.backend.GetSetByName(pokemonMiscShelf)
	if err != nil {
		return false
	}
	co, err := gn.backend.GetUUID(cardID)
	return err == nil && co.SetCode != shelf.Code
}

// finishPrinted reports whether the printing a product resolved to was sold in
// the finish the product names. An id the catalog cannot place says nothing
// either way and is left alone.
func finishPrinted(b *mtgmatcher.Backend, cardID string, foil, etched bool) bool {
	co, err := b.GetUUID(cardID)
	if err != nil {
		return true
	}
	switch {
	case etched:
		return co.HasFinish(mtgmatcher.FinishEtched)
	case foil:
		return co.HasFinish(mtgmatcher.FinishFoil)
	}
	return co.HasFinish(mtgmatcher.FinishNonfoil)
}

// release decides the products the crawl held back, now that it has been
// everywhere. A body naming a set the display name does not is this
// storefront's own shelf code where the product's other finish carries the
// same body - a promo pack is coded "ppm21" where the name writes PPM21, on
// both of them, and a Special Guest keeps its own code under either name -
// and a body belonging to another product where the two finishes disagree
// about it. Game Nerdz resolves a body by matching a product's number
// against other skus as a substring, so MTG-WOE-199-WC5VKQJZA2 takes the
// body, the price and the finish of MTG-MOC-103-QXUYX199FD, whose hash
// happens to spell 199. The twin escapes because its own sku string does not
// match, and that is what makes the pair worth comparing.
func (gn *Gamenerdz) release(mode string, state *crawlState) {
	var dropped int
	for _, product := range state.held {
		if len(state.bodies[skuFamily(product)]) > 1 {
			dropped++
			continue
		}
		err := gn.processProduct(mode, product)
		if err != nil {
			gn.printf("process error: %s", err.Error())
		}
	}
	state.held = nil
	if dropped > 0 {
		gn.printf("%s dropped %d products whose body belongs to another", mode, dropped)
	}
}

// crawlState is what one mode's passes accumulate together: the products any
// pass already processed, and the rarity and finish vocabularies harvested
// from the rows themselves, so a narrower slice never has to be known ahead
// of time.
type crawlState struct {
	seen     map[string]bool
	rarities map[string]bool
	finishes map[string]bool
	// A product whose body names a set its own display name does not is
	// held back until the crawl has been everywhere: what tells a body
	// belonging to another product from a shelf this storefront simply
	// codes its own way is whether the finish twin beside it carries the
	// same body. See release.
	bodies map[string]map[string]bool
	held   []GNProduct
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
		bodies:   map[string]map[string]bool{},
	}
	err := gn.widen(ctx, mode, map[string]string{}, gn.axes(), state)
	if err != nil {
		return err
	}
	gn.release(mode, state)
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
			if gn.backend.Game == mtgmatcher.GameMagic {
				family := skuFamily(product)
				if family != "" {
					if state.bodies[family] == nil {
						state.bodies[family] = map[string]bool{}
					}
					state.bodies[family][strings.ToLower(string(product.ProductData.Set))] = true
					if !bodyNamesOwnSet(product) {
						state.held = append(state.held, product)
						continue
					}
				}
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

		mtgban.WorkerPool(ctx, gn.maxConcurrency, pageNums,
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

	if !gn.disableRetail {
		err := gn.scrape(ctx, modeRetail)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err))
		} else {
			gn.inventoryDate = time.Now()
		}
	}

	if !gn.disableBuylist {
		err := gn.scrape(ctx, modeBuylist)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, err))
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
	info.Game = gn.backend.Game
	return
}
