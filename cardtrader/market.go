package cardtrader

import (
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type CardtraderMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	MaxConcurrency int

	client    *CTAuthClient
	inventory mtgban.InventoryRecord

	blueprints map[int]*Blueprint
}

func NewScraperMarket(token string) *CardtraderMarket {
	ct := CardtraderMarket{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.MaxConcurrency = 1
	ct.client = NewCTAuthClient(token)
	return &ct
}

func (ct *CardtraderMarket) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CTM] "+format, a...)
	}
}

func (ct *CardtraderMarket) processEntry(channel chan<- resultChan, expansionId int) error {
	allProducts, err := ct.client.ProductsForExpansion(expansionId)
	if err != nil {
		return err
	}

	for id, products := range allProducts {
		blueprint, found := ct.blueprints[id]
		if !found {
			ct.printf("blueprint %d not found", id)
			continue
		}

		theCard, err := preprocess(blueprint)
		if err != nil {
			continue
		}

		err = processProducts(channel, theCard, products)
		if err != nil {
			ct.printf("%v", err)
			ct.printf("%q", theCard)
			ct.printf("%d %q", blueprint.Id, blueprint)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ct.printf("- %s", card)
				}
			}
		}
	}

	return nil
}

func formatBlueprints(blueprints []Blueprint, inExpansions []Expansion) (map[int]*Blueprint, map[int]string) {
	// Create a map to be able to retrieve edition name in the blueprint
	formatted := map[int]*Blueprint{}
	expansions := map[int]string{}
	for i := range blueprints {
		if blueprints[i].CategoryId != 1 || blueprints[i].GameId != 1 {
			continue
		}

		// Keep track of blueprints as they are more accurate that the
		// information found in product
		formatted[blueprints[i].Id] = &blueprints[i]

		// Load expansions array
		_, found := expansions[blueprints[i].ExpansionId]
		if !found {
			for j := range inExpansions {
				if inExpansions[j].Id == blueprints[i].ExpansionId {
					expansions[blueprints[i].ExpansionId] = inExpansions[j].Name
				}
			}
		}

		// The name is missing from the blueprints endpoint, fill it with data
		// retrieved from the expansions endpoint
		formatted[blueprints[i].Id].Expansion.Name = expansions[blueprints[i].ExpansionId]

		// Move the blueprint properties from the custom structure from blueprints
		// to the place as expected by preprocess()
		formatted[blueprints[i].Id].Properties = formatted[blueprints[i].Id].FixedProperties
	}

	return formatted, expansions
}

func (ct *CardtraderMarket) scrape() error {
	expansionsRaw, err := ct.client.Expansions()
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d expansions", len(expansionsRaw))

	blueprintsRaw, err := ct.client.Blueprints()
	if err != nil {
		return err
	}
	ct.printf("Found %d blueprints", len(blueprintsRaw))

	blueprints, expansions := formatBlueprints(blueprintsRaw, expansionsRaw)
	ct.blueprints = blueprints
	ct.printf("Parsing %d mtg elements", len(expansions))

	expansionIds := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansionId := range expansionIds {
				err := ct.processEntry(results, expansionId)
				if err != nil {
					ct.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for id, exp := range expansions {
			ct.printf("Processing %s (%d)", exp, id)
			expansionIds <- id
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
			if entry.Conditions == result.invEntry.Conditions {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		err := ct.inventory.Add(result.cardId, result.invEntry)
		if err != nil {
			ct.printf("%s", err.Error())
			continue
		}
	}

	ct.InventoryDate = time.Now()

	return nil
}

func (ct *CardtraderMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(ct.inventory) > 0 {
		return ct.inventory, nil
	}

	err := ct.scrape()
	if err != nil {
		return nil, err
	}

	return ct.inventory, nil
}

func (ct *CardtraderMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CTM"
	info.InventoryTimestamp = ct.InventoryDate
	info.CountryFlag = "🇪🇺"
	return
}
