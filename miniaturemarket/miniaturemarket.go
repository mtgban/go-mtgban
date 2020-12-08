package miniaturemarket

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	Affiliate string
	client    *MMClient
	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.client = NewMMClient()
	mm.inventory = mtgban.InventoryRecord{}
	mm.buylist = mtgban.BuylistRecord{}
	mm.MaxConcurrency = defaultConcurrency
	return &mm
}

const (
	defaultConcurrency = 6

	firstPage = 1
	lastPage  = 10
)

type respChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (mm *Miniaturemarket) printf(format string, a ...interface{}) {
	if mm.LogCallback != nil {
		mm.LogCallback("[MM] "+format, a...)
	}
}

func (mm *Miniaturemarket) processPage(channel chan<- respChan, start int) error {
	resp, err := mm.client.GetInventory(start)
	if err != nil {
		return nil
	}

	for _, product := range resp.Response.Products {
		product.UUID = strings.TrimSuffix(product.UUID, "-ROOT")

		theCard, err := preprocess(product.Title, product.UUID)
		if err != nil {
			continue
		}

		for _, variant := range product.Variants {
			if variant.Quantity == 0 || variant.Price <= 0 {
				continue
			}

			theCard.Foil = strings.HasPrefix(variant.Title, "Foil")

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				mm.printf("%v", err)
				mm.printf("%q", theCard)
				mm.printf("%q", product)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						mm.printf("- %s", card)
					}
				}
				continue
			}

			cond := variant.Title
			switch cond {
			case "Near Mint", "Foil Near Mint", "Foil Near MInt":
				cond = "NM"
			case "Played", "Foil Played":
				cond = "MP"
			default:
				mm.printf("Unsupported %s condition", cond)
				continue
			}

			link := product.URL
			if mm.Affiliate != "" {
				link += "?utm_source=" + mm.Affiliate + "&utm_medium=feed&utm_campaign=mtg_singles"
			}

			channel <- respChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      variant.Price,
					Quantity:   variant.Quantity,
					URL:        link,
				},
			}
		}
	}

	return nil
}

func (mm *Miniaturemarket) scrape() error {
	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	totalProducts, err := mm.client.NumberOfProducts()
	if err != nil {
		return err
	}
	mm.printf("Parsing %d items", totalProducts)

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for start := range pages {
				err = mm.processPage(channel, start)
				if err != nil {
					mm.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < totalProducts; i += MMDefaultResultsPerPage {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := mm.inventory.Add(record.cardId, record.invEntry)
		// Do not print an error if we expect a duplicate due to the sorting
		if err != nil {
			mm.printf("%v", err)
			continue
		}
	}

	mm.inventoryDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) processEntry(channel chan<- respChan, page int) error {
	buyback, err := mm.client.BuyBackPage(MMCategoryMtgSingles, page)
	if err != nil {
		return err
	}

	for _, card := range buyback {
		if card.MtgCondition == "" ||
			card.MtgSet == "Bulk MTG" ||
			card.MtgRarity == "Sealed Product" {
			continue
		}

		switch card.MtgCondition {
		case "Near Mint", "Foil Near Mint", "Foil Near MInt":
		default:
			mm.printf("Unsupported %s condition", card.MtgCondition)
			continue
		}

		price, err := strconv.ParseFloat(card.Price, 64)
		if err != nil {
			return err
		}

		if price <= 0 {
			continue
		}

		theCard, err := preprocess(card.Name, card.SKU)
		if err != nil {
			continue
		}

		theCard.Foil = card.IsFoil

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			mm.printf("%v", err)
			mm.printf("%q", theCard)
			mm.printf("%q", card)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					mm.printf("- %s", card)
				}
			}
			continue
		}

		var priceRatio, sellPrice float64

		invCards := mm.inventory[cardId]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		channel <- respChan{
			cardId: cardId,
			buyEntry: &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: price * 1.3,
				Quantity:   0,
				PriceRatio: priceRatio,
				URL:        "https://www.miniaturemarket.com/buyback/",
			},
		}
	}

	return nil
}

func (mm *Miniaturemarket) parseBL() error {
	pages := make(chan int)
	results := make(chan respChan)
	var wg sync.WaitGroup

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := mm.processEntry(results, page)
				if err != nil {
					mm.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := firstPage; i <= lastPage; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		err := mm.buylist.Add(result.cardId, result.buyEntry)
		if err != nil {
			mm.printf("%s", err.Error())
			continue
		}
	}

	mm.buylistDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(mm.inventory) > 0 {
		return mm.inventory, nil
	}

	err := mm.scrape()
	if err != nil {
		return nil, err
	}

	return mm.inventory, nil
}

func (mm *Miniaturemarket) Buylist() (mtgban.BuylistRecord, error) {
	if len(mm.buylist) > 0 {
		return mm.buylist, nil
	}

	err := mm.parseBL()
	if err != nil {
		return nil, err
	}

	return mm.buylist, nil
}

func grading(_ string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	grade = map[string]float64{
		"SP": 0.75, "MP": 0, "HP": 0,
	}
	if entry.BuyPrice <= 0.08 {
		grade = map[string]float64{
			"SP": 0.4, "MP": 0, "HP": 0,
		}
	} else if entry.BuyPrice <= 0.1 {
		grade = map[string]float64{
			"SP": 0.5, "MP": 0, "HP": 0,
		}
	} else if entry.BuyPrice <= 0.15 {
		grade = map[string]float64{
			"SP": 0.66, "MP": 0, "HP": 0,
		}
	}
	return
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MM"
	info.InventoryTimestamp = mm.inventoryDate
	info.BuylistTimestamp = mm.buylistDate
	info.Grading = grading
	return
}
