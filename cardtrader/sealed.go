package cardtrader

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"golang.org/x/exp/slices"
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

	blueprints map[int]*Blueprint
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

func (ct *CardtraderSealed) processSealedProducts(channel chan<- resultChan, uuid string, products []Product, shareCode string, rate float64) error {
	for _, product := range products {
		productId := uuid

		if product.Properties.Language != "en" {
			continue
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

		if product.Properties.Foil {
			blueprint, found := ct.blueprints[product.BlueprintId]
			if !found {
				continue
			}

			if strings.Contains(blueprint.Name, "Secret Lair") {
				var err error
				productId, err = preprocessSealed("SLD", blueprint.Name+" Foil Edition")
				if err != nil {
					ct.printf("No association for '%s'", blueprint.Name)
					continue
				}
			}
		}

		qty := product.Quantity
		if product.Bundle {
			qty *= 4
		}

		link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintId)
		if shareCode != "" {
			link += "?share_code=" + shareCode
		}

		price := float64(product.Price.Cents) / 100
		if product.Price.Currency != "USD" {
			price *= rate
		}

		channel <- resultChan{
			cardId: productId,
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

	return nil
}

func (ct *CardtraderSealed) processEntry(channel chan<- resultChan, expansionId int, expansionName string) error {
	allProducts, err := ct.client.ProductsForExpansion(expansionId)
	if err != nil {
		return err
	}

	var warned []string

	for id, products := range allProducts {
		blueprint, found := ct.blueprints[id]
		if !found {
			continue
		}

		uuid, err := preprocessSealed(expansionName, blueprint.Name)
		if err != nil {
			if err.Error() == "unknown edition" {
				return fmt.Errorf("%s: %s", expansionName, err.Error())
			}
			if err.Error() == "unsupported" {
				continue
			}
			if slices.Contains(warned, blueprint.Name) {
				continue
			}
			warned = append(warned, blueprint.Name)
			ct.printf("No association for '%s' in '%s'", blueprint.Name, expansionName)
			continue
		}

		ct.processSealedProducts(channel, uuid, products, ct.ShareCode, ct.exchangeRate)
	}

	return nil
}

func (ct *CardtraderSealed) scrape() error {
	expansionsRaw, err := ct.client.Expansions()
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d global sets", len(expansionsRaw))

	var blueprintsRaw []Blueprint
	for _, exp := range expansionsRaw {
		if exp.GameId != GameIdMagic {
			continue
		}
		if ct.TargetEdition != "" && exp.Name != ct.TargetEdition {
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

	blueprints, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw, true)
	ct.blueprints = blueprints
	ct.printf("Parsing %d expansions", len(expansions))

	expansionIds := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansionId := range expansionIds {
				err := ct.processEntry(results, expansionId, expansions[expansionId])
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
			if ct.TargetEdition != "" && expName != ct.TargetEdition {
				continue
			}
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
