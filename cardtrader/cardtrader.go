package cardtrader

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	exchangeRates map[string]float64
	client        *CTAuthClient

	inventory mtgban.InventoryRecord

	blueprints map[int]*Blueprint

	gameId int
}

var availableMarketNames = []string{
	"Card Trader", "Card Trader Zero", "Card Trader 1DR",
}

var name2shorthand = map[string]string{
	"Card Trader":      "CT",
	"Card Trader Zero": "CT0",
	"Card Trader 1DR":  "CT1DR",
}

func NewScraperMarket(gameId int, token string) (*CardtraderMarket, error) {
	ct := CardtraderMarket{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.client = NewCTAuthClient(token)
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

		// Build the per-game input card; the match and error handling below are
		// shared. Magic reuses the blueprint-derived theCard (applying the
		// product language), Lorcana builds one from the product's number.
		switch ct.gameId {
		case GameIdMagic:
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
		case GameIdLorcana:
			if product.Properties.LorcanaLanguage != "en" {
				continue
			}
			theCard = &mtgmatcher.InputCard{
				Name:      blueprint.Name,
				Edition:   blueprint.Expansion.Name,
				Variation: product.Properties.Number,
				Foil:      product.Properties.LorcanaFoil,
			}
		case GameIdRiftbound:
			if product.Properties.RiftboundLanguage != "en" {
				continue
			}
			// A listing copies the collector number when it is created and
			// never refreshes it, so a blueprint corrected later leaves its
			// older listings quoting a number that now belongs to a
			// different card. The blueprint is the authoritative one.
			number := blueprint.Properties.Number
			if number == "" {
				number = product.Properties.Number
			}
			theCard = &mtgmatcher.InputCard{
				Name:      blueprint.Name,
				Edition:   blueprint.Expansion.Name,
				Variation: number,
				Foil:      product.Properties.RiftboundFoil,
			}
		default:
			ct.printf("unsupported game %d", ct.gameId)
			return
		}

		// The blueprint carries the TCGplayer product id, which names the
		// printing outright instead of inferring it. It is worth trying
		// first: for Lorcana it resolves cards the name and number cannot,
		// and for the Riftbound promos it lands on the promo printing where
		// the edition alone leaves the base one. Magic keeps its own
		// preprocessing, and a blueprint without an id falls through.
		var cardId string
		if ct.gameId != GameIdMagic && blueprint.TCGplayerId != 0 {
			cardId, _ = mtgmatcher.MatchId(fmt.Sprint(blueprint.TCGplayerId), theCard.Foil)
		}

		if cardId == "" {
			var err error
			cardId, err = mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				ct.printf("%v", err)
				ct.printf("%q", theCard)
				ct.printf("%d %q", bpId, blueprint)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					for _, probe := range alias.Probe() {
						co, _ := mtgmatcher.GetUUID(probe)
						ct.printf("- %s", co)
					}
				}
				continue
			}
		}

		// Foil listings share the plain id; adopt the foil id when one exists
		// (Magic only: Lorcana's finish is already carried on the input).
		if ct.gameId == GameIdMagic && product.Properties.MTGFoil && mtgmatcher.HasFoilPrinting(theCard.Name) {
			if cardIdFoil, e := mtgmatcher.MatchId(cardId, true); e == nil {
				cardId = cardIdFoil
			}
		}

		qty := product.Quantity
		if product.Bundle {
			qty *= 4
		}

		link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintId)
		if ct.ShareCode != "" {
			link += "?share_code=" + ct.ShareCode
		}

		price, err := priceToUSD(product.Price.Cents, product.Price.Currency, ct.exchangeRates)
		if err != nil {
			ct.printf("%v for blueprint %d", err, product.BlueprintId)
			continue
		}

		// Assign a seller name as required by Market
		sellerName := availableMarketNames[0]
		if product.User.SinglesZero {
			sellerName = availableMarketNames[1]
			if strings.Contains(strings.ToLower(product.User.Name), "day ready") {
				sellerName = availableMarketNames[2]
			}
		} else {
			// Skip non-professional users with excessive shipping options due to geo
			if product.User.UserType == "normal" {
				switch product.User.CountryCode {
				case "GB", "ES", "CH":
					continue
				}
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
					"SubSellerGeo":  product.User.CountryCode,
				},
			},
		}
	}

	return
}

func (ct *CardtraderMarket) processExpansion(ctx context.Context, channel chan<- resultChan, expansionId int) error {
	allProducts, err := ct.client.ProductsForExpansion(ctx, expansionId)
	if err != nil {
		return err
	}

	for id, products := range allProducts {
		ct.processProducts(channel, id, products)
	}

	return nil
}

func (ct *CardtraderMarket) Load(ctx context.Context) error {
	rates, err := exchangeRates(ctx)
	if err != nil {
		return err
	}
	ct.exchangeRates = rates

	expansionsRaw, err := ct.client.Expansions(ctx)
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

		bp, err := ct.client.Blueprints(ctx, exp.Id)
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

	type expItem struct {
		id   int
		name string
	}
	expItems := make([]expItem, 0, len(expansions))
	for id, name := range expansions {
		expItems = append(expItems, expItem{id, name})
	}

	mtgban.WorkerPool(ctx, ct.MaxConcurrency, expItems,
		func(ctx context.Context, item expItem, results chan<- resultChan) error {
			ct.printf("Processing %s [%d]", item.name, item.id)
			return ct.processExpansion(ctx, results, item.id)
		},
		func(result resultChan) {
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
				return
			}

			var err error
			if ct.KeepDuplicates {
				err = ct.inventory.AddRelaxed(result.cardId, result.invEntry)
			} else {
				err = ct.inventory.Add(result.cardId, result.invEntry)
			}
			if err != nil {
				ct.printf("%s", err.Error())
			}
		},
		ct.printf,
	)

	ct.inventoryDate = time.Now()

	return nil
}

func (ct *CardtraderMarket) Inventory() mtgban.InventoryRecord {
	return ct.inventory
}

func (tcg *CardtraderMarket) MarketNames() []string {
	return availableMarketNames
}

func (ct *CardtraderMarket) InfoForScraper(name string) mtgban.ScraperInfo {
	info := ct.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

func (ct *CardtraderMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CTMarket"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.Family = "CT"
	switch ct.gameId {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	case GameIdRiftbound:
		info.Game = mtgban.GameRiftbound
	}
	return
}
