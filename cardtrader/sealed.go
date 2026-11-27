package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// Sealed prices sealed product from Card Trader, under the same
// storefronts as the singles.
type Sealed struct {
	logCallback    mtgban.LogCallbackFunc
	maxConcurrency int
	shareCode      string

	// Only retrieve data from a single edition
	targetEdition string

	exchangeRates map[string]float64
	client        *CTAuthClient
	backend       *matcher.Backend

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord

	gameID int
}

// NewScraperSealed returns a sealed scraper for the datastore's game,
// authenticated with a full API token.
func NewScraperSealed(b *matcher.Backend, token string) (*Sealed, error) {
	id, err := sealedGameID(b.Game)
	if err != nil {
		return nil, err
	}
	ct := Sealed{}
	ct.inventory = mtgban.InventoryRecord{}
	// API is strongly rated limited, hardcode a lower amount
	ct.maxConcurrency = 2
	ct.client = NewCTAuthClient(token)
	ct.backend = b
	ct.gameID = id
	return &ct, nil
}

func (ct *Sealed) printf(format string, a ...any) {
	if ct.logCallback != nil {
		ct.logCallback("[CTSealed] "+format, a...)
	}
}

func (ct *Sealed) processEntry(ctx context.Context, channel chan<- resultChan, expansionID int, expansionName string, productMap map[int][]string) error {
	allProducts, err := ct.client.ProductsForExpansion(ctx, expansionID)
	if err != nil {
		return err
	}

	for _, products := range allProducts {
		for _, product := range products {
			uuids, found := productMap[product.BlueprintID]
			if !found {
				continue
			}

			if gameLanguage(ct.gameID, product) != "en" {
				continue
			}

			uuid := uuids[0]
			if gameFoil(ct.gameID, product) && len(uuids) > 1 {
				uuid = uuids[1]
			}

			switch {
			case product.Quantity < 1,
				product.OnVacation,
				product.Bundle,
				product.Properties.Altered,
				product.Graded:
				continue
			case notTheListedCopy(product.Description),
				matcher.Contains(product.Description, "empty booster"),
				matcher.Contains(product.Description, "empty box"),
				matcher.Contains(product.Description, "opened"),
				matcher.Contains(product.Description, "only cards"),
				matcher.Contains(product.Description, "missing"),
				matcher.Contains(product.Description, "deck box only"):
				continue
			}

			link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintID)
			if ct.shareCode != "" {
				link += "?share_code=" + ct.shareCode
			}

			price, err := priceToUSD(product.Price.Cents, product.Price.Currency, ct.exchangeRates)
			if err != nil {
				ct.printf("%v for blueprint %d", err, product.BlueprintID)
				continue
			}

			// Assign a seller name as required by Market
			sellerName := availableMarketNames[0]
			if product.User.SealedZero {
				sellerName = availableMarketNames[1]
				if strings.Contains(strings.ToLower(product.User.Name), "day ready") {
					sellerName = availableMarketNames[2]
				}
			}

			channel <- resultChan{
				cardID: uuid,
				invEntry: &mtgban.InventoryEntry{
					Conditions: mtgban.NM,
					Price:      price,
					Quantity:   product.Quantity,
					URL:        link,
					SellerName: sellerName,
					Bundle:     product.User.SealedZero,
					OriginalID: fmt.Sprint(product.BlueprintID),
					InstanceID: fmt.Sprint(product.ID),
					CustomFields: map[string]string{
						"SubSellerName": product.User.Name,
						"SubSellerGeo":  product.User.CountryCode,
					},
				},
			}
		}
	}

	return nil
}

// sealedIndex is what the datastore's sealed identifiers say about a
// blueprint: its own cardtraderId, and the TCGplayer id both sides carry.
type sealedIndex struct {
	byID  map[int][]string
	byTCG map[int][]string
	// holders is every product a cardtraderId reaches, outside Magic: no other
	// blueprint is placed on one.
	holders map[string]bool
}

