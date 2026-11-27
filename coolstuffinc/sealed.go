package coolstuffinc

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// Sealed prices Cool Stuff Inc's sealed product.
type Sealed struct {
	logCallback mtgban.LogCallbackFunc
	partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	maxConcurrency int

	productMap map[string]string
	// priced holds how each product a run priced was placed, by its id.
	priced map[string]mtgban.SealedVia

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	disableRetail  bool
	disableBuylist bool

	client  *http.Client
	shelf   string
	backend *matcher.Backend
}

// sealedHoldsOwnID is the refusal of a row whose name reaches a product the
// datastore already lists an id for.
const sealedHoldsOwnID = "product holds its own id"

// holdsOwnID reports whether the datastore gives the product a csiId.
func (csi *Sealed) holdsOwnID(uuid string) bool {
	co, err := csi.backend.GetUUID(uuid)
	return err == nil && co.Identifiers["csiId"] != ""
}

// NewScraperSealed returns a sealed scraper for the datastore's game.
func NewScraperSealed(b *matcher.Backend) (*Sealed, error) {
	game := b.Game
	shelf, ok := csiGames[game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	csi := Sealed{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.client = newCSIHTTPClient()
	csi.maxConcurrency = defaultConcurrency

	csi.productMap = map[string]string{}
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		id, found := co.Identifiers["csiId"]
		if !found {
			continue
		}
		csi.productMap[id] = co.UUID
	}
	csi.shelf = shelf
	csi.backend = b
	return &csi, nil
}

func (csi *Sealed) printf(format string, a ...any) {
	if csi.logCallback != nil {
		csi.logCallback("[CSISealed] "+format, a...)
	}
}

const sealedURL = "https://www.coolstuffinc.com/sq/2293832?page=1&sb=price|desc"

func (csi *Sealed) numOfPages(ctx context.Context) (int, error) {
	link := sealedURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	err = checkStatus(resp)
	if err != nil {
		return 0, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	text := doc.Find(".search-result-links").Text()
	text = strings.TrimPrefix(strings.Split(text, " Results")[0], "1 - ")

	fields := strings.Split(text, " of ")
	if len(fields) != 2 {
		return 0, errors.New("unknown page format")
	}

	resultsPerPage, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, err
	}

	resultsTotal, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, err
	}

	return resultsTotal/resultsPerPage + 1, nil
}

func (csi *Sealed) processSealedPage(ctx context.Context, channel chan<- responseChan, page int) error {
	csi.printf("Processing page %d", page)

	u, err := url.Parse(sealedURL)
	if err != nil {
		return err
	}

	v := u.Query()
	v.Set("page", fmt.Sprint(page))
	u.RawQuery = v.Encode()

	link := u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := csi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = checkStatus(resp)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(".main-container").Each(func(i int, s *goquery.Selection) {
		productName := s.Find(`span[itemprop="name"]`).Text()
		path, _ := s.Find(`a[class="productLink"]`).Attr("href")

		csiID := strings.TrimPrefix(path, "/p/")

		match := csi.matchProduct(csiID, productName, fromSearch)
		if !placed(match) {
			return
		}
		uuid := match.UUID

		qtyStr := s.Find(`span[class="card-qty"]`).Text()
		qtyStr = strings.TrimSuffix(qtyStr, "+")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			qty = 20
		}

		priceStr := s.Find(`b[itemprop="price"]`).First().Text()
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%s: %s", productName, err.Error())
			return
		}

		link := "https://coolstuffinc.com" + path
		if csi.partner != "" {
			link += "?utm_referrer=" + csi.partner
		}

		out := responseChan{
			cardID: uuid,
			invEntry: &mtgban.InventoryEntry{
				Price:    price,
				Quantity: qty,
				URL:      link,
			},
			storeID: csiID,
			via:     match.Via,
		}

		channel <- out
	})

	return nil
}

