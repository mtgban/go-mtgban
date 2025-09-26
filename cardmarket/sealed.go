package cardmarket

import (
	"context"
	"fmt"
	"slices"
	"sync"
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

	inventoryDate time.Time
	exchangeRate  float64

	inventory mtgban.InventoryRecord

	client *MKMClient
	gameId int
}

func (mkm *CardMarketSealed) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMSealed] "+format, a...)
	}
}

func NewScraperSealed(appToken, appSecret string) (*CardMarketSealed, error) {
	mkm := CardMarketSealed{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	mkm.gameId = GameIdMagic
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
				cardId: uuid,
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

	productList, err := GetProductListSealed(ctx, mkm.gameId)
	if err != nil {
		return err
	}
	mkm.printf("Loaded %d mkm products", len(productList))

	var productIds []int
	for _, product := range productList {
		_, found := productMap[product.IdProduct]
		if !found {
			continue
		}
		if mkm.TargetProduct != "" && mkm.TargetProduct != product.Name {
			continue
		}
		productIds = append(productIds, product.IdProduct)
	}
	mkm.printf("Mapped %d mkm products to sealed products", len(productIds))

	products := make(chan int)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for idProduct := range products {
				uuids := productMap[idProduct]
				co, err := mtgmatcher.GetUUID(uuids[0])
				if err != nil {
					continue
				}
				if mkm.TargetEdition != "" && mkm.TargetEdition != co.Edition && mkm.TargetEdition != co.SetCode {
					continue
				}

				mkm.printf("Processing %s (%d/%d)...", co, slices.Index(productIds, idProduct)+1, len(productIds))

				err = mkm.processProduct(ctx, channel, idProduct, uuids)
				if err != nil {
					mkm.printf("%s (%d) %s", co, idProduct, err.Error())
					continue
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, id := range productIds {
			products <- id
		}
		close(products)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := mkm.inventory.AddStrict(result.cardId, &result.entry)
		if err != nil {
			_, cerr := mtgmatcher.GetUUID(result.cardId)
			if cerr != nil {
				mkm.printf("%s - %s: %s", result.entry.OriginalId, cerr.Error(), result.cardId)
				continue
			}
			mkm.printf("%d - %s", result.ogId, err.Error())
			continue
		}
	}

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
	return
}