// blueprintMatch is matchBlueprint's answer for one blueprint.
type blueprintMatch struct {
	// uuids is every product the blueprint prices: a foil listing goes on
	// the second where there are two.
	uuids []string
	how   mtgban.SealedVia
	// reason is the drop reason of a skipped blueprint, or the resolver's
	// refusal.
	reason string
}

// sealedUnlinked is the reason for a Magic blueprint no id links. Load never
// counts it, and MatchSealedCatalog does not match Magic.
const sealedUnlinked = "unlinked id"

// subsumedReason is the refusal reported for a blueprint the pruning dropped.
const subsumedReason = "subsumed by another listing"

// matchBlueprint settles one sealed blueprint. The datastore's own
// cardtraderId index settles a blueprint for good; one it does not answer is
// routed through the TCGplayer id both sides carry. The leftovers resolve by
// name, the way the cardmarket sealed scraper resolves the same leftovers -
// except for Magic, whose sealed namespace is too collision-prone to trust a
// name alone. Outside Magic, neither fallback lands on a product that holds
// a cardtraderId of its own. An answer reads its own blueprint alone, so the
// order the blueprints are asked in changes nothing.
func (ct *Sealed) matchBlueprint(id int, bp *Blueprint, index sealedIndex) blueprintMatch {
	if uuids, found := index.byID[id]; found {
		return blueprintMatch{uuids: uuids, how: mtgban.SealedByID}
	}
	match := ct.matchUnlinked(bp, index)
	if slices.ContainsFunc(match.uuids, func(uuid string) bool { return index.holders[uuid] }) {
		return blueprintMatch{how: mtgban.SealedRefused, reason: sealedHoldsOwnID}
	}
	return match
}

// sealedHoldsOwnID is the refusal of a blueprint that its TCGplayer id or its
// name puts on a product whose cardtraderId the datastore already lists.
const sealedHoldsOwnID = "product holds its own CardTrader id"

// matchUnlinked settles a blueprint whose id the datastore lacks.
func (ct *Sealed) matchUnlinked(bp *Blueprint, index sealedIndex) blueprintMatch {
	// An unlinked blueprint carries a zero id; the singles path skips
	// those, and looking one up would funnel every such listing onto
	// whatever product shares the missing link.
	if bp.TCGplayerID != 0 {
		if uuids, found := index.byTCG[bp.TCGplayerID]; found {
			return blueprintMatch{uuids: uuids, how: mtgban.SealedByBridge}
		}
	}
	if ct.gameID == GameMagic {
		return blueprintMatch{how: mtgban.SealedSkipped, reason: sealedUnlinked}
	}

	// Cardtrader links only some of its sealed blueprints to a
	// TCGplayer product, and the rest carry no id to route through -
	// most of a set's boxes and kits among them. A sealed catalog is
	// named things. The accessories sharing the sealed side - binders,
	// dice, sleeves, empty boxes - name no product of ours and drop
	// out here.
	skipped, why := sealedNamePassSkips(bp)
	if skipped {
		return blueprintMatch{how: mtgban.SealedSkipped, reason: why}
	}
	// The expansion is the other half of what CardTrader knows about
	// a product: it shelves the two print runs of a Flesh and Blood
	// set apart ("Crucible of War - Unlimited") while naming both
	// blueprints the same, so a name that reaches both runs and
	// refuses can still be settled by the shelf it sits on.
	uuid, err := ct.backend.ResolveSealedWithHint(bp.Name, bp.Expansion.Name)
	// The set code some storefronts open a sealed name with is the
	// vendor's filing, not a word of the product: the resolver's
	// vocabulary is built from set names and never sees a code, so
	// the letters read as the vendor naming something else. Retried
	// and never substituted, so the trim can turn a refusal into an
	// answer and can never turn one answer into another.
	if err != nil {
		trimmed := sealedNameWithoutShelfCode(bp)
		if trimmed != bp.Name {
			uuid, err = ct.backend.ResolveSealedWithHint(trimmed, bp.Expansion.Name)
		}
	}
	if err != nil {
		return blueprintMatch{how: mtgban.SealedRefused, reason: err.Error()}
	}
	return blueprintMatch{uuids: []string{uuid}, how: mtgban.SealedByName}
}

