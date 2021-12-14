package cardsphere

import (
	"errors"
	"fmt"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

const (
	defaultConcurrency = 4
	buylistURL         = "https://www.cardsphere.com/sets"
)

var gradingMap = map[string]float64{
	"NM": 1,
	"SP": 0.9,
	"MP": 0.75,
	"HP": 0.6,
}

type CardsphereIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	buylistDate    time.Time
	MaxConcurrency int

	buylist mtgban.BuylistRecord
}

func NewScraperIndex() *CardsphereIndex {
	cs := CardsphereIndex{}
	cs.buylist = mtgban.BuylistRecord{}
	cs.MaxConcurrency = defaultConcurrency
	return &cs
}

type responseChan struct {
	cardId  string
	blEntry *mtgban.BuylistEntry
}

func (cs *CardsphereIndex) printf(format string, a ...interface{}) {
	if cs.LogCallback != nil {
		cs.LogCallback("[CSphereIndex] "+format, a...)
	}
}

func (cs *CardsphereIndex) parseBL() error {
	results := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.cardsphere.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: cs.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//cs.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML(`ul[class="list-unstyled"]`, func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL("")
		isEditionList := strings.HasSuffix(link, "/sets")

		pageTitle := e.DOM.ParentsUntil("~").Find(`h3[class="text-center"]`).Text()
		edition := strings.TrimSpace(pageTitle)

		if isEditionList {
			e.ForEach(`li`, func(_ int, el *colly.HTMLElement) {
				//editionName := el.ChildText("a")
				editionPath := el.ChildAttr("a", "href")

				c.Visit(e.Request.AbsoluteURL(editionPath))
			})
			return
		}

		e.ForEach(`li`, func(_ int, el *colly.HTMLElement) {
			link := el.ChildAttr(`span:nth-child(1) a[class="cardpeek"]`, "href")

			cardName := el.ChildText("span:nth-child(1)")
			priceNonFoilStr := el.ChildText("span:nth-child(2)")
			priceFoilStr := el.ChildText("span:nth-child(3)")

			theCard, err := preprocess(cardName, edition)
			if err != nil {
				return
			}

			for i, priceStr := range []string{priceNonFoilStr, priceFoilStr} {
				price, err := mtgmatcher.ParsePrice(priceStr)
				if err != nil {
					continue
				}

				theCard.Foil = i == 1

				cardId, err := mtgmatcher.Match(theCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					continue
				} else if err != nil {
					cs.printf("%v", err)
					cs.printf("%v", theCard)
					cs.printf("%s | %s", cardName, edition)

					var alias *mtgmatcher.AliasingError
					if errors.As(err, &alias) {
						probes := alias.Probe()
						for _, probe := range probes {
							card, _ := mtgmatcher.GetUUID(probe)
							cs.printf("- %s", card)
						}
					}
					return
				}

				card, _ := mtgmatcher.GetUUID(cardId)
				if (!theCard.Foil && !card.HasFinish(mtgjson.FinishNonfoil)) ||
					(theCard.Foil && !card.HasFinish(mtgjson.FinishFoil)) {
					continue
				}

				out := responseChan{
					cardId: cardId,
					blEntry: &mtgban.BuylistEntry{
						BuyPrice: price,
						Quantity: 0,
						URL:      e.Request.AbsoluteURL(link),
					},
				}

				results <- out
			}
		})
	})

	c.Visit(buylistURL)

	go func() {
		c.Wait()
		close(results)
	}()

	lastTime := time.Now()

	for result := range results {
		err := cs.buylist.Add(result.cardId, result.blEntry)
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

func (cs *CardsphereIndex) Buylist() (mtgban.BuylistRecord, error) {
	if len(cs.buylist) > 0 {
		return cs.buylist, nil
	}

	err := cs.parseBL()
	if err != nil {
		return nil, err
	}

	return cs.buylist, nil
}

func grading(cardId string, entry mtgban.BuylistEntry) map[string]float64 {
	return gradingMap
}

func (cs *CardsphereIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardsphere Index"
	info.Shorthand = "CSphereIndex"
	info.BuylistTimestamp = cs.buylistDate
	info.Grading = grading
	info.NoCredit = true
	info.MetadataOnly = true
	return
}
