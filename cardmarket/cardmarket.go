package cardmarket

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type requestChan struct {
	ProductId string
	Expansion string
}

type responseChan struct {
	ogId   string
	cardId string
	entry  mtgban.InventoryEntry
}

type CardMarketIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int
	exchangeRate   float64

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	client *MKMClient
}

func (mkm *CardMarketIndex) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

func NewScraperIndex(appToken, appSecret string) (*CardMarketIndex, error) {
	mkm := CardMarketIndex{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.marketplace = map[string]mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mkm.exchangeRate = rate
	return &mkm, nil
}

func (mkm *CardMarketIndex) processEntry(channel chan<- responseChan, req requestChan) error {
	product, err := mkm.client.MKMProduct(req.ProductId)
	if err != nil {
		return err
	}

	theCard, err := Preprocess(product.Name, product.Number, product.Expansion.Name)
	if err != nil {
		_, ok := err.(*PreprocessError)
		if ok {
			return err
		}
		return nil
	}
	cardId, err := mtgmatcher.Match(theCard)
	if err != nil {
		if theCard.Edition == "Pro Tour Collector Set" || strings.HasPrefix(theCard.Edition, "World Championship Decks") {
			return nil
		}

		mkm.printf("%v", err)
		mkm.printf("%q", theCard)
		mkm.printf("%v", product)
		alias, ok := err.(*mtgmatcher.AliasingError)
		if ok {
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

	names := []string{
		"MKM Low", "MKM Trend",
	}
	prices := []float64{
		product.PriceGuide["LOWEX"], product.PriceGuide["TREND"],
	}
	foilprices := []float64{
		product.PriceGuide["LOWFOIL"], product.PriceGuide["TRENDFOIL"],
	}

	card, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return err
	}

	// If card is not foil, add prices from the prices array, then check
	// if there is a foil printing, and add prices from the foilprices array.
	// If a card is foil-only then we just use foilprices data.
	if !card.Foil {
		link.RawQuery = v.Encode()

		for i := range names {
			if prices[i] == 0 || product.CountArticles-product.CountFoils == 0 {
				continue
			}
			out := responseChan{
				ogId:   req.ProductId,
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i] * mkm.exchangeRate,
					Quantity:   product.CountArticles - product.CountFoils,
					URL:        link.String(),
					SellerName: names[i],
				},
			}

			channel <- out
		}

		if foilprices[0] != 0 || foilprices[1] != 0 {
			v.Set("isFoil", "Y")
			link.RawQuery = v.Encode()

			theCard.Foil = true
			cardIdFoil, err := mtgmatcher.Match(theCard)
			if err != nil {
				return nil
			}
			// If the id is the same it means that the card was really nonfoil-only
			if cardId != cardIdFoil && product.CountFoils != 0 {
				for i := range names {
					if foilprices[i] == 0 {
						continue
					}
					out := responseChan{
						ogId:   req.ProductId,
						cardId: cardIdFoil,
						entry: mtgban.InventoryEntry{
							Conditions: "NM",
							Price:      foilprices[i] * mkm.exchangeRate,
							Quantity:   product.CountFoils,
							URL:        link.String(),
							SellerName: names[i],
						},
					}

					channel <- out
				}
			}
		}
	} else {
		v.Set("isFoil", "Y")
		link.RawQuery = v.Encode()

		for i := range names {
			if foilprices[i] == 0 || product.CountFoils == 0 {
				continue
			}
			out := responseChan{
				ogId:   req.ProductId,
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      foilprices[i] * mkm.exchangeRate,
					Quantity:   product.CountFoils,
					URL:        link.String(),
					SellerName: names[i],
				},
			}

			channel <- out
		}
	}

	return nil
}

func (mkm *CardMarketIndex) scrape() error {
	list, err := mkm.client.ListProductIds()
	if err != nil {
		return err
	}

	mkm.printf("Parsing %d product ids", len(list))

	products := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for product := range products {
				err := mkm.processEntry(channel, product)
				if err != nil {
					mkm.printf("id %s returned %s", product.ProductId, err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, pair := range list {
			//num, _ := strconv.Atoi(pair.ProductId)
			products <- requestChan{
				ProductId: pair.ProductId,
			}
		}
		close(products)

		wg.Wait()
		close(channel)
	}()

	lastTime := time.Now()
	for result := range channel {
		err := mkm.inventory.AddStrict(result.cardId, &result.entry)
		if err != nil {
			card, cerr := mtgmatcher.GetUUID(result.cardId)
			if cerr != nil {
				mkm.printf("%s - %s: %s", result.ogId, cerr.Error(), result.cardId)
				continue
			}
			// Skip WCD, too many errors
			if card.Edition == "Pro Tour Collector Set" || strings.HasPrefix(card.Edition, "World Championship Decks") {
				continue
			}
			mkm.printf("%s - %s", result.ogId, err.Error())
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			mkm.printf("Still going %s/%s, last processed card: %s", result.ogId, list[len(list)-1].ProductId, card)
			lastTime = time.Now()
		}
	}

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

func (mkm *CardMarketIndex) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(mkm.inventory) == 0 {
		_, err := mkm.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := mkm.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range mkm.inventory {
		for i := range mkm.inventory[card] {
			if mkm.inventory[card][i].SellerName == sellerName {
				if mkm.inventory[card][i].Price == 0 {
					continue
				}
				if mkm.marketplace[sellerName] == nil {
					mkm.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				mkm.marketplace[sellerName][card] = append(mkm.marketplace[sellerName][card], mkm.inventory[card][i])
			}
		}
	}

	if len(mkm.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return mkm.marketplace[sellerName], nil
}

func (mkm *CardMarketIndex) IntializeInventory(reader io.Reader) error {
	inventory, err := mtgban.LoadInventoryFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	mkm.inventory = inventory

	mkm.printf("Loaded inventory from file")

	return nil
}

func (mkm *CardMarketIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = mkm.inventoryDate
	info.MetadataOnly = true
	return
}