// shared is the answer as every sealed scraper reports it. Of several uuids,
// it names the first, the one processEntry prices a nonfoil listing onto.
func (match blueprintMatch) shared(id int, bp *Blueprint) mtgban.SealedMatch {
	out := mtgban.SealedMatch{
		StoreID: strconv.Itoa(id),
		Name:    bp.Name,
		Via:     match.how,
		Reason:  match.reason,
	}
	if len(match.uuids) > 0 {
		out.UUID = match.uuids[0]
	}
	return out
}

// sealedPlacement is what placeBlueprints builds from the blueprints.
type sealedPlacement struct {
	// productMap is the products to price by blueprint id, every product
	// the datastore's cardtraderIds name included.
	productMap map[int][]string
	// matches holds what each blueprint settled as, with the pruned ones
	// refused: what MatchSealedCatalog reports.
	matches map[int]mtgban.SealedMatch
}

// buildProductMap links each sealed blueprint to the datastore product it
// prices, as placeBlueprints settles it.
func (ct *Sealed) buildProductMap(blueprints map[int]*Blueprint) map[int][]string {
	return ct.placeBlueprints(blueprints).productMap
}

// placeBlueprints settles every blueprint through matchBlueprint, logging
// each one the name pass drops, and prunes the names that reached a product
// another blueprint names in fewer words. Every blueprint is settled before
// the pruning looks at any of them, as it was when the id, bridge and name
// passes ran one after the other.
func (ct *Sealed) placeBlueprints(blueprints map[int]*Blueprint) sealedPlacement {
	productMap := ct.backend.BuildSealedProductMap("cardtraderId")
	ct.printf("Loaded %d sealed products", len(productMap))
	index := sealedIndex{
		byID:  maps.Clone(productMap),
		byTCG: ct.backend.BuildSealedProductMap("tcgplayerProductId"),
	}
	if ct.gameID != GameMagic {
		index.holders = map[string]bool{}
		for _, uuids := range index.byID {
			for _, uuid := range uuids {
				index.holders[uuid] = true
			}
		}
	}

	ids := slices.Sorted(maps.Keys(blueprints))
	settled := make(map[int]blueprintMatch, len(ids))
	matches := make(map[int]mtgban.SealedMatch, len(ids))
	var bridged int
	for _, id := range ids {
		match := ct.matchBlueprint(id, blueprints[id], index)
		settled[id] = match
		matches[id] = match.shared(id, blueprints[id])
		if match.how == mtgban.SealedByBridge {
			productMap[id] = match.uuids
			bridged++
		}
	}
	ct.printf("Mapped %d sealed products through the TCGplayer id", bridged)

	if ct.gameID == GameMagic {
		return sealedPlacement{productMap: productMap, matches: matches}
	}

	var resolved int
	dropped := map[string]int{}
	named := map[string][]int{}
	for _, id := range ids {
		match := settled[id]
		switch match.how {
		case mtgban.SealedSkipped, mtgban.SealedRefused:
			// A name the resolver turns down is the whole reason this
			// scraper prices a fraction of the catalog, so say which
			// name and which refusal, the way the singles path does.
			ct.printf("%q (%d): %s", blueprints[id].Name, id, match.reason)
			dropped[match.reason]++
		case mtgban.SealedByName:
			productMap[id] = match.uuids
			named[match.uuids[0]] = append(named[match.uuids[0]], id)
			resolved++
		}
	}
	pruned := ct.pruneSubsumed(blueprints, productMap, named)
	resolved -= pruned
	if pruned > 0 {
		dropped["names a product built on another"] += pruned
	}
	if resolved > 0 {
		ct.printf("Resolved %d more sealed products by name", resolved)
	}
	for _, reason := range slices.Sorted(maps.Keys(dropped)) {
		ct.printf("Dropped %d blueprints: %s", dropped[reason], reason)
	}

	for id, match := range matches {
		_, kept := productMap[id]
		if match.UUID != "" && !kept {
			match.UUID, match.Via, match.Reason = "", mtgban.SealedRefused, subsumedReason
			matches[id] = match
		}
	}
	return sealedPlacement{productMap: productMap, matches: matches}
}

