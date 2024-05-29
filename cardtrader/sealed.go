package cardtrader

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type CardtraderSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	ShareCode      string

	// Only retrieve data from a single edition
	TargetEdition string

	exchangeRate float64
	client       *CTAuthClient

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

func NewScraperSealed(token string) (*CardtraderSealed, error) {
	ct := CardtraderSealed{}
	ct.inventory = mtgban.InventoryRecord{}
	// API is strongly rated limited, hardcode a lower amount
	ct.MaxConcurrency = 2
	ct.client = NewCTAuthClient(token)

	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	ct.exchangeRate = rate
	return &ct, nil
}

func (ct *CardtraderSealed) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CTSealed] "+format, a...)
	}
}

func (ct *CardtraderSealed) processEntry(channel chan<- resultChan, expansionId int, expansionName string, productMap map[int][]string) error {
	allProducts, err := ct.client.ProductsForExpansion(expansionId)
	if err != nil {
		return err
	}

	for _, products := range allProducts {
		for _, product := range products {
			uuids, found := productMap[product.BlueprintId]
			if !found {
				continue
			}

			if product.Properties.Language != "en" {
				continue
			}

			uuid := uuids[0]
			if product.Properties.Foil && len(uuids) > 1 {
				uuid = uuids[1]
			}

			switch {
			case product.Quantity < 1,
				product.OnVacation,
				product.Properties.Altered:
				continue
			case mtgmatcher.Contains(product.Description, "ita"),
				mtgmatcher.Contains(product.Description, "empty box"),
				mtgmatcher.Contains(product.Description, "deck box only"):
				continue
			}

			qty := product.Quantity
			if product.Bundle {
				qty *= 4
			}

			link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintId)
			if ct.ShareCode != "" {
				link += "?share_code=" + ct.ShareCode
			}

			price := float64(product.Price.Cents) / 100
			if product.Price.Currency != "USD" {
				price *= ct.exchangeRate
			}

			channel <- resultChan{
				cardId: uuid,
				invEntry: &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price,
					Quantity:   qty,
					URL:        link,
					SellerName: product.User.Name,
					Bundle:     product.User.SealedZero,
					OriginalId: fmt.Sprint(product.BlueprintId),
					InstanceId: fmt.Sprint(product.Id),
				},
			}
		}
	}

	return nil
}

func (ct *CardtraderSealed) scrape() error {
	productMap := map[int][]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || !co.Sealed {
			continue
		}
		ctId := co.Identifiers["cardtraderId"]

		// Some products do not carry an id because they are already assigned
		// For specific cases, look for them since we have the canonical number
		if ctId == "" && co.SetCode == "SLD" && strings.HasSuffix(co.Name, " Foil") {
			uuids, err := mtgmatcher.SearchSealedEquals(strings.TrimSuffix(co.Name, " Foil"))
			if err != nil {
				continue
			}
			subco, err := mtgmatcher.GetUUID(uuids[0])
			if err != nil {
				continue
			}
			ctId = subco.Identifiers["cardtraderId"]
		}
		cardtraderId, err := strconv.Atoi(ctId)
		if err != nil {
			continue
		}
		// We also know that nonfoil comes before foil since product names are sorted
		// so we can guarantee that the first element is nonfoil, and the second one
		// is actually foil
		productMap[cardtraderId] = append(productMap[cardtraderId], uuid)
	}
	ct.printf("Loaded %d sealed products", len(productMap))

	expansionsRaw, err := ct.client.Expansions()
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d global sets", len(expansionsRaw))

	if ct.TargetEdition != "" {
		ct.printf("-> only targeting edition %s", ct.TargetEdition)
	}

	var blueprintsRaw []Blueprint
	for _, exp := range expansionsRaw {
		if exp.GameId != GameIdMagic {
			continue
		}
		if ct.TargetEdition != "" && exp.Name != ct.TargetEdition && exp.Code != strings.ToLower(ct.TargetEdition) {
			continue
		}

		bp, err := ct.client.Blueprints(exp.Id)
		if err != nil {
			ct.printf("skipping %d %s due to %s", exp.Id, exp.Name, err.Error())
			continue
		}
		blueprintsRaw = append(blueprintsRaw, bp...)
	}
	ct.printf("Found %d blueprints", len(blueprintsRaw))

	_, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw, true)
	ct.printf("Parsing %d expansions", len(expansions))

	expansionIds := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansionId := range expansionIds {
				err := ct.processEntry(results, expansionId, expansions[expansionId], productMap)
				if err != nil {
					ct.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		num := 1
		for id, expName := range expansions {
			ct.printf("Processing %s (%d/%d) [%d]", expName, num, len(expansions), id)
			expansionIds <- id
			num++
		}
		close(expansionIds)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		// Only keep one offer per condition
		skip := false
		entries := ct.inventory[result.cardId]
		for _, entry := range entries {
			if entry.Conditions == result.invEntry.Conditions && entry.Bundle == result.invEntry.Bundle {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Assign a seller name as required by Market
		result.invEntry.SellerName = "Card Trader Sealed"
		if result.invEntry.Bundle {
			result.invEntry.SellerName = "Card Trader Zero Sealed"
		}
		var err error
		err = ct.inventory.Add(result.cardId, result.invEntry)
		if err != nil {
			ct.printf("%s", err.Error())
			continue
		}
	}

	ct.inventoryDate = time.Now()

	return nil
}

func (ct *CardtraderSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(ct.inventory) > 0 {
		return ct.inventory, nil
	}

	err := ct.scrape()
	if err != nil {
		return nil, err
	}

	return ct.inventory, nil
}

var availableMarketNamesSealed = []string{
	"Card Trader Sealed", "Card Trader Zero Sealed",
}

func (tcg *CardtraderSealed) MarketNames() []string {
	return availableMarketNamesSealed
}

func (ct *CardtraderSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader Sealed"
	info.Shorthand = "CTSealed"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.SealedMode = true
	return
}
