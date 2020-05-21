package strikezone

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	maxConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/Magic_the_Gathering_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/List/MagicBuyList.txt"
)

type Strikezone struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time
	BuylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Strikezone {
	sz := Strikezone{}
	sz.inventory = mtgban.InventoryRecord{}
	sz.buylist = mtgban.BuylistRecord{}
	return &sz
}

func (sz *Strikezone) printf(format string, a ...interface{}) {
	if sz.LogCallback != nil {
		sz.LogCallback("[SZ] "+format, a...)
	}
}

type respChan struct {
	card  *mtgdb.Card
	entry *mtgban.InventoryEntry
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
		Parallelism: maxConcurrency,
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
			case "Near Mint":
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

			cc, err := theCard.Match()
			if err != nil {
				sz.printf("%q", theCard)
				sz.printf("%v", err)
				return
			}

			channel <- respChan{
				card: cc,
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
		err := sz.inventory.Add(resp.card, resp.entry)
		if err != nil {
			sz.printf("%v", err)
			continue
		}
	}

	sz.InventoryDate = time.Now()

	return nil
}

func (sz *Strikezone) parseBL() error {
	resp, err := http.Get(szBuylistURL)
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
			return fmt.Errorf("Unsupported buylist format (%d)", len(record))
		}

		cardName := strings.TrimSpace(record[1])
		edition := strings.TrimSpace(record[0])

		notes := strings.TrimSpace(record[2])

		quantity, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			return err
		}

		priceStr := strings.TrimSpace(record[4])
		priceStr = strings.Replace(priceStr, ",", "", 1)
		price, err := strconv.ParseFloat(priceStr, 64)
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

		cc, err := theCard.Match()
		if err != nil {
			sz.printf("%q", theCard)
			sz.printf("%v", err)
			continue
		}

		var sellPrice, priceRatio float64

		invCards := sz.inventory[*cc]
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
			TradePrice: price * 1.3,
			Quantity:   quantity,
			PriceRatio: priceRatio,
			URL:        "http://shop.strikezoneonline.com/TUser?MC=CUSTS&MF=B&BUID=637&ST=D&M=B&CMD=Search&T=" + theCard.Name,
		}
		err = sz.buylist.Add(cc, out)
		if err != nil {
			sz.printf("%v", err)
		}
	}

	sz.BuylistDate = time.Now()

	return nil
}

func (sz *Strikezone) Inventory() (mtgban.InventoryRecord, error) {
	if len(sz.inventory) > 0 {
		return sz.inventory, nil
	}

	start := time.Now()
	sz.printf("Inventory scraping started at %s", start)

	err := sz.scrape()
	if err != nil {
		return nil, err
	}
	sz.printf("Inventory scraping took %s", time.Since(start))

	return sz.inventory, nil
}

func (sz *Strikezone) Buylist() (mtgban.BuylistRecord, error) {
	if len(sz.buylist) > 0 {
		return sz.buylist, nil
	}

	start := time.Now()
	sz.printf("Buylist scraping started at %s", start)

	err := sz.parseBL()
	if err != nil {
		return nil, err
	}
	sz.printf("Buylist scraping took %s", time.Since(start))

	return sz.buylist, nil
}

func (sz *Strikezone) Grading(card mtgdb.Card, entry mtgban.BuylistEntry) (grade map[string]float64) {
	return nil
}

func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	info.InventoryTimestamp = sz.InventoryDate
	info.BuylistTimestamp = sz.BuylistDate
	return
}
