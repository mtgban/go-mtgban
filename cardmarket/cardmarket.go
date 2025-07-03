package cardmarket

import (
	"errors"
	"fmt"
	"net/url"
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
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mkm.exchangeRate = rate
	mkm.gameId = gameId
	return &mkm, nil
}

func (mkm *CardMarketIndex) processEdition(channel chan<- responseChan, idExpansion int) error {
	products, err := mkm.client.MKMProductsInExpansion(idExpansion)
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
		theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
		if err != nil {
			_, ok := err.(*PreprocessError)
			if ok {
				return err
			}
			return nil
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
		cardName := product.Name
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

	link, err := url.Parse("https://www.cardmarket.com" + product.Website)
	if err != nil {
		return err
	}
	v := url.Values{}
	if mkm.Affiliate != "" {
		v.Set("utm_source", mkm.Affiliate)
		v.Set("utm_medium", "text")
		v.Set("utm_campaign", "card_prices")
	}
	// Set English as preferred language, switches to the default one
	// in case the card has a foreign-only printing available
	v.Set("language", "1")

	var index int
	for index = range mkm.priceGuide {
		if mkm.priceGuide[index].IdProduct == product.IdProduct {
			break
		}
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
		link.RawQuery = v.Encode()

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
					URL:        link.String(),
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}

		if foilprices[0] != 0 || foilprices[1] != 0 {
			v.Set("isFoil", "Y")
			link.RawQuery = v.Encode()

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
							URL:        link.String(),
							SellerName: availableIndexNames[i],
							OriginalId: fmt.Sprint(product.IdProduct),
						},
					}

					channel <- out
				}
			}
		}
	} else {
		v.Set("isFoil", "Y")
		link.RawQuery = v.Encode()

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
					URL:        link.String(),
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (mkm *CardMarketIndex) scrape() error {
	priceGuide, err := GetPriceGuide(mkm.gameId)
	if err != nil {
		return err
	}
	mkm.priceGuide = priceGuide

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	list, err := mkm.client.Expansions(mkm.gameId)
	if err != nil {
		return err
	}

	mkm.printf("Parsing %d expansion ids", len(list))

	expansions := make(chan int)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for i := range expansions {
				err := mkm.processEdition(channel, list[i].IdExpansion)
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

	err := mkm.scrape()
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
