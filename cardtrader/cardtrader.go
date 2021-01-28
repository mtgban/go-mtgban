package cardtrader

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type Cardtrader struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	MaxConcurrency int

	authClient  *CTAuthClient
	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	// Custom map of name:id to avoid requesting cards matching those names
	// Name should be normalized with mtgmatcher.Normalize()
	FilterNames map[string]string

	loggedClient *CTLoggedClient
}

func NewScraper(token string) *Cardtrader {
	ct := Cardtrader{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.marketplace = map[string]mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.authClient = NewCTAuthClient(token)
	return &ct
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

func (ct *Cardtrader) processEntry(channel chan<- resultChan, blueprintId int) error {
	filter, err := NewCTClient().ProductsForBlueprint(blueprintId)
	if err != nil {
		return err
	}

	// Skip anything that is not mtg singles
	if filter.Blueprint.CategoryId != 1 ||
		filter.Blueprint.GameId != 1 {
		return nil
	}

	var cardId string
	var cardIdFoil string

	for _, product := range filter.Products {
		switch product.Properties.Language {
		case "en":
		case "it":
			switch filter.Blueprint.Expansion.Name {
			case "Foreign Black Bordered":
			case "Rinascimento":
			default:
				continue
			}
		case "jp":
			switch {
			case filter.Blueprint.Expansion.Name == "Fourth Edition Black Bordered":
			case strings.Contains(filter.Blueprint.Expansion.Name, "Japanese"):
			default:
				continue
			}
		default:
			continue
		}

		if product.Quantity < 1 || product.OnVacation || product.Properties.Altered {
			continue
		}

		switch {
		case mtgmatcher.Contains(product.Description, "ita"),
			mtgmatcher.Contains(product.Description, "oversize"),
			mtgmatcher.Contains(product.Description, "inked"),
			mtgmatcher.Contains(product.Description, "mix"):
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
					card, _ := mtgmatcher.GetUUID(probe)
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
			category: blueprintId,
			cardId:   finalCardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      float64(product.Price.Cents) / 100,
				Quantity:   qty,
				URL:        link,
				SellerName: product.User.Name,
				Bundle:     product.User.Zero,
				OriginalId: fmt.Sprint(filter.Blueprint.Id),
				InstanceId: fmt.Sprint(product.Id),
			},
		}
	}

	return nil
}

func (ct *Cardtrader) scrape() error {
	blueprints, err := ct.authClient.Blueprints()
	if err != nil {
		return err
	}
	ct.printf("Parsing %d blueprints", len(blueprints))

	blueprintIds := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < ct.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for blueprintId := range blueprintIds {
				err := ct.processEntry(results, blueprintId)
				if err != nil {
					ct.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, bp := range blueprints {
			found := true
			if ct.FilterNames != nil {
				_, found = ct.FilterNames[mtgmatcher.Normalize(bp.Name)]
			}
			if found {
				blueprintIds <- bp.Id
			}
		}
		close(blueprintIds)

		wg.Wait()
		close(results)
	}()

	lastTime := time.Now()
	for result := range results {
		err := ct.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			ct.printf("%s", err.Error())
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			ct.printf("Still going %d/%d, last processed card: %s", result.category, blueprints[len(blueprints)-1].Id, card)
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
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	ct.marketplace = market
	ct.inventory = inventory

	ct.printf("Loaded inventory from file")

	return nil
}

func (ct *Cardtrader) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Trader"
	info.Shorthand = "CT"
	info.InventoryTimestamp = ct.InventoryDate
	return
}
