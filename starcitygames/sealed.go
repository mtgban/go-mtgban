package starcitygames

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// Sealed prices SCG's sealed product.
type Sealed struct {
	logCallback   mtgban.LogCallbackFunc
	inventoryDate time.Time
	buylistDate   time.Time

	affiliate string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	productMap map[string]string
	setIDs     map[string]int
	dropped    map[string]int
	client     *SCGClient
	backend    *matcher.Backend
	gameID     int

	// priced holds how each product a run priced was placed, by its sku.
	priced map[string]mtgban.SealedVia
}

// pricedProducts counts the products a run came away with a price for, on
// either side of the counter. Star City Games quotes a buy price on plenty of
// sealed product it holds no stock of, so counting the shelf alone would move
// this number with stock rather than with coverage - the very reading the drop
// tally beside it exists to prevent.
func (scg *Sealed) pricedProducts() int {
	priced := map[string]struct{}{}
	for uuid := range scg.inventory {
		priced[uuid] = struct{}{}
	}
	for uuid := range scg.buylist {
		priced[uuid] = struct{}{}
	}
	return len(priced)
}

// drop records why a catalog product carried no price into the records.
func (scg *Sealed) drop(reason string) {
	if scg.dropped == nil {
		scg.dropped = map[string]int{}
	}
	scg.dropped[reason]++
}

// markPriced records that a catalog product holds a price, and how it was
// placed.
func (scg *Sealed) markPriced(sku string, via mtgban.SealedVia) {
	if scg.priced == nil {
		scg.priced = map[string]mtgban.SealedVia{}
	}
	scg.priced[sku] = via
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

// NewScraperSealed returns a sealed scraper for one game, using the given API
// key.
func NewScraperSealed(b *matcher.Backend, apiKey string) (*Sealed, error) {
	game := b.Game
	gameID, ok := scgGames[game]
	if !ok {
		return nil, fmt.Errorf("unsupported game %q", game)
	}
	scg := Sealed{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(apiKey)
	scg.backend = b
	scg.gameID = gameID
	return &scg, nil
}

func (scg *Sealed) printf(format string, a ...any) {
	if scg.logCallback != nil {
		scg.logCallback("[SCGSealed] "+format, a...)
	}
}

// buildProductMap indexes the sealed products by their SCG id (the catalog SKU)
// so a catalog product can be resolved to its mtgban uuid directly.
func buildProductMap(b *matcher.Backend) map[string]string {
	out := map[string]string{}
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		scgID, found := co.Identifiers["scgId"]
		if !found {
			continue
		}
		out[scgID] = uuid
	}
	return out
}

// sealedHoldsOwnID is the refusal of a listing whose name reaches a product
// the datastore already lists a SKU for.
const sealedHoldsOwnID = "product holds its own sku"

// runTag matches the print run Star City Games writes into a sealed product
// name, where the datastore writes it as a bracketed suffix.
var runTag = regexp.MustCompile(`\s*\((1st Edition|Unlimited)\)\s*`)

// sealedProductName spells a catalog product the way the datastore names the
// same box. Star City Games opens a sealed name with the game it belongs to,
// writes the print run in the middle rather than at the end, and calls a case
// of boxes a "Booster Case" where the datastore calls it a Booster Box Case.
// None of that is the product's identity, and the name-resolution rule needs
// every word of the datastore's name accounted for, so a word only one side
// writes loses the product.
//
// The game prefix only comes off where the catalog's game field is spelled the
// way the name opens, which is Flesh and Blood alone: Riftbound names still
// open "Riftbound: League of Legends TCG - " where the game field says only
// "Riftbound", and Lorcana names open "Lorcana: " with no dash at all. Their
// game words go to the resolver, which forgives words the vendor says over the
// datastore's name - measured over the cached sealed catalogs, every Riftbound
// and Lorcana product resolves to the same printing whether or not its true
// prefix is taken off, while dropping this line loses the four Flesh and Blood
// Hero Decks outright. So the trim earns its place for the one game that needs
// it, and is left alone rather than taught spellings nothing rides on.
func sealedProductName(p CatalogProduct) string {
	name := strings.TrimPrefix(p.Name, p.Game+" - ")

	var run string
	if match := runTag.FindStringSubmatch(name); match != nil {
		run = match[1]
		if run == "Unlimited" {
			run = "Unlimited Edition"
		}
		name = runTag.ReplaceAllString(name, " ")
	}

	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "Booster Case", "Booster Box Case")
	if run != "" {
		name += " [" + run + "]"
	}
	return name
}

