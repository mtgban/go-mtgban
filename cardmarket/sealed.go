package cardmarket

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Sealed prices sealed product from Cardmarket's marketplace,
// reading the listings themselves rather than a price guide.
type Sealed struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	// Optional field to select a single edition to go through
	TargetEdition string
	// Optional field to select a single product name to go through
	TargetProduct string

	// TCGBridge maps a Cardmarket product id to the TCGplayer id of the
	// same sealed product, for datastores that do not catalog cardmarket's
	// own ids (riftbound, lorcana). bantool builds it from cardtrader's
	// blueprints, the one source linking the two marketplaces; the scraper
	// itself stays vendor-pure and receives it as plain data.
	TCGBridge map[int]int

	inventoryDate time.Time
	exchangeRate  float64

	inventory mtgban.InventoryRecord

	client *MKMClient
	gameID int
}

func (mkm *Sealed) printf(format string, a ...any) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMSealed] "+format, a...)
	}
}

// NewScraperSealed returns a sealed scraper for one game, authenticated with an
// app token and secret.
func NewScraperSealed(gameID int, appToken, appSecret string) (*Sealed, error) {
	switch gameID {
	case GameMagic, GameLorcana, GameRiftbound, GameOnePiece, GameYuGiOh, GameFleshAndBlood,
		GamePokemon:
	default:
		return nil, fmt.Errorf("unsupported game %d", gameID)
	}
	mkm := Sealed{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	mkm.gameID = gameID
	return &mkm, nil
}

// List of comments commonly used to describe something that it is not
// actually sealed (usually offered at a lower price)
var notSealedComments = []string{
	"abierto",
	"all cards sleeved",
	"cards only",
	"damaged",
	"deck only",
	"empty",
	"just",
	"missing",
	"no box",
	"no rulebook",
	"no scellé",
	"not sealed",
	"only 60 cards",
	"only box",
	"only cards",
	"only the deck",
	"open",
	"ouvert",
	"sampler",
	"sans",
	"seulement",
	"unsealed",
	"used",
	"without",
}

func (mkm *Sealed) processProduct(ctx context.Context, channel chan<- responseChan, idProduct int, uuids []string) error {
	var done bool
	var page int
	var foundNF, foundF bool

	// Query max 5 pages (500 articles) if prices aren't found
	for !done && page < 5 {
		// We process a tenth of the typical request because we only need the first few results
		// But if there are multiple ids for the same product (ie foil SLDs), then we query more
		entities := MaxEntities / 10
		if len(uuids) > 1 {
			entities = MaxEntities
		}

		articles, err := mkm.client.MKMSimpleArticles(ctx, idProduct, true, page, entities)
		if err != nil {
			return err
		}
		page++

		if len(articles) == 0 {
			break
		}

		for _, article := range articles {
			if article.Price == 0 {
				continue
			}

			uuid := uuids[0]
			if article.IsFoil && len(uuids) > 1 {
				uuid = uuids[1]
			}

			// Skip if we already found the related price
			if len(uuids) > 1 && ((foundNF && !article.IsFoil) || (foundF && article.IsFoil)) {
				continue
			}

			// Skip all the silly non-really-sealed listings
			skip := false
			for _, comment := range notSealedComments {
				if mtgmatcher.Contains(article.Comments, comment) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			link := BuildURL(article.IDProduct, mkm.gameID, mkm.Affiliate, article.IsFoil)
			out := responseChan{
				cardID: uuid,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      article.Price * mkm.exchangeRate,
					Quantity:   article.Count,
					SellerName: article.Seller.Username,
					URL:        link,
					OriginalID: fmt.Sprint(article.IDProduct),
					InstanceID: fmt.Sprint(article.IDArticle),
				},
			}
			channel <- out

			// Only keep the first price found
			// or update what we have found
			if len(uuids) == 1 || (foundNF && foundF) {
				done = true
				break
			} else if article.IsFoil {
				foundF = true
			} else if !article.IsFoil {
				foundNF = true
			}
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mkm *Sealed) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	productMap := mtgmatcher.BuildSealedProductMap("mcmId")
	mkm.printf("Loaded %d sealed products", len(productMap))

	// A datastore that does not catalog cardmarket's own ids resolves by
	// name instead. The TCGplayer bridge settles what an id can settle and
	// the resolver catches the rest, but the bridge is an improvement on the
	// name pass rather than a precondition for it: requiring one left every
	// game whose datastore carries no mcmId priced at nothing at all unless
	// CardTrader credentials happened to be configured beside them.
	// Magic is not among them whatever its map looks like: its sealed names
	// collide too readily to be trusted on their own, which is the same
	// reason the CardTrader sealed scraper stops its name pass there.
	nameFallback := len(productMap) == 0 && mkm.gameID != GameMagic
	if nameFallback && len(mkm.TCGBridge) > 0 {
		tcgMap := mtgmatcher.BuildSealedProductMap("tcgplayerProductId")
		for mkmID, tcgID := range mkm.TCGBridge {
			uuids, found := tcgMap[tcgID]
			if !found {
				continue
			}
			productMap[mkmID] = uuids
		}
		mkm.printf("Bridged %d sealed products through the TCGplayer id", len(productMap))
	}

	productList, err := GetProductListSealed(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	mkm.printf("Loaded %d mkm products", len(productList))

	// The products Cardmarket marks as an earlier print run, keyed by what
	// the name says apart from that mark, so a plain name can be recognised
	// as the run its marked sibling is not.
	earlyRunSiblings := map[string]bool{}
	var printRuns printRunIndex
	if nameFallback {
		printRuns = newPrintRunIndex()
		for _, product := range productList {
			if namesEarlyPrintRun(product.Name) {
				earlyRunSiblings[sealedBaseKey(product.Name)] = true
			}
		}
	}

	var resolved int
	var productIDs []int
	dropped := map[string]int{}
	named := map[string][]int{}
	names := map[int]string{}
	for _, product := range productList {
		if mkm.TargetProduct != "" && mkm.TargetProduct != product.Name {
			continue
		}
		if nameFallback {
			if shelf, holds := sealedShelfHoldsNoProduct(product.CategoryName); holds {
				dropped[shelf]++
				continue
			}
			// The English-only datastores never carry the printings made
			// for another market, whose prices must not land on the
			// English product's uuid - and the bridge links them there,
			// since a blueprint lists an English and a non-English id
			// together.
			if sealedIsForeignPrinting(product.Name) {
				continue
			}
			// A product Cardmarket keeps in two print runs beats the
			// bridge, which speaks through blueprints that lump them: the
			// "(Pre-Errata)" boxes and the reboxed ones share a single
			// blueprint, and a single TCGplayer id between them, so the
			// bridge can only put both on one run and leave the other
			// unpriced.
			//
			// Which run a name means is not in the name - "Romance Dawn
			// Booster Box" says nothing - but it is in the catalogue: a
			// plain name is the later boxing exactly when Cardmarket also
			// lists the same product marked as the earlier one.
			early := namesEarlyPrintRun(product.Name)
			if early || earlyRunSiblings[sealedBaseKey(product.Name)] {
				if uuid, named := printRuns.resolve(product.Name, early); named {
					productMap[product.IDProduct] = []string{uuid}
				}
			}
		}
		_, found := productMap[product.IDProduct]
		if !found && nameFallback {
			uuid, err := mkm.resolveSealedName(product.Name)
			// A name the resolver turns down is the whole reason this
			// scraper prices a fraction of the catalog, so say which
			// name and which refusal, the way the singles path does.
			if err != nil {
				mkm.printf("%q (%d): %s", product.Name, product.IDProduct, err)
				dropped[err.Error()]++
				continue
			}
			productMap[product.IDProduct] = []string{uuid}
			named[uuid] = append(named[uuid], product.IDProduct)
			resolved++
			found = true
		}
		if !found {
			mkm.printf("%q (%d): no id links it to a product", product.Name, product.IDProduct)
			dropped["unlinked id"]++
			continue
		}
		names[product.IDProduct] = product.Name
		productIDs = append(productIDs, product.IDProduct)
	}
	productIDs, subsumed := mkm.pruneSubsumed(names, productMap, named, productIDs)
	if subsumed > 0 {
		resolved -= subsumed
		dropped["names a product built on another"] += subsumed
	}
	if resolved > 0 {
		mkm.printf("Resolved %d more sealed products by name", resolved)
	}
	for _, reason := range slices.Sorted(maps.Keys(dropped)) {
		mkm.printf("Dropped %d products: %s", dropped[reason], reason)
	}
	mkm.printf("Mapped %d mkm products to sealed products", len(productIDs))

	mtgban.WorkerPool(ctx, mkm.MaxConcurrency, productIDs,
		func(ctx context.Context, idProduct int, channel chan<- responseChan) error {
			uuids := productMap[idProduct]
			co, err := mtgmatcher.GetUUID(uuids[0])
			if err != nil {
				return nil
			}
			if mkm.TargetEdition != "" && mkm.TargetEdition != co.Edition && mkm.TargetEdition != co.SetCode {
				return nil
			}

			mkm.printf("Processing %s (%d/%d)...", co, slices.Index(productIDs, idProduct)+1, len(productIDs))

			err = mkm.processProduct(ctx, channel, idProduct, uuids)
			if err != nil {
				mkm.printf("%s (%d) %s", co, idProduct, err.Error())
			}
			return nil
		},
		func(result responseChan) {
			err := mkm.inventory.AddStrict(result.cardID, &result.entry)
			if err != nil {
				_, cerr := mtgmatcher.GetUUID(result.cardID)
				if cerr != nil {
					mkm.printf("%s - %s: %s", result.entry.OriginalID, cerr.Error(), result.cardID)
					return
				}
				mkm.printf("%d - %s", result.ogID, err.Error())
			}
		},
		mkm.printf,
	)

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.printf("Total number of products found: %d", len(mkm.inventory))
	mkm.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mkm *Sealed) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *Sealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardmarket"
	info.Shorthand = "MKMSealed"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.SealedMode = true
	switch mkm.gameID {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameYuGiOh:
		info.Game = mtgban.GameYuGiOh
	case GameFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	}
	return
}

// resolveSealedName names the sealed product a Cardmarket name describes. The
// name as written is asked for first and answers for most of the catalog; what
// is left over is not a different product but the same one spelled the way the
// marketplace spells it, so the two spellings it differs by are tried in turn.
func (mkm *Sealed) resolveSealedName(name string) (string, error) {
	uuid, err := mtgmatcher.ResolveSealed(name)
	if err == nil {
		return uuid, nil
	}
	if uuid, found := resolveSealedRun(name); found {
		return uuid, nil
	}
	if renamed, found := sealedRenamed(mkm.gameID, name); found {
		uuid, rerr := mtgmatcher.ResolveSealed(renamed)
		if rerr == nil {
			return uuid, nil
		}
	}
	// The refusal reported is the one the name as written earned: that is
	// the name the catalog holds and the one a reader has to go looking for.
	return "", err
}

// pruneSubsumed drops the products that reached a product another entry of the
// catalog names in fewer words, and returns how many it dropped. Which names
// those are is mtgmatcher.SealedNameSubsumed's question; this walks the
// products that reached each one and asks it.
//
// A product the bridge or the print-run index answered for is asked about but
// never dropped: an id is what the two catalogs agree on, and a name cannot
// overrule it.
func (mkm *Sealed) pruneSubsumed(names map[int]string, productMap map[int][]string, named map[string][]int, productIDs []int) ([]int, int) {
	var pruned int
	for uuid, ids := range named {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		var beside []string
		for id, uuids := range productMap {
			// A uuid the bridge answered for can be held by a product
			// this pass never saw, and an unknown name says nothing
			// about the ones it stands beside.
			name, known := names[id]
			if known && len(uuids) > 0 && uuids[0] == uuid {
				beside = append(beside, name)
			}
		}
		for _, id := range ids {
			if !mtgmatcher.SealedNameSubsumed(names[id], beside, co.Edition) {
				continue
			}
			mkm.printf("%q (%d): names a product built on %s", names[id], id, uuid)
			delete(productMap, id)
			pruned++
		}
	}
	if pruned == 0 {
		return productIDs, 0
	}
	// The worker pool walks the ids rather than the map, so a product
	// dropped from one and left in the other would still be priced.
	return slices.DeleteFunc(productIDs, func(id int) bool {
		_, kept := productMap[id]
		return !kept
	}), pruned
}

// asiaRegionMark is how Cardmarket says a product is the Asian market's
// printing rather than the English one. The wording is matched short of the
// word it usually ends on, which the catalog has misspelt ("Asia Region
// Lega") often enough to matter.
const asiaRegionMark = "asia region"

// sealedShelves are the catalog's own headings for the things it sells that no
// datastore holds a product for: the collectible coins packed inside a Pokemon
// box rather than sold as one, and the event tickets. Each game prefixes its
// own name onto the heading, so the tail is what names the shelf.
//
// A heading earns its place here only by resolving nothing at all - 689 rows
// across the games, every one of them otherwise reported as a name nobody
// could place. The lots very nearly joined them, on 2 of some 200 resolving,
// until those two turned out to be real: Cardmarket files the Basic Energy Box
// and the Charizard Ultra-Premium Collection under Lot, and shelving the
// heading would have stopped pricing two products to quiet a log. What the
// datastore is merely missing stays off this list and keeps saying so.
var sealedShelves = map[string]string{
	"Coins":         "coins, which are not a sealed product",
	"Event Tickets": "an event ticket rather than a product",
}

// sealedShelfHoldsNoProduct reports whether a catalog heading is one of them,
// and what to say about it.
func sealedShelfHoldsNoProduct(category string) (string, bool) {
	for shelf, reason := range sealedShelves {
		if strings.HasSuffix(category, " "+shelf) {
			return reason, true
		}
	}
	return "", false
}

// sealedIsForeignPrinting reports whether a storefront's product name marks a
// printing an English-only datastore does not carry: one in another language,
// or one made for another market.
//
// The market is worth saying separately from the language because Cardmarket
// does: it files the Asian printings of One Piece under their English names
// with a parenthetical, and a parenthetical is the kind of thing a resolver
// is free to forgive. Naming them here is what keeps them off the English
// row however much the spellings below loosen.
func sealedIsForeignPrinting(name string) bool {
	return mtgmatcher.SealedIsLanguageVariant(name) ||
		strings.Contains(strings.ToLower(name), asiaRegionMark)
}

// sealedRename rewrites the name a marketplace sells a product under into the
// one the datastore files it as.
type sealedRename struct {
	vendor  *regexp.Regexp
	product string
}

// sealedRenames are the products a marketplace sells under a name of its own.
// Cardmarket lists One Piece's Premium Booster sets as "The Best", the name
// Bandai gives them in Japan, and follows it with the rest of the product's
// wording unchanged - so the vendor's name for the set is all that has to
// come off for the rest to be read as usual.
//
// A rename is keyed by game because a marketplace's word for one game's
// product says nothing about another's.
var sealedRenames = map[int][]sealedRename{
	GameOnePiece: {
		{regexp.MustCompile(`(?i)^the best\b`), "Premium Booster"},
	},
}

// sealedRenamed returns the name with the marketplace's own word for the
// product replaced by the datastore's, and whether any applied.
func sealedRenamed(gameID int, name string) (string, bool) {
	for _, rename := range sealedRenames[gameID] {
		if rename.vendor.MatchString(name) {
			return rename.vendor.ReplaceAllString(name, rename.product), true
		}
	}
	return "", false
}

// editionRuns are the print runs a datastore splits a reprinted product into,
// spelled the way it spells them inside the product's own name. Both rows
// carry the bracket, so a marketplace name that says nothing about the run
// reaches neither of them - it is missing a word both candidates have.
var editionRuns = []string{"1st Edition", "Unlimited Edition"}

// namedRunRe matches the run a Cardmarket name spells out for itself. Flesh
// and Blood is the catalog that does: it files each run as its own expansion
// ("Tales of Aria - First") and writes the expansion's tail into the name of
// every product under it. Alpha is Welcome to Rathe's word for the run every
// other set calls First, exactly as fabPrintRun reads it for the singles.
var namedRunRe = regexp.MustCompile(`(?i)(?: - (First|Unlimited|Alpha)\b:?| (First|Unlimited|Alpha) Edition\b)`)

// namedRunBrackets translate that word into the datastore's bracket.
var namedRunBrackets = map[string]string{
	"first":     "1st Edition",
	"alpha":     "1st Edition",
	"unlimited": "Unlimited Edition",
}

// editionRunSpellings returns the datastore spellings of a name the resolver
// turned down. A name that says which run it is has exactly one: its own
// wording with the marketplace's run word traded for the datastore's bracket.
// A name that says nothing has one per run, and which of them is meant is
// decided by whether they name the same product, below.
func editionRunSpellings(name string) []string {
	if match := namedRunRe.FindStringSubmatch(name); match != nil {
		word := match[1]
		if word == "" {
			word = match[2]
		}
		plain := strings.Join(strings.Fields(namedRunRe.ReplaceAllString(name, " ")), " ")
		return []string{plain + " [" + namedRunBrackets[strings.ToLower(word)] + "]"}
	}
	spellings := make([]string, 0, len(editionRuns))
	for _, run := range editionRuns {
		spellings = append(spellings, name+" ["+run+"]")
	}
	return spellings
}

// resolveSealedRun names the run a Cardmarket product is, for a name the
// resolver reached nothing with. A spelling counts only when it lands on a
// product saying the very words it does: the bracket is a word the vendor
// never wrote, and a candidate reached by handing it one has to answer for
// everything else the vendor did write. Two runs answering equally well is a
// name that does not say which it is, and stays unresolved.
func resolveSealedRun(name string) (string, bool) {
	var found string
	seen := map[string]bool{}
	for _, spelling := range editionRunSpellings(name) {
		uuid, err := mtgmatcher.ResolveSealed(spelling)
		if err != nil {
			continue
		}
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		if !sealedSaysSameWords(spelling, co.Name) {
			continue
		}
		if !seen[uuid] {
			seen[uuid] = true
			found = uuid
		}
	}
	if len(seen) != 1 {
		return "", false
	}
	return found, true
}

// sealedWordRe splits a name the way the sealed resolver does, and
// sealedFillerWords drop the same words it drops.
var sealedWordRe = regexp.MustCompile(`[a-z0-9]+`)

var sealedFillerWords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "and": true,
	"disney": true, "lorcana": true, "riftbound": true, "league": true,
	"legends": true, "tcg": true, "trading": true, "game": true,
	"card": true, "cards": true, "one": true, "piece": true,
	"bandai": true, "pokemon": true,
}

// sealedWords reduces a name to the words that carry it, folding the
// spellings a marketplace and a datastore differ on without meaning anything
// by it - a box is a display, a booster is a pack.
//
// This is a copy of what the resolver does, on purpose and deliberately no
// looser than it: every fold made here the resolver makes too, so a word this
// one fails to fold costs a match rather than buying a wrong one. It is not
// the resolver's job either way - the resolver is asked which product a name
// describes, and this is asked whether two names describe it in the same
// words, which is a question only a caller adding words of its own has.
func sealedWords(name string) []string {
	set := map[string]bool{}
	for _, word := range sealedWordRe.FindAllString(strings.ToLower(name), -1) {
		if sealedFillerWords[word] {
			continue
		}
		switch word {
		case "box", "boxes":
			word = "display"
		case "booster", "boosters", "packs":
			word = "pack"
		case "decks":
			word = "deck"
		case "blisters":
			word = "blister"
		case "versus":
			word = "vs"
		case "volume":
			word = "vol"
		}
		set[word] = true
	}
	return slices.Sorted(maps.Keys(set))
}

// sealedSaysSameWords reports whether two names say the same words.
func sealedSaysSameWords(a, b string) bool {
	return slices.Equal(sealedWords(a), sealedWords(b))
}

// printRunRe matches the qualifier a datastore adds to a product it holds in
// more than one print run: "Romance Dawn - Booster Box (Wave 1 - Blue)".
//
// The wording is the datastore's, and only One Piece writes it so far, which
// is why this lives beside the scraper that has to read it rather than in the
// matcher: a run is not a thing the matcher knows about, and a marketplace
// calling one "(Pre-Errata)" is not a thing any datastore knows about.
var printRunRe = regexp.MustCompile(`\(\s*wave\s+([0-9]+)[^)]*\)`)

// earlyPrintRunWords are how a storefront says it means the first boxing of a
// product that was later reboxed. Cardmarket files the original Romance Dawn
// boxes as "(Pre-Errata)" and leaves the reboxed ones plain.
//
// An ordinal is deliberately not one of these. A storefront writes plenty of
// them for other reasons - "1st Anniversary Tournament Pack" - and a run is
// named for the errata that caused the reboxing wherever one is named at all.
var earlyPrintRunWords = map[string]bool{
	"errata": true,
}

// namesEarlyPrintRun reads the words as written, not as sealedNameTokens
// leaves them: that one drops the run mark, which is the very word being
// looked for here.
func namesEarlyPrintRun(name string) bool {
	for _, tok := range splitWords(name) {
		if earlyPrintRunWords[tok] {
			return true
		}
	}
	return false
}

func splitWords(name string) []string {
	return strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// sealedNameTokens reduces a name to the words that carry it, dropping the
// print-run marks, the counts, and any word said twice: "Booster Box Case
// (12x Booster Box) (Pre-Errata)" and "Booster Box Case (Wave 1 - Blue)" both
// come down to the same words, which is what lets one be recognised as the
// other - and the first says "Booster Box" twice.
func sealedNameTokens(name string) []string {
	seen := map[string]bool{}
	var out []string
	for _, tok := range splitWords(name) {
		switch tok {
		case "pre", "errata", "wave":
			continue
		}
		if sealedCountRe.MatchString(tok) || seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

// printRunIndex holds the sealed products a datastore keeps in more than one
// print run, keyed by what their names say apart from the run.
type printRunIndex map[string][]printRunEntry

type printRunEntry struct {
	uuid string
	wave int
}

func newPrintRunIndex() printRunIndex {
	index := printRunIndex{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		match := printRunRe.FindStringSubmatch(strings.ToLower(co.Name))
		if match == nil {
			continue
		}
		wave, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		key := strings.Join(sealedNameTokens(printRunRe.ReplaceAllString(strings.ToLower(co.Name), "")), " ")
		index[key] = append(index[key], printRunEntry{uuid: uuid, wave: wave})
	}
	return index
}

// resolve names the run a vendor's product is, with the caller saying whether
// it holds the earliest: a name never says which run it is, and only a reader
// of a whole catalogue can tell, having seen the sibling that names the run
// this one is not.
func (index printRunIndex) resolve(name string, early bool) (string, bool) {
	entries := index[strings.Join(sealedNameTokens(name), " ")]
	if len(entries) < 2 {
		return "", false
	}
	best := entries[0]
	for _, entry := range entries[1:] {
		if (early && entry.wave < best.wave) || (!early && entry.wave > best.wave) {
			best = entry
		}
	}
	return best.uuid, true
}

// sealedBaseKey reduces a product name to what it says apart from the print
// run and how many the box holds, so the two runs of one product answer to
// one key.
func sealedBaseKey(name string) string {
	return strings.Join(sealedNameTokens(name), " ")
}

var sealedCountRe = regexp.MustCompile(`^\d+x?$`)
