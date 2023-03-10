package bigorbitcards

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
	inventoryURL       = "https://www.bigorbitcards.co.uk/magic-the-gathering/?features_hash=7-Y&items_per_page=48"
)

type BigOrbitCards struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	exchangeRate  float64
	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

func NewScraper() (*BigOrbitCards, error) {
	bigoc := BigOrbitCards{}
	bigoc.inventory = mtgban.InventoryRecord{}
	bigoc.MaxConcurrency = defaultConcurrency
	rate, err := mtgban.GetExchangeRate("GBP")
	if err != nil {
		return nil, err
	}
	// The API returns the inverse value for GBP
	bigoc.exchangeRate = 1 / rate

	return &bigoc, nil
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (bigoc *BigOrbitCards) printf(format string, a ...interface{}) {
	if bigoc.LogCallback != nil {
		bigoc.LogCallback("[bigoc] "+format, a...)
	}
}

func (bigoc *BigOrbitCards) scrape() error {
	results := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.bigorbitcards.co.uk"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: bigoc.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		bigoc.printf("Visiting %s", r.URL.String())
	})

	// Callback for the list of editions
	c.OnHTML(`div[class="ty-menu__submenu-item-header"]`, func(e *colly.HTMLElement) {
		edition := e.ChildText("div a")
		switch edition {
		case "Art Series",
			"Artefact",
			"Artifact",
			"Artifacts",
			"Black",
			"Blue",
			"Card Sets",
			"Colourless",
			"Double-Faced Proxy",
			"Foils",
			"Green",
			"Kilo of Cards",
			"Land",
			"Multi-Coloured",
			"Multi-coloured",
			"Red",
			"Repacks",
			"Scheme",
			"Token",
			"Tokens",
			"White":
			return
		default:
			if strings.Contains(edition, "/") {
				return
			}
			log.Println(edition)
		}
		link := e.ChildAttr("div a", "href")
		c.Visit(e.Request.AbsoluteURL(link))
	})

	// Callback for the list of cards in a single edition
	c.OnHTML(`div[class="ty-pagination"]`, func(e *colly.HTMLElement) {
		link := e.ChildAttr(`a[class="ty-pagination__item ty-pagination__btn ty-pagination__next cm-history cm-ajax ty-pagination__right-arrow"]`, "href")
		c.Visit(e.Request.AbsoluteURL(link))
	})

	// Callback for the list of offers of a single card
	c.OnHTML(`div[class="ty-compact-list"]`, func(e *colly.HTMLElement) {
		e.ForEach(`div[class="ty-compact-list__content"]`, func(_ int, el *colly.HTMLElement) {
			cardName := el.ChildText(`bdi a`)
			link := el.ChildAttr("a", "href")
			edition := el.ChildText(`div[style="margin-bottom:0;"] span span[class="ty-control-group__item"]`)
			priceAndItems := el.ChildText(`div[class="ty-control-group product-list-field"] div span span[style="margin-right:5px;"]`)
			if !strings.Contains(priceAndItems, "items") {
				return
			}
			qtyStr := strings.TrimSpace(strings.Split(priceAndItems, "items")[0])
			priceStr := strings.TrimPrefix(strings.TrimSpace(strings.Split(priceAndItems, "items")[1]), "£")

			theCard, err := preprocess(cardName, edition)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				return
			} else if err != nil {
				// Ignore errors from basic lands
				if mtgmatcher.IsBasicLand(cardName) {
					return
				}
				bigoc.printf("%v", err)
				bigoc.printf("%v", theCard)
				bigoc.printf("%s | %s", cardName, edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						bigoc.printf("- %s", card)
					}
				}
				return
			}

			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				bigoc.printf("%s", err.Error())
				return
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				bigoc.printf("%s", err.Error())
				return
			}

			if price == 0.0 || qty == 0 {
				return
			}

			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price * bigoc.exchangeRate,
					Quantity:   qty,
					URL:        link,
				},
			}

			results <- out
		})
	})

	c.Visit(inventoryURL)

	go func() {
		c.Wait()
		close(results)
	}()

	lastTime := time.Now()

	for result := range results {
		err := bigoc.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			bigoc.printf("%v", err)
			continue
		}
		// This would be better with a select, but for now just print a message
		// that we're still alive every minute
		if time.Now().After(lastTime.Add(60 * time.Second)) {
			card, _ := mtgmatcher.GetUUID(result.cardId)
			bigoc.printf("Still going, last processed card: %s", card)
			lastTime = time.Now()
		}
	}

	bigoc.inventoryDate = time.Now()

	return nil
}

func (bigoc *BigOrbitCards) Inventory() (mtgban.InventoryRecord, error) {
	if len(bigoc.inventory) > 0 {
		return bigoc.inventory, nil
	}

	err := bigoc.scrape()
	if err != nil {
		return nil, err
	}

	return bigoc.inventory, nil
}

func (bigoc *BigOrbitCards) Info() (info mtgban.ScraperInfo) {
	info.Name = "Big Orbit Cards"
	info.Shorthand = "BIGOC"
	info.InventoryTimestamp = &bigoc.inventoryDate
	return
}
