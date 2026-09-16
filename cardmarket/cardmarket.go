// Package cardmarket scrapes Cardmarket, both the price-guide index and
// sealed product, across every game they carry.
package cardmarket

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type responseChan struct {
	ogID   int
	cardID string
	entry  mtgban.InventoryEntry
	// byName marks a price whose printing was named rather than looked
	// up by id, which is a guess however well guarded; see namedLast.
	byName bool
	// product is what was priced, for the collector to tell a twin of a
	// product already priced from a disagreement worth reporting.
	product *cm.Product
	// tally carries an edition's walked and refused counts in place of a
	// price, one record per edition, so the pool's single collector can
	// count the run without the workers sharing anything.
	tally   bool
	walked  int
	refused int
	// foreign is how many of the refused were products of a catalog we do
	// not carry, which the run reports as its own figure: they are shelves
	// nobody can act on, where the rest of the refusals are work.
	foreign int
}

// namedLast holds back the prices whose printing was named until every
// price looked up by an id is in.
//
// The catalogs keep one expansion per old set where the datastore keeps a
// printing per print run, so the set guard puts every run's product on the
// base set's printing, and a run the bridge already priced is a run a name
// can reach too. AddUnique keeps whichever price arrives first, so without
// the wait the winner is whichever expansion the pool happened to walk
// first - and half the time that hands a verified printing over to a guess
// about a different one. Waiting decides it instead: the guess is offered
// only where nothing verified stands.
type namedLast struct {
	add     func(responseChan)
	results []responseChan
	// held is what priced each printing so far, keyed by uuid and by the
	// name of the price column, so a named price for a printing another
	// product of the same name already holds gives way silently: it is
	// the same card sold again on another shelf, and the inventory would
	// only refuse it out loud.
	held map[string]*cm.Product
	// twin says whether two products are the same card sold twice, for
	// the games whose shelves do that; nil leaves every collision to the
	// inventory.
	twin func(a, b *cm.Product) bool
	// face says whether a product names one face of the fused printing
	// it is beside, the other way a shelf sells one card twice
	face  func(product *cm.Product, cardID string) bool
	twins int
	// The run's tally, summed from the editions' records; the collector
	// runs on one goroutine, so plain counts are all this takes.
	walked  int
	refused int
	foreign int
}

// collect takes one result, holding it back or counting it into the run's
// tally. Every price waits for flush, the named ones and the rest alike:
// the pool walks the expansions in whatever order they finish, and which of
// two products reaching one printing arrived first would otherwise be
// decided by that, so a promo sold on two shelves took one shelf's price
// this run and the other's next. flush puts them in the catalog's order.
func (n *namedLast) collect(result responseChan) {
	if result.tally {
		n.walked += result.walked
		n.refused += result.refused
		n.foreign += result.foreign
		return
	}
	n.results = append(n.results, result)
}

// hold records what priced a printing's column, and reports whether the
// price is the first of its product for it: a product named like the one
// already there is the same card sold again on another shelf, and the
// inventory would only refuse it out loud.
func (n *namedLast) hold(result responseChan) bool {
	if n.held == nil {
		n.held = map[string]*cm.Product{}
	}
	key := result.cardID + "|" + result.entry.SellerName
	if holder := n.held[key]; holder != nil && result.product != nil {
		if (n.twin != nil && n.twin(holder, result.product)) || (n.face != nil && n.face(result.product, result.cardID)) {
			n.twins++
			return false
		}
	}
	n.held[key] = result.product
	return true
}

// flush adds everything held back, the prices looked up by id first and
// the named ones after them, each in the order of the catalog - expansion,
// then product - and reports how many named prices went in and how many
// gave way to a product already priced.
func (n *namedLast) flush() (added, twins int) {
	sort.SliceStable(n.results, func(i, j int) bool {
		a, b := n.results[i], n.results[j]
		if a.byName != b.byName {
			return !a.byName
		}
		return productBefore(a.product, b.product)
	})
	before := n.twins
	for i := range n.results {
		if !n.hold(n.results[i]) {
			continue
		}
		n.add(n.results[i])
		if n.results[i].byName {
			added++
		}
	}
	return added, n.twins - before
}

