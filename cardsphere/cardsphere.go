package cardsphere

import (
	"context"
	"fmt"
	"strings"
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

func (cs *Cardsphere) processPage(ctx context.Context, results chan<- responseChan, offset int) error {
	offers, err := cs.client.GetOfferList(ctx, offset)
	if err != nil {
		return err
	}

	for _, offer := range offers {
		// Look for the right Id
		masterId := fmt.Sprint(offer.MasterId)
		ids, _ := mtgmatcher.SearchEquals(offer.CardName)
		if len(ids) == 0 {
			continue
		}

		for _, finish := range offer.Finishes {
			var foundId string
			for _, id := range ids {
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					continue
				}
				if (co.Identifiers["cardsphereId"] == masterId && finish != "F") ||
					(co.Identifiers["cardsphereFoilId"] == masterId && finish == "F") {
					foundId = id
					break
				}
			}
			if foundId == "" {
				continue
			}

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

func (cs *Cardsphere) Load(ctx context.Context) error {
	offsets := make([]int, 0, csMaxOffset/100)
	for i := 0; i < csMaxOffset; i += 100 {
		offsets = append(offsets, i)
	}

	lastTime := time.Now()
	mtgban.WorkerPool(ctx, cs.MaxConcurrency, offsets,
		func(ctx context.Context, offset int, results chan<- responseChan) error {
			err := cs.processPage(ctx, results, offset)
			if err != nil {
				return fmt.Errorf("offset %d: %s", offset, err.Error())
			}
			time.Sleep(3 * time.Second)
			return nil
		},
		func(result responseChan) {
			// Only keep one offer per condition
			entries := cs.buylist[result.cardId]
			for _, entry := range entries {
				if entry.Conditions == result.blEntry.Conditions {
					return
				}
			}

			err := cs.buylist.AddRelaxed(result.cardId, result.blEntry)
			if err != nil {
				cs.printf("%v", err)
				return
			}
			// This would be better with a select, but for now just print a message
			// that we're still alive every minute
			if time.Now().After(lastTime.Add(60 * time.Second)) {
				card, _ := mtgmatcher.GetUUID(result.cardId)
				cs.printf("Still going, last processed card: %s", card)
				lastTime = time.Now()
			}
		},
		cs.printf,
	)

	cs.buylistDate = time.Now()

	return nil
}

func (cs *Cardsphere) Buylist() mtgban.BuylistRecord {
	return cs.buylist
}

func (cs *Cardsphere) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardsphere"
	info.Shorthand = "CS"
	// Rebuild the cash out fee
	info.CreditMultiplier = 1.1
	info.BuylistTimestamp = &cs.buylistDate
	return
}
