package channelfireball

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	maxConcurrency  = 8
	cfbInventoryURL = "https://store.channelfireball.com/catalog/magic_singles/8"
	cfbBuylistURL   = "https://store.channelfireball.com/buylist/magic_singles/8"

	modeInventory = "inventory"
	modeBuylist   = "buylist"
)

type Channelfireball struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time
	BuylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Channelfireball {
	cfb := Channelfireball{}
	cfb.inventory = mtgban.InventoryRecord{}
	cfb.buylist = mtgban.BuylistRecord{}
	return &cfb
}

type resultChan struct {
	err  error
	card cfbCard
}

func (cfb *Channelfireball) printf(format string, a ...interface{}) {
	if cfb.LogCallback != nil {
		cfb.LogCallback("[CFB] "+format, a...)
	}
}

func (cfb *Channelfireball) scrape(mode string) error {
	channel := make(chan cfbCard)

	c := colly.NewCollector(
		colly.AllowedDomains("store.channelfireball.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: maxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		q := r.URL.Query()
		if q.Get("page") == "" {
			//cfb.printf("Visiting %s", r.URL.String())
		}
	})

	// Callback for links on scraped pages (edition names)
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		// Consider only "/{buylist,catalog}/magic_singles-<catergory>-<edition>/<id>" links
		linkDepth := strings.Count(link, "/")
		ok := (mode == modeInventory && strings.HasPrefix(link, "/catalog")) ||
			(mode == modeBuylist && strings.HasPrefix(link, "/buylist"))
		if linkDepth == 3 && ok {
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})
	c.OnHTML("form[class='add-to-cart-form']", func(e *colly.HTMLElement) {
		// Skip out of stock items
		dataVid := e.Attr("data-vid")
		if dataVid == "" {
			return
		}

		priceStr := e.Attr("data-price")
		priceStr = strings.Replace(priceStr, "$", "", 1)
		priceStr = strings.Replace(priceStr, ",", "", 1)
		cardPrice, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			cfb.printf("%v", err)
			return
		}

		urlId := e.Attr("data-id")
		cardName := e.Attr("data-name")
		edition := e.Attr("data-category")
		cond := e.Attr("data-variant")
		fields := strings.Split(cond, ", ")
		cond = fields[0]
		if len(fields) > 1 && fields[1] != "English" {
			return
		}
		switch cond {
		case "NM-Mint":
			cond = "NM"
		case "Slightly Played":
			cond = "SP"
		case "Moderately Played":
			cond = "MP"
		case "Damaged":
			cond = "HP"
		default:
			cfb.printf("Unsupported %s condition", cond)
			return
		}

		isFoil := false
		if strings.Contains(cardName, " Foil") ||
			// Our Market Research Shows that really long names hide card properties
			dataVid == "1061099" || dataVid == "297099" {
			isFoil = true
		}

		cardName, edition, err = preprocess(cardName, edition)
		if err != nil {
			return
		}

		qty := 0
		e.ForEach("input", func(_ int, elem *colly.HTMLElement) {
			if elem.Attr("class") == "qty" {
				qty, err = strconv.Atoi(elem.Attr("max"))
				if err != nil {
					return
				}
			}
		})
		if err != nil {
			cfb.printf("%v", err)
			return
		}

		card := cfbCard{
			URLId:      urlId,
			Key:        dataVid,
			Name:       cardName,
			Edition:    edition,
			Foil:       isFoil,
			Conditions: cond,
			Price:      cardPrice,
			Quantity:   qty,
		}

		channel <- card
	})

	if mode == modeInventory {
		c.Visit(cfbInventoryURL)
	} else if mode == modeBuylist {
		c.Visit(cfbBuylistURL)
	} else {
		return fmt.Errorf("Unsupported mode %s", mode)
	}

	go func() {
		c.Wait()
		close(channel)
	}()

	// The same pattern is repeated exactly 3 times, store the simple key for
	// the processed cards and skip the duplication
	processed := map[string]bool{}

	for card := range channel {
		if processed[card.Key] {
			continue
		}
		processed[card.Key] = true

		variant := ""
		cardName := card.Name
		if card.Name != "Erase (Not the Urza's Legacy One)" {
			variants := mtgdb.SplitVariants(cardName)
			cardName = variants[0]
			if len(variants) > 1 {
				variant = variants[1]
			}
		}

		theCard := &mtgdb.Card{
			Name:      cardName,
			Edition:   card.Edition,
			Variation: variant,
			Foil:      card.Foil,
		}
		cc, err := theCard.Match()
		if err != nil {
			cfb.printf("%q", theCard)
			cfb.printf("%q", card)
			cfb.printf("%v", err)
			continue
		}

		if mode == modeInventory {
			if card.Quantity > 0 && card.Price > 0 {
				out := &mtgban.InventoryEntry{
					Conditions: card.Conditions,
					Price:      card.Price,
					Quantity:   card.Quantity,
					URL:        cfbInventoryURL + "/" + card.URLId,
				}
				err := cfb.inventory.Add(cc, out)
				if err != nil {
					cfb.printf("%v", err)
				}
			}
		}
		if mode == modeBuylist {
			if card.Quantity > 0 && card.Price > 0 && card.Conditions == "NM" {
				var sellPrice, priceRatio float64

				invCards := cfb.inventory[*cc]
				for _, invCard := range invCards {
					if invCard.Conditions == "NM" {
						sellPrice = invCard.Price
						break
					}
				}

				if sellPrice > 0 {
					priceRatio = card.Price / sellPrice * 100
				}

				out := &mtgban.BuylistEntry{
					BuyPrice:   card.Price,
					TradePrice: card.Price * 1.3,
					Quantity:   card.Quantity,
					PriceRatio: priceRatio,
					URL:        cfbBuylistURL + "/" + card.URLId,
				}
				err := cfb.buylist.Add(cc, out)
				if err != nil {
					cfb.printf("%v", err)
				}
			}
		}
	}

	if mode == modeInventory {
		cfb.InventoryDate = time.Now()
	} else if mode == modeBuylist {
		cfb.BuylistDate = time.Now()
	}

	return nil
}

