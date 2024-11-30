package cardsphere

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 2
	baseUrl            = "https://www.cardsphere.com/cards/"
	csMaxOffset        = 10000
)

var gradingMap = map[string]float64{
	"NM": 1,
	"SP": 0.9,
	"MP": 0.75,
	"HP": 0.6,
}

type Cardsphere struct {
	LogCallback    mtgban.LogCallbackFunc
	buylistDate    time.Time
	MaxConcurrency int

	client  *CardSphereClient
	buylist mtgban.BuylistRecord
}

func NewScraper(token string) *Cardsphere {
	cs := Cardsphere{}
	cs.buylist = mtgban.BuylistRecord{}
	cs.MaxConcurrency = defaultConcurrency
	cs.client = NewCardSphereClient(token)
	return &cs
}

func (cs *Cardsphere) printf(format string, a ...interface{}) {
	if cs.LogCallback != nil {
		cs.LogCallback("[CS] "+format, a...)
	}
}

type responseChan struct {
	cardId  string
	blEntry *mtgban.BuylistEntry
}

func (cs *Cardsphere) processPage(results chan<- responseChan, offset int) error {
	offers, err := cs.client.GetOfferList(offset)
	if err != nil {
		return err
	}

	for _, offer := range offers {
		// Look for the right Id
		ids, _ := mtgmatcher.SearchEquals(offer.CardName)
		if len(ids) == 0 {
			continue
		}

		var foundId string
		for _, id := range ids {
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				continue
			}
			if co.Identifiers["cardsphereId"] == fmt.Sprint(offer.MasterId) {
				foundId = id
				break
			}
		}

		if foundId == "" {
			continue
		}

		for _, finish := range offer.Finishes {
			cardId, err := mtgmatcher.MatchId(foundId, finish == "F", strings.Contains(offer.Sets[0].Name, "Etched"))
			if err != nil {
				continue
			}

			price := float64(offer.MaxOffer) / 100.0
			indexPrice := float64(offer.MaxIndex) / 100.0
			var priceRatio float64
			if indexPrice > 0 {
				priceRatio = price / indexPrice * 100
			}

			for _, cond := range offer.Conditions {
				conditions := ""
				switch cond {
				case 40:
					conditions = "NM"
				case 30:
					conditions = "SP"
				case 20:
					conditions = "MP"
				case 10:
					conditions = "HP"
				default:
					cs.printf("Unsupported %s condition for %s", cond, foundId)
					continue
				}

				price *= gradingMap[conditions]
				if int(price*100) > offer.Balance {
					continue
				}

				out := responseChan{
					cardId: cardId,
					blEntry: &mtgban.BuylistEntry{
						// Account for processing fees and cash out fee
						BuyPrice:   price * 0.87,
						Conditions: conditions,
						Quantity:   offer.Quantity,
						PriceRatio: priceRatio,
						URL:        fmt.Sprintf("%s%d", baseUrl, offer.MasterId),
						VendorName: offer.UserDisplay,
					},
				}

				results <- out
			}
		}
	}

	return nil
}

func (cs *Cardsphere) parseBL() error {
	results := make(chan responseChan)
	offsets := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < cs.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for offset := range offsets {
				err := cs.processPage(results, offset)
				if err != nil {
					cs.printf("offset %d: %s", offset, err.Error())
				}
				time.Sleep(3 * time.Second)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < csMaxOffset; i += 100 {
			offsets <- i
		}
		close(offsets)

		wg.Wait()
		close(results)
	}()

	lastTime := time.Now()
	for result := range results {
		// Only keep one offer per condition
		var skip bool
		entries := cs.buylist[result.cardId]
		for _, entry := range entries {
			if entry.Conditions == result.blEntry.Conditions {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		err := cs.buylist.AddRelaxed(result.cardId, result.blEntry)
		if err != nil {
			cs.printf("%v", err)
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			cs.printf("Still going, last processed card: %s", card)
			lastTime = time.Now()
		}
	}

	cs.buylistDate = time.Now()

	return nil
}

func (cs *Cardsphere) Buylist() (mtgban.BuylistRecord, error) {
	if len(cs.buylist) > 0 {
		return cs.buylist, nil
	}

	err := cs.parseBL()
	if err != nil {
		return nil, err
	}

	return cs.buylist, nil
}

func (cs *Cardsphere) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardsphere"
	info.Shorthand = "CS"
	// Rebuild the cash out fee
	info.CreditMultiplier = 1.1
	info.BuylistTimestamp = &cs.buylistDate
	return
}
