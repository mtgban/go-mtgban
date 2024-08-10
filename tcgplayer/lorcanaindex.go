package tcgplayer

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TCGLorcanaIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]TCGGroup

	category            int
	categoryName        string
	categoryDisplayName string

	groups []string

	client *TCGClient
}

func (tcg *TCGLorcanaIndex) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if tcg.groups[0] != "Cards" {
			tag += "{" + strings.Join(tcg.groups, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

func NewLorcanaIndex(publicId, privateId string) (*TCGLorcanaIndex, error) {
	tcg := TCGLorcanaIndex{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency

	tcg.category = CategoryLorcana

	check, err := tcg.client.TCGCategoriesDetails([]int{tcg.category})
	if err != nil {
		return nil, err
	}
	if len(check) == 0 {
		return nil, errors.New("empty categories response")
	}
	tcg.categoryName = check[0].Name
	tcg.categoryDisplayName = check[0].DisplayName
	tcg.groups = []string{"Cards"}

	return &tcg, nil
}

func (tcg *TCGLorcanaIndex) processPage(channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(tcg.category, tcg.groups, false, page, MaxLimit)
	if err != nil {
		return err
	}

	productMap := map[int]TCGProduct{}
	ids := make([]string, 0, len(products))
	for _, product := range products {
		ids = append(ids, fmt.Sprint(product.ProductId))
		productMap[product.ProductId] = product
	}

	results, err := tcg.client.TCGPricesForIds(ids)
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

		cardName := mtgmatcher.SplitVariants(productMap[result.ProductId].Name)[0]
		cardIds, err := GenericSearch(cardName, result.SubTypeName, product.GetNumber())
		if err != nil {
			continue
		}
		if len(cardIds) != 1 {
			tcg.printf("%d %s got ids: %s", result.ProductId, cardName, cardIds)
			for _, uuid := range cardIds {
				co, _ := mtgmatcher.GetUUID(uuid)
				tcg.printf("%s: %s", uuid, co)
			}
			tcg.printf("%+v", result)
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
			link := TCGPlayerProductURL(result.ProductId, result.SubTypeName, tcg.Affiliate, "", "", isDirect)

			out := genericChan{
				key: cardIds[0],
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

func (tcg *TCGLorcanaIndex) scrape() error {
	editions, err := tcg.client.EditionMap(tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	totals, err := tcg.client.TotalProducts(tcg.category, []string{"Cards"})
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
				err := tcg.processPage(channel, page)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < totals; i += MaxLimit {
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

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGLorcanaIndex) MarketNames() []string {
	return availableIndexNames[:len(availableIndexNames)-1]
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
