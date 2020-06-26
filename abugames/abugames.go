package abugames

import (
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	defaultConcurrency = 8
)

type ABUGames struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	BuylistDate    time.Time
	MaxConcurrency int

	client *ABUClient

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *ABUGames {
	abu := ABUGames{}
	abu.inventory = mtgban.InventoryRecord{}
	abu.buylist = mtgban.BuylistRecord{}
	abu.client = NewABUClient()
	abu.MaxConcurrency = defaultConcurrency
	return &abu
}

type resultChan struct {
	card     *mtgdb.Card
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (abu *ABUGames) printf(format string, a ...interface{}) {
	if abu.LogCallback != nil {
		abu.LogCallback("[ABU] "+format, a...)
	}
}

func (abu *ABUGames) processEntry(channel chan<- resultChan, page int) error {
	product, err := abu.client.GetProduct(page)
	if err != nil {
		return err
	}

	duplicate := map[string]bool{}

	for _, group := range product.Grouped.ProductId.Groups {
		// Use a different grading scale if MINT is present
		hasMintSeparate := false
		for _, card := range group.Doclist.Cards {
			if card.Condition == "MINT" {
				hasMintSeparate = true
				break
			}
		}
		for _, card := range group.Doclist.Cards {
			// Deprecated value
			if card.Condition == "SP" {
				continue
			}

			cond := card.Condition
			switch cond {
			case "MINT":
				cond = "NM"
			case "NM":
				cond = "NM"
				if hasMintSeparate {
					cond = "SP"
				}
			case "PLD":
				cond = "SP"
				if hasMintSeparate {
					cond = "MP"
				}
			case "HP":
				cond = "MP"
				if hasMintSeparate {
					cond = "HP"
				}
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

			var invEntry *mtgban.InventoryEntry
			var buyEntry *mtgban.BuylistEntry

			if card.SellQuantity > 0 && card.SellPrice > 0 {
				notes := "https://abugames.com/magic-the-gathering/singles?search=\"" + card.SimpleTitle
				if card.Edition != "Promo" {
					notes += "\"&magic_edition=[\"" + card.Edition + "\"]"
				}
				if theCard.Foil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				invEntry = &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      card.SellPrice,
					Quantity:   card.SellQuantity,
					URL:        notes,
				}
			}

			if card.BuyQuantity > 0 && card.BuyPrice > 0 && card.TradePrice > 0 && card.Condition == "NM" {
				var priceRatio float64
				if card.SellPrice > 0 {
					priceRatio = card.BuyPrice / card.SellPrice * 100
				}

				notes := "https://abugames.com/buylist/magic-the-gathering/singles?search=\"" + card.SimpleTitle
				if card.Edition != "Promo" {
					notes += "\"&magic_edition=[\"" + card.Edition + "\"]"
				}
				if theCard.Foil {
					notes += "&card_style=[\"Foil\"]"
				} else {
					notes += "&card_style=[\"Normal\"]"
				}

				buyEntry = &mtgban.BuylistEntry{
					BuyPrice:   card.BuyPrice,
					TradePrice: card.TradePrice,
					Quantity:   card.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        notes,
				}
			}

			if invEntry != nil || buyEntry != nil {
				channel <- resultChan{
					card:     cc,
					invEntry: invEntry,
					buyEntry: buyEntry,
				}
			}

			duplicate[card.Id] = true
		}
	}

	return nil
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

	for i := 0; i < abu.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := abu.processEntry(results, page)
				if err != nil {
					abu.printf("%v", err)
				}
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
		if result.invEntry != nil {
			err = abu.inventory.Add(result.card, result.invEntry)
			if err != nil {
				abu.printf(err.Error())
			}
		}
		if result.buyEntry != nil {
			err = abu.buylist.Add(result.card, result.buyEntry)
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

	err := abu.scrape()
	if err != nil {
		return nil, err
	}

	return abu.inventory, nil
}

func (abu *ABUGames) Buylist() (mtgban.BuylistRecord, error) {
	if len(abu.buylist) > 0 {
		return abu.buylist, nil
	}

	err := abu.scrape()
	if err != nil {
		return nil, err
	}

	return abu.buylist, nil
}

func (abu *ABUGames) Info() (info mtgban.ScraperInfo) {
	info.Name = "ABU Games"
	info.Shorthand = "ABU"
	info.InventoryTimestamp = abu.InventoryDate
	info.BuylistTimestamp = abu.BuylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
