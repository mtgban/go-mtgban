package cardmarket

import (
	"context"
	"fmt"
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

// CardMarketSealed prices sealed product from Cardmarket's marketplace,
// reading the listings themselves rather than a price guide.
type CardMarketSealed struct {
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

func (mkm *CardMarketSealed) printf(format string, a ...any) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMSealed] "+format, a...)
	}
}

// NewScraperSealed returns a sealed scraper for one game, authenticated with an
// app token and secret.
func NewScraperSealed(gameID int, appToken, appSecret string) (*CardMarketSealed, error) {
	switch gameID {
	case GameIdMagic, GameIdLorcana, GameIdRiftbound, GameIdOnePiece, GameIdYugioh, GameIdFleshAndBlood,
		GameIdPokemon:
	default:
		return nil, fmt.Errorf("unsupported game %d", gameID)
	}
	mkm := CardMarketSealed{}
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

func (mkm *CardMarketSealed) processProduct(ctx context.Context, channel chan<- responseChan, idProduct int, uuids []string) error {
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

			link := BuildURL(article.IdProduct, GameIdMagic, mkm.Affiliate, article.IsFoil)
			out := responseChan{
				cardID: uuid,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      article.Price * mkm.exchangeRate,
					Quantity:   article.Count,
					SellerName: article.Seller.Username,
					URL:        link,
					OriginalId: fmt.Sprint(article.IdProduct),
					InstanceId: fmt.Sprint(article.IdArticle),
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
func (mkm *CardMarketSealed) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	productMap := mtgmatcher.BuildSealedProductMap("mcmId")
	mkm.printf("Loaded %d sealed products", len(productMap))

	// A datastore that does not catalog cardmarket's own ids resolves over
	// the TCGplayer bridge instead, with the sealed-name resolver catching
	// what the bridge does not link yet.
	nameFallback := false
	if len(productMap) == 0 && len(mkm.TCGBridge) > 0 {
		nameFallback = true
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
	var productIds []int
	for _, product := range productList {
		if mkm.TargetProduct != "" && mkm.TargetProduct != product.Name {
			continue
		}
		if nameFallback {
			// The English-only datastores never carry the language
			// variants, whose prices must not land on the English
			// product's uuid - and the bridge links them there, since a
			// blueprint lists an English and a non-English id together.
			if mtgmatcher.SealedIsLanguageVariant(product.Name) {
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
					productMap[product.IdProduct] = []string{uuid}
				}
			}
		}
		_, found := productMap[product.IdProduct]
		if !found && nameFallback {
			uuid, err := mtgmatcher.ResolveSealed(product.Name)
			if err != nil {
				continue
			}
			productMap[product.IdProduct] = []string{uuid}
			resolved++
			found = true
		}
		if !found {
			continue
		}
		productIds = append(productIds, product.IdProduct)
	}
	if resolved > 0 {
		mkm.printf("Resolved %d more sealed products by name", resolved)
	}
	mkm.printf("Mapped %d mkm products to sealed products", len(productIds))

	mtgban.WorkerPool(ctx, mkm.MaxConcurrency, productIds,
		func(ctx context.Context, idProduct int, channel chan<- responseChan) error {
			uuids := productMap[idProduct]
			co, err := mtgmatcher.GetUUID(uuids[0])
			if err != nil {
				return nil
			}
			if mkm.TargetEdition != "" && mkm.TargetEdition != co.Edition && mkm.TargetEdition != co.SetCode {
				return nil
			}

			mkm.printf("Processing %s (%d/%d)...", co, slices.Index(productIds, idProduct)+1, len(productIds))

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
					mkm.printf("%s - %s: %s", result.entry.OriginalId, cerr.Error(), result.cardID)
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
func (mkm *CardMarketSealed) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *CardMarketSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardmarket"
	info.Shorthand = "MKMSealed"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.SealedMode = true
	switch mkm.gameID {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	case GameIdRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameIdOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameIdYugioh:
		info.Game = mtgban.GameYuGiOh
	case GameIdFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	case GameIdPokemon:
		info.Game = mtgban.GamePokemon
	}
	return
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