func (csi *Sealed) scrape(ctx context.Context) error {
	totalPages, err := csi.numOfPages(ctx)
	if err != nil {
		return err
	}
	csi.printf("Found %d pages", totalPages)

	pageNums := make([]int, totalPages)
	for i := range pageNums {
		pageNums[i] = i + 1
	}

	walkCtx, walk := newWalk(ctx)
	defer walk.stop()
	mtgban.WorkerPool(walkCtx, csi.maxConcurrency, pageNums,
		func(ctx context.Context, page int, results chan<- responseChan) error {
			err := csi.processSealedPage(ctx, results, page)
			if err != nil {
				err = fmt.Errorf("page %d: %w", page, err)
			}
			return walk.add(err)
		},
		func(record responseChan) {
			err := csi.inventory.Add(record.cardID, record.invEntry)
			if err != nil {
				csi.printf("%s", err.Error())
				return
			}
			csi.markPriced(record.storeID, record.via)
		},
		csi.printf,
	)

	err = walk.lostError(ctx, len(pageNums), "pages")
	if err != nil {
		return err
	}

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *Sealed) parseBL(ctx context.Context) error {
	products, err := GetBuylist(ctx, csi.client, csi.shelf)
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName != "Box" {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", csi.shelf)
		v.Set("a", "1")
		v.Set("name", product.Name)
		u.RawQuery = v.Encode()
		link := u.String()

		match := csi.matchProduct(product.PID, product.Name, fromBuylist)
		if !placed(match) {
			continue
		}
		uuid := match.UUID

		buyPrice, err := matcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[uuid]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		buyEntry := mtgban.BuylistEntry{
			BuyPrice:   buyPrice,
			PriceRatio: priceRatio,
			Quantity:   maxBuylistQuantity,
			URL:        link,
			InstanceID: product.PPQID,
		}

		err = csi.buylist.Add(uuid, &buyEntry)
		if err != nil {
			csi.printf("%s", err.Error())
			continue
		}
		csi.markPriced(product.PID, match.Via)
	}

	csi.buylistDate = time.Now()

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (csi *Sealed) SetConfig(opt mtgban.ScraperOptions) {
	csi.disableRetail = opt.DisableRetail
	csi.disableBuylist = opt.DisableBuylist
}

// markPriced records that a product holds a price, and how it was placed.
func (csi *Sealed) markPriced(pid string, via mtgban.SealedVia) {
	if csi.priced == nil {
		csi.priced = map[string]mtgban.SealedVia{}
	}
	csi.priced[pid] = via
}

// Load fetches everything this scraper offers, then says how many products
// it priced, by how each was placed. See mtgban.Scraper.
func (csi *Sealed) Load(ctx context.Context) error {
	csi.priced = nil
	err := csi.load(ctx)
	byID, bridged, named := pricedByVia(csi.priced)
	csi.printf("Priced %d products by id, %d through a bridge, %d by name", byID, bridged, named)
	return err
}

// pricedByVia counts the products of a run by how each was placed.
func pricedByVia(priced map[string]mtgban.SealedVia) (byID, bridged, named int) {
	for _, via := range priced {
		switch via {
		case mtgban.SealedByID:
			byID++
		case mtgban.SealedByBridge:
			bridged++
		case mtgban.SealedByName:
			named++
		}
	}
	return byID, bridged, named
}

// load is the work Load does before that line.
func (csi *Sealed) load(ctx context.Context) error {
	// The saved-query page and the buylist exist for Magic alone; the
	// other games ride the same set-facet search the singles use, with
	// the product ids and the sealed-name resolver telling the sealed
	// rows apart from the card ones.
	if csi.backend.Game != matcher.GameMagic {
		var errs []error
		if !csi.disableRetail {
			if err := csi.scrapeBysets(ctx); err != nil {
				errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err))
			}
		}
		if !csi.disableBuylist {
			if err := csi.parseBL(ctx); err != nil {
				errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, err))
			}
		}
		return errors.Join(errs...)
	}

	var errs []error

	if !csi.disableRetail {
		err := csi.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err))
		}
	}

	if !csi.disableBuylist {
		err := csi.parseBL(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, err))
		}
	}

	return errors.Join(errs...)
}