// productBefore orders two products the way the catalog files them, by
// expansion and then by product id, with a result carrying no product last.
func productBefore(a, b *cm.Product) bool {
	if a == nil || b == nil {
		return a != nil && b == nil
	}
	if a.Expansion.IDExpansion != b.Expansion.IDExpansion {
		return a.Expansion.IDExpansion < b.Expansion.IDExpansion
	}
	if a.ExpansionName != b.ExpansionName {
		return a.ExpansionName < b.ExpansionName
	}
	return a.IDProduct < b.IDProduct
}

// Index prices singles from Cardmarket's price guide, the low and
// trend numbers rather than any one seller's listing.
type Index struct {
	resolver

	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int
	exchangeRate   float64

	inventory mtgban.InventoryRecord

	// priceGuide holds one game's published prices, indexed by the product
	// id they belong to: a run asks for one product's prices tens of
	// thousands of times, once per product in the catalog.
	priceGuide map[int]cm.PriceGuide

	game mtgban.Game
}

var availableIndexNames = []string{
	"MKM Low", "MKM Trend",
}

var name2shorthand = map[string]string{
	"MKM Low":   "MKMLow",
	"MKM Trend": "MKMTrend",
}

// mkmGames is the only way into these scrapers: a game names its Cardmarket
// id here or it is not one this package is read for.
var mkmGames = map[mtgban.Game]int{
	mtgban.GameMagic:         cm.GameMagic,
	mtgban.GameLorcana:       cm.GameLorcana,
	mtgban.GameRiftbound:     cm.GameRiftbound,
	mtgban.GameOnePiece:      cm.GameOnePiece,
	mtgban.GameYuGiOh:        cm.GameYuGiOh,
	mtgban.GameFleshAndBlood: cm.GameFleshAndBlood,
	mtgban.GamePokemon:       cm.GamePokemon,
}

