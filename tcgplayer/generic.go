package tcgplayer

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"time"

	"github.com/kodabb/go-mtgban/mtgban"
)

type TCGPlayerGeneric struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	editions map[int]string

	category            int
	categoryName        string
	categoryDisplayName string

	groups []string

	client *TCGClient
}

func (tcg *TCGPlayerGeneric) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if tcg.groups[0] != "Cards" {
			tag += "{" + strings.Join(tcg.groups, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

func NewScraperGeneric(publicId, privateId string, category int, groups ...string) (*TCGPlayerGeneric, error) {
	tcg := TCGPlayerGeneric{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	check, err := tcg.client.TCGCategoriesDetails([]int{category})
	if err != nil {
		return nil, err
	}
	if len(check) == 0 {
		return nil, errors.New("empty categories response")
	}
	tcg.category = category
	tcg.categoryName = check[0].Name
	tcg.categoryDisplayName = check[0].DisplayName

	tcg.groups = groups
	if len(tcg.groups) == 0 {
		tcg.groups = []string{"Cards"}
	}

	return &tcg, nil
}

type genericChan struct {
	key   string
	entry mtgban.InventoryEntry
}

func (tcg *TCGPlayerGeneric) processPage(channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(tcg.category, tcg.groups, false, page, MaxLimit)
	if err != nil {
		return err
	}

	prodMap := map[int]TCGProduct{}
	ids := make([]string, 0, len(products))
	for _, product := range products {
		ids = append(ids, fmt.Sprint(product.ProductId))
		prodMap[product.ProductId] = product
	}

	results, err := tcg.client.TCGPricesForIds(ids)
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

		link := "https://www.tcgplayer.com/product/" + fmt.Sprint(result.ProductId)
		link += "&Printing=" + result.SubTypeName

		if tcg.Affiliate != "" {
			link += fmt.Sprintf("&utm_campaign=affiliate&utm_medium=%s&utm_source=%s&partner=%s", tcg.Affiliate, tcg.Affiliate, tcg.Affiliate)
		}

		keys := []string{
			fmt.Sprint(result.ProductId),
			prodMap[result.ProductId].Name,
			tcg.editions[prodMap[result.ProductId].GroupId],
			result.SubTypeName,
		}

		for i := range names {
			if prices[i] == 0 {
				continue
			}
			out := genericChan{
				key: strings.Join(keys, "|"),
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: names[i],
					Bundle:     i == 3,
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGPlayerGeneric) scrape() error {
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

func (tcg *TCGPlayerGeneric) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) == 0 {
		_, err := tcg.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := tcg.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range tcg.inventory {
		for i := range tcg.inventory[card] {
			if tcg.inventory[card][i].SellerName == sellerName {
				if tcg.inventory[card][i].Price == 0 {
					continue
				}
				if tcg.marketplace[sellerName] == nil {
					tcg.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				tcg.marketplace[sellerName][card] = append(tcg.marketplace[sellerName][card], tcg.inventory[card][i])
			}
		}
	}

	if len(tcg.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return tcg.marketplace[sellerName], nil
}

func (tcg *TCGPlayerGeneric) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	tcg.marketplace = market
	tcg.inventory = inventory

	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerGeneric) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player - " + tcg.categoryDisplayName
	info.Shorthand = "TCG+" + tcg.categoryName
	info.InventoryTimestamp = tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true

	if tcg.groups[0] != "Cards" {
		info.Name += " " + strings.Join(tcg.groups, ",")
		info.Shorthand += "+" + strings.Join(tcg.groups, ",")
	}
	return
}