// matchCatalog is MatchSealedCatalog past the fetch: one match per blueprint,
// sorted by blueprint id.
func (ct *Sealed) matchCatalog(blueprints map[int]*Blueprint) []mtgban.SealedMatch {
	placed := ct.placeBlueprints(blueprints)
	out := make([]mtgban.SealedMatch, 0, len(placed.matches))
	for _, id := range slices.Sorted(maps.Keys(placed.matches)) {
		out = append(out, placed.matches[id])
	}
	return out
}

// sealedGameID is the Card Trader catalog number of a game the sealed
// scraper is built for.
func sealedGameID(game matcher.Game) (int, error) {
	id, found := ctGames[game]
	// An unknown game would not error anywhere later: its listings would
	// simply all fail the language read and the scraper would run empty.
	// Palworld is the one game the shared table carries that this side has
	// never been built for, so it is refused here rather than left out of
	// the table the market scraper reads too.
	if !found || game == matcher.GamePalworld {
		return 0, fmt.Errorf("unsupported game %q", game)
	}
	return id, nil
}

// newSealedMatcher is the part of a sealed scraper the matching needs: the
// backend and its game, which must not be Magic, whose sealed map is ids
// alone. It holds no client and no credential.
func newSealedMatcher(b *matcher.Backend) (*Sealed, error) {
	id, err := sealedGameID(b.Game)
	if err != nil {
		return nil, err
	}
	if id == GameMagic {
		return nil, fmt.Errorf("unsupported game %q", b.Game)
	}
	// API is strongly rated limited, as NewScraperSealed has it
	return &Sealed{backend: b, gameID: id, maxConcurrency: 2}, nil
}

// MatchSealedCatalog fetches the sealed blueprints of the backend's game and
// reports how each one is matched to the datastore, the way Load matches it,
// pricing nothing. It refuses Magic and the games the sealed scraper is not
// built for. It is strict where Load is tolerant: an expansion that fails to
// fetch fails the call, naming how many were skipped and the first, rather
// than leaving its blueprints out of the answer.
func MatchSealedCatalog(ctx context.Context, b *matcher.Backend, token string) ([]mtgban.SealedMatch, error) {
	ct, err := newSealedMatcher(b)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, errors.New("missing Card Trader token")
	}
	return ct.matchSealedCatalog(ctx, NewCTAuthClient(token))
}

// matchSealedCatalog is the export's fetch and match, over client.
func (ct *Sealed) matchSealedCatalog(ctx context.Context, client *CTAuthClient) ([]mtgban.SealedMatch, error) {
	var skips expansionSkips
	blueprintsRaw, expansionsRaw, err := BlueprintsForGameID(ctx, client, ct.gameID, "", ct.maxConcurrency, skips.logf)
	if err != nil {
		return nil, err
	}
	err = skips.err()
	if err != nil {
		return nil, err
	}
	blueprints, _ := FormatBlueprints(blueprintsRaw, expansionsRaw, true)
	// An empty answer would read as every id of the game withdrawn.
	if len(blueprints) == 0 {
		return nil, fmt.Errorf("no sealed blueprints for %q", ct.backend.Game)
	}
	return ct.matchCatalog(blueprints), nil
}

// sealedNameWithoutShelfCode takes the letters of a set code off the head of a
// product name, when they are the letters of the code its own shelf opens
// with. The number stays: the catalog spells it into the product's own name
// ("Starter Deck 10: Generation Pulse"), and dropping it loses the one word
// that tells the starter decks apart.
func sealedNameWithoutShelfCode(bp *Blueprint) string {
	name := codedShelf.FindStringSubmatch(bp.Name)
	if name == nil {
		return bp.Name
	}
	shelf := codedShelf.FindStringSubmatch(bp.Expansion.Name)
	if shelf == nil || !strings.EqualFold(name[1], shelf[1]) || name[2] != shelf[2] {
		return bp.Name
	}
	return name[2] + " " + bp.Name[len(name[0]):]
}

