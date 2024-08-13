package cardtrader

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
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

	inventory mtgban.InventoryRecord

	blueprints map[int]*Blueprint

	gameId int
}

var availableMarketNames = []string{
	"Card Trader", "Card Trader Zero", "Card Trader 1DR",
}

func NewScraperMarket(gameId int, token string) (*CardtraderMarket, error) {
	ct := CardtraderMarket{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.client = NewCTAuthClient(token)

	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	ct.exchangeRate = rate

	ct.gameId = gameId
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

var langMap = map[string]string{
	"en":    "",
	"fr":    "French",
	"de":    "German",
	"es":    "Spanish",
	"it":    "Italian",
	"jp":    "Japanese",
	"kr":    "Korean",
	"pt":    "Portuguese",
	"ru":    "Russian",
	"zh-cn": "Chinese",
	"zh-tw": "Chinese",
}

func (ct *CardtraderMarket) processProducts(channel chan<- resultChan, bpId int, products []Product) {
	blueprint, found := ct.blueprints[bpId]
	if !found {
		return
	}

	var theCard *mtgmatcher.InputCard
	if ct.gameId == GameIdMagic {
		var err error
		theCard, err = Preprocess(blueprint)
		if err != nil {
			return
		}
	}

	for _, product := range products {
		switch {
		case product.Quantity < 1,
			product.OnVacation,
			product.Properties.Altered:
			continue
		case mtgmatcher.Contains(product.Description, "ita"),
			mtgmatcher.Contains(product.Description, "mix"):
			continue
		}

		cond := product.Properties.Condition
		if product.Properties.Signed ||
			mtgmatcher.Contains(product.Description, "signed") ||
			mtgmatcher.Contains(product.Description, "inked") ||
			mtgmatcher.Contains(product.Description, "stamp") ||
			mtgmatcher.Contains(product.Description, "poor") ||
			mtgmatcher.Contains(product.Description, "water") {
			cond = "Poor"
		}

		conditions, found := condMap[cond]
		if !found {
			ct.printf("unsupported %s condition", cond)
			continue
		}

		var cardId string
		var err error

		if ct.gameId == GameIdMagic {
			lang := product.Properties.MTGLanguage
			if lang != "" {
				lang, found = langMap[strings.ToLower(lang)]
				if !found {
					ct.printf("unsupported '%s' language", product.Properties.MTGLanguage)
					ct.printf("%s '%q'", theCard, product)
					continue
				}
				theCard.Language = lang
			}

			cardId, err = mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				ct.printf("%v", err)
				ct.printf("%q", theCard)
				ct.printf("%d %q", bpId, blueprint)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						ct.printf("- %s", card)
					}
				}
				break
			}

			if product.Properties.MTGFoil && mtgmatcher.HasFoilPrinting(theCard.Name) {
				cardIdFoil, err := mtgmatcher.MatchId(cardId, true)
				if err == nil {
					cardId = cardIdFoil
				}
			}
		} else if ct.gameId == GameIdLorcana {
			if product.Properties.LorcanaLanguage != "en" {
				continue
			}

			cardName := blueprint.Name
			collectorNumber := strings.TrimLeft(product.Properties.Number, "0")

			cardId, err = mtgmatcher.SimpleSearch(cardName, collectorNumber, product.Properties.LorcanaFoil)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				ct.printf("%v", err)
				ct.printf("%+v", blueprint)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					ct.printf("%s got ids: %s", cardName, probes)
					for _, probe := range probes {
						co, _ := mtgmatcher.GetUUID(probe)
						ct.printf("%s: %s", probe, co)
					}
				}
				continue
			}
		} else {
			ct.printf("unsupported game %d", ct.gameId)
			return
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

		// Assign a seller name as required by Market
		sellerName := availableMarketNames[0]
		if product.User.SinglesZero {
			sellerName = availableMarketNames[1]
			if strings.Contains(strings.ToLower(product.User.Name), "day ready") {
				sellerName = availableMarketNames[2]
			}
		}

		channel <- resultChan{
			cardId: cardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      price,
				Quantity:   qty,
				URL:        link,
				SellerName: sellerName,
				Bundle:     product.User.SinglesZero,
				OriginalId: fmt.Sprint(product.BlueprintId),
				InstanceId: fmt.Sprint(product.Id),
				CustomFields: map[string]string{
					"SubSellerName": product.User.Name,
				},
			},
		}
	}

	return
}

func (ct *CardtraderMarket) processExpansion(channel chan<- resultChan, expansionId int) error {
	allProducts, err := ct.client.ProductsForExpansion(expansionId)
	if err != nil {
		return err
	}

	for id, products := range allProducts {
		ct.processProducts(channel, id, products)
	}

	return nil
}

func (ct *CardtraderMarket) scrape() error {
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
		if exp.GameId != ct.gameId {
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

	blueprints, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw, false)
	ct.blueprints = blueprints
	ct.printf("Parsing %d expansions with %d blueprints", len(expansions), len(blueprints))

	expansionIds := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansionId := range expansionIds {
				err := ct.processExpansion(results, expansionId)
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
			if entry.Conditions == result.invEntry.Conditions && entry.SellerName == result.invEntry.SellerName {
				skip = true
				break
			}
		}
		if skip && !ct.KeepDuplicates {
			continue
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

func (tcg *CardtraderMarket) MarketNames() []string {
	return availableMarketNames
}

func (ct *CardtraderMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CT"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.Family = "CT"
	switch ct.gameId {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
