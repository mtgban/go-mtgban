package cardmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type CardMarketSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	exchangeRate  float64

	inventory mtgban.InventoryRecord

	client *MKMClient
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
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mkm.exchangeRate = rate
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
	"unsealed",
	"used",
	"without",
}

func (mkm *CardMarketSealed) processProduct(channel chan<- responseChan, idProduct int, uuid string) error {
	u, err := url.Parse("https://www.cardmarket.com/en/Magic/Products/Search")
	if err != nil {
		return err
	}

	var done bool
	var i int
	for !done {
		// We process a tenth of the typical request because we only need the first few results
		articles, err := mkm.client.MKMArticles(idProduct, false, i, mkmMaxEntities/10)
		if err != nil {
			return err
		}
		i++

		if len(articles) == 0 {
			break
		}

		for _, article := range articles {
			if article.Price == 0 {
				continue
			}

			// Filter by language (the search option in the API seems to have no effect)
			if article.Language.LanguageName != "English" {
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

			v := url.Values{}
			v.Set("searchString", article.Product.Name)
			if mkm.Affiliate != "" {
				v.Set("utm_source", mkm.Affiliate)
				v.Set("utm_medium", "text")
				v.Set("utm_campaign", "card_prices")
			}
			v.Set("language", "1")
			u.RawQuery = v.Encode()

			out := responseChan{
				cardId: uuid,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      article.Price * mkm.exchangeRate,
					Quantity:   article.Count,
					SellerName: article.Seller.Username,
					URL:        u.String(),
					OriginalId: fmt.Sprint(article.IdProduct),
					InstanceId: fmt.Sprint(article.IdArticle),
				},
			}
			channel <- out

			// Only keep the first price found
			done = true
			break
		}
	}

	return nil
}

func (mkm *CardMarketSealed) scrape() error {
	productMap := map[int]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || !co.Sealed || co.Identifiers["mcmId"] == "" {
			continue
		}
		mcmId, err := strconv.Atoi(co.Identifiers["mcmId"])
		if err != nil {
			continue
		}
		productMap[mcmId] = uuid
	}
	mkm.printf("Loaded %d sealed products", len(productMap))

	productList, err := mkm.client.MKMProductList()
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
				uuid := productMap[idProduct]
				co, _ := mtgmatcher.GetUUID(uuid)
				mkm.printf("Processing %s (%d/%d)...", co, slices.Index(productIds, idProduct)+1, len(productIds))

				err = mkm.processProduct(channel, idProduct, uuid)
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
			mkm.printf("%s - %s", result.ogId, err.Error())
			continue
		}
	}

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.printf("Total number of products found: %d", len(mkm.inventory))
	mkm.inventoryDate = time.Now()
	return nil
}

func (mkm *CardMarketSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(mkm.inventory) > 0 {
		return mkm.inventory, nil
	}

	err := mkm.scrape()
	if err != nil {
		return nil, err
	}

	return mkm.inventory, nil
}

func (mkm *CardMarketSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardmarket"
	info.Shorthand = "MKMSealed"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.SealedMode = true
	return
}
