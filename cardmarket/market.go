package cardmarket

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Market prices singles from Cardmarket's own listings, live, rather than
// from the price guide's low and trend columns: for each printing it holds
// the cheapest price per condition from a seller inside the European Union
// (see excludedCountries). This is a real market price at the cost of one
// signed call per product instead of one bulk download per game - the API
// tolerates almost no in-flight parallelism per app token, so Load walks
// the catalog strictly sequentially rather than pooling workers the way
// Sealed does.
//
// It also holds a second, independent cheapest-per-condition view for
// German and Dutch Powerseller listings alone, under its own MarketNames
// bucket - see marketPowersellerName.
type Market struct {
	resolver

	logCallback mtgban.LogCallbackFunc
	affiliate   string

	// BanPriceKey authenticates the mtgban price snapshot Load reads to
	// restrict the games marketFilterParams covers to the cards worth a
	// live call. Magic, Pokemon and YuGiOh need it to fit a nightly budget
	// at all - Load refuses to run any of them without one, rather than
	// starting a walk it cannot finish (see marketFilterRequired). Lorcana,
	// Riftbound, Flesh and Blood and One Piece fit their own budget
	// unfiltered too, so a missing key there is a real degradation, not a
	// hard failure: Load falls back to running them unfiltered and logs
	// that it did, rather than refusing to run at all over a filter that
	// only saves call volume, not correctness, for those four.
	banPriceKey string

	inventoryDate time.Time
	exchangeRate  float64
	inventory     mtgban.InventoryRecord

	client *cm.Client

	// liveExpansionsCache and liveExpansionsTried back liveExpansions: the
	// live call is attempted at most once per Load, its result (success or
	// failure) reused for every gap the rest of that run hits. Reset at
	// the top of Load so a Market reused across more than one run tries
	// again rather than replaying a stale failure or a stale list forever.
	liveExpansionsCache map[int]cm.Expansion
	liveExpansionsTried bool

	// bounced counts requests since the last one that came back a usable
	// answer - a price, or a clean empty result. Load runs strictly
	// sequentially, so this needs no lock: exactly one goroutine ever
	// touches it.
	bounced int

	game mtgban.Game
}

func (mkm *Market) printf(format string, a ...any) {
	if mkm.logCallback != nil {
		mkm.logCallback("[MKMMarket] "+format, a...)
	}
}

// maxBounced is how many requests in a row may fail before Load gives up on
// the run rather than grinding through the rest of the catalog. Retry-After
// and the client's own backoff already absorb an ordinary 429 inside one
// call; a request that still errors after that policy is exhausted is
// Cardmarket rejecting this token outright, not a blip one more product
// would clear.
const maxBounced = 20

// errTooManyBounces marks a run stopped by the failsafe above.
var errTooManyBounces = errors.New("too many consecutive failed requests")

// bounce counts one failed request and reports whether the run should stop.
func (mkm *Market) bounce() bool {
	mkm.bounced++
	return mkm.bounced >= maxBounced
}

