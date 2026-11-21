// Package trolltrader scrapes Troll Trader Cards, a UK Magic store whose
// storefront names every printing by MTGJSON set code and collector number.
package trolltrader

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
	"github.com/mtgban/go-mtgban/mtgban"
)

const defaultConcurrency = 4

// storefrontPace is how often the storefront is asked for a page.
const storefrontPace = 1100 * time.Millisecond

// TrollTrader prices the Magic singles Troll Trader Cards sells. The store
// quotes pounds; every price is converted to dollars at the day's rate.
type TrollTrader struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int

	backend *matcher.Backend

	client *ttClient
	rate   float64

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraper returns a scraper matching against b.
func NewScraper(b *matcher.Backend) *TrollTrader {
	tt := TrollTrader{backend: b}
	tt.inventory = mtgban.InventoryRecord{}
	tt.client = newClient()
	tt.maxConcurrency = defaultConcurrency
	return &tt
}

func (tt *TrollTrader) printf(format string, a ...any) {
	if tt.logCallback != nil {
		tt.logCallback("[TT] "+format, a...)
	}
}

// ttCondition reads the store's grades on its own scale: Excellent shows a
// few white spots, Light Played shows general play, Played any wear.
func ttCondition(s string) (mtgban.Condition, error) {
	switch s {
	case "Excellent":
		return mtgban.SP, nil
	case "Good", "Light Played":
		return mtgban.MP, nil
	case "Played":
		return mtgban.HP, nil
	default:
		return mtgban.ParseCondition(s)
	}
}

// finishes maps the store's finish words to the flags the matcher reads.
// Signed copies are left out.
var finishes = map[string]struct{ foil, etched bool }{
	"Non-Foil": {},
	"Foil":     {foil: true},
	"Etched":   {foil: true, etched: true},
}

// match resolves the printing a set code and collector number name, checking
// the answer is the printing asked for. The store files a set's tokens and
// emblems under the set's own code, where the datastore keeps them in its
// token set, and numbers a two-sided token by both faces, "20/12".
func (tt *TrollTrader) match(name, setCode, number, finish string) (string, error) {
	flags, found := finishes[finish]
	if !found {
		return "", matcher.ErrUnsupported
	}
	// Some titles keep the backslash escaping a quote
	name = strings.ReplaceAll(name, `\"`, `"`)
	// The store files an emblem as "Emblem - <planeswalker>", which the
	// datastore names "<planeswalker> Emblem"
	walker, found := strings.CutPrefix(name, "Emblem - ")
	if found {
		name = walker + " Emblem"
	}

	tokenSet := magic.SetTokenSetCode(tt.backend, setCode)
	if strings.Contains(name, " // ") && strings.Contains(number, "/") && tokenSet != "" {
		for face := range strings.SplitSeq(number, "/") {
			id := magic.MatchNativeTokenPair(tt.backend, tokenSet, face, name)
			if id == "" {
				id = magic.MatchTokenPairingBySetNumber(tt.backend, tokenSet, face, name, flags.foil)
			}
			if id == "" {
				continue
			}
			cardID, err := tt.backend.MatchID(id, flags.foil)
			if err == nil {
				return cardID, nil
			}
		}
		return "", matcher.ErrUnsupported
	}

	cardID, err := tt.matchIn(name, setCode, number, flags.foil, flags.etched)
	// A Secret Lair crossover is titled by its flavor name first,
	// "Amunet, Tyrants' End - Queen Marchesa"
	_, cardName, found := strings.Cut(name, " - ")
	if err != nil && found {
		namedID, namedErr := tt.matchIn(cardName, setCode, number, flags.foil, flags.etched)
		if namedErr == nil {
			return namedID, nil
		}
	}
	if err != nil && tokenSet != "" {
		tokenID, tokenErr := tt.matchIn(name, tokenSet, number, flags.foil, flags.etched)
		if tokenErr == nil {
			return tokenID, nil
		}
	}
	return cardID, err
}

// matchIn resolves a printing by name, set and number, refusing one that
// lands anywhere else.
func (tt *TrollTrader) matchIn(name, setCode, number string, foil, etched bool) (string, error) {
	in := matcher.InputCard{
		Name:      name,
		Edition:   setCode,
		Variation: number,
		Foil:      foil,
	}
	if etched {
		in.Finish = matcher.FinishEtched
	}
	cardID, err := tt.backend.Match(&in)
	if err != nil {
		return "", err
	}
	co, err := tt.backend.GetUUID(cardID)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(co.SetCode, setCode) || !sameNumber(number, co.Number, etched) {
		return "", fmt.Errorf("%s %s landed on %s %s", setCode, number, co.SetCode, co.Number)
	}
	return cardID, nil
}

// sameNumber reports whether the store's number names the printing the
// datastore numbers landed. The store numbers a foil the datastore files
// apart, as "89★", the same as the plain card, and suffixes an etched one
// "273e" the datastore does not.
func sameNumber(number, landed string, etched bool) bool {
	return landed == number || landed == number+"★" || etched && number == landed+"e"
}

func (tt *TrollTrader) logError(err error, name, setCode, number, finish string) {
	if errors.Is(err, matcher.ErrUnsupported) {
		return
	}
	tt.printf("%v", err)
	tt.printf("%q [%s %s] %s", name, setCode, number, finish)
	var alias *matcher.AliasingError
	if errors.As(err, &alias) {
		for _, probe := range alias.Probe() {
			co, _ := tt.backend.GetUUID(probe)
			tt.printf("- %s", co)
		}
	}
}

// titleRE reads a product title, "Name (number)" and whatever qualifier
// follows, " - Surge-Foil" or " (Display Commander)", which the number
// already speaks for.
var titleRE = regexp.MustCompile(`^(.+?) ?\(([^()]+)\)`)

