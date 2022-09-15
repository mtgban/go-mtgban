package mtgstocks

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type MTGStocksIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
}

func (stks *MTGStocksIndex) printf(format string, a ...interface{}) {
	if stks.LogCallback != nil {
		stks.LogCallback("[STKSIndex] "+format, a...)
	}
}

func NewScraperIndex() *MTGStocksIndex {
	stks := MTGStocksIndex{}
	stks.inventory = mtgban.InventoryRecord{}
	stks.marketplace = map[string]mtgban.InventoryRecord{}
	stks.MaxConcurrency = defaultConcurrency
	return &stks
}

var availableIndexNames = []string{
	"Stocks TCG Mid", "Stocks TCG Market",
}

func (stks *MTGStocksIndex) processEntry(channel chan<- responseChan, id int, edition string) error {
	printings, err := GetPrints(id)
	if err != nil {
		return err
	}

	for _, printing := range printings {
		theCard, err := preprocess(printing.Name, edition, printing.Foil)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			switch edition {
			// Skip set full of errors or with missing variants
			case "Alliances",
				"Fallen Empires",
				"Guilds of Ravnica: Guild Kits",
				"Homelands",
				"Starter 2000",
				"Ravnica Allegiance: Guild Kits",
				"World Championship Decks":
				continue
			// And of course, lands!
			default:
				if theCard.IsBasicLand() {
					continue
				}
			}
			stks.printf("%s", err.Error())
			stks.printf("%q", theCard)
			stks.printf("%q %s", printing, edition)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					stks.printf("- %s", card)
				}
			}
			continue
		}

		// Prices are smushed together, so we check whether there is a foil
		// version with a second match, and use it only if it's different than
		// the main printing (the other prices are empty when the card is foil-only)
		cardIdFoil, err := mtgmatcher.MatchId(cardId, !printing.Foil)
		if err != nil {
			cardIdFoil = cardId
		}

		link, err := getLink(printing.Slug)
		if err != nil {
			stks.printf("invalid data type used for %s", printing.Name)
		}

		// Sorted as availableIndexNames (for regular and foil)
		prices := []float64{
			printing.LatestPrice.Avg, printing.LatestPrice.Market, printing.LatestPrice.Foil, printing.LatestPrice.MarketFoil,
		}

		// Skip the empty prices
		var priceFiltered []float64
		for i := range prices {
			if prices[i] == 0 {
				continue
			}
			priceFiltered = append(priceFiltered, prices[i])
		}
		// If no price available, try one last chance, even if it's empty
		// to keep the ancillary information thus found (id/url)
		if len(priceFiltered) == 0 {
			priceFiltered = append(priceFiltered, printing.PreviousPrice)
		}

		for i := range priceFiltered {
			theId := cardId
			// If there are 4 prices, then we add the last two only when
			// the cardId is different between foil and nonfoil
			if i > 1 {
				if cardId == cardIdFoil {
					continue
				}
				theId = cardIdFoil
			}
			out := responseChan{
				cardId: theId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Quantity:   1,
					Price:      priceFiltered[i],
					URL:        link,
					SellerName: availableIndexNames[i%2],
				},
			}

			channel <- out
		}
	}

	return nil
}

func (stks *MTGStocksIndex) scrape() error {
	editions, err := GetSets()
	if err != nil {
		return err
	}
	stks.printf("Found %d editions", len(editions))

	sets := make(chan MTGStocksSet)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < stks.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for set := range sets {
				stks.printf("Processing %s", set.Name)
				err := stks.processEntry(channel, set.ID, set.Name)
				if err != nil {
					stks.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, edition := range editions {
			sets <- edition
		}
		close(sets)

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

func (stks *MTGStocksIndex) Inventory() (mtgban.InventoryRecord, error) {
	if len(stks.inventory) > 0 {
		return stks.inventory, nil
	}

	err := stks.scrape()
	if err != nil {
		return nil, err
	}

	return stks.inventory, nil
}

func (stks *MTGStocksIndex) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
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

func (tcg *MTGStocksIndex) MarketNames() []string {
	return availableIndexNames
}

func (stks *MTGStocksIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGStocksIndex"
	info.Shorthand = "STKSIndex"
	info.InventoryTimestamp = &stks.inventoryDate
	info.MetadataOnly = true
	return
}