// NewScraperMarket returns a live-listing scraper matching against b,
// authenticated with an app token and secret.
func NewScraperMarket(b *mtgmatcher.Backend, appToken, appSecret string) (*Market, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	id, found := mkmGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	mkm := Market{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = cm.NewClient(appToken, appSecret)
	mkm.game = game
	mkm.resolver.backend = b
	mkm.resolver.gameID = id
	mkm.resolver.printf = mkm.printf
	return &mkm, nil
}

// excludedCountries are the sellers Market drops for being based outside of
// the European Union and having too high shipping costs, as the alpha-2 code
// Cardmarket answers article.Seller.Address.Country with. sellerCountry
// cannot ask the API to exclude them server-side - confirmed exhaustively
// unable to carry more than one bare id, let alone a negation - so this is
// applied to every listing read back instead.
var excludedCountries = map[string]bool{
	"GB": true, // United Kingdom
	"CH": true, // Switzerland
	"NO": true, // Norway
	"IS": true, // Iceland
}

// mkmPowersellerCountries names the two countries whose Powerseller
// listings (isCommercial == 2 - confirmed directly against two real
// accounts: 1 reads "Professional", 2 reads "Powerseller" on Cardmarket's
// own seller pages) are also held under their own MarketNames bucket - see
// marketPowersellerName and queryOnePrinting.
var mkmPowersellerCountries = map[string]bool{
	"D":  true, // Germany
	"NL": true, // Netherlands
}

// isPowerseller reports whether an article qualifies for the
// marketPowersellerName bucket - already past acceptArticle's own gate
// (price, finish, condition, excludedCountries), so this only adds the
// Powerseller-and-country check on top.
func isPowerseller(article cm.Article) bool {
	return article.Seller.IsCommercial == 2 && mkmPowersellerCountries[article.Seller.Address.Country]
}

// mkmCondition maps Cardmarket's seven-grade condition scale onto mtgban's
// five (MT/NM > NM, EX > SP, GD > MP, LP/PL > HP, PO > PO): five buckets
// cannot hold seven without folding somewhere, and folding at the ends
// leaves the three grades this scraper actually chases - NM, SP, MP -
// each with a row of their own. In practice the article filter's own
// minCondition floor (see queryOnePrinting) already excludes LP, PL and PO
// server-side, so the last two rows rarely see a listing at all.
var mkmCondition = map[string]string{
	"MT": "NM",
	"NM": "NM",
	"EX": "SP",
	"GD": "MP",
	"LP": "HP",
	"PL": "HP",
	"PO": "PO",
}

// marketFinishParam names the server-side filter parameter and its
// article-level flag for the games whose second price column is a real
// second printing, single-axis - one param fully describes the alternate
// printing. Not present for a game's gameID: either the resolver never hands
// Market a distinct cardIDFoil for it (One Piece and Flesh and Blood sell
// each treatment as its own product, so the two ids are always the same
// one - see queryPrintings), or the game crosses more than one axis and is
// handled on its own (Pokemon - see queryPokemonPrintings and
// pokemonFinishPlan; Cardmarket exposes isFirstEd and isReverseHolo as
// independent flags, and a single param cannot name a query that needs
// both).
//
// isFoil's own documentation names only Magic and Pokemon (deprecated), but
// that is about the request-side filter, a separate question from whether
// the response's own article.IsFoil is populated for other games. It is,
// confirmed directly for Lorcana and Riftbound: a spread sample of thirty
// products (fifteen each) showed products selling nothing but foil copies
// at a clear premium (e.g. a Riftbound rare at $15-$1000, a Lorcana one at
// $35-$1174, versus $0.02-$0.03 commons), and products mixing a handful of
// foil listings into a mostly-plain one with the foil listings priced
// distinctly above the plain floor - exactly the shape cardID/cardIDFoil
// being the nonfoil/foil match of the same product predicts, not noise. The
// request-side isFoil param may or may not narrow the result set for these
// two (not separately verified, and not load-bearing either way - see
// queryOnePrinting on why a filter is never trusted without the client-side
// check below).
var marketFinishParam = map[int]string{
	cm.GameMagic:     "isFoil",
	cm.GameYuGiOh:    "isFirstEd",
	cm.GameLorcana:   "isFoil",
	cm.GameRiftbound: "isFoil",
}

// articleFlagValue reads one article's own boolean for a Cardmarket finish
// parameter - a fixed, game-agnostic name (there are only three: isFoil,
// isFirstEd, isReverseHolo), unlike marketFinishParam's per-game single
// choice of which one applies at all.
func articleFlagValue(param string, article *cm.Article) bool {
	switch param {
	case "isFoil":
		return article.IsFoil
	case "isFirstEd":
		return article.IsFirstEd
	case "isReverseHolo":
		return article.IsReverseHolo
	}
	return false
}

// marketLanguages maps a card's own Language field onto Cardmarket's
// idLanguage, defaulting to English for anything without a clean match -
// mtgban's fictional languages (Phyrexian, Quenya), the handful it carries
// that Cardmarket's table does not (Polish), or a plain missing field. This
// is safe to trust directly: the catalog being queried is already listing-
// shaped, one card at a time by its own resolved uuid, so there is no
// print-vs-listing mismatch to guard against the way there would be if a
// language were being read off a Cardmarket response instead.
var marketLanguages = map[string]int{
	"":                    1, // English
	"English":             1,
	"French":              2,
	"German":              3,
	"Spanish":             4,
	"Italian":             5,
	"Chinese Simplified":  6,
	"Japanese":            7,
	"Portuguese":          8,
	"Russian":             9,
	"Korean":              10,
	"Chinese Traditional": 11,
}

func marketLanguage(language string) int {
	if id, found := marketLanguages[language]; found {
		return id
	}
	return 1
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mkm *Market) Load(ctx context.Context) error {
	err := mkm.checkCatalog()
	if err != nil {
		return err
	}

	// A fresh attempt at the live-expansions enrichment every Load, rather
	// than trusting a cache (or a remembered failure) from a previous run
	// on a reused Market instance - see liveExpansions.
	mkm.liveExpansionsCache = nil
	mkm.liveExpansionsTried = false

	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	var candidates map[string]bool
	if _, filtered := marketFilterParams[mkm.gameID]; filtered {
		switch {
		case mkm.banPriceKey != "":
			snap, err := loadBanSnapshot(ctx, mkm.game, mkm.banPriceKey)
			if err != nil {
				return fmt.Errorf("loading the price snapshot to pre-filter this catalog: %w", err)
			}
			candidates = marketCandidates(mkm.backend, mkm.gameID, snap)
			mkm.printf("Restricting to %d of this game's uuids, from the price snapshot", len(candidates))
		case marketFilterRequired[mkm.gameID]:
			return fmt.Errorf("%s needs a pre-filtered candidate set to fit its scrape budget, and BanPriceKey is not set", mkm.game)
		default:
			mkm.printf("BanPriceKey not set - running %s unfiltered rather than pre-filtered", mkm.game)
		}
	}

	return mkm.walkCatalog(ctx, candidates)
}

// liveExpansions answers every expansion Cardmarket's live API currently
// has for this game, to fill a gap in mkm.catalog: mkm.catalog is built
// from MTGJSON's own CardmarketIdentifiers.json export, which is
// currently missing 88 of Magic's 761 real expansions (mostly
// individually-Cardmarket-ID'd Secret Lair drops MTGJSON never mapped,
// plus a handful of genuinely new sets it hasn't caught up to yet) -
// without this, every product under one of those 88 falls back to the
// literal edition string "expansion <id>", which nothing downstream
// recognises, so every one of those cards fails to resolve.
//
// This is a best-effort enrichment, not a hard dependency: with the
// current gap set, essentially every Magic run hits at least one empty
// Catalog entry, so treating a failed call here as fatal would turn one
// flaky Expansions response into zero inventory for the run, not just a
// missed enrichment for the gap set it was trying to help. A failure is
// logged and cached the same as a success - tried, not retried - so one
// bad response degrades back to today's "expansion <id>" placeholder for
// every remaining gap this run hits, rather than one bad response
// retrying once per gap (up to all 88, each through the client's own
// retry/backoff policy).
func (mkm *Market) liveExpansions(ctx context.Context) map[int]cm.Expansion {
	if mkm.liveExpansionsTried {
		return mkm.liveExpansionsCache
	}
	mkm.liveExpansionsTried = true

	live, err := mkm.client.Expansions(ctx, mkm.gameID)
	if err != nil {
		mkm.printf("could not fetch the live expansion list to fill Catalog's gaps: %v", err)
		return nil
	}
	byID := make(map[int]cm.Expansion, len(live))
	for _, exp := range live {
		byID[exp.IDExpansion] = exp
	}
	mkm.liveExpansionsCache = byID
	return byID
}

// resolveExpansionEntry substitutes entry from live when Catalog has no
// name for expansionID, split out from liveExpansions so the substitution
// itself can be tested without a live network call.
func resolveExpansionEntry(entry cm.CatalogExpansion, expansionID int, live map[int]cm.Expansion) cm.CatalogExpansion {
	if entry.Name != "" {
		return entry
	}
	if liveEntry, found := live[expansionID]; found {
		return cm.CatalogExpansion{Name: liveEntry.Name, Code: liveEntry.SetCode}
	}
	return entry
}

// walkCatalog prices every product of the id map, and of the product list
// beside it, expansion by expansion - the same shape Index.walkCatalog
// walks the catalog in, since resolving a Cardmarket product to a printing
// is exactly the same problem for both scrapers (see resolver). What
// differs is what happens once a product resolves: Index reads its price
// off the published guide, this asks the product's own live listings.
func (mkm *Market) walkCatalog(ctx context.Context, candidates map[string]bool) error {
	products := make(map[int]cm.CatalogProduct, len(mkm.catalog.Data.Products))
	for id, product := range mkm.catalog.Data.Products {
		products[id] = product
	}
	list, err := cm.DownloadProductListSingles(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	var unmapped int
	for _, entry := range list {
		_, found := products[entry.IDProduct]
		if found {
			continue
		}
		products[entry.IDProduct] = cm.CatalogProduct{ExpansionID: entry.ExpansionID, Name: entry.Name}
		unmapped++
	}
	mkm.printf("%d products of the list are not in the map and resolve by name", unmapped)

	byExpansion := map[int][]int{}
	for id, product := range products {
		byExpansion[product.ExpansionID] = append(byExpansion[product.ExpansionID], id)
	}

	var items []cm.Expansion
	for expansionID := range byExpansion {
		entry := mkm.catalog.Data.Expansions[expansionID]
		if entry.Name == "" {
			entry = resolveExpansionEntry(entry, expansionID, mkm.liveExpansions(ctx))
		}
		name := entry.Name
		if name == "" {
			name = fmt.Sprintf("expansion %d", expansionID)
		}
		if mkm.targetEdition != "" && name != mkm.targetEdition {
			continue
		}
		items = append(items, cm.Expansion{IDExpansion: expansionID, Name: name, SetCode: entry.Code})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].IDExpansion < items[j].IDExpansion })

	// See idmap.go's walkCatalog for why the non-English programs are
	// dropped here rather than left to the resolver.
	switch mkm.gameID {
	case cm.GameOnePiece, cm.GameYuGiOh:
		kept := items[:0]
		for _, exp := range items {
			if strings.HasSuffix(exp.SetCode, "-JP") || foreignShelf(exp.Name) {
				continue
			}
			kept = append(kept, exp)
		}
		items = kept
		if mkm.gameID == cm.GameOnePiece {
			mkm.shelved = shelvedSets(mkm.backend, items)
		}
	}

	mkm.printf("Parsing %d expansion ids from the id map", len(items))

	// A cancellable copy of ctx: the worker below cancels it the moment the
	// failsafe trips, which is what actually stops the walk from dispatching
	// further expansions - returning an error from one worker call would
	// otherwise just be logged, not treated as a reason to stop.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var stopped error

	total := len(items)
	var processed int
	walked, refused, foreign := mkm.collectPrices(ctx, items, func(ctx context.Context, exp cm.Expansion, channel chan<- responseChan) error {
		processed++
		err := mkm.walkExpansion(ctx, exp, byExpansion[exp.IDExpansion], products, candidates, channel, processed, total)
		if errors.Is(err, errTooManyBounces) {
			stopped = err
			cancel()
		}
		return err
	})
	if stopped != nil {
		return stopped
	}

	mkm.printf("Walked %d products, %d of which named no printing of ours", walked, refused)
	if foreign > 0 {
		mkm.printf("%d of those were products of a catalog we do not carry", foreign)
	}
	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.printf("Total number of prices found: %d", len(mkm.inventory))
	mkm.inventoryDate = time.Now()
	return nil
}

