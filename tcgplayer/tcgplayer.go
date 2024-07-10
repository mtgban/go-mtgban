package tcgplayer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"golang.org/x/exp/slices"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

type TCGPlayerMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	Affiliate      string
	MaxConcurrency int

	// The cache data defining SKU data, if not set it will be loaded
	// from the default location on mtgjson website.
	SKUsData map[string][]mtgjson.TCGSku

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *TCGClient
}

type marketChan struct {
	UUID      string
	Condition string
	Printing  string
	Finish    string
	ProductId int
	SkuId     int
	Language  string
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
	bl     mtgban.BuylistEntry
}

const (
	allSkusURL       = "https://mtgjson.com/api/v5/TcgplayerSkus.json.bz2"
	allSkusBackupURL = "https://mtgjson.com/api/v5_backup/TcgplayerSkus.json.bz2"
)

var availableMarketNames = []string{
	"TCG Player", "TCG Direct",
}

var skuConditions = map[string]string{
	"NEAR MINT":         "NM",
	"LIGHTLY PLAYED":    "SP",
	"MODERATELY PLAYED": "MP",
	"HEAVILY PLAYED":    "HP",
	"DAMAGED":           "PO",
}

func (tcg *TCGPlayerMarket) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGMkt] "+format, a...)
	}
}

func NewScraperMarket(publicId, privateId string) *TCGPlayerMarket {
	tcg := TCGPlayerMarket{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerMarket) processEntry(channel chan<- responseChan, reqs []marketChan, mode string) error {
	ids := make([]string, len(reqs))
	for i := range reqs {
		ids[i] = fmt.Sprint(reqs[i].SkuId)
	}

	var results []TCGSKUPrice
	var err error

	// Retrieve a list of skus with their prices
	if mode == "inventory" {
		results, err = tcg.client.TCGPricesForSKUs(ids)
	} else if mode == "buylist" {
		results, err = tcg.client.TCGBuylistPricesForSKUs(ids)
	}
	if err != nil {
		return err
	}

	for _, result := range results {
		var req marketChan
		for _, req = range reqs {
			if result.SkuId == req.SkuId {
				break
			}
		}

		isFoil := req.Printing == "FOIL"
		isEtched := req.Finish == "FOIL ETCHED"
		cardId, err := mtgmatcher.MatchId(req.UUID, isFoil, isEtched)
		if err != nil {
			tcg.printf("%s - (tcgId:%d / uuid:%s)", err.Error(), req.ProductId, req.UUID)
			continue
		}

		// Skip impossible entries, such as listing mistakes that list a foil
		// price for a foil-only card
		co, _ := mtgmatcher.GetUUID(cardId)
		if !co.Etched &&
			((co.Foil && req.Printing != "FOIL") ||
				(!co.Foil && req.Printing != "NON FOIL")) {
			continue
		}

		cond, found := skuConditions[req.Condition]
		if !found {
			tcg.printf("unknown condition %d for %d", req.Condition, req.SkuId)
			continue
		}

		if mode == "inventory" {
			// Sorted as in availableMarketNames
			prices := []float64{
				result.LowestListingPrice, result.DirectLowPrice,
			}
			printing := "Normal"
			if req.Printing == "FOIL" {
				printing = "Foil"
			}
			for i := range availableMarketNames {
				if prices[i] == 0 {
					continue
				}

				isDirect := i == 1
				link := TCGPlayerProductURL(req.ProductId, printing, tcg.Affiliate, cond, req.Language, isDirect)

				out := responseChan{
					cardId: cardId,
					entry: mtgban.InventoryEntry{
						Conditions: cond,
						Price:      prices[i],
						Quantity:   1,
						URL:        link,
						SellerName: availableMarketNames[i],
						Bundle:     isDirect,
						OriginalId: fmt.Sprint(req.ProductId),
						InstanceId: fmt.Sprint(result.SkuId),
					},
				}

				channel <- out
			}
		} else if mode == "buylist" {
			price := result.BuylistPrices.High
			if price == 0 {
				continue
			}

			var sellPrice, priceRatio float64
			var backupPrice float64

			// Find the NM Market price of the same card id, if missing for
			// whatever reason use the tcg direct one
			invCards := tcg.inventory[cardId]
			for _, invCard := range invCards {
				if invCard.Conditions != "NM" {
					continue
				}
				if invCard.SellerName == "TCG Player" {
					backupPrice = invCard.Price
				}
				if invCard.SellerName == "TCG Direct" {
					sellPrice = invCard.Price
					break
				}
			}
			if sellPrice == 0 {
				sellPrice = backupPrice
			}

			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}
			out := responseChan{
				cardId: cardId,
				bl: mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   price,
					Quantity:   0,
					PriceRatio: priceRatio,
					URL:        "https://store.tcgplayer.com/buylist",
					OriginalId: fmt.Sprint(req.ProductId),
					InstanceId: fmt.Sprint(result.SkuId),
				},
			}

			channel <- out

		}
	}

	return nil
}

