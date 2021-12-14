package cardtrader

import (
	"errors"
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
	ShareCode      string

	exchangeRate float64

	authClient  *CTAuthClient
	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	// Custom map of ids to consider
	FilterId map[string]string

	loggedClient *CTLoggedClient
	client       *CTClient
}

func NewScraper(token string) (*Cardtrader, error) {
	ct := Cardtrader{}
	ct.inventory = mtgban.InventoryRecord{}
	ct.marketplace = map[string]mtgban.InventoryRecord{}
	ct.MaxConcurrency = defaultConcurrency
	ct.client = NewCTClient()
	ct.authClient = NewCTAuthClient(token)

	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	ct.exchangeRate = rate

	return &ct, nil
}

func (ct *Cardtrader) printf(format string, a ...interface{}) {
	if ct.LogCallback != nil {
		ct.LogCallback("[CTF] "+format, a...)
	}
}

type resultChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

var condMap = map[string]string{
	"Mint":              "NM",
	"Near Mint":         "NM",
	"Slightly Played":   "SP",
	"Moderately Played": "MP",
	"Played":            "HP",
	"Heavily Played":    "HP",
	"Poor":              "PO",
}

func processProducts(channel chan<- resultChan, theCard *mtgmatcher.Card, products []Product, shareCode string, rate float64) error {
	var cardId string
	var cardIdFoil string

	for _, product := range products {
		switch product.Properties.Language {
		case "en":
		case "it":
			switch product.Expansion.Name {
			case "Foreign Black Bordered":
			case "Rinascimento":
			case "Legends Italian":
			case "The Dark Italian":
			default:
				continue
			}
		case "jp":
			switch product.Expansion.Name {
			case "Chronicles Japanese",
				"Fourth Edition Black Bordered",
				"Magic Premiere Shop",
				"Mystical Archive: Japanese alternate-art",
				"War of the Spark: Japanese alternate-art Planeswalker",
				"War of the Spark: Japanese alternate-art Planeswalker Promo":
			case "Ikoria: Lair of Behemoths Collectors":
				switch theCard.Name {
				case "Crystalline Giant",
					"Battra, Dark Destroyer",
					"Mothra's Great Cocoon":
				default:
					continue
				}
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
			mtgmatcher.Contains(product.Description, "stamp"),
			mtgmatcher.Contains(product.Description, "mix"):
			continue
		}

		var err error
		if cardId == "" {
			cardId, err = mtgmatcher.Match(theCard)
			if err != nil {
				return err
			}
		}

		if cardIdFoil == "" && product.Properties.Foil && mtgmatcher.HasFoilPrinting(theCard.Name) {
			theCard.Foil = true
			cardIdFoil, err = mtgmatcher.Match(theCard)
			if err != nil {
				return err
			}
		}

		cond := product.Properties.Condition
		if product.Properties.Signed {
			cond = "Heavily Played"
		}
		if product.Properties.Altered {
			continue
		}
		conditions, found := condMap[cond]
		if !found {
			return fmt.Errorf("Unsupported %s condition", cond)
		}

		finalCardId := cardId
		if product.Properties.Foil {
			finalCardId = cardIdFoil
		}

		qty := product.Quantity
		if product.Bundle {
			qty *= 4
		}

		link := "https://www.cardtrader.com/cards/" + fmt.Sprint(product.BlueprintId)
		if shareCode != "" {
			link += "?share_code=" + shareCode
		}

		price := float64(product.Price.Cents) / 100
		if product.Price.Currency != "USD" {
			price *= rate
		}
		channel <- resultChan{
			cardId: finalCardId,
			invEntry: &mtgban.InventoryEntry{
				Conditions: conditions,
				Price:      price,
				Quantity:   qty,
				URL:        link,
				SellerName: product.User.Name,
				Bundle:     product.User.Zero,
				OriginalId: fmt.Sprint(product.BlueprintId),
				InstanceId: fmt.Sprint(product.Id),
			},
		}
	}

	return nil
}

func (ct *Cardtrader) processEntry(channel chan<- resultChan, blueprintId int) error {
	filter, err := ct.client.ProductsForBlueprint(blueprintId)
	if err != nil {
		return err
	}

	theCard, err := Preprocess(&filter.Blueprint)
	if err != nil {
		return nil
	}

	err = processProducts(channel, theCard, filter.Products, ct.ShareCode, ct.exchangeRate)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		ct.printf("%q", theCard)
		ct.printf("%d %q", filter.Blueprint.Id, filter.Blueprint)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := mtgmatcher.GetUUID(probe)
				ct.printf("- %s", card)
			}
		}
	}
	return err
}

func (ct *Cardtrader) scrape() error {
	expansionsRaw, err := ct.authClient.Expansions()
	if err != nil {
		return err
	}
	ct.printf("Retrieved %d expansions", len(expansionsRaw))

	var blueprintsRaw []Blueprint
	for _, exp := range expansionsRaw {
		if exp.GameId != GameIdMagic {
			continue
		}

		bp, err := ct.authClient.Blueprints(exp.Id)
		if err != nil {
			return err
		}
		blueprintsRaw = append(blueprintsRaw, bp...)
	}
	total := len(blueprintsRaw)
	ct.printf("Found %d blueprints", total)

	blueprints, _ := FormatBlueprints(blueprintsRaw, expansionsRaw)

	if ct.FilterId != nil {
		total = len(ct.FilterId)
		ct.printf("Filtering to %d entries", total)
	}

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
			if ct.FilterId != nil {
				theCard, err := Preprocess(bp)
				if err != nil {
					continue
				}
				cardId, err := mtgmatcher.Match(theCard)
				if err != nil {
					continue
				}

				_, found := ct.FilterId[cardId]
				_, foundFoil := ct.FilterId[cardId+"_f"]
				if !found && !foundFoil {
					continue
				}
			}

			blueprintIds <- bp.Id
		}
		close(blueprintIds)

		wg.Wait()
		close(results)
	}()

	lastTime := time.Now()
	current := 0
	for result := range results {
		if ct.inventory[result.cardId] == nil {
			if !strings.HasSuffix(result.cardId, "_f") {
				current++
			}
		}
		err := ct.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			ct.printf("%s", err.Error())
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			ct.printf("Still going %d/%d, last processed card: %s", current, total, card)
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

func (ct *Cardtrader) InitializeInventory(reader io.Reader) error {
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
	info.Name = "Card Trader Full"
	info.Shorthand = "CTF"
	info.InventoryTimestamp = ct.InventoryDate
	return
}
