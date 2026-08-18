package cardtrader

import (
	"context"
	"fmt"
	"strings"
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

	exchangeRates map[string]float64
	client        *CTAuthClient

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	gameID        int
}

func NewScraperSealed(gameID int, token string) (*CardtraderSealed, error) {
	// An unknown game would not error anywhere later: its listings would
	// simply all fail the language read and the scraper would run empty.
	switch gameID {
	case GameIdMagic, GameIdLorcana, GameIdRiftbound, GameIdOnePiece:
	default:
		return nil, fmt.Errorf("unsupported game %d", gameID)
	}
	ct := CardtraderSealed{}
	ct.inventory = mtgban.InventoryRecord{}
	// API is strongly rated limited, hardcode a lower amount
	ct.MaxConcurrency = 2
	ct.client = NewCTAuthClient(token)
	ct.gameID = gameID
	return &ct, nil
}

func (ct *CardtraderSealed) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CTSealed] "+format, a...)
	}
}

func (ct *CardtraderSealed) processEntry(ctx context.Context, channel chan<- resultChan, expansionID int, expansionName string, productMap map[int][]string) error {
	allProducts, err := ct.client.ProductsForExpansion(ctx, expansionID)
	if err != nil {
		return err
	}

	for _, products := range allProducts {
		for _, product := range products {
			uuids, found := productMap[product.BlueprintId]
			if !found {
				continue
			}

			if gameLanguage(ct.gameID, product) != "en" {
				continue
			}

			uuid := uuids[0]
			if gameFoil(ct.gameID, product) && len(uuids) > 1 {
				uuid = uuids[1]
			}

			switch {
			case product.Quantity < 1,
				product.OnVacation,
				product.Bundle,
				product.Properties.Altered:
				continue
			case mtgmatcher.Contains(product.Description, "ita"),
				mtgmatcher.Contains(product.Description, "empty booster"),
				mtgmatcher.Contains(product.Description, "empty box"),
				mtgmatcher.Contains(product.Description, "opened"),
				mtgmatcher.Contains(product.Description, "only cards"),
				mtgmatcher.Contains(product.Description, "missing"),
				mtgmatcher.Contains(product.Description, "deck box only"):
				continue
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
			if product.User.SealedZero {
				sellerName = availableMarketNames[1]
				if strings.Contains(strings.ToLower(product.User.Name), "day ready") {
					sellerName = availableMarketNames[2]
				}
			}

			channel <- resultChan{
				cardID: uuid,
				invEntry: &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price,
					Quantity:   product.Quantity,
					URL:        link,
					SellerName: sellerName,
					Bundle:     product.User.SealedZero,
					OriginalId: fmt.Sprint(product.BlueprintId),
					InstanceId: fmt.Sprint(product.Id),
					CustomFields: map[string]string{
						"SubSellerName": product.User.Name,
						"SubSellerGeo":  product.User.CountryCode,
					},
				},
			}
		}
	}

	return nil
}

func (ct *CardtraderSealed) Load(ctx context.Context) error {
	rates, err := exchangeRates(ctx)
	if err != nil {
		return err
	}
	ct.exchangeRates = rates

	productMap := mtgmatcher.BuildSealedProductMap("cardtraderId")
	ct.printf("Loaded %d sealed products", len(productMap))

	if ct.TargetEdition != "" {
		ct.printf("-> only targeting edition %s", ct.TargetEdition)
	}
	blueprintsRaw, expansionsRaw, err := BlueprintsForGame(ctx, ct.client, ct.gameID, ct.TargetEdition, ct.printf)
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d global sets", len(expansionsRaw))
	ct.printf("Found %d blueprints", len(blueprintsRaw))

	blueprints, expansions := FormatBlueprints(blueprintsRaw, expansionsRaw, true)
	ct.printf("Parsing %d expansions", len(expansions))

	// A datastore that does not catalog cardtrader's own ids (riftbound)
	// still names every sealed product by its TCGplayer id; route the
	// blueprints through that to the same blueprintId-to-uuids shape.
	if len(productMap) == 0 {
		tcgMap := mtgmatcher.BuildSealedProductMap("tcgplayerProductId")
		for id, bp := range blueprints {
			// An unlinked blueprint carries a zero id; the singles path
			// skips those, and looking one up would funnel every such
			// listing onto whatever product shares the missing link.
			if bp.TCGplayerId == 0 {
				continue
			}
			uuids, found := tcgMap[bp.TCGplayerId]
			if !found {
				continue
			}
			productMap[id] = uuids
		}
		ct.printf("Mapped %d sealed products through the TCGplayer id", len(productMap))
	}

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
			return ct.processEntry(ctx, results, item.id, item.name, productMap)
		},
		func(result resultChan) {
			// Only keep one offer per condition
			skip := false
			entries := ct.inventory[result.cardID]
			for _, entry := range entries {
				if entry.Conditions == result.invEntry.Conditions && entry.Bundle == result.invEntry.Bundle {
					skip = true
					break
				}
			}
			if skip {
				return
			}

			err := ct.inventory.Add(result.cardID, result.invEntry)
			if err != nil {
				ct.printf("%s", err.Error())
			}
		},
		ct.printf,
	)

	ct.inventoryDate = time.Now()

	return nil
}

func (ct *CardtraderSealed) Inventory() mtgban.InventoryRecord {
	return ct.inventory
}

func (ct *CardtraderSealed) MarketNames() []string {
	// Riftbound has no sealed 1DR listings, and an always-empty seller
	// reads as a broken scrape downstream, failing the run.
	if ct.gameID == GameIdRiftbound {
		return availableMarketNames[:2]
	}
	return availableMarketNames
}

var name2shorthandSealed = map[string]string{
	"Card Trader":      "CTSealed",
	"Card Trader Zero": "CT0Sealed",
	"Card Trader 1DR":  "CT1DRSealed",
}

func (ct *CardtraderSealed) InfoForScraper(name string) mtgban.ScraperInfo {
	info := ct.Info()
	info.Name = name
	info.Shorthand = name2shorthandSealed[name]
	return info
}

func (ct *CardtraderSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader Sealed"
	info.Shorthand = "CTSealedWrapper"
	info.InventoryTimestamp = &ct.inventoryDate
	info.CountryFlag = "EU"
	info.SealedMode = true
	switch ct.gameID {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	case GameIdRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameIdOnePiece:
		info.Game = mtgban.GameOnePiece
	}
	return
}