// collectPrices runs worker over every expansion and adds what it produces
// straight to the inventory - unlike Index's own collectPrices, nothing is
// held back to arbitrate between an id-resolved price and a named one,
// because nothing here competes for one printing's one price the way the
// price guide's columns do: every listing this scraper adds is one
// seller's own, so two products both naming a printing ordinarily means two
// sellers' worth of prices for it. AddStrict keeps both only while they
// actually disagree on seller, condition or price; when two products'
// cheapest listing for a condition lands on the very same seller at the
// very same price - plausible when Cardmarket's own catalog has split one
// physical product across two ids - it folds the second into the first's
// Quantity instead of adding a second line, and only drops a listing
// outright (ErrDuplicateEntry, silently ignored below) when the URL,
// quantity and bundle also match exactly. Twin products are still caught
// before ever reaching this, at the resolution step in walkExpansion, so as
// not to query the same live listings twice.
//
// Sequential by construction (concurrency 1, not a configurable field):
// the API tolerates almost no in-flight parallelism per token, so pooling
// workers the way Sealed does would only manufacture 429s here.
func (mkm *Market) collectPrices(ctx context.Context, items []cm.Expansion, worker func(context.Context, cm.Expansion, chan<- responseChan) error) (walked, refused, foreign int) {
	mtgban.WorkerPool(ctx, 1, items, worker, func(result responseChan) {
		if result.tally {
			walked += result.walked
			refused += result.refused
			foreign += result.foreign
			return
		}
		err := mkm.inventory.AddStrict(result.cardID, &result.entry)
		if err != nil && !errors.Is(err, mtgban.ErrDuplicateEntry) {
			mkm.printf("%d - %s", result.ogID, err.Error())
		}
	}, mkm.printf)
	return walked, refused, foreign
}

