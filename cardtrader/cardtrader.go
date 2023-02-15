package cardtrader

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type CardtraderMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int
	ShareCode      string

	// Only retrieve data from a single edition
	TargetEdition string

	// Keep same-conditions entries
	KeepDuplicates bool

	exchangeRate float64
	client       *CTAuthClient
	loggedClient *CTLoggedClient

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	blueprints map[int]*Blueprint
}

var availableMarketNames = []string{
	"Card Trader", "Card Trader Zero",
}

func NewScraperMarket(token string) (*CardtraderMarket, error) {
	ct := CardtraderMarket{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.marketplace = map[string]mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.client = NewCTAuthClient(token)

	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	ct.exchangeRate = rate
	return &ct, nil
}

func (ct *CardtraderMarket) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CT] "+format, a...)
	}
}

type resultChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

var condMap = map[string]string{
	"":                  "NM",
	"Mint":              "NM",
	"Near Mint":         "NM",
	"Slightly Played":   "SP",
	"Moderately Played": "MP",
	"Played":            "HP",
	"Heavily Played":    "HP",
	"Poor":              "PO",
}

func processProducts(channel chan<- resultChan, theCard *mtgmatcher.Card, products []Product, shareCode string, rate float64) error {
	var cardId string
	var cardIdFoil string

	for _, product := range products {
		if mtgmatcher.SkipLanguage(theCard.Name, product.Expansion.Name, product.Properties.Language) {
			continue
		}

		if product.Quantity < 1 || product.OnVacation || product.Properties.Altered {
			continue
		}

		switch {
		case mtgmatcher.Contains(product.Description, "ita"),
			mtgmatcher.Contains(product.Description, "signed"),
			mtgmatcher.Contains(product.Description, "inked"),
			mtgmatcher.Contains(product.Description, "stamp"),
			mtgmatcher.Contains(product.Description, "mix"):
			continue
		}

		var err error
		if cardId == "" {
			cardId, err = mtgmatcher.Match(theCard)
			if err != nil {
				return err
			}
		}

		if cardIdFoil == "" && product.Properties.Foil && mtgmatcher.HasFoilPrinting(theCard.Name) {
			// The function retuns empty string on error
			cardIdFoil, _ = mtgmatcher.MatchId(cardId, true)
		}

		cond := product.Properties.Condition
		if product.Properties.Signed {
			cond = "Heavily Played"
		}
		if product.Properties.Altered {
			continue
		}
		conditions, found := condMap[cond]
		if !found {
			return fmt.Errorf("Unsupported %s condition", cond)
		}

		finalCardId := cardId
		if product.Properties.Foil && cardIdFoil != "" {
			finalCardId = cardIdFoil
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
			cardId: finalCardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      price,
				Quantity:   qty,
				URL:        link,
				SellerName: product.User.Name,
				Bundle:     product.User.Zero,
				OriginalId: fmt.Sprint(product.BlueprintId),
				InstanceId: fmt.Sprint(product.Id),
			},
		}
	}

	return nil
}

func (ct *CardtraderMarket) processEntry(channel chan<- resultChan, expansionId int) error {
	allProducts, err := ct.client.ProductsForExpansion(expansionId)
	if err != nil {
		return err
	}

	for id, products := range allProducts {
		blueprint, found := ct.blueprints[id]
		if !found {
			continue
		}

		theCard, err := Preprocess(blueprint)
		if err != nil {
			continue
		}

		err = processProducts(channel, theCard, products, ct.ShareCode, ct.exchangeRate)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			ct.printf("%v", err)
			ct.printf("%q", theCard)
			ct.printf("%d %q", blueprint.Id, blueprint)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
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

func FormatBlueprints(blueprints []Blueprint, inExpansions []Expansion) (map[int]*Blueprint, map[int]string) {
	// Create a map to be able to retrieve edition name in the blueprint
	formatted := map[int]*Blueprint{}
	expansions := map[int]string{}
	for i := range blueprints {
		switch blueprints[i].GameId {
		case GameIdMagic:
		default:
			continue
		}
		switch blueprints[i].CategoryId {
		case CategoryMagicSingles, CategoryMagicTokens, CategoryMagicOversized:
		default:
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
		// to the place as expected by Preprocess()
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

	blueprints, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw)
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
		for id, expName := range expansions {
			if ct.TargetEdition != "" && expName != ct.TargetEdition {
				continue
			}
			ct.printf("Processing %s (%d)", expName, id)
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
			if entry.Conditions == result.invEntry.Conditions && entry.Bundle == result.invEntry.Bundle {
				skip = true
				break
			}
		}
		if skip && !ct.KeepDuplicates {
			continue
		}

		// Assign a seller name as required by Market
		result.invEntry.SellerName = "Card Trader"
		if result.invEntry.Bundle {
			result.invEntry.SellerName = "Card Trader Zero"
		}
		var err error
		if ct.KeepDuplicates {
			err = ct.inventory.AddRelaxed(result.cardId, result.invEntry)
		} else {
			err = ct.inventory.Add(result.cardId, result.invEntry)
		}
		if err != nil {
			ct.printf("%s", err.Error())
			continue
		}
	}

	// Sort to keep NM-SP-MP-HP-PO order
	conds := map[string]int{"NM": 0, "SP": 1, "MP": 2, "HP": 3, "PO": 4}
	for cardId := range ct.inventory {
		sort.Slice(ct.inventory[cardId], func(i, j int) bool {
			return conds[ct.inventory[cardId][i].Conditions] < conds[ct.inventory[cardId][j].Conditions]
		})
	}

	ct.inventoryDate = time.Now()

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

func (ct *CardtraderMarket) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(ct.inventory) == 0 {
		_, err := ct.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := ct.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range ct.inventory {
		for i := range ct.inventory[card] {
			if ct.inventory[card][i].SellerName == sellerName {
				if ct.inventory[card][i].Price == 0 {
					continue
				}
				if ct.marketplace[sellerName] == nil {
					ct.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				ct.marketplace[sellerName][card] = append(ct.marketplace[sellerName][card], ct.inventory[card][i])
			}
		}
	}

	if len(ct.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return ct.marketplace[sellerName], nil
}

func (ct *CardtraderMarket) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	ct.marketplace = market
	ct.inventory = inventory

	ct.printf("Loaded inventory from file")

	return nil
}

func (tcg *CardtraderMarket) MarketNames() []string {
	return availableMarketNames
}

func (ct *CardtraderMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CT"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	return
}
