// Package cardsphere scrapes Cardsphere, whose prices come from standing
// offers rather than listings.
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
	baseURL            = "https://www.cardsphere.com/cards/"
	csMaxOffset        = 10000
)

var gradingMap = map[string]float64{
	"NM": 1,
	"SP": 0.9,
	"MP": 0.75,
	"HP": 0.6,
}

// Cardsphere prices what Cardsphere's members offer to pay, which is a set of
// standing offers rather than a storefront's buylist.
type Cardsphere struct {
	LogCallback    mtgban.LogCallbackFunc
	buylistDate    time.Time
	MaxConcurrency int

	client  *Client
	buylist mtgban.BuylistRecord
}

// NewScraper returns a scraper authenticated with the given token.
func NewScraper(token string) *Cardsphere {
	cs := Cardsphere{}
	cs.buylist = mtgban.BuylistRecord{}
	cs.MaxConcurrency = defaultConcurrency
	cs.client = NewClient(token)
	return &cs
}

func (cs *Cardsphere) printf(format string, a ...any) {
	if cs.LogCallback != nil {
		cs.LogCallback("[CS] "+format, a...)
	}
}

type responseChan struct {
	cardID  string
	blEntry *mtgban.BuylistEntry
}

func (cs *Cardsphere) processPage(ctx context.Context, results chan<- responseChan, offset int) error {
	offers, err := cs.client.GetOfferList(ctx, offset)
	if err != nil {
		return err
	}

	for _, offer := range offers {
		// Look for the right Id
		masterID := fmt.Sprint(offer.MasterId)
		ids, _ := mtgmatcher.SearchEquals(offer.CardName)
		if len(ids) == 0 {
			continue
		}

		for _, finish := range offer.Finishes {
			var foundID string
			for _, id := range ids {
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					continue
				}
				if (co.Identifiers["cardsphereId"] == masterID && finish != "F") ||
					(co.Identifiers["cardsphereFoilId"] == masterID && finish == "F") {
					foundID = id
					break
				}
			}
			if foundID == "" {
				continue
			}

			// Sets is decoded straight from the API and an offer may carry
			// none, so the finish cannot be read from it unconditionally
			var etched bool
			if len(offer.Sets) > 0 {
				etched = strings.Contains(offer.Sets[0].Name, "Etched")
			}

			cardID, err := mtgmatcher.MatchId(foundID, finish == "F", etched)
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
					cs.printf("Unsupported %s condition for %s", cond, foundID)
					continue
				}

				// Derive from the offer price rather than updating it in
				// place: the offer covers every condition it lists, so a
				// deduction applied here would carry into the next one
				condPrice := price * gradingMap[conditions]
				if int(condPrice*100) > offer.Balance {
					continue
				}

				out := responseChan{
					cardID: cardID,
					blEntry: &mtgban.BuylistEntry{
						// Account for processing fees and cash out fee
						BuyPrice:   condPrice * 0.87,
						Conditions: conditions,
						Quantity:   offer.Quantity,
						PriceRatio: priceRatio,
						URL:        fmt.Sprintf("%s%d", baseURL, offer.MasterId),
						VendorName: offer.UserDisplay,
					},
				}

				results <- out
			}
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
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
			entries := cs.buylist[result.cardID]
			for _, entry := range entries {
				if entry.Conditions == result.blEntry.Conditions {
					return
				}
			}

			err := cs.buylist.AddRelaxed(result.cardID, result.blEntry)
			if err != nil {
				cs.printf("%v", err)
				return
			}
			// This would be better with a select, but for now just print a message
			// that we're still alive every minute
			if time.Now().After(lastTime.Add(60 * time.Second)) {
				card, _ := mtgmatcher.GetUUID(result.cardID)
				cs.printf("Still going, last processed card: %s", card)
				lastTime = time.Now()
			}
		},
		cs.printf,
	)

	cs.buylistDate = time.Now()

	return nil
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (cs *Cardsphere) Buylist() mtgban.BuylistRecord {
	return cs.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (cs *Cardsphere) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardsphere"
	info.Shorthand = "CS"
	// Rebuild the cash out fee
	info.CreditMultiplier = 1.1
	info.BuylistTimestamp = &cs.buylistDate
	return
}