// walkExpansion resolves and prices every product of one expansion, the
// same way idmap.go's walkCatalog does for Index, swapping emitPrices'
// read off the price guide for queryPrintings' live calls, and skipping a
// product the offline pre-filter (candidates) leaves out.
func (mkm *Market) walkExpansion(ctx context.Context, exp cm.Expansion, ids []int, products map[int]cm.CatalogProduct, candidates map[string]bool, channel chan<- responseChan, index, total int) error {
	mkm.printf("Processing %s (%d) [%d/%d]", exp.Name, exp.IDExpansion, index, total)
	sort.Ints(ids)

	results := make([]resolved, 0, len(ids))
	for _, id := range ids {
		results = append(results, mkm.resolveMapped(id, products[id], exp))
	}
	if mkm.gameID == cm.GameFleshAndBlood {
		mkm.disownBridged(results)
	}
	if same := sameProduct(mkm.gameID); same != nil {
		twinsAmong(results, same, faceOf(mkm.backend, mkm.gameID))
	}

	var refusedNames []string
	named := map[string]int{}
	var twins, foreign, refusals, skipped, priced int
	for i := range results {
		r := &results[i]
		id, mapped := r.product.IDProduct, products[r.product.IDProduct]
		err := r.err
		if err == nil && r.cardID != "" {
			if !mkm.marketCandidateHit(candidates, r.cardID, r.cardIDFoil) {
				skipped++
			} else {
				err = mkm.queryPrintings(ctx, channel, r.product, r.cardID, r.cardIDFoil, r.byName)
				if errors.Is(err, errTooManyBounces) {
					return err
				}
				if err == nil {
					priced++
				}
			}
		}
		switch {
		case errors.Is(err, errTwin):
			twins++
		case errors.Is(err, errForeign):
			foreign++
		case errors.Is(err, errNoPrinting):
			refusals++
			key := fmt.Sprintf("%q (%s) in %s", mkm.refusalName(mapped.Name), mapped.Number, exp.Name)
			if at, seen := named[key]; seen {
				refusedNames[at] += "+"
				continue
			}
			named[key] = len(refusedNames)
			refusedNames = append(refusedNames, fmt.Sprintf("%d %s", id, key))
		case err != nil:
			mkm.printf("product id %d returned %s", id, err)
		}
	}

	mkm.reportRefused(exp.Name, len(ids), refusedNames, twins, foreign)
	if skipped > 0 {
		mkm.printf("%s: %d of %d products skipped, outside the pre-filter's candidates", exp.Name, skipped, len(ids))
	}
	// Unconditional, unlike the lines above: those only appear when there is
	// something to explain, so an edition with nothing skipped or refused
	// used to end its walk in silence - indistinguishable in the log from
	// one that priced nothing at all. This is the one line every edition
	// gets, so "Processing X" is always followed by what happened to it.
	mkm.printf("%s: priced %d/%d products", exp.Name, priced, len(ids))
	channel <- responseChan{tally: true, walked: len(ids), refused: refusals + twins + foreign, foreign: foreign}
	return nil
}