// sealedQueries are the searches the sealed catalog is read through, one per
// sealed-product word.
var sealedQueries = []string{
	"booster", "deck", "trove", "gift", "bundle",
	"case", "box", "kit", "vault", "collection",
}

// scrapeBysets walks every set of the game through the same search the
// singles scraper uses; the sealed rows resolve by id or through the
// sealed-name resolver, the card rows resolve to nothing and drop out.
func (csi *Sealed) scrapeBysets(ctx context.Context) error {
	// One search per sealed-product word: the store's own names carry
	// them ("Booster Box", "Illumineer's Trove"), and a handful of
	// searches covers the whole sealed catalog where the set facets do
	// not - lorcana sealed carries no ItemSet at all
	queries := sealedQueries

	// The queries overlap ("Booster Box" answers booster and box both);
	// the first sighting of a product wins
	seen := map[string]bool{}
	var rows atomic.Int64
	walkCtx, walk := newWalk(ctx)
	defer walk.stop()
	mtgban.WorkerPool(walkCtx, csi.maxConcurrency, queries,
		func(ctx context.Context, query string, channel chan<- responseChan) error {
			found, err := csi.processSealedSearch(ctx, channel, query)
			rows.Add(int64(found))
			if err != nil {
				err = fmt.Errorf("%s: %w", query, err)
			}
			return walk.add(err)
		},
		func(record responseChan) {
			key := record.cardID + "|" + record.invEntry.OriginalID
			if seen[key] {
				return
			}
			seen[key] = true
			err := csi.inventory.Add(record.cardID, record.invEntry)
			if err != nil {
				csi.printf("%v", err)
				return
			}
			csi.markPriced(record.storeID, record.via)
		},
		csi.printf,
	)

	err := walk.lostError(ctx, len(queries), "searches")
	if err != nil {
		return err
	}

	// Rows, not results: a game can have nothing in stock that resolves,
	// but a refused search answers its form again with no rows at all.
	if rows.Load() == 0 {
		return errors.New("no products found in any search")
	}

	csi.inventoryDate = time.Now()

	return nil
}

// searchSealed runs a product-name search without the singles' rarity
// facets, which sealed products have none of and would be filtered out
// by. The name route rather than the set facet: lorcana sealed carries
// no ItemSet at all, where riftbound's does.
func searchSealed(ctx context.Context, client *http.Client, shelf, query string) (*SearchResult, error) {
	v := url.Values{}
	v.Set("name", query)
	v.Set("f[Artist][]", "")
	v.Add("f[Cost][]", "")
	v.Add("f[Cost][]", "")
	v.Set("f[Number][]", "")
	v.Set("f[Type][]", "")
	v.Set("f[Card+Text][]", "")
	v.Set("notes", "")
	v.Set("sign-Cost", "<")
	v.Set("sign-Power", "<")
	v.Set("f[Power][]", "")
	v.Set("sign-Toughness", "<")
	v.Set("f[Toughness][]", "")
	v.Set("sign-Loyalty", "<")
	v.Set("f[Loyalty][]", "")
	v.Set("signprice", "<")
	v.Set("price", "")
	// No instock option either: combined with a name search it returns
	// nothing at all, and the offer rows carry their own stock state
	// No rarity facets: sealed products carry no rarity, and any rarity
	// constraint excludes them outright
	v.Set("f[Rarity][]", "")
	v.Set("f[ItemSet][]", "")
	v.Set("s", shelf)
	v.Set("page", "1")
	v.Set("resultsPerPage", "25")
	v.Set("submit", "Search")

	return postSearch(ctx, client, v)
}

