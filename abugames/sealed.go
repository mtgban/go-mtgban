package abugames

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type ABUGamesSealed struct {
	LogCallback mtgban.LogCallbackFunc

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	productMap map[string]string
	client     *ABUClient

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraperSealed() *ABUGamesSealed {
	abu := ABUGamesSealed{}
	abu.inventory = mtgban.InventoryRecord{}
	abu.buylist = mtgban.BuylistRecord{}
	abu.MaxConcurrency = defaultConcurrency
	abu.client = NewABUClient()

	abu.productMap = map[string]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		id, found := co.Identifiers["abuId"]
		if !found {
			continue
		}
		abu.productMap[id] = co.UUID
	}
	return &abu
}

func (abu *ABUGamesSealed) printf(format string, a ...interface{}) {
	if abu.LogCallback != nil {
		abu.LogCallback("[ABUSealed] "+format, a...)
	}
}

func (abu *ABUGamesSealed) processEntry(ctx context.Context, channel chan<- resultChan, page int) error {
	response, err := abu.client.GetSealedProduct(ctx, page)
	if err != nil {
		return err
	}

	for _, doc := range response.Response.Docs {
		productId, found := abu.productMap[doc.Id]
		if !found {
			continue
		}

		var invEntry *mtgban.InventoryEntry
		var buyEntry *mtgban.BuylistEntry
		var tradeEntry *mtgban.BuylistEntry

		u, err := url.Parse("https://abugames.com")
		if err != nil {
			return err
		}

		// This works differently than the singles search
		v := url.Values{}
		v.Set("search", doc.DisplayTitle)
		u.RawQuery = v.Encode()

		if doc.SellQuantity > 0 && doc.SellPrice > 0 {
			u.Path = "/magic-the-gathering/packs"

			invEntry = &mtgban.InventoryEntry{
				Price:      doc.SellPrice,
				Quantity:   doc.SellQuantity,
				URL:        u.String(),
				OriginalId: doc.Id,
			}
		}

		if doc.BuyQuantity > 0 && doc.BuyPrice > 0 {
			var priceRatio float64
			if doc.SellPrice > 0 {
				priceRatio = doc.BuyPrice / doc.SellPrice * 100
			}

			u.Path = "/buylist/packs"

			buyEntry = &mtgban.BuylistEntry{
				BuyPrice:   doc.BuyPrice,
				Quantity:   doc.BuyQuantity,
				PriceRatio: priceRatio,
				URL:        u.String(),
				OriginalId: doc.Id,
				VendorName: availableTraderNames[0],
			}

			if doc.SellPrice > 0 {
				priceRatio = doc.TradePrice / doc.SellPrice * 100
			}
			tradeEntry = &mtgban.BuylistEntry{
				BuyPrice:   doc.TradePrice,
				Quantity:   doc.BuyQuantity,
				PriceRatio: priceRatio,
				URL:        u.String(),
				OriginalId: doc.Id,
				VendorName: availableTraderNames[1],
			}
		}

		if invEntry != nil || buyEntry != nil {
			channel <- resultChan{
				cardId:     productId,
				invEntry:   invEntry,
				buyEntry:   buyEntry,
				tradeEntry: tradeEntry,
			}
		}
	}

	return nil
}

func (abu *ABUGamesSealed) Load(ctx context.Context) error {
	count, err := abu.client.GetTotalSealedItems(ctx)
	if err != nil {
		return err
	}
	abu.printf("Parsing %d entries", count)

	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < abu.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := abu.processEntry(ctx, results, page)
				if err != nil {
					abu.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < count; i += maxEntryPerRequest {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.invEntry != nil {
			err = abu.inventory.AddRelaxed(result.cardId, result.invEntry)
			if err != nil {
				abu.printf("%s", &result.theCard)
				abu.printf("%s", err.Error())
			}
		}
		if result.buyEntry != nil {
			err = abu.buylist.AddRelaxed(result.cardId, result.buyEntry)
			if err != nil {
				abu.printf("%s", &result.theCard)
				abu.printf("%s", err.Error())
			}
		}
		if result.tradeEntry != nil {
			err = abu.buylist.AddRelaxed(result.cardId, result.tradeEntry)
			if err != nil {
				abu.printf("%s", &result.theCard)
				abu.printf("%s", err.Error())
			}
		}
	}

	abu.inventoryDate = time.Now()
	abu.buylistDate = time.Now()

	return nil
}

func (abu *ABUGamesSealed) Inventory() mtgban.InventoryRecord {
	return abu.inventory
}

func (abu *ABUGamesSealed) Buylist() mtgban.BuylistRecord {
	return abu.buylist
}

func (abu *ABUGamesSealed) TraderNames() []string {
	return availableTraderNames
}

var name2shorthandSealed = map[string]string{
	"ABU Games":          "ABUGamesSealed",
	"ABU Games (credit)": "ABUCreditSealed",
}

func (abu *ABUGamesSealed) InfoForScraper(name string) mtgban.ScraperInfo {
	info := abu.Info()
	info.Name = name
	info.Shorthand = name2shorthandSealed[name]
	return info
}

func (abu *ABUGamesSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "ABU Games"
	info.Shorthand = "ABUSealed"
	info.InventoryTimestamp = &abu.inventoryDate
	info.BuylistTimestamp = &abu.buylistDate
	info.SealedMode = true
	return
}
