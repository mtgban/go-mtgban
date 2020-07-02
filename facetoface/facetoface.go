package facetoface

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
	defaultConcurrency = 8

	ftfInventoryURL = "https://www.facetofacegames.com/catalog/test-magic_singles/8"
	ftfBuylistURL   = "https://www.facetofacegames.com/buylist/test-magic_singles/8"

	modeInventory = "inventory"
	modeBuylist   = "buylist"
)

type FaceToFace struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	exchangeRate float64

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() (*FaceToFace, error) {
	ftf := FaceToFace{}
	ftf.inventory = mtgban.InventoryRecord{}
	ftf.buylist = mtgban.BuylistRecord{}
	rate, err := mtgban.GetExchangeRate("CAD")
	if err != nil {
		return nil, err
	}
	ftf.exchangeRate = rate
	ftf.MaxConcurrency = defaultConcurrency
	return &ftf, nil
}

type resultChan struct {
	err  error
	card ftfCard
}

func (ftf *FaceToFace) printf(format string, a ...interface{}) {
	if ftf.LogCallback != nil {
		ftf.LogCallback("[FTF] "+format, a...)
	}
}

func (ftf *FaceToFace) scrape(mode string) error {
	channel := make(chan ftfCard)

	c := colly.NewCollector(
		colly.AllowedDomains("www.facetofacegames.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: ftf.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		q := r.URL.Query()
		if q.Get("page") == "" {
			//ftf.printf("Visiting %s", r.URL.String())
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
		priceStr = strings.Replace(priceStr, "CAD$ ", "", 1)
		priceStr = strings.Replace(priceStr, ",", "", 1)
		cardPrice, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			ftf.printf("%v", err)
			return
		}

		urlId := e.Attr("data-id")
		switch urlId {
		// Duplicated Starter 1999 lands
		case "575953", "576003", "575943", "575993", "575893",
			"88668", "88664", "88669", "88663", "88662", "88667",
			"88661", "70420", "70421", "70419", "70418":
			return
		}
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
		case "Moderately Played- Signed",
			"Heavily Played- Signed",
			"Slightly Played- Signed",
			"NM-Mint-Signed",
			"Heavily Played":
			cond = "HP"
		default:
			ftf.printf("Unsupported %s condition", cond)
			return
		}

		isFoil := false
		if strings.Contains(cardName, " Foil") || strings.Contains(cardName, "-Foil") ||
			edition == "Foil Promos" ||
			// Our Market Research Shows that really long names hide card properties
			dataVid == "257147" || dataVid == "9685216" || dataVid == "9685217" {
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
			ftf.printf("%v", err)
			return
		}

		card := ftfCard{
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
		c.Visit(ftfInventoryURL)
	} else if mode == modeBuylist {
		c.Visit(ftfBuylistURL)
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
			switch {
			// Ignore errors coming from sets with too much duplication
			case card.Edition == "Alliances",
				card.Edition == "Fallen Empires",
				strings.HasPrefix(card.Edition, "WCD"):
			// Ignore errors coming from lands
			case theCard.IsBasicLand():
				switch card.Edition {
				case "3rd Edition",
					"4th Edition",
					"5th Edition",
					"Alpha",
					"Battle Royale",
					"Beta",
					"Collector's Edition - Domestic",
					"Collector's Edition - International",
					"Ice Age",
					"Mirage",
					"Portal 1",
					"Portal Second Age",
					"Tempest",
					"Unlimited":
					continue
				}
				fallthrough
			default:
				ftf.printf("%q", theCard)
				ftf.printf("%q", card)
				ftf.printf("%v", err)
			}
			continue
		}

		if mode == modeInventory {
			if card.Quantity > 0 && card.Price > 0 {
				out := &mtgban.InventoryEntry{
					Conditions: card.Conditions,
					Price:      card.Price * ftf.exchangeRate,
					Quantity:   card.Quantity,
					URL:        ftfInventoryURL + "/" + card.URLId,
				}
				err := ftf.inventory.Add(cc, out)
				if err != nil {
					ftf.printf("%v", err)
				}
			}
		}
		if mode == modeBuylist {
			if card.Quantity > 0 && card.Price > 0 && card.Conditions == "NM" {
				var sellPrice, priceRatio float64

				invCards := ftf.inventory[*cc]
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
					BuyPrice:   card.Price * ftf.exchangeRate,
					TradePrice: card.Price * ftf.exchangeRate * 1.3,
					Quantity:   card.Quantity,
					PriceRatio: priceRatio,
					URL:        ftfBuylistURL + "/" + card.URLId,
				}
				err := ftf.buylist.Add(cc, out)
				if err != nil {
					ftf.printf("%v", err)
				}
			}
		}
	}

	if mode == modeInventory {
		ftf.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		ftf.buylistDate = time.Now()
	}

	return nil
}

func (ftf *FaceToFace) Inventory() (mtgban.InventoryRecord, error) {
	if len(ftf.inventory) > 0 {
		return ftf.inventory, nil
	}

	err := ftf.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return ftf.inventory, nil
}

func (ftf *FaceToFace) Buylist() (mtgban.BuylistRecord, error) {
	if len(ftf.buylist) > 0 {
		return ftf.buylist, nil
	}

	err := ftf.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return ftf.buylist, nil
}

func (ftf *FaceToFace) Info() (info mtgban.ScraperInfo) {
	info.Name = "Face to Face"
	info.Shorthand = "FTF"
	info.CountryFlag = "🇨🇦"
	info.InventoryTimestamp = ftf.inventoryDate
	info.BuylistTimestamp = ftf.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
