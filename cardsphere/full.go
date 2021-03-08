package cardsphere

import (
	"fmt"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	baseUrl = "https://www.cardsphere.com/cards/"

	csMaxOffset = 80000
)

type CardsphereFull struct {
	LogCallback    mtgban.LogCallbackFunc
	buylistDate    time.Time
	MaxConcurrency int

	client  *CardSphereClient
	buylist mtgban.BuylistRecord
}

func NewScraperFull(email, password string) (*CardsphereFull, error) {
	cs := CardsphereFull{}
	cs.buylist = mtgban.BuylistRecord{}
	cs.MaxConcurrency = defaultConcurrency
	client, err := NewCardSphereClient(email, password)
	if err != nil {
		return nil, err
	}
	cs.client = client
	return &cs, nil
}

func (cs *CardsphereFull) printf(format string, a ...interface{}) {
	if cs.LogCallback != nil {
		cs.LogCallback("[CSF] "+format, a...)
	}
}

func (cs *CardsphereFull) processPage(results chan<- responseChan, offset int) error {
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

		cardName := offer.CardName
		edition := offer.Sets[0].Name
		foil := offer.Finishes[0] == "F"

		theCard, err := preprocess(cardName, edition)
		if err != nil {
			continue
		}
		theCard.Foil = foil

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			cs.printf("%v", err)
			cs.printf("%v", theCard)
			cs.printf("%v", offer)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
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

func (cs *CardsphereFull) parseBL() error {
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

func (cs *CardsphereFull) Buylist() (mtgban.BuylistRecord, error) {
	if len(cs.buylist) > 0 {
		return cs.buylist, nil
	}

	err := cs.parseBL()
	if err != nil {
		return nil, err
	}

	return cs.buylist, nil
}

func (cs *CardsphereFull) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardsphere"
	info.Shorthand = "CS"
	info.BuylistTimestamp = cs.buylistDate
	info.NoCredit = true
	info.MultiCondBuylist = true
	return
}