// marketMaxPages caps how many pages of a product's listings queryOnePrinting
// reads chasing NM, SP and MP: Content-Range says the true total up front,
// so this only ever matters for a product with far more listings than any
// real one carries.
const marketMaxPages = 20

// marketPowersellerExtraPages is how many pages past the point the main
// bucket's NM/SP/MP are all held queryOnePrinting keeps going, chasing at
// least one Powerseller listing before giving up on this product entirely -
// see shouldStopPaging. Bounded rather than open-ended: gating the stop on
// full Powerseller completeness would mean every product with no
// qualifying seller (most of them) pages all the way to Content-Range
// coverage instead of stopping early, which is exactly the budget this
// scraper is built around not spending. A few extra pages is a fixed,
// small cost paid once per product; unbounded completeness-chasing is not.
const marketPowersellerExtraPages = 4

// shouldStopPaging decides whether queryOnePrinting's page loop can stop
// once the main bucket's own NM/SP/MP are all held. mainSatisfiedAt is the
// page index that first became true on, or -1 if it has not yet. Stops
// immediately once a Powerseller listing has been found (nothing more to
// chase), or once marketPowersellerExtraPages have been spent looking
// without finding one - whichever comes first; the two other stopping
// conditions in queryOnePrinting (an empty page, Content-Range coverage)
// apply independently of this one and are not this function's concern.
func shouldStopPaging(mainDone bool, mainSatisfiedAt, page int, foundPowerseller bool) bool {
	if !mainDone {
		return false
	}
	return foundPowerseller || page-mainSatisfiedAt >= marketPowersellerExtraPages
}

