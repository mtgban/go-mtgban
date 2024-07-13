package cardmarket

import (
	"errors"
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

	inventory mtgban.InventoryRecord

	client *MKMClient
}

var availableIndexNames = []string{
	"MKM Low", "MKM Trend",
}

func (mkm *CardMarketIndex) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

func NewScraperIndex(appToken, appSecret string) (*CardMarketIndex, error) {
	mkm := CardMarketIndex{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mkm.exchangeRate = rate
	return &mkm, nil
}

func (mkm *CardMarketIndex) processEdition(channel chan<- responseChan, idExpansion int, priceGuide []PriceGuide) error {
	products, err := mkm.client.MKMProductsInExpansion(idExpansion)
	if err != nil {
		return err
	}

	for _, product := range products {
		err := mkm.processProduct(channel, &product, priceGuide)
		if err != nil {
			mkm.printf("product id %d returned %s", product.IdProduct, err)
		}
	}
	return nil
}

func (mkm *CardMarketIndex) processProduct(channel chan<- responseChan, product *MKMProduct, priceGuide []PriceGuide) error {
	theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
	if err != nil {
		_, ok := err.(*PreprocessError)
		if ok {
			return err
		}
		return nil
	}
	var cardIdFoil string
	cardId, err := mtgmatcher.Match(theCard)
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
		mkm.printf("%v|%v|%v", product.Name, product.Number, product.ExpansionName)

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
	for index = range priceGuide {
		if priceGuide[index].IdProduct == product.IdProduct {
			break
		}
	}

	// Sorted as availableIndexNames
	prices := []float64{
		priceGuide[index].LowPrice, priceGuide[index].TrendPrice,
	}
	foilprices := []float64{
		priceGuide[index].FoilLowPrice, priceGuide[index].FoilTrendPrice,
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
				},
			}

			channel <- out
		}

		if foilprices[0] != 0 || foilprices[1] != 0 {
			v.Set("isFoil", "Y")
			link.RawQuery = v.Encode()

			if cardIdFoil == "" {
				theCard, _ = Preprocess(product.Name, product.Number, product.ExpansionName)
				theCard.Foil = true
				cardIdFoil, err = mtgmatcher.Match(theCard)
				if err != nil {
					return nil
				}
			}
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
				},
			}

			channel <- out
		}
	}

	return nil
}

func (mkm *CardMarketIndex) scrape() error {
	priceGuide, err := GetPriceGuide()
	if err != nil {
		return err
	}

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	list, err := mkm.client.FilteredExpansions()
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
				err := mkm.processEdition(channel, list[i].IdExpansion, priceGuide)
				if err != nil {
					mkm.printf("expansion %s (id %d) returned %s", list[i].Name, list[i].IdExpansion, err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := range list {
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

func (mkm *CardMarketIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.MetadataOnly = true
	info.Family = "MKM"
	return
}