// setCodeOf reads the set code off a product's variants, whose SKUs all open
// "cc-<set code>-".
func setCodeOf(product shopifyProduct) string {
	for _, variant := range product.Variants {
		rest, found := strings.CutPrefix(variant.SKU, "cc-")
		if !found {
			continue
		}
		code, _, found := strings.Cut(rest, "-")
		if found && code != "" {
			return strings.ToUpper(code)
		}
	}
	return ""
}

// optionValues returns a variant's finish, condition and language, by the
// product's own option names; a product sold in one language names none.
func optionValues(product shopifyProduct, variant shopifyVariant) (finish, condition, language string) {
	values := []string{variant.Option1, variant.Option2, variant.Option3}
	language = "English"
	for _, option := range product.Options {
		if option.Position < 1 || option.Position > len(values) {
			continue
		}
		value := values[option.Position-1]
		switch option.Name {
		case "Card Finish":
			finish = value
		case "Condition":
			condition = value
		case "Language":
			language = value
		}
	}
	return
}

func (tt *TrollTrader) processProduct(product shopifyProduct) {
	m := titleRE.FindStringSubmatch(product.Title)
	setCode := setCodeOf(product)
	if m == nil || setCode == "" {
		tt.printf("unparsed product %d: %q", product.ID, product.Title)
		return
	}
	name, number := m[1], m[2]
	// A variant the store tells apart by art, "Female - 88a"
	_, after, found := strings.Cut(number, " - ")
	if found {
		number = after
	}
	link := baseURL + "/products/" + product.Handle

	for _, variant := range product.Variants {
		if !variant.Available {
			continue
		}
		finish, condition, language := optionValues(product, variant)
		if language != "English" {
			continue
		}
		price, err := strconv.ParseFloat(variant.Price, 64)
		if err != nil || price <= 0 {
			continue
		}
		cond, err := ttCondition(condition)
		if err != nil {
			tt.printf("unsupported %s condition", condition)
			continue
		}

		cardID, err := tt.match(name, setCode, number, finish)
		if err != nil {
			tt.logError(err, name, setCode, number, finish)
			continue
		}

		err = tt.inventory.Add(cardID, &mtgban.InventoryEntry{
			Conditions: cond,
			Price:      price * tt.rate,
			Quantity:   1,
			URL:        link + "?variant=" + strconv.FormatInt(variant.ID, 10),
			OriginalID: strconv.FormatInt(product.ID, 10),
			InstanceID: strconv.FormatInt(variant.ID, 10),
		})
		if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
			tt.printf("%s", err.Error())
		}
	}
}

// scrapeRetail walks the storefront collection by collection. Shopify walks
// no listing past walkLimit products, and the one holding every single
// is several times that, so the singles are read off the collections of
// each set and the shelves beside them, a product seen twice counted once.
func (tt *TrollTrader) scrapeRetail(ctx context.Context) error {
	// Shopify answers a burst of about 60 and then one request a second,
	// refusing the rest, so every request waits its turn
	tick := time.NewTicker(storefrontPace)
	defer tick.Stop()
	wait := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			return nil
		}
	}

	var handles []string
	for page := 1; ; page++ {
		err := wait(ctx)
		if err != nil {
			return err
		}
		collections, err := tt.client.collections(ctx, page)
		if err != nil {
			return fmt.Errorf("collections page %d: %w", page, err)
		}
		if len(collections) == 0 {
			break
		}
		for _, collection := range collections {
			switch {
			case collection.ProductsCount == 0:
			case collection.ProductsCount > walkLimit:
				tt.printf("skipping %s, %d products are past Shopify's walk", collection.Handle, collection.ProductsCount)
			default:
				handles = append(handles, collection.Handle)
			}
		}
	}

	var failed atomic.Int32
	seen := map[int64]bool{}
	mtgban.WorkerPool(ctx, tt.maxConcurrency, handles,
		func(ctx context.Context, handle string, results chan<- []shopifyProduct) error {
			for page := 1; ; page++ {
				err := wait(ctx)
				if err != nil {
					return err
				}
				products, err := tt.client.products(ctx, handle, page)
				if err != nil {
					failed.Add(1)
					return fmt.Errorf("%s page %d: %w", handle, page, err)
				}
				results <- products
				if len(products) < productsPerPage {
					return nil
				}
			}
		},
		func(products []shopifyProduct) {
			for _, product := range products {
				if seen[product.ID] || !isMagicSingle(product) {
					continue
				}
				seen[product.ID] = true
				tt.processProduct(product)
			}
		},
		tt.printf,
	)

	err := ctx.Err()
	if err != nil {
		return err
	}
	tt.printf("%d singles over %d collections", len(seen), len(handles))
	n := failed.Load()
	if n > 0 {
		return fmt.Errorf("%d of %d collections failed", n, len(handles))
	}
	if len(seen) == 0 {
		return errors.New("no collection holds a Magic single")
	}
	return nil
}

// isMagicSingle reports whether a product is a Magic single, which the
// storefront files under the game's own vendor, beside other games' singles
// and the sealed product in the same collections.
func isMagicSingle(product shopifyProduct) bool {
	return product.ProductType == "Singles" && product.Vendor == "Magic: The Gathering"
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tt *TrollTrader) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "GBP")
	if err != nil {
		return err
	}
	tt.rate = rate

	err = tt.scrapeRetail(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err)
	}
	tt.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tt *TrollTrader) Inventory() mtgban.InventoryRecord {
	return tt.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (tt *TrollTrader) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll Trader Cards"
	info.Shorthand = "TTC"
	info.Game = tt.backend.Game
	info.CountryFlag = "GB"
	info.NoQuantityInventory = true
	info.InventoryTimestamp = &tt.inventoryDate
	return
}