func (cfb *Channelfireball) Inventory() (mtgban.InventoryRecord, error) {
	if len(cfb.inventory) > 0 {
		return cfb.inventory, nil
	}

	start := time.Now()
	cfb.printf("Inventory scraping started at %s", start)

	err := cfb.scrape(modeInventory)
	if err != nil {
		return nil, err
	}
	cfb.printf("Inventory scraping took %s", time.Since(start))

	return cfb.inventory, nil
}

func (cfb *Channelfireball) Buylist() (mtgban.BuylistRecord, error) {
	if len(cfb.buylist) > 0 {
		return cfb.buylist, nil
	}

	start := time.Now()
	cfb.printf("Buylist scraping started at %s", start)

	err := cfb.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}
	cfb.printf("Buylist scraping took %s", time.Since(start))

	return cfb.buylist, nil
}

var fourHorsemenDate = time.Date(1993, time.August, 1, 0, 0, 0, 0, time.UTC)
var premodernDate = time.Date(1994, time.August, 1, 0, 0, 0, 0, time.UTC)
var modernDate = time.Date(2003, time.July, 1, 0, 0, 0, 0, time.UTC)

func (cfb *Channelfireball) Grading(card mtgdb.Card, entry mtgban.BuylistEntry) (grade map[string]float64) {
	set, err := mtgdb.Set(card.Edition)
	if err != nil {
		return nil
	}
	setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
	if err != nil {
		return nil
	}

	switch {
	case card.Foil:
		grade = map[string]float64{
			"SP": 0.7, "MP": 0.5, "HP": 0.3,
		}
	case setDate.After(fourHorsemenDate) && setDate.Before(premodernDate.AddDate(0, 0, -1)):
		grade = map[string]float64{
			"SP": 0.5, "MP": 0.25, "HP": 0.1,
		}
	case setDate.After(premodernDate) && setDate.Before(modernDate.AddDate(0, 0, -1)):
		grade = map[string]float64{
			"SP": 0.7, "MP": 0.5, "HP": 0.3,
		}
	case setDate.After(modernDate):
		grade = map[string]float64{
			"SP": 0.8, "MP": 0.6, "HP": 0.4,
		}
	}

	return
}

func (cfb *Channelfireball) Info() (info mtgban.ScraperInfo) {
	info.Name = "Channel Fireball"
	info.Shorthand = "CFB"
	info.InventoryTimestamp = cfb.InventoryDate
	info.BuylistTimestamp = cfb.BuylistDate
	return
}
