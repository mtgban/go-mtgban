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
	"github.com/kodabb/go-mtgban/mtgjson"
)

const (
	maxConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/Magic_the_Gathering_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/List/MagicBuyList.txt"
)

var untaggedTags = []string{
	"2011 Holiday",
	"2015 Judge Promo",
	"BIBB",
	"Convention Foil M19",
	"Draft Weekend",
	"FNM",
	"Full Box Promo",
	"GP Promo",
	"Grand Prix 2018",
	"Holiday Promo",
	"Judge Promo",
	"Judge 2020",
	"League Promo",
	"MagicFest 2019",
	"MagicFest 2020",
	"MCQ Promo",
	"Media Promo",
	"Players Tour Qualifier PTQ Promo",
	"Prerelease",
	"SDCC 2015",
	"Shooting Star Promo",
	"Standard Showdown 2017",
	"Store Champ",
	"Store Championship",
}

// StrikezoneBuylist is the Scraper for the Strikezone Online vendor.
type Strikezone struct {
	LogCallback mtgban.LogCallbackFunc

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

// NewBuylist initializes a Scraper for retriving buylist information, using
// the passed-in client to make http connections.
func NewScraper(db mtgjson.MTGDB) *Strikezone {
	sz := Strikezone{}
	sz.db = db
	sz.inventory = map[string][]mtgban.InventoryEntry{}
	sz.buylist = map[string]mtgban.BuylistEntry{}
	sz.norm = mtgban.NewNormalizer()
	return &sz
}

func (sz *Strikezone) printf(format string, a ...interface{}) {
	if sz.LogCallback != nil {
		sz.LogCallback(format, a...)
	}
}

func (sz *Strikezone) scrape() error {
	channel := make(chan szCard)

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
		var cardName, foil, cond, qty, price string
		edition := e.ChildText("h1")

		e.ForEach("table.rtti tr", func(_ int, el *colly.HTMLElement) {
			cardName = el.ChildText("td:nth-child(1)")
			foil = el.ChildText("td:nth-child(4)")
			cond = el.ChildText("td:nth-child(5)")
			qty = el.ChildText("td:nth-child(6)")
			price = el.ChildText("td:nth-child(7)")
			if cardName == "" {
				return
			}

			cardPrice, _ := strconv.ParseFloat(price, 64)
			quantity, _ := strconv.Atoi(qty)
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

			isFoil := foil == "Foil"

			cn, found := cardTable[cardName]
			if found {
				cardName = cn
			}

			// Sometimes the buylist specifies tags at the end of the card name,
			// but without parenthesis, so make sure they are present.
			for _, tag := range untaggedTags {
				if strings.HasSuffix(cardName, tag) {
					cardName = strings.Replace(cardName, tag, "("+tag+")", 1)
					break
				}
			}

			card := szCard{
				Name:       cardName,
				Edition:    edition,
				IsFoil:     isFoil,
				Conditions: cond,
				Price:      cardPrice,
				Quantity:   quantity,
			}

			channel <- card
		})
	})

	c.Visit(szInventoryURL)

	go func() {
		c.Wait()
		close(channel)
	}()

	for card := range channel {
		cc, err := sz.convert(&card)
		if err != nil {
			sz.printf("%v", err)
			continue
		}

		if card.Quantity > 0 && card.Price > 0 {
			out := mtgban.InventoryEntry{
				Card:       *cc,
				Conditions: card.Conditions,
				Price:      card.Price,
				Quantity:   card.Quantity,
			}
			err := sz.InventoryAdd(out)
			if err != nil {
				sz.printf("%v", err)
				continue
			}
		}
	}

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
		cardSet := strings.TrimSpace(record[0])

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
		if price <= 0 {
			continue
		}

		// skip duplicates, with less than NM conditions
		if strings.Contains(notes, "Play") {
			continue
		}

		// skip tokens, too many variations
		if strings.Contains(cardName, "Token") {
			continue
		}

		isFoil := strings.Contains(notes, "Foil")

		cn, found := cardTable[cardName]
		if found {
			cardName = cn
		}

		// Sometimes the buylist specifies tags at the end of the card name,
		// but without parenthesis, so make sure they are present.
		for _, tag := range untaggedTags {
			if strings.HasSuffix(cardName, tag) {
				cardName = strings.Replace(cardName, tag, "("+tag+")", 1)
				break
			}
		}

		switch {
		case strings.HasPrefix(cardName, "Snow-Cover "):
			cardName = strings.Replace(cardName, "Snow-Cover ", "Snow-Covered ", 1)
		}

		card := &szCard{
			Name:    cardName,
			Edition: cardSet,
			IsFoil:  isFoil,
		}

		cc, err := sz.convert(card)
		if err != nil {
			sz.printf("%v", err)
			continue
		}

		if quantity > 0 && price > 0 {
			var sellPrice, priceRatio, qtyRatio float64
			sellQty := 0

			invCards := sz.inventory[cc.Id]
			for _, invCard := range invCards {
				if invCard.Conditions == "NM" {
					sellPrice = invCard.Price
					sellQty = invCard.Quantity
					break
				}
			}

			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}
			if sellQty > 0 {
				qtyRatio = float64(quantity) / float64(sellQty) * 100
			}

			out := mtgban.BuylistEntry{
				Card:          *cc,
				Conditions:    "NM",
				BuyPrice:      price,
				TradePrice:    price * 1.3,
				Quantity:      quantity,
				PriceRatio:    priceRatio,
				QuantityRatio: qtyRatio,
			}
			err := sz.BuylistAdd(out)
			if err != nil {
				sz.printf("%v", err)
			}
		}
	}

	return nil
}

func (sz *Strikezone) InventoryAdd(card mtgban.InventoryEntry) error {
	entries, found := sz.inventory[card.Id]
	if found {
		for _, entry := range entries {
			if entry.Conditions == card.Conditions && entry.Price == card.Price {
				return fmt.Errorf("Attempted to add a duplicate inventory card:\n-new: %v\n-old: %v", card, entry)
			}
		}
	}

	sz.inventory[card.Id] = append(sz.inventory[card.Id], card)
	return nil
}

func (sz *Strikezone) Inventory() (map[string][]mtgban.InventoryEntry, error) {
	if len(sz.inventory) > 0 {
		return sz.inventory, nil
	}

	sz.printf("Empty inventory, scraping started")

	err := sz.scrape()
	if err != nil {
		return nil, err
	}

	return sz.inventory, nil
}

func (sz *Strikezone) BuylistAdd(card mtgban.BuylistEntry) error {
	entry, found := sz.buylist[card.Id]
	if found {
		if entry.BuyPrice == card.BuyPrice {
			return fmt.Errorf("Attempted to add a duplicate buylist card:\n-new: %v\n-old: %v", card, entry)
		}
	}

	sz.buylist[card.Id] = card
	return nil
}

func (sz *Strikezone) Buylist() (map[string]mtgban.BuylistEntry, error) {
	if len(sz.buylist) > 0 {
		return sz.buylist, nil
	}

	sz.printf("Empty buylist, scraping started")
	err := sz.parseBL()
	if err != nil {
		return nil, err
	}
	return sz.buylist, nil
}

func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	return
}
