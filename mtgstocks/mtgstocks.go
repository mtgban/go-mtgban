package mtgstocks

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type MTGStocks struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
}

type requestChan struct {
	name     string
	interest Interest
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
}

var availableNames = []string{
	"Average Day", "Average Week", "Market Day", "Market Week",
}

func (stks *MTGStocks) printf(format string, a ...interface{}) {
	if stks.LogCallback != nil {
		stks.LogCallback("[STKS] "+format, a...)
	}
}

func NewScraper() *MTGStocks {
	stks := MTGStocks{}
	stks.inventory = mtgban.InventoryRecord{}
	stks.marketplace = map[string]mtgban.InventoryRecord{}
	stks.MaxConcurrency = defaultConcurrency
	return &stks
}

func (stks *MTGStocks) processEntry(channel chan<- responseChan, req requestChan) error {
	if req.interest.Percentage < 0 {
		return nil
	}

	theCard, err := preprocess(req.interest.Print.Name, req.interest.Print.SetName, req.interest.Foil)
	if err != nil {
		return nil
	}

	cardId, err := mtgmatcher.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		switch theCard.Edition {
		case "Alliances",
			"Fallen Empires",
			"Homelands",
			"World Championship Decks":
			return nil
		default:
			if mtgmatcher.IsBasicLand(theCard.Name) {
				return nil
			}
		}

		stks.printf("%q", theCard)
		stks.printf("%s | %s | %v", req.interest.Print.Name, req.interest.Print.SetName, req.interest.Foil)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := mtgmatcher.GetUUID(probe)
				stks.printf("- %s", card)
			}
		}
		return err
	}

	link, err := getLink(req.interest.Print.Slug)
	if err != nil {
		stks.printf("invalid data type used for %s", req.interest.Print.Name)
	}
	out := responseChan{
		cardId: cardId,
		entry: mtgban.InventoryEntry{
			Price:      req.interest.PresentPrice,
			Quantity:   1,
			URL:        link,
			SellerName: req.name + " " + strings.Title(req.interest.InterestType),
		},
	}

	channel <- out

	return nil
}

func (stks *MTGStocks) scrape() error {
	averages, err := AverageInterests()
	if err != nil {
		return err
	}
	markets, err := MarketInterests()
	if err != nil {
		return err
	}

	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < stks.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := stks.processEntry(channel, page)
				if err != nil {
					stks.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, interest := range averages.Foil {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range averages.Normal {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range markets.Foil {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		for _, interest := range markets.Normal {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := stks.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			stks.printf("%s", err.Error())
			continue
		}
	}

	stks.inventoryDate = time.Now()

	return nil
}

func (stks *MTGStocks) Inventory() (mtgban.InventoryRecord, error) {
	if len(stks.inventory) > 0 {
		return stks.inventory, nil
	}

	err := stks.scrape()
	if err != nil {
		return nil, err
	}

	return stks.inventory, nil
}

func (stks *MTGStocks) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(stks.inventory) == 0 {
		_, err := stks.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := stks.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range stks.inventory {
		for i := range stks.inventory[card] {
			if stks.inventory[card][i].SellerName == sellerName {
				if stks.inventory[card][i].Price == 0 {
					continue
				}
				if stks.marketplace[sellerName] == nil {
					stks.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				stks.marketplace[sellerName][card] = append(stks.marketplace[sellerName][card], stks.inventory[card][i])
			}
		}
	}

	if len(stks.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return stks.marketplace[sellerName], nil
}

func (tcg *MTGStocks) MarketNames() []string {
	return availableNames
}

func (stks *MTGStocks) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGStocks"
	info.Shorthand = "STKS"
	info.InventoryTimestamp = &stks.inventoryDate
	info.MetadataOnly = true
	return
}
