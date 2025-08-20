package tcgplayer

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	tcgplayer "github.com/mtgban/go-tcgplayer"
)

type TCGPlayerGeneric struct {
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

func (tcg *TCGPlayerGeneric) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Contains(tcg.productTypes, tcgplayer.ProductTypesSingles[0]) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

func NewScraperGeneric(publicId, privateId string, category int, productTypes ...string) (*TCGPlayerGeneric, error) {
	if publicId == "" || privateId == "" {
		return nil, fmt.Errorf("missing authentication data")
	}
	tcg := TCGPlayerGeneric{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = tcgplayer.NewClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency

	check, err := tcg.client.GetCategoriesDetails([]int{category})
	if err != nil {
		return nil, err
	}
	if len(check) == 0 {
		return nil, errors.New("empty categories response")
	}

	tcg.category = category
	tcg.categoryName = check[0].Name
	tcg.categoryDisplayName = check[0].DisplayName

	tcg.productTypes = productTypes
	if len(tcg.productTypes) == 0 {
		tcg.productTypes = tcgplayer.ProductTypesSingles
	}

	return &tcg, nil
}

type genericChan struct {
	key   string
	entry mtgban.InventoryEntry
}

func (tcg *TCGPlayerGeneric) processPage(channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(tcg.category, tcg.productTypes, false, page)
	if err != nil {
		return err
	}

	prodMap := map[int]tcgplayer.Product{}
	ids := make([]int, len(products))
	for i, product := range products {
		ids[i] = product.ProductId
		prodMap[product.ProductId] = product
	}

	results, err := tcg.client.GetMarketPricesByProducts(ids)
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
			fmt.Sprint(result.ProductId),
			prodMap[result.ProductId].Name,
			tcg.editions[prodMap[result.ProductId].GroupId].Name,
			result.SubTypeName,
		}

		for i := range names {
			if prices[i] == 0 {
				continue
			}

			isDirect := names[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductId, result.SubTypeName, tcg.Affiliate, "", "", isDirect)

			out := genericChan{
				key: strings.Join(keys, "|"),
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: names[i],
					Bundle:     isDirect,
					OriginalId: fmt.Sprint(result.ProductId),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGPlayerGeneric) scrape() error {
	editions, err := EditionMap(tcg.client, tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	totals, err := tcg.client.TotalProducts(tcg.category, tcg.productTypes)
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

func (tcg *TCGPlayerGeneric) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerGeneric) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCGplayer - " + tcg.categoryDisplayName
	info.Shorthand = "TCG+" + tcg.categoryName
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true

	if !slices.Contains(tcg.productTypes, tcgplayer.ProductTypesSingles[0]) {
		info.Name += " " + strings.Join(tcg.productTypes, ",")
		info.Shorthand += "+" + strings.Join(tcg.productTypes, ",")
	}
	return
}