// accessoryWords name what CardTrader files in each of its accessory
// categories. A category alone is not enough to turn a blueprint down:
// CardTrader files real sealed product in them by mistake often enough to
// matter - the Snorlax and Morpeko Pin Collections sit under Memorabilia, and
// Legendary Collection's Gameboard Editions beside them - and a blueprint
// resolving to the product it actually is should keep doing so.
//
// A blueprint that is filed as an accessory and named as one is the case
// there is no doubt about. Every sealed product a set holds shares its
// wording, so a deck box named for the collection it was packed in resolves
// to that collection and prices a ten-euro box at the collection's fifty:
// "Marnie Premium Tournament Collection Deck Box" is not the Marnie Premium
// Tournament Collection Box, whatever the resolver forgives.
var accessoryWords = map[int]string{
	CategoryMagicDeckBoxes: "deck box", CategoryYuGiOhDeckBoxes: "deck box",
	CategoryPokemonDeckBoxes: "deck box", CategoryLorcanaDeckBoxes: "deck box",
	CategoryOnePieceDeckBoxes: "deck box", CategoryRiftboundDeckBoxes: "deck box",
	CategoryGundamDeckBoxes: "deck box", CategoryPalworldDeckBoxes: "deck box",

	CategoryMagicSleeves: "sleeve", CategoryYuGiOhSleeves: "sleeve",
	CategoryPokemonSleeves: "sleeve", CategoryLorcanaSleeves: "sleeve",
	CategoryOnePieceSleeves: "sleeve", CategoryRiftboundSleeves: "sleeve",
	CategoryFleshAndBloodSleeves: "sleeve",
	CategoryGundamSleeves:        "sleeve", CategoryPalworldSleeves: "sleeve",

	CategoryMagicPlaymats: "playmat", CategoryYuGiOhPlaymats: "playmat",
	CategoryPokemonPlaymats: "playmat", CategoryLorcanaPlaymats: "playmat",
	CategoryOnePiecePlaymats: "playmat", CategoryRiftboundPlaymats: "playmat",
	CategoryFleshAndBloodPlaymats: "playmat",
	CategoryGundamPlaymats:        "playmat", CategoryPalworldPlaymats: "playmat",

	CategoryMagicAlbums: "album", CategoryYuGiOhAlbums: "album",
	CategoryPokemonAlbums: "album", CategoryLorcanaAlbums: "album",
	CategoryOnePieceAlbums: "album", CategoryRiftboundAlbums: "album",

	CategoryMagicDice: "dice", CategoryYuGiOhDice: "dice",
	CategoryPokemonDice: "dice", CategoryFleshAndBloodDice: "dice",
	CategoryGundamDice: "dice",

	CategoryMagicDividers: "divider", CategoryYuGiOhDividers: "divider",
	CategoryPokemonDividers: "divider",
}

// sealedNamePassSkips reports whether a blueprint must not be asked of the
// sealed-name resolver at all, and what to call the refusal in the tally.
func sealedNamePassSkips(bp *Blueprint) (bool, string) {
	if matcher.SealedIsLanguageVariant(bp.Name) {
		return true, "language variant"
	}
	word, filed := accessoryWords[bp.CategoryID]
	if filed && strings.Contains(strings.ToLower(bp.Name), word) {
		return true, "accessory"
	}
	return false, ""
}

