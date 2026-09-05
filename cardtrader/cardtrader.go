// Package cardtrader scrapes Card Trader, both the marketplace and sealed
// product, across every game they carry.
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

// Market prices singles from Card Trader, splitting the result into
// the storefronts they sell under: the marketplace itself, Zero, and 1DR.
type Market struct {
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

	gameID int
}

var availableMarketNames = []string{
	"Card Trader", "Card Trader Zero", "Card Trader 1DR",
}

var name2shorthand = map[string]string{
	"Card Trader":      "CT",
	"Card Trader Zero": "CT0",
	"Card Trader 1DR":  "CT1DR",
}

// NewScraperMarket returns a market scraper for one game, authenticated with a
// full API token.
func NewScraperMarket(gameID int, token string) (*Market, error) {
	ct := Market{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.client = NewCTAuthClient(token)
	ct.gameID = gameID
	return &ct, nil
}

func (ct *Market) printf(format string, a ...any) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CT] "+format, a...)
	}
}

type resultChan struct {
	cardID   string
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

func (ct *Market) processProducts(channel chan<- resultChan, bpID int, products []Product) {
	blueprint, found := ct.blueprints[bpID]
	if !found {
		return
	}

	var theCard *mtgmatcher.InputCard
	if ct.gameID == GameMagic {
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
		switch ct.gameID {
		case GameMagic:
			lang := product.Properties.MTGLanguage
			if lang != "" {
				lang, found = langMap[strings.ToLower(lang)]
				if !found {
					ct.printf("unsupported '%s' language", product.Properties.MTGLanguage)
					ct.printf("%s %+v", theCard, product)
					continue
				}
				theCard.Language = lang
			}
		case GameLorcana, GameRiftbound, GameOnePiece, GameYuGiOh, GameFleshAndBlood,
			GamePokemon, GameGundam, GamePalworld:
			if gameLanguage(ct.gameID, product) != "en" {
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
				Name:      gameName(ct.gameID, blueprint),
				Edition:   blueprint.Expansion.Name,
				Variation: gameVariation(ct.gameID, blueprint, number),
				// A listing is one printing in one finish, and the games
				// whose properties name it say which; the flag beside it
				// keeps carrying the plain foilness like everywhere else.
				Finish: gameFinish(ct.gameID, product),
				Foil:   gameFoil(ct.gameID, product),
			}
		default:
			ct.printf("unsupported game %d", ct.gameID)
			return
		}

		// The blueprint carries the TCGplayer product id, which names the
		// printing outright instead of inferring it. It is worth trying
		// first: for Lorcana it resolves cards the name and number cannot,
		// and for the Riftbound promos it lands on the promo printing where
		// the edition alone leaves the base one. Magic keeps its own
		// preprocessing, and a blueprint without an id falls through.
		var cardID string
		if ct.gameID != GameMagic && blueprint.TCGplayerID != 0 {
			// A named finish reaches the sibling the flag cannot: the flag
			// has one bit and lands on the product's foil default, where the
			// name says which of its treatments the listing prices. A name
			// the product is sold in no printing of falls back to the flag,
			// which answers with the default rather than nothing.
			if theCard.Finish != "" {
				cardID, _ = mtgmatcher.MatchIDFinish(fmt.Sprint(blueprint.TCGplayerID), theCard.Finish)
			}
			if cardID == "" {
				cardID, _ = mtgmatcher.MatchID(fmt.Sprint(blueprint.TCGplayerID), theCard.Foil)
			}
		}

		if cardID == "" {
			var err error
			cardID, err = mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				ct.printf("%v", err)
				ct.printf("%q", theCard)
				ct.printf("%d %+v", bpID, blueprint)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					for _, probe := range alias.Probe() {
						co, _ := mtgmatcher.GetUUID(probe)
						ct.printf("- %s", co)
					}
				}
				continue
			}
			// A promotional shelf sells no ordinary card, so an answer
			// carrying no promotional label is the number having spoken
			// alone; promoShelfNeedsLabel says when that is worth refusing,
			// and today only Gundam ever says so.
			if promoShelfNeedsLabel(ct.gameID, blueprint) {
				co, err := mtgmatcher.GetUUID(cardID)
				if err != nil || len(co.PromoTypes) == 0 {
					continue
				}
			}
		}

		// Magic only: the other games carry the finish on the input already.
		if ct.gameID == GameMagic && product.Properties.MTGFoil {
			cardID = foilPrintingID(cardID, theCard.Name)
		}

		qty := product.Quantity
		if product.Bundle {
			qty *= 4
		}

		link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintID)
		if ct.ShareCode != "" {
			link += "?share_code=" + ct.ShareCode
		}

		price, err := priceToUSD(product.Price.Cents, product.Price.Currency, ct.exchangeRates)
		if err != nil {
			ct.printf("%v for blueprint %d", err, product.BlueprintID)
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
			cardID: cardID,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      price,
				Quantity:   qty,
				URL:        link,
				SellerName: sellerName,
				Bundle:     product.User.SinglesZero,
				OriginalID: fmt.Sprint(product.BlueprintID),
				InstanceID: fmt.Sprint(product.ID),
				CustomFields: map[string]string{
					"SubSellerName": product.User.Name,
					"SubSellerGeo":  product.User.CountryCode,
				},
			},
		}
	}
}

func (ct *Market) processExpansion(ctx context.Context, channel chan<- resultChan, expansionID int) error {
	allProducts, err := ct.client.ProductsForExpansion(ctx, expansionID)
	if err != nil {
		return err
	}

	for id, products := range allProducts {
		ct.processProducts(channel, id, products)
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (ct *Market) Load(ctx context.Context) error {
	rates, err := mtgban.GetExchangeRates(ctx)
	if err != nil {
		return err
	}
	ct.exchangeRates = rates

	if ct.TargetEdition != "" {
		ct.printf("-> only targeting edition %s", ct.TargetEdition)
	}
	blueprintsRaw, expansionsRaw, err := BlueprintsForGame(ctx, ct.client, ct.gameID, ct.TargetEdition, ct.printf)
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d global sets", len(expansionsRaw))
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
			entries := ct.inventory[result.cardID]
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
				err = ct.inventory.AddRelaxed(result.cardID, result.invEntry)
			} else {
				err = ct.inventory.Add(result.cardID, result.invEntry)
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

// Inventory returns what Load collected. See mtgban.Seller.
func (ct *Market) Inventory() mtgban.InventoryRecord {
	return ct.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (ct *Market) MarketNames() []string {
	return availableMarketNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (ct *Market) InfoForScraper(name string) mtgban.ScraperInfo {
	info := ct.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (ct *Market) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CTMarket"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.Family = "CT"
	switch ct.gameID {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameYuGiOh:
		info.Game = mtgban.GameYuGiOh
	case GameFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	case GamePokemon:
		info.Game = mtgban.GamePokemon
	case GameGundam:
		info.Game = mtgban.GameGundam
	case GamePalworld:
		info.Game = mtgban.GamePalworld
	}
	return
}
