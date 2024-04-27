package cardsphere

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 4
	baseUrl            = "https://www.cardsphere.com/cards/"
	csMaxOffset        = 80000
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

func NewScraper(email, password string) (*Cardsphere, error) {
	cs := Cardsphere{}
	cs.buylist = mtgban.BuylistRecord{}
	cs.MaxConcurrency = defaultConcurrency
	client, err := NewCardSphereClient(email, password)
	if err != nil {
		return nil, err
	}
	cs.client = client
	return &cs, nil
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
	offers, err := cs.client.GetOfferListByMaxAbsolute(offset)
	if err != nil {
		return err
	}

	for _, offer := range offers {
		skip := true
		for _, lang := range offer.Languages {
			if lang == "EN" {
				skip = false
				break
			}
		}

		// When multiple printings are reported, it's impossible to tell
		// apart which price between max and mi belong to which edition
		if len(offer.Sets) > 1 || len(offer.Finishes) > 1 {
			skip = true
		}

		if skip {
			continue
		}

		theCard, err := preprocess(offer)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			cs.printf("%v", err)
			cs.printf("%v", theCard)
			cs.printf("%v", offer)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					cs.printf("- %s", card)
				}
			}
			break
		}

		price := float64(offer.MaxOffer) / 100
		indexPrice := float64(offer.MaxIndex) / 100
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
				cs.printf("Unsupported %s condition for %s", cond, theCard)
				continue
			}

			price *= gradingMap[conditions]
			if int(price*100) > offer.Balance {
				continue
			}

			out := responseChan{
				cardId: cardId,
				blEntry: &mtgban.BuylistEntry{
					BuyPrice:   price,
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
					cs.printf("%s", err.Error())
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
	info.BuylistTimestamp = &cs.buylistDate
	info.NoCredit = true
	return
}