func (scg *Sealed) processProduct(p CatalogProduct) {
	// A single malformed product must never abort the whole catalog stream;
	// recover, log, and skip it.
	defer func() {
		if r := recover(); r != nil {
			scg.printf("recovered from panic on %q (sku=%s): %v", p.Name, p.SKU, r)
		}
	}()

	// This scraper handles sealed only; singles have their own scraper,
	// and the catalog also carries supplies and bulk lots. product_type
	// states it outright, where the sku prefix only implied it.
	if !scg.isSealedOf(p) {
		return
	}

	match := scg.matchProduct(p)
	if match.UUID == "" {
		// The name resolver is the whole reason this scraper prices a
		// fraction of the sealed catalog for the products with no sku in the
		// datastore, so say which name and which refusal, the way the
		// singles path says which card it could not place. Magic is exempt:
		// it is keyed by sku and never asks the resolver, so its whole
		// unlisted catalog would be named here for nothing.
		if match.Via == mtgban.SealedRefused {
			scg.printf("%q (sku=%s): %s", sealedProductName(p), p.SKU, match.Reason)
		}
		scg.drop(match.Reason)
		return
	}
	uuid := match.UUID

	link := SCGProductURL(p.URL, "", scg.affiliate)

	// The buylist link points at the sell-your-cards page for this product.
	// Sealed products carry no catalog set, so match the set off the product
	// name; fall back to retail if nothing matches.
	buyURL := link
	ids := setIDsForProduct(scg.backend, scg.setIDs, p.Name, p.SKU)
	if len(ids) > 0 {
		buyURL = SCGBuylistURL(scg.gameID, p.Name, p.Language, ids)
	}

	for _, v := range p.Variants {
		retailPrice, _ := matcher.ParsePrice(v.Price)

		if retailPrice > 0 && v.Qty > 0 {
			entry := &mtgban.InventoryEntry{
				Price:      retailPrice,
				Quantity:   v.Qty,
				OriginalID: p.SKU,
				InstanceID: v.SKU,
				URL:        SCGProductURL(p.URL, v.SKU, scg.affiliate),
			}
			err := scg.inventory.Add(uuid, entry)
			if err != nil {
				scg.printf("%s", err.Error())
			} else {
				scg.markPriced(p.SKU, match.Via)
			}
		}

		if buyPrice, err := matcher.ParsePrice(v.SellListPrice); err == nil && buyPrice > 0 {
			var priceRatio float64
			if retailPrice > 0 {
				priceRatio = buyPrice / retailPrice * 100
			}

			entry := &mtgban.BuylistEntry{
				BuyPrice:   buyPrice,
				PriceRatio: priceRatio,
				URL:        buyURL,
				OriginalID: v.SKU,
			}
			err = scg.buylist.Add(uuid, entry)
			if err != nil {
				scg.printf("%s", err.Error())
			} else {
				scg.markPriced(p.SKU, match.Via)
			}
		}
	}
}

// sealedUnlisted is why a Magic product the datastore holds no sku for is
// dropped; Magic sealed is matched by sku alone.
const sealedUnlisted = "sku the datastore does not carry"

// matchProduct decides which datastore product a sealed catalog product is,
// before anything about its stock or price is read. Sealed products are keyed
// by their SKU, which mtgban stores as the scgId; outside Magic a SKU the
// datastore does not carry resolves by name instead, English only, unique or
// nothing, and never onto a product that holds a SKU of its own.
func (scg *Sealed) matchProduct(p CatalogProduct) mtgban.SealedMatch {
	match := mtgban.SealedMatch{StoreID: p.SKU, Name: p.Name}
	if uuid, found := scg.productMap[p.SKU]; found {
		match.UUID = uuid
		match.Via = mtgban.SealedByID
		return match
	}
	if scg.backend.Game == matcher.GameMagic {
		match.Via = mtgban.SealedSkipped
		match.Reason = sealedUnlisted
		return match
	}
	if p.Language != "" && p.Language != "English" {
		match.Via = mtgban.SealedSkipped
		match.Reason = "not sold in English"
		return match
	}
	if matcher.SealedIsLanguageVariant(p.Name) {
		match.Via = mtgban.SealedSkipped
		match.Reason = "language variant"
		return match
	}
	uuid, err := scg.backend.ResolveSealed(sealedProductName(p))
	if err != nil {
		match.Via = mtgban.SealedRefused
		match.Reason = err.Error()
		return match
	}
	co, err := scg.backend.GetUUID(uuid)
	if err == nil && co.Identifiers["scgId"] != "" {
		match.Via = mtgban.SealedRefused
		match.Reason = sealedHoldsOwnID
		return match
	}
	match.UUID = uuid
	match.Via = mtgban.SealedByName
	return match
}

// isSealedOf reports whether a catalog product is one this scraper reads:
// sealed, of its game.
func (scg *Sealed) isSealedOf(p CatalogProduct) bool {
	return p.ProductType == ProductTypeSealed && gameFromCatalog(p.Game) == scg.gameID
}

// newSealedMatcher builds the scraper the export matches through, refusing
// Magic, which is matched by sku alone, and every game the sealed scraper
// is not registered for.
func newSealedMatcher(b *matcher.Backend) (*Sealed, error) {
	gameID, ok := scgGames[b.Game]
	if !ok || !slices.Contains(starcitygamesGames, b.Game) {
		return nil, fmt.Errorf("no sealed scraper for game %q", b.Game)
	}
	if b.Game == matcher.GameMagic {
		return nil, errors.New("magic sealed is matched by sku alone")
	}
	return &Sealed{backend: b, gameID: gameID, productMap: buildProductMap(b)}, nil
}