// processSealedSearch pages through one query's results, pricing every
// row whose id the datastore carries or whose name the sealed-name resolver
// recognizes. English only: language-variant names are skipped before
// resolution. It answers how many rows the search returned, priced or not.
func (csi *Sealed) processSealedSearch(ctx context.Context, channel chan<- responseChan, query string) (int, error) {
	return csi.pageSealedSearch(ctx, query, func(s *goquery.Selection) {
		row := sealedSearchRow(s)
		pid := row.pid
		match := csi.matchProduct(pid, row.name, fromSearch)
		if !placed(match) {
			return
		}
		uuid := match.UUID

		link := "https://www.coolstuffinc.com/p/" + pid
		if csi.partner != "" {
			link += "?utm_referrer=" + csi.partner
		}

		// The stock state is schema markup on the row; the row's
		// visible text mentions being out of stock even on in-stock
		// rows, through template markup, and must not be trusted
		availability, _ := s.Find(`[itemprop="availability"]`).Attr("content")
		if !strings.Contains(availability, "InStock") {
			return
		}

		// Sealed rows carry no quantity span; an in-stock row
		// without one sells at least a single copy
		qty := 1
		qtyStr := s.Find(`span[class="card-qty"]`).Text()
		qtyStr = strings.TrimSpace(strings.TrimSuffix(qtyStr, "+"))
		if qtyStr != "" {
			var err error
			qty, err = strconv.Atoi(qtyStr)
			if err != nil || qty == 0 {
				return
			}
		}

		// The price rides in the content attribute, the visible text
		// being nothing but a formatting placeholder
		priceStr, _ := s.Find(`[itemprop="price"]`).Attr("content")
		if priceStr == "" {
			priceStr = strings.TrimSpace(s.Find(`b[itemprop="price"]`).Text())
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		channel <- responseChan{
			cardID: uuid,
			invEntry: &mtgban.InventoryEntry{
				Conditions: mtgban.NM,
				Price:      price,
				Quantity:   qty,
				URL:        link,
				OriginalID: pid,
			},
			storeID: pid,
			via:     match.Via,
		}
	})
}

// pageSealedSearch pages through one query's results, handing each row to
// read, and answers how many rows the search returned.
func (csi *Sealed) pageSealedSearch(ctx context.Context, query string, read func(*goquery.Selection)) (int, error) {
	result, err := searchSealed(ctx, csi.client, csi.shelf, query)
	if err != nil {
		return 0, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(result.Data))
	if err != nil {
		return 0, fmt.Errorf("page 1: %w", err)
	}
	next := result.NextLink

	// A shelf that ends on its first page is already read; one that does
	// not is read again from the top at the larger page size.
	if next != "" {
		wide, wideNext := widenSearchPage(ctx, csi.client, next)
		if wide != nil {
			doc = wide
			next = wideNext
		}
	}

	found := 0
	for page := 1; ; page++ {
		rows := doc.Find(searchRowSelector)
		found += rows.Length()
		rows.Each(func(i int, s *goquery.Selection) {
			read(s)
		})

		if next == "" {
			break
		}

		doc, err = fetchSearchPage(ctx, csi.client, next)
		if err != nil {
			return found, fmt.Errorf("page %d: %w", page+1, err)
		}
		next = searchNextLink(doc)
	}

	return found, nil
}

// sealedRow is one product row of the sealed search, by its id and name.
type sealedRow struct {
	pid  string
	name string
}

// sealedSearchRow reads a search row's id and name.
func sealedSearchRow(s *goquery.Selection) sealedRow {
	pid, _ := s.Find(`span[class="rating-display "]`).Attr("data-pid")
	return sealedRow{
		pid:  pid,
		name: strings.TrimSpace(s.Find(`span[itemprop="name"]`).Text()),
	}
}

// sealedFeed names the feed a product row comes from, since the two spell
// a product's name apart.
type sealedFeed int

const (
	fromSearch sealedFeed = iota
	fromBuylist
)

// sealedUnlinked is the reason a Magic product the datastore holds no id
// for is dropped; Magic sealed is matched by id alone.
const sealedUnlinked = "unlinked id"

// matchProduct decides which datastore product one row of a feed is, before
// anything about its stock or price is read. An id the datastore carries
// settles the row; outside Magic the others are placed by name, unique or
// nothing, skipping the language variants the datastores never carry.
func (csi *Sealed) matchProduct(pid, name string, feed sealedFeed) mtgban.SealedMatch {
	match := mtgban.SealedMatch{StoreID: pid, Name: name}
	if uuid, found := csi.productMap[pid]; found {
		match.UUID = uuid
		match.Via = mtgban.SealedByID
		return match
	}
	if csi.backend.Game == matcher.GameMagic {
		match.Via = mtgban.SealedSkipped
		match.Reason = sealedUnlinked
		return match
	}

	productName := name
	if feed == fromSearch {
		if csi.backend.Game == matcher.GameYuGiOh {
			// The storefront leads its yugioh sealed listings with the
			// game's own name, which the canonical names never carry.
			productName = strings.TrimPrefix(productName, "Yu-Gi-Oh!")
			productName = strings.TrimPrefix(strings.TrimSpace(productName), "- ")
		}
		if csi.backend.Game == matcher.GameLorcana {
			productName = lorcanaSealedWords.Replace(productName)
		}
		if productName == "" {
			match.Via = mtgban.SealedSkipped
			match.Reason = "no name"
			return match
		}
	}
	if matcher.SealedIsLanguageVariant(productName) {
		match.Via = mtgban.SealedSkipped
		match.Reason = "language variant"
		return match
	}
	uuid, err := csi.backend.ResolveSealed(productName)
	if err != nil {
		// A card row, or a product the datastore does not carry
		match.Via = mtgban.SealedRefused
		match.Reason = err.Error()
		return match
	}
	if csi.holdsOwnID(uuid) {
		csi.printf("%q (%s): %s", productName, pid, sealedHoldsOwnID)
		match.Via = mtgban.SealedRefused
		match.Reason = sealedHoldsOwnID
		return match
	}
	match.UUID = uuid
	match.Via = mtgban.SealedByName
	return match
}

// placed reports whether a match landed on a product.
func placed(match mtgban.SealedMatch) bool {
	return match.UUID != ""
}

// newSealedMatcher builds the scraper the export matches through, refusing
// Magic, which is matched by id alone, and every game the sealed scraper is
// not registered for.
func newSealedMatcher(b *matcher.Backend) (*Sealed, error) {
	if !slices.Contains(sealedGames, b.Game) {
		return nil, fmt.Errorf("no sealed scraper for game %q", b.Game)
	}
	if b.Game == matcher.GameMagic {
		return nil, errors.New("magic sealed is matched by id alone")
	}
	return NewScraperSealed(b)
}

// MatchSealedCatalog pulls the sealed products of the backend's game from
// the storefront's sealed searches and its buylist, the way Load does, and
// answers what the scraper's matching decides for each, by store id. It
// prices nothing and needs no credential. The search mixes card rows in, so
// a search row is reported only when it is placed; every buylist box is
// reported, placed, skipped or refused. A row with no id is left out. When
// the two feeds place one id on different products, the export refuses it
// and names both, while the scraper prices each feed's own answer. A lost
// search, searches answering no row at all, or rows none of which carries a
// product id, is an error.
func MatchSealedCatalog(ctx context.Context, b *matcher.Backend) ([]mtgban.SealedMatch, error) {
	csi, err := newSealedMatcher(b)
	if err != nil {
		return nil, err
	}
	return csi.matchSealedCatalog(ctx)
}

// matchSealedCatalog is the export's fetch and match, over csi's client.
func (csi *Sealed) matchSealedCatalog(ctx context.Context) ([]mtgban.SealedMatch, error) {
	search, err := csi.searchSealedRows(ctx)
	if err != nil {
		return nil, err
	}
	buylist, err := GetBuylist(ctx, csi.client, csi.shelf)
	if err != nil {
		return nil, err
	}
	return csi.matchCatalog(search, buylist), nil
}

// searchSealedRows runs scrapeBysets' searches and answers every row they
// return, failing as scrapeBysets does on a lost search or on none
// answering a row, and on rows none of which carries a product id.
func (csi *Sealed) searchSealedRows(ctx context.Context) ([]sealedRow, error) {
	var all []sealedRow
	walkCtx, walk := newWalk(ctx)
	defer walk.stop()
	mtgban.WorkerPool(walkCtx, csi.maxConcurrency, sealedQueries,
		func(ctx context.Context, query string, results chan<- []sealedRow) error {
			rows, err := csi.sealedSearchRows(ctx, query)
			if err != nil {
				return walk.add(fmt.Errorf("%s: %w", query, err))
			}
			results <- rows
			return walk.add(nil)
		},
		func(rows []sealedRow) {
			all = append(all, rows...)
		},
		nil,
	)

	err := walk.lostError(ctx, len(sealedQueries), "searches")
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, errors.New("no products found in any search")
	}
	// The export leaves out a row with no id, so rows none of which carries
	// one would report the buylist alone, every search-only id as gone.
	var withPID int
	for _, row := range all {
		if row.pid != "" {
			withPID++
		}
	}
	if withPID == 0 {
		return nil, fmt.Errorf("none of %d search rows carries a product id: the markup may have changed", len(all))
	}
	return all, nil
}

