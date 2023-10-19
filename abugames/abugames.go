package abugames

import (
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type ABUGames struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
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
	theCard  mtgmatcher.Card
	cardId   string
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
		for _, card := range group.Doclist.Cards {
			// Deprecated value
			if card.Condition == "SP" {
				continue
			}

			cond := card.Condition
			switch cond {
			case "MINT":
				continue
			case "NM":
				cond = "NM"
			case "PLD":
				cond = "SP"
			case "HP":
				cond = "MP"
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

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// There is a bunch of non-existing prerelease cards from mh2
				if theCard.Variation == "Prerelease" {
					continue
				}
				abu.printf("%v", theCard)
				abu.printf("%v", card)
				abu.printf("%v", err)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						abu.printf("- %s", card)
					}
				}
				continue
			}

			var invEntry *mtgban.InventoryEntry
			var buyEntry *mtgban.BuylistEntry

			// For URL genration searchQuery needs to be in plaintext, not URL-encoded
			searchQuery := "&search=" + card.SimpleTitle

			u, err := url.Parse("https://abugames.com")
			if err != nil {
				return err
			}

			if card.SellQuantity > 0 && card.SellPrice > 0 {
				u.Path = "/magic-the-gathering/singles"

				v := url.Values{}
				v.Set("magic_edition", "[\""+card.Edition+"\"]")
				v.Set("card_style", "[\"Normal\"]")
				if theCard.Foil {
					v.Set("card_style", "[\"Foil\"]")
				}
				u.RawQuery = v.Encode()

				invEntry = &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      card.SellPrice,
					Quantity:   card.SellQuantity,
					URL:        u.String() + searchQuery,
				}
			}

			if card.BuyQuantity > 0 && card.BuyPrice > 0 && card.TradePrice > 0 {
				var priceRatio float64
				if card.SellPrice > 0 {
					priceRatio = card.BuyPrice / card.SellPrice * 100
				}

				u.Path = "/buylist/magic-the-gathering/singles"

				v := url.Values{}
				v.Set("magic_edition", "[\""+card.Edition+"\"]")
				v.Set("card_style", "[\"Normal\"]")
				if theCard.Foil {
					v.Set("card_style", "[\"Foil\"]")
				}
				u.RawQuery = v.Encode()

				buyEntry = &mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   card.BuyPrice,
					TradePrice: card.TradePrice,
					Quantity:   card.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        u.String() + searchQuery,
				}
			}

			if invEntry != nil || buyEntry != nil {
				channel <- resultChan{
					theCard:  *theCard,
					cardId:   cardId,
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
			err = abu.inventory.AddRelaxed(result.cardId, result.invEntry)
			if err != nil {
				abu.printf("%s", &result.theCard)
				abu.printf("%s", err.Error())
			}
		}
		if result.buyEntry != nil {
			err = abu.buylist.AddRelaxed(result.cardId, result.buyEntry)
			if err != nil {
				abu.printf("%s", &result.theCard)
				abu.printf("%s", err.Error())
			}
		}
	}

	abu.inventoryDate = time.Now()
	abu.buylistDate = time.Now()

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
	info.InventoryTimestamp = &abu.inventoryDate
	info.BuylistTimestamp = &abu.buylistDate
	return
}
