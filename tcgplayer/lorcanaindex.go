package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	tcgplayer "github.com/mtgban/go-tcgplayer"
)

type TCGLorcanaIndex struct {
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

func (tcg *TCGLorcanaIndex) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Contains(tcg.productTypes, tcgplayer.ProductTypesSingles[0]) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

func NewLorcanaIndex(publicId, privateId string) (*TCGLorcanaIndex, error) {
	client, err := tcgplayer.NewClient(publicId, privateId)
	if err != nil {
		return nil, err
	}

	tcg := TCGLorcanaIndex{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency
	tcg.category = tcgplayer.CategoryLorcana

	check, err := tcg.client.GetCategoriesDetails(context.TODO(), []int{tcg.category})
	if err != nil {
		return nil, err
	}
	if len(check) == 0 {
		return nil, errors.New("empty categories response")
	}

	tcg.categoryName = check[0].Name
	tcg.categoryDisplayName = check[0].DisplayName
	tcg.productTypes = tcgplayer.ProductTypesSingles

	return &tcg, nil
}

func (tcg *TCGLorcanaIndex) processPage(ctx context.Context, channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(ctx, tcg.category, tcg.productTypes, false, page)
	if err != nil {
		return err
	}

	productMap := map[int]tcgplayer.Product{}
	ids := make([]int, len(products))
	for i, product := range products {
		ids[i] = product.ProductId
		productMap[product.ProductId] = product
	}

	results, err := tcg.client.GetMarketPricesByProducts(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		product, found := productMap[result.ProductId]
		if !found {
			continue
		}

		cardName := productMap[result.ProductId].Name
		number := GetProductNumber(&product)
		cardId, err := mtgmatcher.SimpleSearch(cardName, number, result.SubTypeName != "Normal")
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			tcg.printf("%v", err)
			tcg.printf("%+v", result)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				tcg.printf("%d %s got ids: %s", product.ProductId, cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					tcg.printf("%s: %s", probe, co)
				}
			}
			continue
		}

		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, result.DirectLowPrice,
		}

		for i := range prices {
			if prices[i] == 0 {
				continue
			}

			isDirect := availableIndexNames[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductId, result.SubTypeName, tcg.Affiliate, "", "", isDirect)

			out := genericChan{
				key: cardId,
				entry: mtgban.InventoryEntry{
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableIndexNames[i],
					Bundle:     isDirect,
					OriginalId: fmt.Sprint(result.ProductId),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGLorcanaIndex) scrape(ctx context.Context) error {
	editions, err := EditionMap(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	totals, err := tcg.client.TotalProducts(ctx, tcg.category, []string{"Cards"})
	if err != nil {
		return err
	}
	tcg.printf("Found %d products", totals)

	pages := make(chan int)
	channel := make(chan genericChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {

			for page := range pages {
				err := tcg.processPage(ctx, channel, page)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < totals; i += tcgplayer.MaxItemsInResponse {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.inventory.Add(result.key, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGLorcanaIndex) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape(context.TODO())
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGLorcanaIndex) MarketNames() []string {
	return availableIndexNames[:len(availableIndexNames)-1]
}

func (tcg *TCGLorcanaIndex) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

func (tcg *TCGLorcanaIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.Game = mtgban.GameLorcana
	return
}
