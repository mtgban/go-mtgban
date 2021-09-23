package strikezone

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/Magic_the_Gathering_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/List/MagicBuyList.txt"
)

type Strikezone struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Strikezone {
	sz := Strikezone{}
	sz.inventory = mtgban.InventoryRecord{}
	sz.buylist = mtgban.BuylistRecord{}
	sz.MaxConcurrency = defaultConcurrency
	return &sz
}

func (sz *Strikezone) printf(format string, a ...interface{}) {
	if sz.LogCallback != nil {
		sz.LogCallback("[SZ] "+format, a...)
	}
}

type respChan struct {
	cardId string
	entry  *mtgban.InventoryEntry
}

func (sz *Strikezone) scrape() error {
	channel := make(chan respChan)

	c := colly.NewCollector(
		colly.AllowedDomains("shop.strikezoneonline.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: sz.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//sz.printf("Visiting %s", r.URL.String())
	})

	// Callback for links on scraped pages (edition names)
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		if strings.Contains(link, "/Category/") &&
			!strings.HasSuffix(link, "_ByTable.html") &&
			!strings.HasSuffix(link, "_ByRarity.html") &&
			!strings.HasSuffix(link, "Games.html") &&
			!strings.HasSuffix(link, "Magic_Booster_Boxes.html") &&
			!strings.HasSuffix(link, "Fat_Packs.html") &&
			!strings.HasSuffix(link, "Preconstructed_Decks.html") {
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// Callback for when a scraped page contains a form element
	c.OnHTML("body", func(e *colly.HTMLElement) {
		var cardName, pathURL, notes, cond, qty, price string
		edition := e.ChildText("h1")

		e.ForEach("table.rtti tr", func(_ int, el *colly.HTMLElement) {
			cardName = el.ChildText("td:nth-child(1)")
			pathURL = el.ChildAttr("a", "href")
			notes = el.ChildText("td:nth-child(4)")
			cond = el.ChildText("td:nth-child(5)")
			qty = el.ChildText("td:nth-child(6)")
			price = el.ChildText("td:nth-child(7)")
			if cardName == "" {
				return
			}

			cardPrice, _ := strconv.ParseFloat(price, 64)
			if cardPrice <= 0 {
				return
			}

			quantity, _ := strconv.Atoi(qty)
			if quantity <= 0 {
				return
			}

			switch cond {
			case "Near Mint", "Mint":
				cond = "NM"
			case "Light Play":
				cond = "SP"
			case "Medium Play":
				cond = "MP"
			case "Heavy Play":
				cond = "HP"
			default:
				sz.printf("Unsupported %s condition", cond)
				return
			}

			// skip tokens, too many variations
			if strings.Contains(cardName, "Token") {
				return
			}

			theCard, err := preprocess(cardName, edition, notes)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				sz.printf("%v", err)
				sz.printf("%q", theCard)
				sz.printf("%s|%s|%s", cardName, edition, notes)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						sz.printf("- %s", card)
					}
				}
				return
			}

			channel <- respChan{
				cardId: cardId,
				entry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      cardPrice,
					Quantity:   quantity,
					URL:        "http://shop.strikezoneonline.com" + pathURL,
				},
			}
		})
	})

	c.Visit(szInventoryURL)

	go func() {
		c.Wait()
		close(channel)
	}()

	for resp := range channel {
		err := sz.inventory.Add(resp.cardId, resp.entry)
		if err != nil {
			sz.printf("%v", err)
			continue
		}
	}

	sz.inventoryDate = time.Now()

	return nil
}

func (sz *Strikezone) parseBL() error {
	resp, err := cleanhttp.DefaultClient().Get(szBuylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.Comma = '	'
	r.LazyQuotes = true

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 5 {
			return fmt.Errorf("unsupported buylist format (%d)", len(record))
		}

		cardName := strings.TrimSpace(record[1])
		edition := strings.TrimSpace(record[0])

		notes := strings.TrimSpace(record[2])

		quantity, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			return err
		}

		price, err := mtgmatcher.ParsePrice(record[4])
		if err != nil {
			return err
		}

		// skip invalid offers
		if price <= 0 || quantity <= 0 {
			continue
		}

		// skip duplicates, with less than NM conditions
		if strings.Contains(notes, "Play") {
			continue
		}

		theCard, err := preprocess(cardName, edition, notes)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			sz.printf("%v", err)
			sz.printf("%q", theCard)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					sz.printf("- %s", card)
				}
			}
			continue
		}

		var sellPrice, priceRatio float64

		invCards := sz.inventory[cardId]
		for _, invCard := range invCards {
			if invCard.Conditions == "NM" {
				sellPrice = invCard.Price
				break
			}
		}

		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		out := &mtgban.BuylistEntry{
			BuyPrice:   price,
			Quantity:   quantity,
			PriceRatio: priceRatio,
			URL:        "http://shop.strikezoneonline.com/TUser?MC=CUSTS&MF=B&BUID=637&ST=D&M=B&CMD=Search&T=" + url.QueryEscape(theCard.Name),
		}
		err = sz.buylist.Add(cardId, out)
		if err != nil {
			sz.printf("%v", err)
		}
	}

	sz.buylistDate = time.Now()

	return nil
}

func (sz *Strikezone) Inventory() (mtgban.InventoryRecord, error) {
	if len(sz.inventory) > 0 {
		return sz.inventory, nil
	}

	err := sz.scrape()
	if err != nil {
		return nil, err
	}

	return sz.inventory, nil
}

func (sz *Strikezone) Buylist() (mtgban.BuylistRecord, error) {
	if len(sz.buylist) > 0 {
		return sz.buylist, nil
	}

	err := sz.parseBL()
	if err != nil {
		return nil, err
	}

	return sz.buylist, nil
}

func grading(_ string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	return nil
}

func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	info.InventoryTimestamp = sz.inventoryDate
	info.BuylistTimestamp = sz.buylistDate
	info.Grading = grading
	info.NoCredit = true
	return
}