// queryPrintings prices the printing(s) one product resolved to, from the
// product's own live listings: cardID alone for a game that sells each
// treatment as its own product, or when the resolver found no separate
// printing to split cardIDFoil from it; both, queried separately, for the
// games whose second column is a genuine second printing and whose
// distinguishing signal this scraper actually knows - see marketFinishParam.
// A game not in marketFinishParam at all would leave cardIDFoil unpriced
// here rather than guessed at from an unfiltered, unverified mix of
// listings; every game the resolver ever hands a distinct cardIDFoil for
// (Magic, YuGiOh, Lorcana, Riftbound) currently has one. Pokemon crosses
// two axes rather than one and is handled entirely on its own - see
// queryPokemonPrintings.
func (mkm *Market) queryPrintings(ctx context.Context, channel chan<- responseChan, product *cm.Product, cardID, cardIDFoil string, byName bool) error {
	if mkm.gameID == cm.GamePokemon {
		return mkm.queryPokemonPrintings(ctx, channel, product, cardID, byName)
	}

	finish, verified := marketFinishParam[mkm.gameID]
	var baseFlags map[string]bool
	if verified {
		baseFlags = map[string]bool{finish: false}
	}
	// A foil-only card resolves cardID to the same uuid as cardIDFoil below.
	if isPLSTFoil(mkm.backend, cardID) {
		return nil
	}
	err := mkm.queryOnePrinting(ctx, channel, product, cardID, byName, baseFlags)
	if err != nil {
		return err
	}
	if cardIDFoil == "" || cardIDFoil == cardID {
		return nil
	}
	if !verified {
		mkm.printf("id %d: %s has a second printing (%s) this scraper does not price yet", product.IDProduct, cardID, cardIDFoil)
		return nil
	}
	if isPLSTFoil(mkm.backend, cardIDFoil) {
		return nil
	}
	return mkm.queryOnePrinting(ctx, channel, product, cardIDFoil, byName, map[string]bool{finish: true})
}

// isPLSTFoil reports whether cardID is a foil printing from The List: it
// reprints a card under whatever treatment its original printing had, and
// sellers list its foil copies under the wrong finish often enough that
// pricing them isn't worth the noise.
func isPLSTFoil(b *mtgmatcher.Backend, cardID string) bool {
	co, err := b.GetUUID(cardID)
	return err == nil && co.SetCode == "PLST" && co.Foil
}

// queryPokemonPrintings prices every printing pokemonFinishPlan resolved for
// cardID, one live query per occupied Cardmarket cell - see pokemonFinishPlan
// on why this cannot be the plain cardID/cardIDFoil pair every other game
// uses.
func (mkm *Market) queryPokemonPrintings(ctx context.Context, channel chan<- responseChan, product *cm.Product, cardID string, byName bool) error {
	for _, target := range pokemonFinishPlan(mkm.backend, cardID) {
		flags := map[string]bool{"isFirstEd": target.isFirstEd, "isReverseHolo": target.isReverseHolo}
		if err := mkm.queryOnePrinting(ctx, channel, product, target.cardID, byName, flags); err != nil {
			return err
		}
	}
	return nil
}

// marketCandidateHit reports whether any of a product's priceable uuids
// clears the pre-filter. Every other game resolves to at most the plain
// cardID/cardIDFoil pair, but Pokemon can resolve to more (see
// pokemonFinishPlan), so its own full finish plan is asked rather than just
// the two - a candidate hiding behind a third or fourth uuid must not read
// as skipped.
func (mkm *Market) marketCandidateHit(candidates map[string]bool, cardID, cardIDFoil string) bool {
	if candidates == nil {
		return true
	}
	if candidates[cardID] || candidates[cardIDFoil] {
		return true
	}
	if mkm.gameID == cm.GamePokemon {
		for _, target := range pokemonFinishPlan(mkm.backend, cardID) {
			if candidates[target.cardID] {
				return true
			}
		}
	}
	return false
}

