package tcgplayer

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-tcgplayer"
)

// Generic prices any partner API category by number, for the games
// and product types that have no scraper of their own.
type Generic struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]tcgplayer.Group

	category            int
	categoryName        string
	categoryDisplayName string

	productTypes []string

	client *tcgplayer.Client
}

func (tcg *Generic) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Equal(tcg.productTypes, tcgplayer.SinglesProductTypes(tcg.category)) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

// NewScraperGeneric returns a scraper for one category id, optionally narrowed
// to the named product types.
func NewScraperGeneric(publicID, privateID string, category int, productTypes ...string) (*Generic, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}
	tcg := Generic{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency
	tcg.category = category

	tcg.productTypes = productTypes
	if len(tcg.productTypes) == 0 {
		tcg.productTypes = tcgplayer.SinglesProductTypes(category)
	}

	return &tcg, nil
}

type genericChan struct {
	key   string
	entry mtgban.InventoryEntry
}

func (tcg *Generic) processPage(ctx context.Context, channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(ctx, tcg.category, tcg.productTypes, false, page)
	if err != nil {
		return err
	}

	prodMap := map[int]tcgplayer.Product{}
	ids := make([]int, len(products))
	for i, product := range products {
		ids[i] = product.ProductID
		prodMap[product.ProductID] = product
	}

	results, err := tcg.client.GetMarketPricesByProducts(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, result.DirectLowPrice,
		}
		names := []string{
			"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
		}

		keys := []string{
			fmt.Sprint(result.ProductID),
			prodMap[result.ProductID].Name,
			tcg.editions[prodMap[result.ProductID].GroupID].Name,
			result.SubTypeName,
		}

		for i := range names {
			if prices[i] == 0 {
				continue
			}

			isDirect := names[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.Affiliate, "", "", isDirect)

			out := genericChan{
				key: strings.Join(keys, "|"),
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: names[i],
					Bundle:     isDirect,
					OriginalID: fmt.Sprint(result.ProductID),
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Generic) Load(ctx context.Context) error {
	// Initialize data for debug logs
	var err error
	tcg.categoryName, tcg.categoryDisplayName, err = GetCategoryNames(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}

	editions, err := EditionMap(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	totals, err := tcg.client.TotalProducts(ctx, tcg.category, tcg.productTypes)
	if err != nil {
		return err
	}
	tcg.printf("Found %d products", totals)

	pageNums := make([]int, 0, totals/tcgplayer.MaxItemsInResponse+1)
	for i := 0; i < totals; i += tcgplayer.MaxItemsInResponse {
		pageNums = append(pageNums, i)
	}

	mtgban.WorkerPool(ctx, tcg.MaxConcurrency, pageNums,
		func(ctx context.Context, page int, channel chan<- genericChan) error {
			return tcg.processPage(ctx, channel, page)
		},
		func(result genericChan) {
			err := tcg.inventory.Add(result.key, &result.entry)
			if err != nil {
				tcg.printf("%s", err.Error())
			}
		},
		tcg.printf,
	)

	tcg.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *Generic) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *Generic) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCGplayer - " + tcg.categoryDisplayName
	info.Shorthand = "TCG+" + tcg.categoryName
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true

	if !slices.Equal(tcg.productTypes, tcgplayer.SinglesProductTypes(tcg.category)) {
		info.Name += " " + strings.Join(tcg.productTypes, ",")
		info.Shorthand += "+" + strings.Join(tcg.productTypes, ",")
	}
	return
}
