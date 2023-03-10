package wizardscupboard

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	wcInventoryURL = "https://www.wizardscupboard.com/singles-c-100.html"
)

type Wizardscupboard struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
}

func NewScraper() *Wizardscupboard {
	wc := Wizardscupboard{}
	wc.inventory = mtgban.InventoryRecord{}
	wc.MaxConcurrency = defaultConcurrency
	return &wc
}

func (wc *Wizardscupboard) printf(format string, a ...interface{}) {
	if wc.LogCallback != nil {
		wc.LogCallback("[WC] "+format, a...)
	}
}

type respChan struct {
	cardId string
	entry  *mtgban.InventoryEntry
}

func (wc *Wizardscupboard) scrape() error {
	channel := make(chan respChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.wizardscupboard.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.URLFilters(
			regexp.MustCompile(`https://www.wizardscupboard.com/singles-.+`),
			regexp.MustCompile(`https://www.wizardscupboard.com/foils-.+`),
		),

		colly.Async(true),
	)

	// Callback for links on scraped pages (edition names)
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		u, _ := url.Parse(link)
		q := u.Query()
		if q.Get("osCsid") != "" {
			return
		}
		if q.Get("action") != "" {
			return
		}
		if q.Get("products_id") != "" {
			return
		}
		if q.Get("page") == "" {
			q.Set("page", "1")
		}

		q.Del("sort")
		q.Set("sort", "1a")

		u.RawQuery = q.Encode()
		link = u.String()

		if strings.HasPrefix(link, "https://www.wizardscupboard.com/singles-") ||
			strings.HasPrefix(link, "https://www.wizardscupboard.com/foils-") {

			err := c.Visit(e.Request.AbsoluteURL(link))
			if err != nil {
				if err != colly.ErrAlreadyVisited {
					wc.printf("error while linking: %s", err.Error())
				}
			}
		}
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: wc.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//log.Printf("Visiting %s\n", r.URL.String())
	})

	// Callback for when a scraped page contains a form element
	c.OnHTML(`table.productListing`, func(e *colly.HTMLElement) {
		e.ForEach(`tr`, func(_ int, elem *colly.HTMLElement) {
			if elem.ChildAttr("td", "class") != "productListing-data" {
				return
			}

			link := strings.TrimSpace(elem.ChildAttr("td:nth-child(1) a", "href"))
			cardName := strings.TrimSpace(elem.ChildText("td:nth-child(1)"))
			edition := strings.TrimSpace(elem.ChildText("td:nth-child(2)"))
			notes := strings.TrimSpace(elem.ChildText("td:nth-child(5)"))
			qtyStr := elem.ChildText("td:nth-child(6)")
			priceStr := strings.TrimSpace(elem.ChildText("td:nth-child(7)"))

			if priceStr == "" || qtyStr == "" {
				return
			}
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				wc.printf("%s %s", cardName, err.Error())
				return
			}

			if price <= 0 {
				return
			}

			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				wc.printf("%s %s", cardName, err.Error())
				return
			}

			if qty < 1 {
				return
			}

			conditions, err := parseConditions(notes)
			if err != nil {
				return
			}

			theCard, err := preprocess(cardName, edition, notes)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				return
			} else if err != nil {
				// Skip logging errors for basic lands
				if !mtgmatcher.IsBasicLand(cardName) {
					wc.printf("%v", err)
					wc.printf("%s", theCard)
					wc.printf("'%s' '%s' '%s'", cardName, edition, notes)

					var alias *mtgmatcher.AliasingError
					if errors.As(err, &alias) {
						probes := alias.Probe()
						for _, probe := range probes {
							card, _ := mtgmatcher.GetUUID(probe)
							wc.printf("- %s", card)
						}
					}
				}
				return
			}

			channel <- respChan{
				cardId: cardId,
				entry: &mtgban.InventoryEntry{
					Price:      price,
					Conditions: conditions,
					Quantity:   qty,
					URL:        link,
				},
			}
		})
	})

	c.Visit(wcInventoryURL)

	go func() {
		c.Wait()
		close(channel)
	}()

	dupes := map[string]bool{}

	for resp := range channel {
		key := resp.cardId + resp.entry.Conditions
		if dupes[key] {
			continue
		}
		dupes[key] = true

		err := wc.inventory.Add(resp.cardId, resp.entry)
		if err != nil {
			wc.printf("%v", err)
			continue
		}
	}

	wc.inventoryDate = time.Now()

	return nil
}

func (wc *Wizardscupboard) Inventory() (mtgban.InventoryRecord, error) {
	if len(wc.inventory) > 0 {
		return wc.inventory, nil
	}

	err := wc.scrape()
	if err != nil {
		return nil, err
	}

	return wc.inventory, nil
}

func (wc *Wizardscupboard) Info() (info mtgban.ScraperInfo) {
	info.Name = "Wizard's Cupboard"
	info.Shorthand = "WC"
	info.InventoryTimestamp = &wc.inventoryDate
	return
}