// acceptArticle reports whether one listing is valid to price at all -
// priced, from a seller who is not on vacation and outside
// excludedCountries, matching every finish flag actually being queried,
// and a recognised condition - and which of
// mtgban's five conditions it counts as. It does not compare against any
// held price: that happens separately, once per bucket that holds its own
// cheapest-so-far (see isCheaper), so a listing that is not the single
// global cheapest can still be the cheapest one that also qualifies for a
// narrower bucket like marketPowersellerName - sharing one held map across
// both would silently starve the narrower one of every listing that is not
// also the overall cheapest.
//
// flags names, for each Cardmarket parameter this query cares about, the
// value every accepted article's own flag must equal - a game with no
// finish signal to verify (or a request with none, like Pokemon's own
// (false, false) cell) passes an empty or nil map, which accepts regardless
// of finish rather than trusting a filter that is documented to fail open
// on a game or value it does not apply to.
func acceptArticle(flags map[string]bool, article cm.Article) (string, bool) {
	if article.Price == 0 {
		return "", false
	}
	if article.Seller.OnVacation {
		return "", false
	}
	if excludedCountries[article.Seller.Address.Country] {
		return "", false
	}
	for param, want := range flags {
		if articleFlagValue(param, &article) != want {
			return "", false
		}
	}
	cond, known := mkmCondition[article.Condition]
	if !known {
		return "", false
	}
	return cond, true
}

// isCheaper reports whether price is a new cheapest for cond in held -
// either nothing is held there yet, or price genuinely beats what is.
// held is one bucket's own record of the cheapest price accepted so far
// per condition, across every product and page scanned; queryOnePrinting
// keeps a separate held map per bucket precisely so this comparison never
// crosses between them.
//
// Listings come back in a rough price order, not a strictly monotonic
// one: replayed against a real page of results (see
// market_replay_test.go), a heavily bulk-priced product carried runs like
// 0.02, 0.02, 0.03, 0.03, 0.02, ... - the same handful of cent-level
// prices repeating out of order rather than climbing. Comparing against
// the held price instead of trusting the first acceptable listing catches
// that without costing an extra request: the page is scanned in full
// either way.
func isCheaper(held map[string]float64, cond string, price float64) bool {
	current, found := held[cond]
	return !found || price < current
}