func (mkm *Index) printf(format string, a ...any) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper for one game. It prices from the
// published catalog and the public price guide, so it needs no credential.
func NewScraperIndex(game mtgban.Game) (*Index, error) {
	id, found := mkmGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	mkm := Index{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.MaxConcurrency = defaultConcurrency
	mkm.game = game
	mkm.gameID = id
	mkm.resolver.printf = mkm.printf
	return &mkm, nil
}

// versionTail matches the parenthetical Cardmarket tells same-name products
// apart with, which names its own version index and the rarity beside it.
var versionTail = regexp.MustCompile(` \(V\.\d+.*\)$`)

// rarityTail captures the rarity out of that same parenthetical, which is
// the only place Cardmarket writes it.
var rarityTail = regexp.MustCompile(` \(V\.\d+ - ([^)]+)\)$`)

// numberTail matches the digits a collector number ends on, which is the part
// two catalogs numbering the same card agree about.
var numberTail = regexp.MustCompile(`\d+[A-Za-z]?$`)

// nameCode matches the collector number Cardmarket writes at the end of a One
// Piece product name, beside the one it writes in the number field.
var nameCode = regexp.MustCompile(`\(([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\)$`)

// onePieceNumber picks the collector number a One Piece product is asked
// with, out of the two Cardmarket writes for it.
//
// The number field is sometimes another card's ("Sanji (OP01-013)" filed
// under ST13-016, "Corrida Coliseum (OP04-096)" under OP04-092) and
// sometimes a typo of the name's own code with a digit doubled ("ST13-0151",
// "P-0611"). The code inside the name is the card's, so it answers for the
// product when the two disagree.
//
// Only on a shelf naming a set of ours, though. Cardmarket sells its promos
// in buckets no set of ours answers for - "Judge Promos", "Winner Cards",
// "Premium Bandai Products" - and with nothing to hold it to a set the
// matcher reaches past the edition and lands on the ordinary booster
// printing of that number, which is a mass-printed card wearing a promo's
// price. A refusal says less but claims nothing.
func onePieceNumber(name, number, expansion string) string {
	fields := nameCode.FindStringSubmatch(name)
	if fields == nil || strings.EqualFold(fields[1], number) {
		return number
	}
	if _, err := mtgmatcher.GetSetByName(expansion); err != nil {
		return number
	}
	return fields[1]
}

// shelvedSets names, for each set of ours, the one expansion of this catalog
// that sells it. A set no expansion names is absent, and so is an expansion
// naming no set.
func shelvedSets(list []cm.Expansion) map[string]string {
	shelved := make(map[string]string, len(list))
	for _, exp := range list {
		set, err := mtgmatcher.GetSetByName(exp.Name)
		if err != nil {
			continue
		}
		shelved[set.Code] = exp.Name
	}
	return shelved
}

// numberPrefix returns the letters a collector number opens on, the set code
// aside: "EN005" yields "EN", "DCR-005" yields "", "SGX1-END19" yields "END".
func numberPrefix(number string) string {
	if _, tail, dashed := strings.Cut(number, "-"); dashed {
		number = tail
	}
	return strings.ToUpper(number[:len(number)-len(strings.TrimLeft(number, letters))])
}

// letters spells the alphabet a collector number's prefix is drawn from.
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// otherPrintRun reports whether a product's collector number names a print
// run other than the one the answer's number belongs to.
//
// Cardmarket sells the North American print of an old Yu-Gi-Oh set ("005"),
// the European ("EN005") and the Asian ("A005") as three products of one
// expansion, and files the special-edition promos beside them ("SP1"), where
// the datastore carries the single row the set is numbered by ("DCR-005").
// The matcher reads none of those prefixed numbers - its own numbering opens
// on digits or on a set code - so it drops them and answers the name alone,
// which puts every run on that one row. The prefix has to be the answer's
// own, the region infix the datastore writes and Cardmarket omits aside
// ("D19" is "SGX1-END19", "EN005" is not "DCR-005").
func otherPrintRun(number, full string) bool {
	prefix := numberPrefix(number)
	return prefix != "" && !strings.HasSuffix(numberPrefix(full), prefix)
}

// productFinish names the printing a product is, for the catalogs that sell
// each printing as its own product rather than as a column beside the card.
func productFinish(gameID int, product *cm.Product) string {
	if gameID == cm.GameFleshAndBlood {
		return fabFinish(product.ExpansionName, product.Name)
	}
	return ""
}

// foreignShelves are the tails Cardmarket appends to a set's name when it
// shelves that set's non-English printings apart from the English ones. They
// are separate catalogs of the same cards, and the datastore carries only the
// English ones, so a price from one of these shelves would land on a printing
// it is not.
var foreignShelves = []string{"(Japanese)", "(Korean)", "(PMT)"}

// foreignShelf reports whether an expansion name wears one of those tails.
func foreignShelf(name string) bool {
	for _, tail := range foreignShelves {
		if strings.HasSuffix(name, tail) {
			return true
		}
	}
	return false
}

// emitPrices lands a product's guide prices on the printings resolved for
// it, the plain columns on one and the foil columns on the other, in the
// games that split them.
func (mkm *Index) emitPrices(channel chan<- responseChan, product *cm.Product, cardID, cardIDFoil string, byName bool) error {
	// Look for the price presence
	guide, found := mkm.priceGuide[product.IDProduct]
	if !found {
		return fmt.Errorf("IdProduct %d not found in cm.PriceGuide", product.IDProduct)
	}

	// Sorted as availableIndexNames
	prices := []float64{guide.LowPrice, guide.TrendPrice}
	foilLow, foilTrend := guide.SecondPrinting(mkm.gameID)
	foilprices := []float64{foilLow, foilTrend}

	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return err
	}

	// A catalog that gives each treatment its own product prices one
	// printing per product, and the product's own columns are that
	// printing's whatever its finish - there is no second column for them
	// to be in. Every other catalog keeps a second printing beside the
	// first and splits the two across the columns: the foil beside the
	// plain card, or in Pokemon's guide the reverse holo beside whatever
	// the card's own printing is, holo or plain or a print run. The finish
	// the loader stored says which side of that split the printing is on;
	// the foil flag cannot, a Pokemon holo being a foil to the flag and a
	// printing of its own to the guide - which is how the holos were priced
	// from the reverse's columns and the reverses from nothing.
	perTreatment := mkm.gameID == cm.GameFleshAndBlood || mkm.gameID == cm.GameOnePiece
	second := co.Finish != mtgmatcher.FinishNonfoil
	if mkm.gameID == cm.GamePokemon {
		// This is reverse-holo-only - the same blindness Market's own
		// resolveProduct had until it was fixed to cross both of Pokemon's
		// finish axes (see pokemonFinishPlan) - so a 1st Edition printing
		// still reads as "not the second column" here regardless of print
		// run. Closing mtgban/go-mtgban#641 does not change this line: the
		// price guide has no 1st-Edition column at all, unlike Market's
		// per-listing live query, so the fix is below instead - filing the
		// same blended first-pair price under every uuid this product's
		// "not reverse holo" side actually resolves to, not splitting this
		// check itself into a third case.
		second = co.Finish == mtgmatcher.NormalizeFinish(pokemonReverseHolo)
	}

	// A printing on the first side takes the first pair and hands the
	// second pair to the printing beside it; one on the second side is
	// priced by the second pair alone.
	if perTreatment || !second {
		link := cm.BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, false)

		// The first pair's target(s): cardID alone for every other game,
		// but for Pokemon the guide's low/trend blend every listing of the
		// product that is not specifically Reverse Holofoil - Unlimited,
		// 1st Edition, and both their Holofoil crossings all read as one
		// number here, because the guide has no column of its own for the
		// print-run axis at all (mtgban/go-mtgban#641). Filing the same
		// blended number under every one of those uuids is a real
		// improvement on filing it under only one and leaving the rest
		// unpriced entirely, even though it cannot separate what a live
		// listing would: Market's own per-listing query can tell 1st
		// Edition apart from Unlimited (see pokemonFinishPlan) because it
		// asks Cardmarket per finish; the price guide never breaks the two
		// out to begin with, so there is no better number to give either
		// one here.
		targets := []string{cardID}
		if mkm.gameID == cm.GamePokemon {
			for _, target := range pokemonFinishPlan(cardID) {
				if target.isReverseHolo || target.cardID == cardID {
					continue
				}
				targets = append(targets, target.cardID)
			}
		}

		for _, id := range targets {
			for i := range availableIndexNames {
				if prices[i] == 0 {
					continue
				}

				out := responseChan{
					ogID:    product.IDProduct,
					product: product,
					cardID:  id,
					byName:  byName,
					entry: mtgban.InventoryEntry{
						Conditions: "NM",
						Price:      prices[i] * mkm.exchangeRate,
						URL:        link,
						SellerName: availableIndexNames[i],
						OriginalID: fmt.Sprint(product.IDProduct),
					},
				}

				channel <- out
			}
		}

		if !perTreatment && (foilprices[0] != 0 || foilprices[1] != 0) {
			link := cm.BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, true)

			// An empty foil id means the card has no foil printing (Match
			// errored on the foil probe), so residual foil prices in the
			// guide have nothing to attach to
			if cardIDFoil != "" && cardID != cardIDFoil {
				for i := range availableIndexNames {
					if foilprices[i] == 0 {
						continue
					}
					out := responseChan{
						ogID:    product.IDProduct,
						product: product,
						cardID:  cardIDFoil,
						byName:  byName,
						entry: mtgban.InventoryEntry{
							Conditions: "NM",
							Price:      foilprices[i] * mkm.exchangeRate,
							URL:        link,
							SellerName: availableIndexNames[i],
							OriginalID: fmt.Sprint(product.IDProduct),
						},
					}

					channel <- out
				}
			}
		}
	} else {
		link := cm.BuildURL(product.IDProduct, mkm.gameID, mkm.Affiliate, true)

		for i := range availableIndexNames {
			if foilprices[i] == 0 {
				continue
			}
			out := responseChan{
				ogID:    product.IDProduct,
				product: product,
				cardID:  cardID,
				byName:  byName,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      foilprices[i] * mkm.exchangeRate,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalID: fmt.Sprint(product.IDProduct),
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mkm *Index) Load(ctx context.Context) error {
	err := mkm.checkCatalog()
	if err != nil {
		return err
	}

	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	priceGuide, err := cm.DownloadPriceGuide(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	// An empty guide is a download that went wrong, not a day with no
	// prices, and walking the catalog against it logs one miss per product.
	if len(priceGuide) == 0 {
		return errors.New("empty price guide")
	}
	mkm.priceGuide = make(map[int]cm.PriceGuide, len(priceGuide))
	for _, entry := range priceGuide {
		mkm.priceGuide[entry.IDProduct] = entry
	}

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	return mkm.walkCatalog(ctx)
}

// collectPrices runs worker over every expansion and files what it produces
// into the inventory, prices whose printing was named last. It is where the
// wait namedLast describes is actually taken: the pool hands its results to
// the collector rather than to the inventory, so a named price cannot win a
// printing merely by being walked first.
func (mkm *Index) collectPrices(ctx context.Context, items []cm.Expansion, worker func(context.Context, cm.Expansion, chan<- responseChan) error) (walked, refused, foreign int) {
	// The bridge is keyed by the Cardmarket id and valued by the TCGplayer
	// one, and a cardtrader blueprint names every Cardmarket product it
	// sells as, so nothing stops two products from resolving to one
	// printing. An index wants a single price per name per uuid, and a
	// second one is worth the log line the callback already prints rather
	// than a second row no consumer can choose between.
	add := mkm.inventory.AddStrict
	switch mkm.gameID {
	case cm.GameYuGiOh, cm.GameFleshAndBlood, cm.GamePokemon:
		add = mkm.inventory.AddUnique
	}

	addOne := func(result responseChan) {
		err := add(result.cardID, &result.entry)
		if err != nil {
			card, cerr := mtgmatcher.GetUUID(result.cardID)
			if cerr != nil {
				mkm.printf("%d - %s: %s", result.ogID, cerr.Error(), result.cardID)
				return
			}
			// Skip too many errors
			if mtgmatcher.IsToken(card.Name) ||
				card.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(card.Edition, "World Championship Decks") {
				return
			}
			mkm.printf("%d - %s", result.ogID, err.Error())
		}
	}

	collector := namedLast{add: addOne, twin: sameProduct(mkm.gameID), face: faceOf(mkm.gameID)}

	mtgban.WorkerPool(ctx, mkm.MaxConcurrency, items, worker, collector.collect, mkm.printf)

	added, _ := collector.flush()
	mkm.printf("Adding %d prices whose printing was named", added)
	if collector.twins > 0 {
		mkm.printf("%d prices gave way to a product of the same name already priced", collector.twins)
	}
	return collector.walked, collector.refused, collector.foreign
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mkm *Index) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (mkm *Index) MarketNames() []string {
	return availableIndexNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (mkm *Index) InfoForScraper(name string) mtgban.ScraperInfo {
	info := mkm.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *Index) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.MetadataOnly = true
	info.Family = "MKM"
	info.Game = mkm.game
	return
}
