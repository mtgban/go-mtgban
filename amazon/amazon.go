package amazon

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 4
)

type Amazon struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord

	client *AMZClient
}

type respChan struct {
	cardId string
	entry  *mtgban.InventoryEntry
}

func NewScraper(token string) *Amazon {
	amz := Amazon{}
	amz.inventory = mtgban.InventoryRecord{}
	amz.client = NewAMZClient(token)
	amz.MaxConcurrency = defaultConcurrency
	return &amz
}

func (amz *Amazon) printf(format string, a ...interface{}) {
	if amz.LogCallback != nil {
		amz.LogCallback("[AMZ] "+format, a...)
	}
}

func (amz *Amazon) processUUIDs(channel chan<- respChan, ids []string) error {
	results, err := amz.client.GetPrices(ids)
	if err != nil {
		return err
	}

	for uuid, result := range results {
		for tag, price := range result {
			if price == 0 {
				continue
			}

			isFoil := strings.HasSuffix(tag, "Foil")
			cardId, err := mtgmatcher.MatchId(uuid, isFoil)
			if err != nil {
				amz.printf("%s - %s", uuid, err)
				continue
			}

			link := "http://greatermossdogapi.us-east-1.elasticbeanstalk.com/api/v1/purchase/" + uuid
			if strings.HasSuffix(tag, "Foil") {
				link += "/foil"
			} else {
				link += "/nonfoil"
			}

			out := respChan{
				cardId: cardId,
				entry: &mtgban.InventoryEntry{
					Conditions: strings.ToUpper(tag[:2]),
					Price:      price,
					Quantity:   1,
					URL:        link,
				},
			}

			channel <- out
		}
	}

	return nil
}

// Retrieve one price from a set at random to check the server is running
func (amz *Amazon) ping() error {
	sets := mtgmatcher.GetSets()
	for _, set := range sets {
		for _, card := range set.Cards {
			_, err := amz.client.GetPrices([]string{card.UUID})
			return err
		}
	}
	return nil
}

func (amz *Amazon) scrape() error {
	err := amz.ping()
	if err != nil {
		return errors.New("server unreachable")
	}

	pages := make(chan string)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	for i := 0; i < amz.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			buffer := make([]string, 0, 20)
			for uuid := range pages {
				buffer = append(buffer, uuid)
				if len(buffer) == 20 {
					err := amz.processUUIDs(channel, buffer)
					if err != nil {
						amz.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Spillover
			if len(buffer) != 0 {
				err := amz.processUUIDs(channel, buffer)
				if err != nil {
					amz.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetSets()
		for _, set := range sets {
			amz.printf("Processing %s", set.Name)
			for _, card := range set.Cards {
				pages <- card.UUID
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := amz.inventory.Add(record.cardId, record.entry)
		if err != nil {
			amz.printf("%v", err)
			continue
		}
	}

	amz.inventoryDate = time.Now()

	return nil
}

func (amz *Amazon) Inventory() (mtgban.InventoryRecord, error) {
	if len(amz.inventory) > 0 {
		return amz.inventory, nil
	}

	err := amz.scrape()
	if err != nil {
		return nil, err
	}

	return amz.inventory, nil
}

func (amz *Amazon) Info() (info mtgban.ScraperInfo) {
	info.Name = "Amazon"
	info.Shorthand = "AMZ"
	info.InventoryTimestamp = amz.inventoryDate
	info.NoQuantityInventory = true
	return
}
