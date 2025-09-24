package cardmarket

import (
	"context"
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

type responseChan struct {
	ogId   int
	cardId string
	entry  mtgban.InventoryEntry
}

type CardMarketIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int
	exchangeRate   float64

	// Optional field to select a single edition to go through
	TargetEdition string

	inventory mtgban.InventoryRecord

	priceGuide []PriceGuide

	client *MKMClient
	gameId int
}

var availableIndexNames = []string{
	"MKM Low", "MKM Trend",
}

var name2shorthand = map[string]string{
	"MKM Low":   "MKMLow",
	"MKM Trend": "MKMTrend",
}

func (mkm *CardMarketIndex) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

func NewScraperIndex(gameId int, appToken, appSecret string) (*CardMarketIndex, error) {
	mkm := CardMarketIndex{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	mkm.gameId = gameId
	return &mkm, nil
}

func (mkm *CardMarketIndex) processEdition(ctx context.Context, channel chan<- responseChan, idExpansion int) error {
	products, err := mkm.client.MKMProductsInExpansion(ctx, idExpansion)
	if err != nil {
		return err
	}

	for _, product := range products {
		err := mkm.processProduct(channel, &product)
		if err != nil {
			mkm.printf("product id %d returned %s", product.IdProduct, err)
		}
	}
	return nil
}

func (mkm *CardMarketIndex) processProduct(channel chan<- responseChan, product *MKMProduct) error {
	var cardId string
	var cardIdFoil string
	var err error

	switch mkm.gameId {
	case GameIdMagic:
		backupCardId, backupFoilCardId := fallback(product)

		theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
		if err != nil {
			_, ok := err.(*PreprocessError)
			if ok {
				return err
			}
			if backupCardId == "" && backupFoilCardId == "" {
				return nil
			}

			theCard = &mtgmatcher.InputCard{
				Id: backupCardId,
			}
			if backupCardId == "" {
				theCard.Id = backupFoilCardId
				theCard.Foil = true
			}
		}

		cardId, err = mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			if mtgmatcher.IsToken(theCard.Name) ||
				theCard.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(theCard.Edition, "World Championship Decks") {
				return nil
			}

			if backupCardId != "" || backupFoilCardId != "" {
				cardId = backupCardId
				cardIdFoil = backupFoilCardId
				break
			}

			mkm.printf("%v", err)
			mkm.printf("%q", theCard)
			mkm.printf("%v | %v | %v ", product.Name, product.ExpansionName, product.Number)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("- %s", card)
				}
			}
			return err
		}

		cardIdFoil, _ = mtgmatcher.MatchId(cardId, true)
	case GameIdLorcana:
		cardName := strings.Split(product.Name, " (V.")[0]
		number := product.Number

		cardId, err = mtgmatcher.SimpleSearch(cardName, number, false)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			mkm.printf("%v", err)
			mkm.printf("%+v", product)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				mkm.printf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("%s: %s", probe, co)
				}
			}
			return err
		}
		cardIdFoil, _ = mtgmatcher.SimpleSearch(cardName, number, true)
		if cardId == "" {
			cardId = cardIdFoil
		}

		if cardId == "" {
			return nil
		}
	default:
		return errors.New("unsupported game")
	}

	// Look for the price presence
	var index int
	var found bool
	for index = range mkm.priceGuide {
		if mkm.priceGuide[index].IdProduct == product.IdProduct {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("IdProduct %d not found in PriceGuide", product.IdProduct)
	}

	// Sorted as availableIndexNames
	prices := []float64{
		mkm.priceGuide[index].LowPrice, mkm.priceGuide[index].TrendPrice,
	}
	foilprices := []float64{
		mkm.priceGuide[index].FoilLowPrice, mkm.priceGuide[index].FoilTrendPrice,
	}

	co, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return err
	}

	// If card is not foil, add prices from the prices array, then check
	// if there is a foil printing, and add prices from the foilprices array.
	// If a card is foil-only or is etched, then we just use foilprices data.
	if !co.Foil && !co.Etched {
		link := BuildURL(product.IdProduct, mkm.gameId, mkm.Affiliate, false)

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}

			out := responseChan{
				ogId:   product.IdProduct,
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i] * mkm.exchangeRate,
					Quantity:   product.CountArticles - product.CountFoils,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}

		if foilprices[0] != 0 || foilprices[1] != 0 {
			link := BuildURL(product.IdProduct, mkm.gameId, mkm.Affiliate, true)

			// If the id is the same it means that the card was really nonfoil-only
			if cardId != cardIdFoil {
				for i := range availableIndexNames {
					if foilprices[i] == 0 {
						continue
					}
					out := responseChan{
						ogId:   product.IdProduct,
						cardId: cardIdFoil,
						entry: mtgban.InventoryEntry{
							Conditions: "NM",
							Price:      foilprices[i] * mkm.exchangeRate,
							Quantity:   product.CountFoils,
							URL:        link,
							SellerName: availableIndexNames[i],
							OriginalId: fmt.Sprint(product.IdProduct),
						},
					}

					channel <- out
				}
			}
		}
	} else {
		link := BuildURL(product.IdProduct, mkm.gameId, mkm.Affiliate, true)

		for i := range availableIndexNames {
			if foilprices[i] == 0 || product.CountFoils == 0 {
				continue
			}
			out := responseChan{
				ogId:   product.IdProduct,
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      foilprices[i] * mkm.exchangeRate,
					Quantity:   product.CountFoils,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (mkm *CardMarketIndex) scrape(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	priceGuide, err := GetPriceGuide(ctx, mkm.gameId)
	if err != nil {
		return err
	}
	mkm.priceGuide = priceGuide

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	list, err := mkm.client.Expansions(ctx, mkm.gameId)
	if err != nil {
		return err
	}
	list = FilterAndSortExpansions(list)

	mkm.printf("Parsing %d expansion ids", len(list))

	expansions := make(chan int)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for i := range expansions {
				err := mkm.processEdition(ctx, channel, list[i].IdExpansion)
				if err != nil {
					mkm.printf("expansion %s (id %d) returned %s", list[i].Name, list[i].IdExpansion, err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := range list {
			if mkm.TargetEdition != "" && mkm.TargetEdition != list[i].Name {
				continue
			}
			mkm.printf("Processing %s (%d) %d/%d", list[i].Name, list[i].IdExpansion, i+1, len(list))
			expansions <- i
		}
		close(expansions)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := mkm.inventory.AddStrict(result.cardId, &result.entry)
		if err != nil {
			card, cerr := mtgmatcher.GetUUID(result.cardId)
			if cerr != nil {
				mkm.printf("%d - %s: %s", result.ogId, cerr.Error(), result.cardId)
				continue
			}
			// Skip too many errors
			if mtgmatcher.IsToken(card.Name) ||
				card.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(card.Edition, "World Championship Decks") {
				continue
			}
			mkm.printf("%d - %s", result.ogId, err.Error())
			continue
		}
	}

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.inventoryDate = time.Now()
	return nil
}

func (mkm *CardMarketIndex) Inventory() (mtgban.InventoryRecord, error) {
	if len(mkm.inventory) > 0 {
		return mkm.inventory, nil
	}

	err := mkm.scrape(context.TODO())
	if err != nil {
		return nil, err
	}

	return mkm.inventory, nil
}

func (mkm *CardMarketIndex) MarketNames() []string {
	return availableIndexNames
}

func (mkm *CardMarketIndex) InfoForScraper(name string) mtgban.ScraperInfo {
	info := mkm.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

func (mkm *CardMarketIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.MetadataOnly = true
	info.Family = "MKM"
	switch mkm.gameId {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
