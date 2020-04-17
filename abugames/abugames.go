package abugames

import (
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	maxConcurrency = 8
)

type ABUGames struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time
	BuylistDate   time.Time

	client *ABUClient

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *ABUGames {
	abu := ABUGames{}
	abu.inventory = mtgban.InventoryRecord{}
	abu.buylist = mtgban.BuylistRecord{}
	abu.client = NewABUClient()
	return &abu
}

type resultChan struct {
	err       error
	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func (abu *ABUGames) printf(format string, a ...interface{}) {
	if abu.LogCallback != nil {
		abu.LogCallback("[ABU] "+format, a...)
	}
}

func (abu *ABUGames) processEntry(page int) (res resultChan) {
	product, err := abu.client.GetProduct(page)
	if err != nil {
		res.err = err
		return
	}

	res.inventory = mtgban.InventoryRecord{}
	res.buylist = mtgban.BuylistRecord{}

	duplicate := map[string]bool{}

	for _, group := range product.Grouped.ProductId.Groups {
		for _, card := range group.Doclist.Cards {
			// Deprecated value
			if card.Condition == "SP" {
				continue
			}

			cond := card.Condition
			switch cond {
			case "NM", "HP":
			case "MINT":
				cond = "NM"
			case "PLD":
				cond = "SP"
			default:
				abu.printf("Unknown '%s' condition", cond)
				continue
			}

			if duplicate[card.Id] {
				abu.printf("Skipping duplicate card: %s (%s)", card.DisplayTitle, card.Edition)
				continue
			}

			theCard, err := preprocess(&card)
			if err != nil {
				continue
			}

			cc, err := theCard.Match()
			if err != nil {
				abu.printf("%v", theCard)
				abu.printf("%v", card)
				abu.printf("%v", err)
				continue
			}

			if card.SellQuantity > 0 && card.SellPrice > 0 {
				notes := "https://abugames.com/magic-the-gathering/singles?search=\"" + card.SimpleTitle + "\"&magic_edition=[\"" + card.Edition + "\"]"
				if theCard.Foil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				out := &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      card.SellPrice,
					Quantity:   card.SellQuantity,
					URL:        notes,
				}
				res.inventory.Add(cc, out)
			}

			if card.BuyQuantity > 0 && card.BuyPrice > 0 && card.TradePrice > 0 && card.Condition == "NM" {
				var priceRatio, qtyRatio float64
				if card.SellPrice > 0 {
					priceRatio = card.BuyPrice / card.SellPrice * 100
				}
				if card.SellQuantity > 0 {
					qtyRatio = float64(card.BuyQuantity) / float64(card.SellQuantity) * 100
				}

				notes := "https://abugames.com/buylist/magic-the-gathering/singles?search=\"" + card.SimpleTitle + "\"&magic_edition=[\"" + card.Edition + "\"]"
				if theCard.Foil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				out := &mtgban.BuylistEntry{
					Conditions:    cond,
					BuyPrice:      card.BuyPrice,
					TradePrice:    card.TradePrice,
					Quantity:      card.BuyQuantity,
					PriceRatio:    priceRatio,
					QuantityRatio: qtyRatio,
					URL:           notes,
				}
				res.buylist.Add(cc, out)
			}

			duplicate[card.Id] = true
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (abu *ABUGames) scrape() error {
	product, err := abu.client.GetInfo()
	if err != nil {
		return err
	}

	count := product.Grouped.ProductId.Count
	abu.printf("Parsing %d entries", count)

	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- abu.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < count; i += maxEntryPerRequest {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			abu.printf("%v", result.err)
			continue
		}

		for card, entries := range result.inventory {
			for i := range entries {
				err = abu.inventory.Add(&card, &entries[i])
				if err != nil {
					abu.printf(err.Error())
				}
			}
		}
		for card, entry := range result.buylist {
			err = abu.buylist.Add(&card, &entry)
			if err != nil {
				abu.printf(err.Error())
			}
		}
	}

	abu.InventoryDate = time.Now()
	abu.BuylistDate = time.Now()

	return nil
}

func (abu *ABUGames) Inventory() (mtgban.InventoryRecord, error) {
	if len(abu.inventory) > 0 {
		return abu.inventory, nil
	}

	start := time.Now()
	abu.printf("Inventory scraping started at %s", start)

	err := abu.scrape()
	if err != nil {
		return nil, err
	}
	abu.printf("Inventory scraping took %s", time.Since(start))

	return abu.inventory, nil
}

func (abu *ABUGames) Buylist() (mtgban.BuylistRecord, error) {
	if len(abu.buylist) > 0 {
		return abu.buylist, nil
	}

	start := time.Now()
	abu.printf("Buylist scraping started at %s", start)

	err := abu.scrape()
	if err != nil {
		return nil, err
	}
	abu.printf("Buylist scraping took %s", time.Since(start))

	return abu.buylist, nil
}

// Purely estimated
func (abu *ABUGames) Grading(card mtgdb.Card, entry mtgban.BuylistEntry) (grade map[string]float64) {
	grade = map[string]float64{
		"SP": 0.70, "MP": 0.6, "HP": 0.4,
	}
	return
}

func (abu *ABUGames) Info() (info mtgban.ScraperInfo) {
	info.Name = "ABU Games"
	info.Shorthand = "ABU"
	info.InventoryTimestamp = abu.InventoryDate
	info.BuylistTimestamp = abu.BuylistDate
	return
}
