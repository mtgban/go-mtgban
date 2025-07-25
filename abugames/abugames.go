package abugames

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 4
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
	theCard    mtgmatcher.InputCard
	cardId     string
	invEntry   *mtgban.InventoryEntry
	buyEntry   *mtgban.BuylistEntry
	tradeEntry *mtgban.BuylistEntry
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

	var duplicates []string

	for _, group := range product.Grouped.ProductId.Groups {
		for i, doc := range group.Doclist.Docs {
			// Deprecated value
			if doc.Condition == "SP" {
				continue
			}

			cond := doc.Condition
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

			if slices.Contains(duplicates, doc.Id) {
				abu.printf("Skipping duplicate card: %s (%s)", doc.DisplayTitle, doc.Edition)
				continue
			}

			theCard, err := preprocess(&doc)
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// Reduce error reporting for repeated conditions
				if i > 0 {
					continue
				}

				// There are a bunch of non-existing prerelease cards from mh2
				// and promo pack DFC from lci (among others)
				if strings.Contains(theCard.Variation, "Prerelease") ||
					strings.Contains(theCard.Variation, "The List") ||
					strings.Contains(theCard.Variation, "Mystery Booster") ||
					strings.Contains(theCard.Variation, "Promo Pack") {
					continue
				}
				abu.printf("%v", theCard)
				abu.printf("%v", doc)
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

			// Sanity check, a bunch of cards are market as foil when they
			// actually don't have a foil printing, just skip them
			if strings.Contains(doc.DisplayTitle, "FOIL") {
				co, err := mtgmatcher.GetUUID(cardId)
				if err != nil {
					continue
				}
				if !co.Foil {
					continue
				}
			}

			var invEntry *mtgban.InventoryEntry
			var buyEntry *mtgban.BuylistEntry
			var tradeEntry *mtgban.BuylistEntry

			// For URL genration searchQuery needs to be in plaintext, not URL-encoded
			searchQuery := "&search=" + doc.SimpleTitle

			u, err := url.Parse("https://abugames.com")
			if err != nil {
				return err
			}

			v := url.Values{}
			v.Set("magic_edition", "[\""+doc.Edition+"\"]")
			v.Set("card_style", "[\"Normal\"]")
			if theCard.Foil {
				v.Set("card_style", "[\"Foil\"]")
			}
			u.RawQuery = v.Encode()

			if doc.SellQuantity > 0 && doc.SellPrice > 0 {
				u.Path = "/magic-the-gathering/singles"

				invEntry = &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      doc.SellPrice,
					Quantity:   doc.SellQuantity,
					URL:        u.String() + searchQuery,
					OriginalId: group.GroupValue,
					InstanceId: doc.Id,
				}
			}

			if doc.BuyQuantity > 0 && doc.BuyPrice > 0 {
				var priceRatio float64
				if doc.SellPrice > 0 {
					priceRatio = doc.BuyPrice / doc.SellPrice * 100
				}

				u.Path = "/buylist/magic-the-gathering/singles"

				buyEntry = &mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   doc.BuyPrice,
					Quantity:   doc.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        u.String() + searchQuery,
					OriginalId: group.GroupValue,
					InstanceId: doc.Id,
					VendorName: availableTraderNames[0],
				}

				if doc.SellPrice > 0 {
					priceRatio = doc.TradePrice / doc.SellPrice * 100
				}
				tradeEntry = &mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   doc.TradePrice,
					Quantity:   doc.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        u.String() + searchQuery,
					OriginalId: group.GroupValue,
					InstanceId: doc.Id,
					VendorName: availableTraderNames[1],
				}
			}

			if invEntry != nil || buyEntry != nil {
				channel <- resultChan{
					theCard:    *theCard,
					cardId:     cardId,
					invEntry:   invEntry,
					buyEntry:   buyEntry,
					tradeEntry: tradeEntry,
				}
			}

			duplicates = append(duplicates, doc.Id)
		}
	}

	return nil
}

// Scrape returns an array of Entry, containing pricing and card information
func (abu *ABUGames) scrape() error {
	count, err := abu.client.GetTotalItems()
	if err != nil {
		return err
	}
	abu.printf("Parsing %d entries", count)

	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < abu.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				abu.printf("Processing page %d/%d", page/maxEntryPerRequest, count/maxEntryPerRequest)
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
		if result.tradeEntry != nil {
			err = abu.buylist.AddRelaxed(result.cardId, result.tradeEntry)
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

var availableTraderNames = []string{
	"ABU Games",
	"ABU Games (credit)",
}

var name2shorthand = map[string]string{
	"ABU Games":          "ABUGames",
	"ABU Games (credit)": "ABUCredit",
}

func (abu *ABUGames) TraderNames() []string {
	return availableTraderNames
}

func (abu *ABUGames) InfoForScraper(name string) mtgban.ScraperInfo {
	info := abu.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	if info.Shorthand == "ABUCredit" {
		info.CreditMultiplier = 0.6
	}
	return info
}

func (abu *ABUGames) Info() (info mtgban.ScraperInfo) {
	info.Name = "ABU Games"
	info.Shorthand = "ABU"
	info.InventoryTimestamp = &abu.inventoryDate
	info.BuylistTimestamp = &abu.buylistDate
	return
}
