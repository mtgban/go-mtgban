package trollandtoad

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	maxConcurrency = 8
)

type Trollandtoad struct {
	LogCallback mtgban.LogCallbackFunc
	BuylistDate time.Time

	buylist mtgban.BuylistRecord

	client *TATClient
}

func NewScraper() *Trollandtoad {
	tat := Trollandtoad{}
	tat.buylist = mtgban.BuylistRecord{}
	tat.client = NewTATClient()
	return &tat
}

type responseChan struct {
	card     *mtgdb.Card
	buyEntry *mtgban.BuylistEntry
}

func (tat *Trollandtoad) printf(format string, a ...interface{}) {
	if tat.LogCallback != nil {
		tat.LogCallback("[TaT] "+format, a...)
	}
}

func (tat *Trollandtoad) processPage(channel chan<- responseChan, id string) error {
	products, err := tat.client.ProductsForId(id)
	if err != nil {
		return err
	}

	for _, card := range products.Product {
		if !strings.Contains(card.Condition, "Near Mint") {
			continue
		}

		theCard, err := preprocess(card.Name, card.Edition)
		if err != nil {
			continue
		}

		cc, err := theCard.Match()
		if err != nil {
			switch {
			case strings.Contains(card.Edition, "World Championships"):
			default:
				tat.printf("%q", theCard)
				tat.printf("%s ~ %s", card.Name, card.Edition)
				tat.printf("%v", err)
			}
			continue
		}

		price, err := strconv.ParseFloat(card.BuyPrice, 64)
		if err != nil {
			tat.printf("%s %v", card.Name, err)
			continue
		}

		qty, err := strconv.Atoi(card.Quantity)
		if err != nil {
			tat.printf("%s %v", card.Name, err)
			continue
		}

		channel <- responseChan{
			card: cc,
			buyEntry: &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: price * 1.30,
				Quantity:   qty,
			},
		}
	}
	return nil
}

func (tat *Trollandtoad) parseBL() error {
	list, err := tat.client.ListEditions()
	if err != nil {
		return err
	}

	tat.printf("Processing %d editions", len(list))

	editions := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := tat.processPage(results, edition)
				if err != nil {
					tat.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, product := range list {
			// Bulk cards
			if product.CategoryId == "" {
				continue
			}
			editions <- product.CategoryId
		}
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := tat.buylist.Add(record.card, record.buyEntry)
		if err != nil {
			tat.printf(err.Error())
			continue
		}
	}

	return nil
}

func (tat *Trollandtoad) Buylist() (mtgban.BuylistRecord, error) {
	if len(tat.buylist) > 0 {
		return tat.buylist, nil
	}

	start := time.Now()
	tat.printf("Buylist scraping started at %s", start)

	err := tat.parseBL()
	if err != nil {
		return nil, err
	}
	tat.printf("Buylist scraping took %s", time.Since(start))

	return tat.buylist, nil
}

func (tat *Trollandtoad) Grading(card mtgdb.Card, entry mtgban.BuylistEntry) (grade map[string]float64) {
	switch {
	case card.Foil:
		grade = map[string]float64{
			"SP": 0.7, "MP": 0.5, "HP": 0.3,
		}
	}

	return
}

func (tat *Trollandtoad) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll and Toad"
	info.Shorthand = "TaT"
	info.BuylistTimestamp = tat.BuylistDate
	return
}
