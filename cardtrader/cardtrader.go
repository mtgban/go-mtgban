package cardtrader

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	maxCategoryId = 135095

	defaultConcurrency = 8
)

type Cardtrader struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	MaxConcurrency int

	ids         []int
	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
}

func NewScraper(reader io.Reader) (*Cardtrader, error) {
	ct := Cardtrader{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.marketplace = map[string]mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency

	if reader != nil {
		d := json.NewDecoder(reader)
		err := d.Decode(&ct.ids)
		if err != nil {
			return nil, err
		}
	} else {
		ct.ids = make([]int, maxCategoryId)
		for i := range ct.ids {
			ct.ids[i] = i + 1
		}
	}

	return &ct, nil
}

func (ct *Cardtrader) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CT] "+format, a...)
	}
}

type resultChan struct {
	category int
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (ct *Cardtrader) processEntry(channel chan<- resultChan, categoryId int) error {
	filter, err := NewCTClient().GetBlueprints(categoryId)
	if err != nil {
		return err
	}

	// Skip anything that is not mtg singles
	if filter.Blueprint.CategoryId != 1 ||
		filter.Blueprint.GameId != 1 {
		return nil
	}

	if filter.Blueprint.Properties.Language != "en" {
		switch filter.Blueprint.Properties.Language {
		case "it":
			switch filter.Blueprint.Expansion.Name {
			case "Foreign Black Bordered":
			case "Rinascimento":
			default:
				return nil
			}
		case "jp":
			switch {
			case strings.Contains(filter.Blueprint.Expansion.Name, "Japanese"):
			case filter.Blueprint.Expansion.Name == "Fourth Edition Black Bordered":
			default:
				return nil
			}
		default:
			return nil
		}
	}

	var cardId string
	var cardIdFoil string

	for _, product := range filter.Products {
		if product.Properties.Language != "en" || product.Properties.Altered {
			continue
		}

		if product.Quantity < 1 || product.OnVacation {
			continue
		}

		switch {
		case strings.Contains(strings.ToLower(product.Description), "italian"),
			strings.ToLower(product.Description) == "ita",
			strings.Contains(strings.ToLower(product.Description), "oversize"):
			continue
		}

		theCard, err := preprocess(filter.Blueprint)
		if err != nil {
			continue
		}

		if cardId == "" {
			cardId, err = mtgmatcher.Match(theCard)
		}
		if cardIdFoil == "" && product.Properties.Foil {
			theCard.Foil = true
			cardIdFoil, err = mtgmatcher.Match(theCard)
		}
		if err != nil {
			ct.printf("%v", theCard)
			ct.printf("%q", theCard)
			ct.printf("%d %q", filter.Blueprint.Id, filter.Blueprint)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.Unmatch(probe)
					ct.printf("- %s", card)
				}
			}
			return err
		}

		conditions := product.Properties.Condition
		if product.Properties.Signed {
			conditions = "HP"
		}
		switch conditions {
		case "Near Mint", "Mint":
			conditions = "NM"
		case "Slightly Played":
			conditions = "SP"
		case "Moderately Played", "Played":
			conditions = "MP"
		case "Heavily Played", "HP":
			conditions = "HP"
		case "Poor":
			conditions = "PO"
		default:
			ct.printf("Unsupported %s condition", conditions)
			continue
		}

		finalCardId := cardId
		if product.Properties.Foil {
			finalCardId = cardIdFoil
		}

		qty := product.Quantity
		if product.Bundle {
			qty *= 4
		}

		link := "https://www.cardtrader.com/cards/" + fmt.Sprint(filter.Blueprint.Id)

		channel <- resultChan{
			category: categoryId,
			cardId:   finalCardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      float64(product.Price.Cents) / 100,
				Quantity:   qty,
				URL:        link,
				SellerName: product.User.Name,
				Bundle:     product.User.Zero,
			},
		}
	}

	return nil
}

func (ct *Cardtrader) scrape() error {
	categories := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for categoryId := range categories {
				err := ct.processEntry(results, categoryId)
				if err != nil {
					ct.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, id := range ct.ids {
			categories <- id
		}
		close(categories)

		wg.Wait()
		close(results)
	}()

	lastTime := time.Now()
	for result := range results {
		err := ct.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			ct.printf(err.Error())
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.Unmatch(result.cardId)
			ct.printf("Still going %d/%d, last processed card: %s", result.category, ct.ids[len(ct.ids)-1], card)
			lastTime = time.Now()
		}
	}

	ct.InventoryDate = time.Now()

	return nil
}

func (ct *Cardtrader) Inventory() (mtgban.InventoryRecord, error) {
	if len(ct.inventory) > 0 {
		return ct.inventory, nil
	}

	err := ct.scrape()
	if err != nil {
		return nil, err
	}

	return ct.inventory, nil
}

func (ct *Cardtrader) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(ct.inventory) == 0 {
		_, err := ct.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := ct.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range ct.inventory {
		for i := range ct.inventory[card] {
			if ct.inventory[card][i].SellerName == sellerName {
				if ct.inventory[card][i].Price == 0 {
					continue
				}
				if ct.marketplace[sellerName] == nil {
					ct.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				ct.marketplace[sellerName][card] = append(ct.marketplace[sellerName][card], ct.inventory[card][i])
			}
		}
	}

	if len(ct.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return ct.marketplace[sellerName], nil
}

func (ct *Cardtrader) IntializeInventory(reader io.Reader) error {
	inventory, err := mtgban.LoadInventoryFromCSV(reader)
	if err != nil {
		return err
	}

	ct.inventory = mtgban.InventoryRecord{}
	for card := range inventory {
		ct.inventory[card] = inventory[card]

		for i := range ct.inventory[card] {
			sellerName := ct.inventory[card][i].SellerName
			if ct.marketplace[sellerName] == nil {
				ct.marketplace[sellerName] = mtgban.InventoryRecord{}
			}
			ct.marketplace[sellerName][card] = append(ct.marketplace[sellerName][card], ct.inventory[card][i])
		}
	}
	if len(ct.inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}
	ct.printf("Loaded inventory from file")

	return nil
}

func (ct *Cardtrader) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CT"
	info.InventoryTimestamp = ct.InventoryDate
	return
}
