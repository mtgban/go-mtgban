package cardmarket

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type CardMarketFull struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int
	exchangeRate   float64

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	client *MKMClient
}

func (mkm *CardMarketFull) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMF] "+format, a...)
	}
}

func NewScraperFull(appToken, appSecret string) (*CardMarketFull, error) {
	mkm := CardMarketFull{}
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

// TODO
// handle the list/mb1 foils
// check articles loop works more than once
func (mkm *CardMarketFull) processEntry(channel chan<- responseChan, req requestChan) error {
	anyLang := false
	switch req.Expansion {
	case "Dengeki Maoh Promos",
		"Promos", // for Magazine Inserts
		"War of the Spark: Japanese Alternate-Art Planeswalkers",
		"Ikoria: Lair of Behemoths: Extras",
		// "Rinascimento",
		// "Fourth Edition Black Bordered",
		// "Chronlicles Japanese",
		"Foreign Black Bordered":
		anyLang = true
	}

	articles, err := mkm.client.MKMArticles(req.ProductId, anyLang)
	if err != nil {
		return err
	}

	if len(articles) == 0 {
		return nil
	}

	product := articles[0].Product

	theCard, err := Preprocess(product.Name, product.Number, product.Expansion)
	if err != nil {
		return nil
	}

	var cardId string
	var cardIdFoil string

	for _, article := range articles {
		switch article.Language.LanguageName {
		case "English":
		case "Japanese":
			switch product.Expansion {
			//case "Chronlicles Japanese":
			//case "Fourth Edition Black Bordered":
			case "Dengeki Maoh Promos":
			case "Promos": // for Magazine Inserts
			case "War of the Spark: Japanese Alternate-Art Planeswalkers":
			case "Ikoria: Lair of Behemoths: Extras":
				switch product.Name {
				case "Crystalline Giant (V.2)",
					"Battra, Dark Destroyer (V.2)",
					"Mothra's Great Cocoon (V.2)":
				default:
					continue
				}
			default:
				continue
			}
		case "Italian":
			switch product.Expansion {
			//case "Rinascimento":
			case "Foreign Black Bordered":
			default:
				continue
			}
		default:
			continue
		}

		if cardId == "" {
			cardId, err = mtgmatcher.Match(theCard)
		}
		if cardIdFoil == "" && article.IsFoil {
			theCard.Foil = true
			cardIdFoil, err = mtgmatcher.Match(theCard)
		}
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

		finalCardId := cardId
		if article.IsFoil {
			finalCardId = cardIdFoil
		}

		price := article.Price
		qty := article.Count
		if article.IsPlayset {
			price /= 4
			qty *= 4
		}

		cond := article.Condition
		switch cond {
		case "MT", "NM":
			cond = "NM"
		case "EX":
			cond = "SP"
		case "GD", "LP":
			cond = "MP"
		case "PL":
			cond = "HP"
		case "PO":
			cond = "PO"
		default:
			mkm.printf("Unknown '%s' condition", cond)
			continue
		}

		if mtgmatcher.Contains(article.Comments, "alter") ||
			mtgmatcher.Contains(article.Comments, "signed") {
			continue
		}

		// Find common ways to describe misprints
		misprintHack := mtgmatcher.Contains(article.Comments, "misprint") ||
			mtgmatcher.Contains(article.Comments, "miscut") ||
			mtgmatcher.Contains(article.Comments, "crimped") ||
			mtgmatcher.Contains(article.Comments, "square")

		out := responseChan{
			ogId:   req.ProductId,
			cardId: finalCardId,
			entry: mtgban.InventoryEntry{
				Conditions: cond,
				Price:      price * mkm.exchangeRate,
				Quantity:   qty,
				SellerName: article.Seller.Username,

				// Hijack Bundle to propagate this property
				Bundle: misprintHack,
			},
		}

		channel <- out
	}

	return nil
}

func (mkm *CardMarketFull) scrape() error {
	list, err := mkm.client.ListProductIds()
	if err != nil {
		return err
	}
	expansions, err := mkm.client.MKMExpansions()
	if err != nil {
		return err
	}

	mkm.printf("Parsing %d product ids over %d editions", len(list), len(expansions))

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
			exp, found := expansions[pair.ExpansionId]
			if !found {
				mkm.printf("edition id %s not found", pair.ExpansionId)
				continue
			}
			if !exp.IsReleased {
				continue
			}
			//num, _ := strconv.Atoi(id)
			products <- requestChan{
				ProductId: pair.ProductId,
				Expansion: expansions[pair.ExpansionId].Name,
			}
		}
		close(products)

		wg.Wait()
		close(channel)
	}()

	lastTime := time.Now()
	for result := range channel {
		err := mkm.inventory.AddRelaxed(result.cardId, &result.entry)
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

func (mkm *CardMarketFull) Inventory() (mtgban.InventoryRecord, error) {
	if len(mkm.inventory) > 0 {
		return mkm.inventory, nil
	}

	err := mkm.scrape()
	if err != nil {
		return nil, err
	}

	return mkm.inventory, nil
}

func (mkm *CardMarketFull) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
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

func (mkm *CardMarketFull) IntializeInventory(reader io.Reader) error {
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

func (mkm *CardMarketFull) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Full"
	info.Shorthand = "MKMFull"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = mkm.inventoryDate
	return
}
