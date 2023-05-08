package tcgplayer

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TCGSellerInventory struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	sellerKey     string
	sellerName    string
	requestSize   int
	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
}

func (tcg *TCGSellerInventory) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSI_"+tcg.sellerKey+"] "+format, a...)
	}
}

const defaultSellerInventoryConcurrency = 8

func NewScraperForSeller(sellerName string) (*TCGSellerInventory, error) {
	sellerKey, err := SellerName2ID(sellerName)
	if err != nil {
		return nil, err
	}

	tcg := TCGSellerInventory{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.sellerKey = sellerKey
	tcg.sellerName = sellerName

	tcg.MaxConcurrency = defaultSellerInventoryConcurrency

	return &tcg, nil
}

type itemsRecap struct {
	TotalResults int
	Pair         []setCountPair
}

type setCountPair struct {
	Name  string
	Count int
}

func (tcg *TCGSellerInventory) totalItems() (*itemsRecap, error) {
	resp, err := TCGInventoryForSeller(tcg.sellerKey, 0, 0)
	if err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return nil, errors.New("empty response")
	}

	var ret itemsRecap

	ret.TotalResults = resp.Results[0].TotalResults

	for _, aggregation := range resp.Results[0].Aggregations.SetName {
		ret.Pair = append(ret.Pair, setCountPair{
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

func (tcg *TCGSellerInventory) processEntry(channel chan<- responseChan, page int) error {
	resp, err := TCGInventoryForSeller(tcg.sellerKey, tcg.requestSize, page)
	if err != nil {
		return err
	}

	return tcg.processInventory(channel, resp.Results[0].Results)
}

func (tcg *TCGSellerInventory) processEdition(channel chan<- responseChan, setName string, count int) error {
	for i := 0; i <= count/tcg.requestSize; i++ {
		resp, err := TCGInventoryForSeller(tcg.sellerKey, tcg.requestSize, i, setName)
		if err != nil {
			return err
		}

		err = tcg.processInventory(channel, resp.Results[0].Results)
		if err != nil {
			return err
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
			if mtgmatcher.SkipLanguage(result.ProductName, result.SetName, listing.Language) {
				continue
			}

			id := uuid
			switch result.SetName {
			case "Legends", "The Dark":
				id += "_ita"
			case "Strixhaven Mystical Archive":
				num, _ := strconv.Atoi(result.CustomAttributes.Number)
				if listing.Language == "Japanese" && num < 64 {
					continue
				}
			}

			isFoil := listing.Printing == "Foil"
			cardId, err := mtgmatcher.MatchId(id, isFoil)
			if err != nil {
				continue
			}

			cond, found := conditionMap[listing.Condition]
			if !found {
				return fmt.Errorf("condition not found: %s", listing.Condition)
			}

			if listing.Price == 0 || listing.Quantity == 0 {
				continue
			}

			link := TCGPlayerProductURL(int(result.ProductID), listing.Printing, tcg.Affiliate, listing.Language)
			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Price:      listing.Price,
					Quantity:   int(listing.Quantity),
					Conditions: cond,
					URL:        link,
					SellerName: listing.SellerName,
					Bundle:     listing.DirectProduct,
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGSellerInventory) scrape() error {
	ret, err := tcg.totalItems()
	if err != nil {
		return err
	}

	tcg.requestSize = DefaultSellerRequestSize
	tcg.printf("Found %d results for %s", ret.TotalResults, tcg.sellerName)

	results := make(chan responseChan)
	var wg sync.WaitGroup

	if ret.TotalResults < MaxGlobalScrapingValue {
		tcg.printf("Using global scraping")
		pages := make(chan int)
		for i := 0; i < tcg.MaxConcurrency; i++ {
			wg.Add(1)
			go func() {
				for page := range pages {
					tcg.printf("processing page %d/%d", page, ret.TotalResults/tcg.requestSize)
					err := tcg.processEntry(results, page)
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
					tcg.printf("processing edition %s", pair.Name)
					err := tcg.processEdition(results, pair.Name, pair.Count)
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

func (tcg *TCGSellerInventory) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGSellerInventory) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Seller Inventory: " + tcg.sellerName
	info.Shorthand = "TCGSI_" + tcg.sellerKey
	info.InventoryTimestamp = &tcg.inventoryDate
	info.CountryFlag = "EU"
	return
}