// pruneSubsumed drops the blueprints that reached a product another blueprint
// names in fewer words, and returns how many it dropped. Which names those are
// is matcher.SealedNameSubsumed's question; this walks the blueprints that
// reached each product and asks it.
//
// A blueprint the datastore's own ids answered for is asked about but never
// dropped: an id is what the catalogs agree on, and a name cannot overrule it.
func (ct *Sealed) pruneSubsumed(blueprints map[int]*Blueprint, productMap map[int][]string, named map[string][]int) int {
	var pruned int
	for uuid, ids := range named {
		co, err := ct.backend.GetUUID(uuid)
		if err != nil {
			continue
		}
		var beside []string
		for id, uuids := range productMap {
			// A uuid the datastore's ids answered for can be held by a
			// blueprint this pass never saw, and an unknown name says
			// nothing about the ones it stands beside.
			bp, known := blueprints[id]
			if known && len(uuids) > 0 && uuids[0] == uuid {
				beside = append(beside, bp.Name)
			}
		}
		for _, id := range ids {
			if !matcher.SealedNameSubsumed(blueprints[id].Name, beside, co.Edition) {
				continue
			}
			ct.printf("%q (%d): names a product built on %s", blueprints[id].Name, id, uuid)
			delete(productMap, id)
			pruned++
		}
	}
	return pruned
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (ct *Sealed) Load(ctx context.Context) error {
	rates, err := mtgban.GetExchangeRates(ctx)
	if err != nil {
		return err
	}
	ct.exchangeRates = rates

	if ct.targetEdition != "" {
		ct.printf("-> only targeting edition %s", ct.targetEdition)
	}
	blueprintsRaw, expansionsRaw, err := BlueprintsForGameID(ctx, ct.client, ct.gameID, ct.targetEdition, ct.maxConcurrency, ct.printf)
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d global sets", len(expansionsRaw))
	ct.printf("Found %d blueprints", len(blueprintsRaw))

	blueprints, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw, true)
	ct.printf("Parsing %d expansions", len(expansions))

	placed := ct.placeBlueprints(blueprints)
	ct.priceSealed(ctx, placed, expansions)

	ct.inventoryDate = time.Now()

	return nil
}

// priceSealed reads every expansion's listings onto the placed blueprints,
// then says how many blueprints it priced, by how each was placed.
func (ct *Sealed) priceSealed(ctx context.Context, placed sealedPlacement, expansions map[int]string) {
	productMap := placed.productMap

	type expItem struct {
		id   int
		name string
	}
	expItems := make([]expItem, 0, len(expansions))
	for id, name := range expansions {
		expItems = append(expItems, expItem{id, name})
	}

	mtgban.WorkerPool(ctx, ct.maxConcurrency, expItems,
		func(ctx context.Context, item expItem, results chan<- resultChan) error {
			ct.printf("Processing %s [%d]", item.name, item.id)
			return ct.processEntry(ctx, results, item.id, item.name, productMap)
		},
		func(result resultChan) {
			addCheapestOffer(ct.inventory, result, ct.printf)
		},
		ct.printf,
	)

	byID, bridged, named := pricedBlueprints(ct.inventory, placed)
	ct.printf("Priced %d products by id, %d through a bridge, %d by name", byID, bridged, named)
}

// pricedBlueprints counts the placed blueprints the inventory holds an entry
// for, each once however many entries it holds, by how it was placed. A
// blueprint the catalog did not list is placed by the datastore's id. It
// undercounts by design: addCheapestOffer keeps one entry per uuid,
// condition and storefront, so two blueprints priced onto one uuid can
// leave one entry, and the blueprint it dropped is not counted.
func pricedBlueprints(inventory mtgban.InventoryRecord, placed sealedPlacement) (byID, bridged, named int) {
	priced := map[string]bool{}
	for _, entries := range inventory {
		for _, entry := range entries {
			priced[entry.OriginalID] = true
		}
	}
	for id := range placed.productMap {
		if !priced[strconv.Itoa(id)] {
			continue
		}
		via := mtgban.SealedByID
		if match, found := placed.matches[id]; found {
			via = match.Via
		}
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

// Inventory returns what Load collected. See mtgban.Seller.
func (ct *Sealed) Inventory() mtgban.InventoryRecord {
	return ct.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (ct *Sealed) MarketNames() []string {
	return availableMarketNames
}

var name2shorthandSealed = map[string]string{
	"Card Trader":      "CTSealed",
	"Card Trader Zero": "CT0Sealed",
	"Card Trader 1DR":  "CT1DRSealed",
}

// InfoForScraper describes one of the sub-scrapers named above.
func (ct *Sealed) InfoForScraper(name string) mtgban.ScraperInfo {
	info := ct.Info()
	info.Name = name
	info.Shorthand = name2shorthandSealed[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (ct *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader Sealed"
	info.Shorthand = "CTSealedWrapper"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.SealedMode = true
	info.Game = ct.backend.Game
	return
}