// MatchSealedCatalog streams the catalog export the way Load does and
// answers what the scraper's matching decides for each sealed product of the
// backend's game, by sku. It prices nothing. A product with no sku, or one
// so malformed its matching panics, is left out. A failed stream, or a
// catalog holding no product of the game, is an error.
func MatchSealedCatalog(ctx context.Context, b *matcher.Backend, apiKey string) ([]mtgban.SealedMatch, error) {
	scg, err := newSealedMatcher(b)
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, errors.New("missing Star City Games API key")
	}
	return scg.matchSealedCatalog(ctx, NewSCGClient(apiKey))
}

// matchSealedCatalog is the export's stream and match, over client.
func (scg *Sealed) matchSealedCatalog(ctx context.Context, client *SCGClient) ([]mtgban.SealedMatch, error) {
	var products []CatalogProduct
	count, ofGame := 0, 0
	err := client.StreamCatalog(ctx, func() {
		products = nil
		count, ofGame = 0, 0
	}, func(p CatalogProduct) error {
		if gameFromCatalog(p.Game) == scg.gameID {
			ofGame++
		}
		if scg.isSealedOf(p) {
			products = append(products, p)
		}
		count++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("catalog load failed: %w", err)
	}
	if ofGame == 0 {
		return nil, fmt.Errorf("none of %d catalog products maps to %s", count, scg.backend.Game)
	}
	return scg.matchCatalog(products), nil
}

// matchCatalog matches every sealed product of the game in the catalog, one
// answer per sku, a placed answer kept over one placing nothing, sorted by
// sku. A product with no sku, or whose matching panics, is left out.
func (scg *Sealed) matchCatalog(products []CatalogProduct) []mtgban.SealedMatch {
	answers := map[string]mtgban.SealedMatch{}
	for _, p := range products {
		if p.SKU == "" || !scg.isSealedOf(p) {
			continue
		}
		prev, found := answers[p.SKU]
		if found && prev.UUID != "" {
			continue
		}
		match, ok := scg.matchOne(p)
		if !ok || (found && match.UUID == "") {
			continue
		}
		answers[p.SKU] = match
	}
	return slices.SortedFunc(maps.Values(answers), func(a, b mtgban.SealedMatch) int {
		return strings.Compare(a.StoreID, b.StoreID)
	})
}

// matchOne is matchProduct for the export, answering false for a malformed
// product whose matching panics, as processProduct skips one: a refusal
// would have the consumer take back an id the store never withdrew.
func (scg *Sealed) matchOne(p CatalogProduct) (match mtgban.SealedMatch, ok bool) {
	defer func() {
		r := recover()
		if r != nil {
			match, ok = mtgban.SealedMatch{}, false
		}
	}()
	return scg.matchProduct(p), true
}

// Load streams the single catalog export (authenticated with the API key) and
// fills the sealed inventory and buylist in one pass.
func (scg *Sealed) Load(ctx context.Context) error {
	scg.productMap = buildProductMap(scg.backend)
	scg.priced = nil

	setIDs, err := scg.client.SetIDs(ctx, scg.gameID)
	if err != nil {
		scg.printf("could not load set ids for buylist links: %v", err)
	}
	scg.setIDs = setIDs

	count, ofGame := 0, 0
	err = scg.client.StreamCatalog(ctx, func() {
		scg.printf("Catalog stream broke after %d products, downloading it again", count)
		scg.inventory = mtgban.InventoryRecord{}
		scg.buylist = mtgban.BuylistRecord{}
		scg.dropped = nil
		scg.priced = nil
		count, ofGame = 0, 0
	}, func(p CatalogProduct) error {
		if gameFromCatalog(p.Game) == scg.gameID {
			ofGame++
		}
		scg.processProduct(p)
		count++
		if count%5000 == 0 {
			scg.printf("Processed %d products", count)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("catalog load failed: %w", err)
	}
	// Counted over every product type, since a game may have no sealed
	// listed while its singles still prove the spelling maps.
	if ofGame == 0 {
		return fmt.Errorf("none of %d catalog products maps to %s", count, scg.backend.Game)
	}
	scg.printf("Processed %d products total", count)
	// What a sealed run priced is only half of what it saw; without these
	// lines the half it turned down leaves no trace, and a game whose
	// coverage falls looks exactly like a game with nothing more to sell.
	for _, reason := range slices.Sorted(maps.Keys(scg.dropped)) {
		scg.printf("Dropped %d products: %s", scg.dropped[reason], reason)
	}
	scg.printf("Priced %d sealed products", scg.pricedProducts())
	byID, bridged, named := pricedByVia(scg.priced)
	scg.printf("Priced %d products by id, %d through a bridge, %d by name", byID, bridged, named)

	now := time.Now()
	scg.inventoryDate = now
	scg.buylistDate = now
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (scg *Sealed) Inventory() mtgban.InventoryRecord {
	return scg.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (scg *Sealed) Buylist() mtgban.BuylistRecord {
	return scg.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (scg *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCGSealed"
	info.InventoryTimestamp = &scg.inventoryDate
	info.BuylistTimestamp = &scg.buylistDate
	info.SealedMode = true
	info.Game = scg.backend.Game
	return
}