func (tcg *TCGPlayerMarket) scrape(mode string) error {
	skusMap := tcg.SKUsData
	if skusMap == nil {
		var err error
		tcg.printf("Retrieving skus")
		skusMap, err = getAllSKUs()
		if err != nil {
			return err

		}
		tcg.SKUsData = skusMap
	}
	tcg.printf("Found skus for %d %s entries", len(skusMap), mode)

	start := time.Now()

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			buffer := make([]marketChan, 0, maxIdsInRequest)

			for page := range pages {
				// Add our data to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == maxIdsInRequest {
					err := tcg.processEntry(channel, buffer, mode)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntry(channel, buffer, mode)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetAllSets()
		i := 1

		var idsFound []int
		for _, code := range sets {
			set, _ := mtgmatcher.GetSet(code)

			switch set.Code {
			case "4EDALT":
				continue
			}

			tcg.printf("[%s] Scraping %s (%d/%d)", mode, set.Name, i, len(sets))
			i++

			for _, card := range set.Cards {
				uuid := card.UUID
				skus, found := skusMap[uuid]
				if !found {
					continue
				}

				hasNonfoil := card.HasFinish(mtgjson.FinishNonfoil)
				hasFoil := card.HasFinish(mtgjson.FinishFoil)
				hasEtched := card.HasFinish(mtgjson.FinishEtched)

				for _, sku := range skus {
					// Skip sealed products
					if sku.Condition == "UNOPENED" {
						continue
					}
					// Skip non-main languages
					if sku.Language != "ENGLISH" && !strings.Contains(card.Language, mtgmatcher.Title(sku.Language)) {
						continue
					}
					// Extra validation for incorrect data
					if !hasNonfoil && sku.Printing == "NON FOIL" {
						continue
					}
					if !hasFoil && !hasEtched && (sku.Printing == "FOIL" || sku.Finish == "FOIL ETCHED") {
						continue
					}
					if !hasEtched && sku.Finish == "FOIL ETCHED" {
						continue
					}
					// Make sure the right id is parsed
					if sku.Finish != "FOIL ETCHED" && fmt.Sprint(sku.ProductId) != card.Identifiers["tcgplayerProductId"] {
						continue
					}
					// Skip dupes
					if slices.Contains(idsFound, sku.SkuId) {
						continue
					}
					idsFound = append(idsFound, sku.SkuId)

					pages <- marketChan{
						UUID:      uuid,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductId: sku.ProductId,
						SkuId:     sku.SkuId,
						Language:  sku.Language,
					}
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	if mode == "inventory" {
		for result := range channel {
			err := tcg.inventory.AddStrict(result.cardId, &result.entry)
			if err != nil {
				tcg.printf("%s", err.Error())
				continue
			}
		}
		tcg.inventoryDate = time.Now()
	} else if mode == "buylist" {
		for result := range channel {
			err := tcg.buylist.Add(result.cardId, &result.bl)
			if err != nil {
				tcg.printf("%s", err.Error())
				continue
			}
		}
		tcg.buylistDate = time.Now()
	}

	tcg.printf("Took %v", time.Since(start))

	return nil
}

func (tcg *TCGPlayerMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape("inventory")
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerMarket) Buylist() (mtgban.BuylistRecord, error) {
	if len(tcg.buylist) > 0 {
		return tcg.buylist, nil
	}

	err := tcg.scrape("buylist")
	if err != nil {
		return nil, err
	}

	return tcg.buylist, nil
}

func (tcg *TCGPlayerMarket) MarketNames() []string {
	return availableMarketNames
}

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.BuylistTimestamp = &tcg.buylistDate
	info.NoQuantityInventory = true
	return
}

func getAllSKUs() (map[string][]mtgjson.TCGSku, error) {
	resp, err := cleanhttp.DefaultClient().Get(allSkusURL)
	if err != nil {
		resp, err = cleanhttp.DefaultClient().Get(allSkusBackupURL)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	skus, err := LoadTCGSKUs(resp.Body)
	if err != nil {
		return nil, err
	}
	return skus.Data, nil
}
