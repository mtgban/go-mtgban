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

	id2uuid map[string]string

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
	mkm.id2uuid = map[string]string{}
	return &mkm, nil
}

func (mkm *CardMarketFull) processEdition(channel chan<- responseChan, pair *MKMExpansionIdPair) error {
	products, err := mkm.client.MKMProductsInExpansion(pair.IdExpansion)
	if err != nil {
		return err
	}

	for _, product := range products {
		cardName := product.Name
		skipCard := false
		for _, name := range filteredCards {
			if cardName == name {
				skipCard = true
				break
			}
		}
		if skipCard ||
			mtgmatcher.IsToken(cardName) ||
			strings.Contains(cardName, "On Your Turn") {
			continue
		}

		err := mkm.processProduct(channel, &product)
		if err != nil {
			mkm.printf("product id %s returned %s", product.IdProduct, err)
		}
	}
	return nil
}

func (mkm *CardMarketFull) processProduct(channel chan<- responseChan, ogProduct *MKMProduct) error {
	anyLang := false
	switch ogProduct.ExpansionName {
	case "Dengeki Maoh Promos",
		"Promos", // for Magazine Inserts
		"War of the Spark: Japanese Alternate-Art Planeswalkers",
		"Ikoria: Lair of Behemoths: Extras",
		"Legends Italian",
		"The Dark Italian",
		"Rinascimento",
		"Fourth Edition Black Bordered",
		"Chronlicles Japanese",
		"Foreign Black Bordered":
		anyLang = true
	}

	articles, err := mkm.client.MKMArticles(ogProduct.IdProduct, anyLang)
	if err != nil {
		return err
	}

	for _, article := range articles {
		err := mkm.processArticles(channel, &article)
		if err != nil {
			mkm.printf("article %d (product id %d) returned %s", article.IdArticle, article.IdProduct, err.Error())
		}
	}

	return nil
}

func (mkm *CardMarketFull) processArticles(channel chan<- responseChan, article *MKMArticle) error {
	_, found := mkm.priceGuide[article.IdProduct]
	if !found {
		return nil
	}

	switch article.Language.LanguageName {
	case "English":
	case "Japanese":
		switch article.Product.Expansion {
		case "Chronlicles Japanese":
		case "Fourth Edition Black Bordered":
		case "Dengeki Maoh Promos":
		case "Promos": // for Magazine Inserts
		case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		case "Ikoria: Lair of Behemoths: Extras":
			switch article.Product.Name {
			case "Crystalline Giant (V.2)",
				"Battra, Dark Destroyer (V.2)",
				"Mothra's Great Cocoon (V.2)":
			default:
				return nil
			}
		default:
			return nil
		}
	case "Italian":
		switch article.Product.Expansion {
		case "Rinascimento":
		case "Legends Italian":
		case "The Dark Italian":
		case "Foreign Black Bordered":
		default:
			return nil
		}
	default:
		return nil
	}

	// Use a map to avoid repeating the same card match
	key := fmt.Sprintf("%d+%v", article.IdProduct, article.IsFoil)

	cardId, found := mkm.id2uuid[key]
	if !found {
		theCard, err := Preprocess(article.Product.Name, article.Product.Number, article.Product.Expansion)
		if err != nil {
			return nil
		}
		if article.IsFoil {
			theCard.Foil = true
		}

		cardId, err = mtgmatcher.Match(theCard)
		if err != nil {
			if theCard.Edition == "Pro Tour Collector Set" || strings.HasPrefix(theCard.Edition, "World Championship Decks") {
				return nil
			}

			mkm.printf("%v", err)
			mkm.printf("%q", theCard)
			mkm.printf("%v", article.Product)
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

		mkm.id2uuid[key] = cardId
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
		return nil
	}

	if mtgmatcher.Contains(article.Comments, "alter") ||
		mtgmatcher.Contains(article.Comments, "signed") ||
		mtgmatcher.Contains(article.Comments, "artist proof") {
		return nil
	}

	// Find common ways to describe misprints
	misprintHack := mtgmatcher.Contains(article.Comments, "misprint") ||
		mtgmatcher.Contains(article.Comments, "miscut") ||
		mtgmatcher.Contains(article.Comments, "crimped") ||
		mtgmatcher.Contains(article.Comments, "square")

	out := responseChan{
		ogId:   article.IdProduct,
		cardId: cardId,
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

	return nil
}

func (mkm *CardMarketFull) scrape() error {
	list, err := mkm.client.ListExpansionIds()
	if err != nil {
		return err
	}

	mkm.printf("Parsing %d editions", len(list))

	expansions := make(chan MKMExpansionIdPair)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansion := range expansions {
				err := mkm.processEdition(channel, &expansion)
				if err != nil {
					mkm.printf("expansion id %s returned %s", expansion.IdExpansion, err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i, pair := range list {
			mkm.printf("Processing id %d - %s (%d/%d)", pair.IdExpansion, pair.Name, i+1, len(list))
			expansions <- pair
		}
		close(expansions)

		wg.Wait()
		close(channel)
	}()

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
	}

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
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
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	mkm.marketplace = market
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
