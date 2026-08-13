package cardmarket

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

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

func (mkm *CardMarketSealed) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMSealed] "+format, a...)
	}
}

func NewScraperSealed(gameID int, appToken, appSecret string) (*CardMarketSealed, error) {
	switch gameID {
	case GameIdMagic, GameIdLorcana, GameIdRiftbound, GameIdOnePiece:
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

	var resolved int
	var productIds []int
	for _, product := range productList {
		if mkm.TargetProduct != "" && mkm.TargetProduct != product.Name {
			continue
		}
		_, found := productMap[product.IdProduct]
		if !found && nameFallback {
			// The English-only datastores never carry the language
			// variants, whose prices must not land on the English
			// product's uuid.
			if mtgmatcher.SealedIsLanguageVariant(product.Name) {
				continue
			}
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

func (mkm *CardMarketSealed) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

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
	}
	return
}
