package tcgplayer

import (
	"fmt"
	"sync"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"golang.org/x/exp/slices"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	tcgplayer "github.com/mtgban/go-tcgplayer"
)

type TCGPlayerMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	// The cache data defining SKU data, if not set it will be loaded
	// from the default location on mtgjson website.
	SKUsData map[string][]mtgjson.TCGSku

	inventory mtgban.InventoryRecord

	client *tcgplayer.Client
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

var name2shorthand = map[string]string{
	"TCG Low":          "TCGLow",
	"TCG Market":       "TCGMarket",
	"TCG Mid":          "TCGMid",
	"TCG Direct Low":   "TCGDirectLow",
	"TCG Player":       "TCGPlayer",
	"TCG Direct":       "TCGDirect",
	"TCGplayer":        "TCGPlayer",
	"TCGplayer Direct": "TCGDirect",
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
	tcg.client = tcgplayer.NewClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerMarket) processEntry(channel chan<- responseChan, reqs []marketChan) error {
	ids := make([]int, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].SkuId
	}

	// Retrieve a list of skus with their prices
	results, err := tcg.client.GetMarketPricesBySKUs(ids)
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
	}

	return nil
}

func (tcg *TCGPlayerMarket) scrape() error {
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
	tcg.printf("Found skus for %d entries", len(skusMap))

	start := time.Now()

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			buffer := make([]marketChan, 0, tcgplayer.MaxIdsInRequest)

			for page := range pages {
				// Add our data to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == cap(buffer) {
					err := tcg.processEntry(channel, buffer)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntry(channel, buffer)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetAllSets()
		total := len(sets) - 1
		i := 1

		var idsFound []int
		for _, code := range sets {
			set, _ := mtgmatcher.GetSet(code)

			switch set.Code {
			case "4EDALT":
				continue
			}

			tcg.printf("Scraping %s (%d/%d)", set.Name, i, total)
			i++

			for _, card := range set.Cards {
				uuid := card.Identifiers["mtgjsonId"]
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
					if !mtgmatcher.Equals(sku.Language, card.Language) {
						// These two sets contain English sku, skip them
						switch set.Code {
						case "LEGITA", "DRKITA":
							continue
						}
						// Otherwise many Japanese and special cards are listed as English, skip anything else
						if sku.Language != "ENGLISH" {
							continue
						}
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
					// Check for tcgplayerProductId due to non-English cards from duplicated sets
					if sku.Finish != "FOIL ETCHED" && card.Identifiers["tcgplayerProductId"] != "" && fmt.Sprint(sku.ProductId) != card.Identifiers["tcgplayerProductId"] {
						continue
					}
					// Skip dupes
					if slices.Contains(idsFound, sku.SkuId) {
						continue
					}
					idsFound = append(idsFound, sku.SkuId)

					pages <- marketChan{
						UUID:      card.UUID,
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

	for result := range channel {
		err := tcg.inventory.AddStrict(result.cardId, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}
	tcg.inventoryDate = time.Now()

	tcg.printf("Took %v", time.Since(start))

	return nil
}

func (tcg *TCGPlayerMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerMarket) MarketNames() []string {
	return availableMarketNames
}

func (tcg *TCGPlayerMarket) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = &tcg.inventoryDate
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