// sealedSearchRows answers every row one query returns.
func (csi *Sealed) sealedSearchRows(ctx context.Context, query string) ([]sealedRow, error) {
	var rows []sealedRow
	_, err := csi.pageSealedSearch(ctx, query, func(s *goquery.Selection) {
		rows = append(rows, sealedSearchRow(s))
	})
	return rows, err
}

// matchCatalog matches the rows of both feeds, one answer per id: a placed
// answer wins over a skipped or refused one, and two feeds placing one id
// on different products refuse it, naming both. A search row is reported
// only when it is placed, since the search mixes card rows in; of the
// buylist, only its boxes are read, as parseBL reads it, and each is
// reported whatever its outcome. A row of either feed with no id is left
// out.
func (csi *Sealed) matchCatalog(search []sealedRow, buylist []CSIPriceEntry) []mtgban.SealedMatch {
	type answer struct {
		match    mtgban.SealedMatch
		feed     string
		disputed bool
	}
	answers := map[string]*answer{}
	add := func(match mtgban.SealedMatch, feed string) {
		prev, found := answers[match.StoreID]
		if !found {
			answers[match.StoreID] = &answer{match: match, feed: feed}
			return
		}
		if prev.disputed || !placed(match) {
			return
		}
		if !placed(prev.match) {
			prev.match = match
			prev.feed = feed
			return
		}
		if prev.match.UUID == match.UUID {
			return
		}
		prev.match = mtgban.SealedMatch{
			StoreID: prev.match.StoreID,
			Name:    prev.match.Name,
			Via:     mtgban.SealedRefused,
			Reason:  fmt.Sprintf("the %s places it on %s, the %s on %s", prev.feed, prev.match.UUID, feed, match.UUID),
		}
		prev.disputed = true
	}

	for _, row := range search {
		if row.pid == "" {
			continue
		}
		match := csi.matchProduct(row.pid, row.name, fromSearch)
		if !placed(match) {
			continue
		}
		add(match, "search")
	}
	for _, product := range buylist {
		if product.RarityName != "Box" || product.PID == "" {
			continue
		}
		add(csi.matchProduct(product.PID, product.Name, fromBuylist), "buylist")
	}

	matches := make([]mtgban.SealedMatch, 0, len(answers))
	for _, answer := range answers {
		matches = append(matches, answer.match)
	}
	slices.SortFunc(matches, func(a, b mtgban.SealedMatch) int {
		return compareStoreIDs(a.StoreID, b.StoreID)
	})
	return matches
}

// compareStoreIDs orders two ids by their number, falling back to their
// text for an id that is not one.
func compareStoreIDs(a, b string) int {
	x, errA := strconv.Atoi(a)
	y, errB := strconv.Atoi(b)
	if errA != nil || errB != nil || x == y {
		return strings.Compare(a, b)
	}
	return cmp.Compare(x, y)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (csi *Sealed) Inventory() mtgban.InventoryRecord {
	return csi.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (csi *Sealed) Buylist() mtgban.BuylistRecord {
	return csi.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (csi *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSISealed"
	info.Game = csi.backend.Game
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.SealedMode = true
	info.CreditMultiplier = 1.25
	return
}