// queryOnePrinting prices one printing from one product's live listings,
// holding the cheapest price per condition from a seller outside
// excludedCountries (see acceptArticle on why "cheapest", not "first", is
// the one actually held), and stopping once NM, SP and MP are all held: it
// is fine to miss HP and PO chasing them deeper, and the article filter's
// own minCondition floor already excludes both server-side in the common
// case. Nothing is sent to channel until the scan itself is done, so a
// cheaper listing for a condition found on an earlier page can still
// replace it before anything is reported.
//
// flags, for each Cardmarket parameter set true, is also sent to the
// server as a request-side filter - but never trusted alone: Cardmarket's
// filters fail open on a value or a game they do not apply to, silently
// answering the unfiltered list rather than an error, so every listing is
// also checked against the printing actually being priced through its own
// article-level flags before being accepted (see acceptArticle). A flag set
// false is never sent - Cardmarket's own behavior for an explicit false was
// never tested, only omitting the parameter - and is still verified
// accept-side.
func (mkm *Market) queryOnePrinting(ctx context.Context, channel chan<- responseChan, product *cm.Product, cardID string, byName bool, flags map[string]bool) error {
	co, err := mkm.backend.GetUUID(cardID)
	if err != nil {
		return err
	}

	options := map[string]string{
		"minCondition": "GD",
		"minUserScore": "3",
		"isSigned":     "false",
		"isAltered":    "false",
		"idLanguage":   strconv.Itoa(marketLanguage(co.Language)),
	}
	for param, want := range flags {
		if want {
			options[param] = "true"
		}
	}

	held := map[string]float64{}
	entries := map[string]responseChan{}
	heldPS := map[string]float64{}
	entriesPS := map[string]responseChan{}
	mainSatisfiedAt := -1
	for page := 0; page < marketMaxPages; page++ {
		articles, total, _, err := mkm.client.Articles(ctx, product.IDProduct, options, page, cm.MaxEntities)
		if err != nil {
			if mkm.bounce() {
				return fmt.Errorf("%w (%d in a row, last: %v)", errTooManyBounces, mkm.bounced, err)
			}
			return err
		}
		mkm.bounced = 0

		for _, article := range articles {
			cond, ok := acceptArticle(flags, article)
			if !ok {
				continue
			}

			link := cm.BuildURL(article.IDProduct, mkm.gameID, mkm.affiliate, cm.Finish{
				Foil:        article.IsFoil,
				FirstEd:     article.IsFirstEd,
				ReverseHolo: article.IsReverseHolo,
			})
			customFields := map[string]string{
				"SubSellerName": article.Seller.Username,
				"SubSellerGeo":  article.Seller.Address.Country,
			}

			if isCheaper(held, cond, article.Price) {
				held[cond] = article.Price
				entries[cond] = responseChan{
					ogID:    product.IDProduct,
					product: product,
					cardID:  cardID,
					byName:  byName,
					entry: mtgban.InventoryEntry{
						Conditions:   cond,
						Price:        article.Price * mkm.exchangeRate,
						Quantity:     article.Count,
						SellerName:   marketMainName,
						URL:          link,
						OriginalID:   fmt.Sprint(article.IDProduct),
						InstanceID:   fmt.Sprint(article.IDArticle),
						CustomFields: customFields,
					},
				}
			}

			if isPowerseller(article) && isCheaper(heldPS, cond, article.Price) {
				heldPS[cond] = article.Price
				entriesPS[cond] = responseChan{
					ogID:    product.IDProduct,
					product: product,
					cardID:  cardID,
					byName:  byName,
					entry: mtgban.InventoryEntry{
						Conditions:   cond,
						Price:        article.Price * mkm.exchangeRate,
						Quantity:     article.Count,
						SellerName:   marketPowersellerName,
						URL:          link,
						OriginalID:   fmt.Sprint(article.IDProduct),
						InstanceID:   fmt.Sprint(article.IDArticle),
						CustomFields: customFields,
					},
				}
			}
		}

		mainDone := held["NM"] != 0 && held["SP"] != 0 && held["MP"] != 0
		if mainDone && mainSatisfiedAt == -1 {
			mainSatisfiedAt = page
		}
		if shouldStopPaging(mainDone, mainSatisfiedAt, page, len(heldPS) > 0) {
			break
		}
		if len(articles) == 0 {
			break
		}
		if contentRangeCovered(page+1, cm.MaxEntities, total) {
			break
		}
	}

	for _, out := range entries {
		channel <- out
	}
	for _, out := range entriesPS {
		channel <- out
	}
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mkm *Market) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *Market) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardmarket"
	info.Shorthand = "MKM"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.Game = mkm.game
	return
}

// marketMainName and marketPowersellerName are the two sub-sellers Market
// splits its inventory into - see MarketNames. Every entry gets one of
// these two as its SellerName, the same way cardtrader.Market's three
// storefronts do; the article's own username moves to
// CustomFields["SubSellerName"] instead.
const (
	marketMainName        = "Cardmarket"
	marketPowersellerName = "Cardmarket Powersellers"
)

var marketName2Shorthand = map[string]string{
	marketMainName:        "MKM",
	marketPowersellerName: "MKMPS",
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market. Every product query holds a second, independent
// cheapest-per-condition view for marketPowersellerName alongside the main
// one - see queryOnePrinting and shouldStopPaging. Once the main view's
// NM/SP/MP are all held, the page loop spends up to
// marketPowersellerExtraPages more pages specifically chasing at least one
// Powerseller listing before giving up on this product - a bounded, paid-
// once cost, not a guarantee that a printing with genuinely no qualifying
// seller anywhere in its listings will ever be found; that product simply
// contributes nothing to this bucket, same as cardtrader.Market's
// Zero/1DR splits do for a product neither storefront carries.
func (mkm *Market) MarketNames() []string {
	return []string{marketMainName, marketPowersellerName}
}

// InfoForScraper describes one of the sub-scrapers named above.
func (mkm *Market) InfoForScraper(name string) mtgban.ScraperInfo {
	info := mkm.Info()
	info.Name = name
	info.Shorthand = marketName2Shorthand[name]
	return info
}
