package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TCGSellerInventory struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	sellerKeys    []string
	onlyDirect    bool
	requestSize   int
	client        *SellerClient
	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
}

func (tcg *TCGSellerInventory) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("["+tcg.Info().Shorthand+"] "+format, a...)
	}
}

const (
	defaultSellerInventoryConcurrency = 8

	MaxPagesGlobalScrapingValue = 200
)

func NewScraperForSellerIds(sellerKeys []string, onlyDirect bool) *TCGSellerInventory {
	tcg := TCGSellerInventory{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.sellerKeys = sellerKeys
	tcg.onlyDirect = onlyDirect

	tcg.client = NewSellerClient()
	tcg.MaxConcurrency = defaultSellerInventoryConcurrency

	return &tcg
}

type itemsRecap struct {
	TotalResults int
	Pair         []setCountPair
}

type setCountPair struct {
	Idx   int
	Name  string
	Count int
}

func (tcg *TCGSellerInventory) totalItems(ctx context.Context) (*itemsRecap, error) {
	response, err := tcg.client.InventoryForSeller(ctx, tcg.sellerKeys, 0, 0, tcg.onlyDirect, nil)
	if err != nil {
		return nil, err
	}

	if len(response.Results) == 0 {
		return nil, errors.New("empty response")
	}

	var ret itemsRecap

	ret.TotalResults = response.Results[0].TotalResults

	for i, aggregation := range response.Results[0].Aggregations.SetName {
		ret.Pair = append(ret.Pair, setCountPair{
			Idx:   i,
			Name:  aggregation.URLValue,
			Count: int(aggregation.Count),
		})
	}

	return &ret, nil
}

var conditionMap = map[string]string{
	"Near Mint":         "NM",
	"Lightly Played":    "SP",
	"Moderately Played": "MP",
	"Heavily Played":    "HP",
	"Damaged":           "PO",
}

func (tcg *TCGSellerInventory) processEntry(ctx context.Context, channel chan<- responseChan, page int) error {
	for _, finish := range []string{"Normal", "Foil"} {
		response, err := tcg.client.InventoryForSeller(ctx, tcg.sellerKeys, tcg.requestSize, page, tcg.onlyDirect, []string{finish})
		if err != nil {
			tcg.printf("InventoryForSeller (entry) %s %s", finish, err.Error())
			continue
		}
		err = tcg.processInventory(channel, response.Results[0].Results)
		if err != nil {
			tcg.printf("processInventory %s %s", finish, err.Error())
		}
	}
	return nil
}

func (tcg *TCGSellerInventory) processEdition(ctx context.Context, channel chan<- responseChan, setName string, count int) error {
	for i := 0; i <= count/tcg.requestSize; i++ {
		for _, finish := range []string{"Normal", "Foil"} {
			response, err := tcg.client.InventoryForSeller(ctx, tcg.sellerKeys, tcg.requestSize, i, tcg.onlyDirect, []string{finish}, setName)
			if err != nil {
				tcg.printf("InventoryForSeller (edition) %s %s", finish, err.Error())
				continue
			}
			err = tcg.processInventory(channel, response.Results[0].Results)
			if err != nil {
				tcg.printf("processInventory %s %s", finish, err.Error())
			}
		}
	}
	return nil
}

func (tcg *TCGSellerInventory) processInventory(channel chan<- responseChan, results []SellerInventoryResult) error {
	for _, result := range results {
		if result.Sealed {
			continue
		}

		tcgProductID := fmt.Sprint(int(result.ProductID))
		uuid := mtgmatcher.Tcg2UUID(tcgProductID)
		if uuid == "" {
			continue
		}

		for _, listing := range result.Listings {
			isFoil := listing.Printing == "Foil"
			isEtched := strings.Contains(result.ProductName, "Foil Etched")
			cardId, err := mtgmatcher.MatchId(uuid, isFoil, isEtched)
			if err != nil {
				continue
			}

			if listing.Language != "English" {
				co, _ := mtgmatcher.GetUUID(cardId)
				if listing.Language != co.Language {
					continue
				}
			}

			cond, found := conditionMap[listing.Condition]
			if !found {
				return fmt.Errorf("condition not found: %s", listing.Condition)
			}

			if listing.Price == 0 || listing.Quantity == 0 {
				continue
			}

			customFields := map[string]string{
				"sellerKey": listing.SellerKey,
			}

			isDirect := listing.DirectSeller && listing.DirectProduct && listing.DirectInventory > 0
			if isDirect {
				customFields["directInventory"] = fmt.Sprint(int(listing.DirectInventory))
			}

			link := GenerateProductURL(int(result.ProductID), listing.Printing, tcg.Affiliate, listing.Condition, listing.Language, isDirect)

			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Price:        listing.Price,
					Quantity:     int(listing.Quantity),
					Conditions:   cond,
					URL:          link,
					SellerName:   listing.SellerName,
					Bundle:       isDirect,
					OriginalId:   fmt.Sprint(int(listing.ProductID)),
					InstanceId:   fmt.Sprint(int(listing.ProductConditionID)),
					CustomFields: customFields,
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGSellerInventory) Load(ctx context.Context) error {
	ret, err := tcg.totalItems(ctx)
	if err != nil {
		return err
	}

	tcg.requestSize = DefaultSellerRequestSize
	tcg.printf("Found %d results for seller id %s", ret.TotalResults, tcg.Info().Shorthand)

	results := make(chan responseChan)
	var wg sync.WaitGroup

	if ret.TotalResults/tcg.requestSize < MaxPagesGlobalScrapingValue {
		tcg.printf("Using global scraping")
		pages := make(chan int)
		for i := 0; i < tcg.MaxConcurrency; i++ {
			wg.Add(1)
			go func() {
				for page := range pages {
					tcg.printf("processing page %d/%d", page, ret.TotalResults/tcg.requestSize)
					err := tcg.processEntry(ctx, results, page)
					if err != nil {
						tcg.printf("%v", err)
					}
				}
				wg.Done()
			}()
		}

		go func() {
			for i := 0; i <= ret.TotalResults/tcg.requestSize; i++ {
				pages <- i
			}
			close(pages)

			wg.Wait()
			close(results)
		}()
	} else {
		tcg.printf("Using per-edition scraping, this might take a while")
		pairs := make(chan setCountPair)
		for i := 0; i < tcg.MaxConcurrency; i++ {
			wg.Add(1)
			go func() {
				for pair := range pairs {
					tcg.printf("processing edition %d/%d (%s)", pair.Idx+1, len(ret.Pair), pair.Name)
					err := tcg.processEdition(ctx, results, pair.Name, pair.Count)
					if err != nil {
						tcg.printf("%v", err)
					}
				}
				wg.Done()
			}()
		}

		go func() {
			for _, pair := range ret.Pair {
				pairs <- pair
			}
			close(pairs)

			wg.Wait()
			close(results)
		}()

	}

	for result := range results {
		err := tcg.inventory.AddRelaxed(result.cardId, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGSellerInventory) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

func (tcg *TCGSellerInventory) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Seller Inventory"
	tag := strings.Join(tcg.sellerKeys, ",")
	if tcg.onlyDirect {
		if tag != "" {
			tag += "+"
		}
		tag += "direct"
	}
	if tag != "" {
		tag = "_" + tag
	}
	info.Shorthand = "TCGSI" + tag
	info.InventoryTimestamp = &tcg.inventoryDate
	info.CountryFlag = "EU"
	return
}
