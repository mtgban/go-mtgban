package magiccorner

import (
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
)

const (
	maxConcurrency = 8
)

type Magiccorner struct {
	VerboseLog    bool
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time

	exchangeRate float64

	inventory map[string][]mtgban.InventoryEntry
	client    *MCClient
}

func NewScraper() (*Magiccorner, error) {
	mc := Magiccorner{}
	mc.inventory = map[string][]mtgban.InventoryEntry{}
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mc.exchangeRate = rate
	mc.client = NewMCClient()
	return &mc, nil
}

type resultChan struct {
	err   error
	cards []mtgban.InventoryEntry
}

func (mc *Magiccorner) printf(format string, a ...interface{}) {
	if mc.LogCallback != nil {
		mc.LogCallback("[MC] "+format, a...)
	}
}

func (mc *Magiccorner) processEntry(edition MCEdition) (res resultChan) {
	cards, err := mc.client.GetInventoryForEdition(edition)
	if err != nil {
		res.err = err
		return
	}

	printed := false

	// Keep track of the processed ids, and don't add duplicates
	duplicate := map[int]bool{}

	for _, card := range cards {
		if !printed && mc.VerboseLog {
			mc.printf("Processing id %d - %s (%s, code: %s)", edition.Id, edition.Set, card.Extra, card.Code)
			printed = true
		}

		// Skip lands, too many and without a simple solution
		isBasicLand := false
		switch card.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			isBasicLand = true
		}

		for i, v := range card.Variants {
			// Skip duplicate cards
			if duplicate[v.Id] {
				if mc.VerboseLog {
					mc.printf("Skipping duplicate card: %s (%s %s)", card.Name, card.Set, v.Foil)
				}
				continue
			}

			// Only keep English cards and a few other exceptions
			switch v.Language {
			case "EN":
			case "JP":
				if edition.Set != "War of the Spark: Japanese Alternate-Art Planeswalkers" {
					continue
				}
			case "IT":
				if edition.Id != mcRevisedEUFBBId && edition.Id != mcReinassanceId {
					continue
				}
			default:
				continue
			}

			if v.Quantity < 1 {
				continue
			}

			cond := v.Condition
			switch cond {
			case "NM/M":
				cond = "NM"
			case "SP", "HP":
			case "D":
				cond = "PO"
			default:
				mc.printf("Unknown '%s' condition", cond)
				continue
			}

			// The basic lands need custom handling for each edition if they
			// aren't found with other methods, ignore errors until they are
			// added to the variants table.
			printError := true
			if isBasicLand {
				printError = false
			}

			theCard, err := preprocess(&card, i)
			if err != nil {
				continue
			}

			cc, err := theCard.Match()
			if err != nil {
				if printError {
					mc.printf("%q", theCard)
					mc.printf("%q", card)
					mc.printf("%v", err)
				}
				continue
			}

			out := mtgban.InventoryEntry{
				Card:       mtgban.Card2card(cc),
				Conditions: cond,
				Price:      v.Price * mc.exchangeRate,
				Quantity:   v.Quantity,
				Notes:      "https://www.magiccorner.it" + card.URL,
			}

			res.cards = append(res.cards, out)

			duplicate[v.Id] = true
		}
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (mc *Magiccorner) scrape() error {
	editionList, err := mc.client.GetEditionList(true)
	if err != nil {
		return err
	}

	pages := make(chan MCEdition)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- mc.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for _, edition := range editionList {
			pages <- edition
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			mc.printf("%v", result.err)
			continue
		}
		for _, card := range result.cards {
			err = mtgban.InventoryAdd(mc.inventory, card)
			if err != nil {
				mc.printf(err.Error())
				continue
			}
		}
	}

	mc.InventoryDate = time.Now()

	return nil
}

func (mc *Magiccorner) Inventory() (map[string][]mtgban.InventoryEntry, error) {
	if len(mc.inventory) > 0 {
		return mc.inventory, nil
	}

	start := time.Now()
	mc.printf("Inventory scraping started at %s", start)

	err := mc.scrape()
	if err != nil {
		return nil, err
	}
	mc.printf("Inventory scraping took %s", time.Since(start))

	return mc.inventory, nil

}

func (mc *Magiccorner) Info() (info mtgban.ScraperInfo) {
	info.Name = "Magic Corner"
	info.Shorthand = "MC"
	info.InventoryTimestamp = mc.InventoryDate
	return
}
