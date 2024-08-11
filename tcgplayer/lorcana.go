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

type TCGLorcana struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]TCGGroup

	printings map[int]string

	category            int
	categoryName        string
	categoryDisplayName string

	groups []string

	client *TCGClient
}

func (tcg *TCGLorcana) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if tcg.groups[0] != "Cards" {
			tag += "{" + strings.Join(tcg.groups, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

func NewLorcanaScraper(publicId, privateId string) (*TCGLorcana, error) {
	tcg := TCGLorcana{}
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

	tcg.printings = map[int]string{}

	return &tcg, nil
}

func (tcg *TCGLorcana) processPage(channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(tcg.category, tcg.groups, true, page, MaxLimit)
	if err != nil {
		return err
	}

	productMap := map[int]TCGProduct{}
	skuMap := map[int]TCGSKU{}
	var skuIds []string
	for _, product := range products {
		productMap[product.ProductId] = product

		for _, sku := range product.Skus {
			_, found := SKUConditionMap[sku.ConditionId]
			if !found {
				continue
			}
			// Only English
			if sku.LanguageId != 1 {
				continue
			}

			skuIds = append(skuIds, fmt.Sprint(sku.SkuId))
			skuMap[sku.SkuId] = sku
		}
	}

	for i := 0; i < len(skuIds); i += maxIdsInRequest {
		start := i
		end := i + maxIdsInRequest
		if end > len(skuIds) {
			end = len(skuIds)
		}

		results, err := tcg.client.TCGPricesForSKUs(skuIds[start:end])
		if err != nil {
			return err
		}

		for _, result := range results {
			price := result.LowestListingPrice
			if price == 0 {
				continue
			}

			sku := skuMap[result.SkuId]
			product, found := productMap[sku.ProductId]
			if !found {
				continue
			}

			cardName := mtgmatcher.SplitVariants(product.Name)[0]
			cardId, err := mtgmatcher.SimpleSearch(cardName, product.GetNumber(), tcg.printings[sku.PrintingId] != "Normal")
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				tcg.printf("%v", err)
				tcg.printf("%+v", result)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					tcg.printf("%d %s got ids: %s", sku.ProductId, cardName, probes)
					for _, probe := range probes {
						co, _ := mtgmatcher.GetUUID(probe)
						tcg.printf("%s: %s", probe, co)
					}
				}
				continue
			}

			condition := SKUConditionMap[sku.ConditionId]

			link := TCGPlayerProductURL(sku.ProductId, tcg.printings[sku.PrintingId], tcg.Affiliate, condition, "", false)

			out := genericChan{
				key: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: condition,
					Price:      price,
					Quantity:   1,
					URL:        link,
					OriginalId: fmt.Sprint(sku.ProductId),
					InstanceId: fmt.Sprint(sku.SkuId),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGLorcana) scrape() error {
	printings, err := tcg.client.ListCategoryPrintings(tcg.category)
	if err != nil {
		return err
	}
	tcg.printf("Found %d printings", len(printings))
	for _, printing := range printings {
		tcg.printings[printing.PrintingId] = printing.Name
	}

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

func (tcg *TCGLorcana) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGLorcana) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player"
	info.Shorthand = "TCG Player"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.NoQuantityInventory = true
	info.Game = mtgban.GameLorcana
	return
}
