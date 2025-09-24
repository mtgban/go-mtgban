package strikezone

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/%s_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/BuyList/%s.html"

	GameMagic   = "Magic_the_Gathering"
	GameLorcana = "Lorcana"
)

type Strikezone struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	game string
}

func NewScraper(game string) *Strikezone {
	sz := Strikezone{}
	sz.inventory = mtgban.InventoryRecord{}
	sz.buylist = mtgban.BuylistRecord{}
	sz.MaxConcurrency = defaultConcurrency
	sz.game = game
	return &sz
}

func (sz *Strikezone) printf(format string, a ...interface{}) {
	if sz.LogCallback != nil {
		sz.LogCallback("[SZ] "+format, a...)
	}
}

type respChan struct {
	cardId string
	inv    *mtgban.InventoryEntry
	bl     *mtgban.BuylistEntry
}

func (sz *Strikezone) processRow(mode string, channel chan<- respChan, el *colly.HTMLElement, edition string) error {
	var cardName, pathURL, notes, cond, qty, price string

	cardName = el.ChildText("td:nth-child(1)")
	if cardName == "" || cardName == "Name" {
		// No error as empty page may not have anything to process
		return nil
	}

	pathURL = el.ChildAttr("a", "href")

	var cardId string
	var err error
	switch sz.game {
	case GameMagic:
		if mode == "inventory" {
			notes = el.ChildText("td:nth-child(4)")
			cond = el.ChildText("td:nth-child(5)")
			qty = el.ChildText("td:nth-child(6)")
			price = el.ChildText("td:nth-child(7)")
		} else if mode == "buylist" {
			notes = el.ChildText("td:nth-child(4)")
			cond = notes
			qty = el.ChildText("td:nth-child(5)")
			price = el.ChildText("td:nth-child(6)")
		}

		theCard, err := preprocess(cardName, edition, notes)
		if err != nil {
			return nil
		}

		cardId, err = mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			// Skip errors from these sets, there is not enough information
			switch edition {
			case "Secret Lair", "The List", "Mystery Booster":
				return nil
			}
			sz.printf("%q", theCard)
			sz.printf("%s|%s|%s", cardName, edition, notes)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					sz.printf("- %s", card)
				}
			}
			return err
		}
	case GameLorcana:
		notes = el.ChildText("td:nth-child(2)")
		cond = el.ChildText("td:nth-child(4)")
		qty = el.ChildText("td:nth-child(5)")
		price = el.ChildText("td:nth-child(6)")

		foil := strings.Contains(strings.ToLower(cond), "foil")

		cardId, err = mtgmatcher.SimpleSearch(cardName, notes, foil)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			sz.printf("%s|%s|%s", cardName, edition, notes)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					sz.printf("- %s", card)
				}
			}
			return err
		}
	}

	cardPrice, err := mtgmatcher.ParsePrice(price)
	if err != nil || cardPrice <= 0 {
		return err
	}

	quantity, err := strconv.Atoi(qty)
	if err != nil || quantity <= 0 {
		return err
	}

	switch {
	case strings.Contains(cond, "Mint"):
		cond = "NM"
	case strings.Contains(cond, "Light"):
		cond = "SP"
	case strings.Contains(cond, "Medium"):
		cond = "MP"
	case strings.Contains(cond, "Heavy"):
		cond = "HP"
	default:
		return fmt.Errorf("Unsupported %s condition", cond)
	}

	if mode == "inventory" {
		channel <- respChan{
			cardId: cardId,
			inv: &mtgban.InventoryEntry{
				Conditions: cond,
				Price:      cardPrice,
				Quantity:   quantity,
				URL:        "http://shop.strikezoneonline.com" + pathURL,
			},
		}
	} else if mode == "buylist" {
		var sellPrice, priceRatio float64

		invCards := sz.inventory[cardId]
		for _, invCard := range invCards {
			if invCard.Conditions == "NM" {
				sellPrice = invCard.Price
				break
			}
		}

		if sellPrice > 0 {
			priceRatio = cardPrice / sellPrice * 100
		}

		// Some buy pages return wrong results if they have a comma
		cardName = url.QueryEscape(strings.Replace(cardName, ",", "", -1))
		link := "http://shop.strikezoneonline.com/TUser?MC=CUSTS&MF=B&BUID=637&ST=D&M=B&CMD=Search&T=" + cardName

		channel <- respChan{
			cardId: cardId,
			bl: &mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   cardPrice,
				Quantity:   quantity,
				PriceRatio: priceRatio,
				URL:        link,
			},
		}
	}
	return nil
}

func (sz *Strikezone) scrape(ctx context.Context, mode string) error {
	channel := make(chan respChan)

	c := colly.NewCollector(
		colly.AllowedDomains("shop.strikezoneonline.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),

		colly.StdlibContext(ctx),
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

		basePath := "/Category/"
		if mode == "buylist" {
			basePath = "/BuyList/"
		}

		if strings.Contains(link, basePath) &&
			!strings.HasSuffix(link, "_ByTable.html") &&
			!strings.HasSuffix(link, "_ByRarity.html") &&
			!strings.HasSuffix(link, "Games.html") &&
			!strings.HasSuffix(link, "Magic_Booster_Boxes.html") &&
			!strings.HasSuffix(link, "Fat_Packs.html") &&
			!strings.HasSuffix(link, "Gift_Sets_and_Secret_Lairs.html") &&
			!strings.HasSuffix(link, "Preconstructed_Decks.html") {
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	// Callback for when a scraped page contains a form element
	c.OnHTML("body", func(e *colly.HTMLElement) {
		edition := e.ChildText("h1")
		edition = strings.TrimSuffix(edition, " Buy Lists")
		edition = strings.TrimPrefix(edition, "Singles ")

		sz.printf("Parsing %s", edition)

		tableRowName := "table.rtti tr"
		if mode == "buylist" || sz.game == GameLorcana {
			tableRowName = "table.ItemTable tr"
		}

		e.ForEach(tableRowName, func(_ int, el *colly.HTMLElement) {
			err := sz.processRow(mode, channel, el, edition)
			if err != nil {
				sz.printf("cannot process %s %s: %s", mode, edition, err.Error())
				sz.printf("-> %s", e.Request.URL)
			}
		})
	})

	var link string
	if mode == "inventory" {
		link = fmt.Sprintf(szInventoryURL, sz.game)
	} else if mode == "buylist" {
		link = fmt.Sprintf(szBuylistURL, sz.game)
	}
	sz.printf("Visiting %s", link)
	c.Visit(link)

	go func() {
		c.Wait()
		close(channel)
	}()

	for resp := range channel {
		if resp.inv != nil {
			err := sz.inventory.Add(resp.cardId, resp.inv)
			if err != nil {
				sz.printf("%v", err)
			}
		}
		if resp.bl != nil {
			err := sz.buylist.Add(resp.cardId, resp.bl)
			if err != nil {
				sz.printf("%v", err)
			}
		}
	}

	if mode == "inventory" {
		sz.inventoryDate = time.Now()
	} else if mode == "buylist" {
		sz.buylistDate = time.Now()
	}

	return nil
}

func (sz *Strikezone) Inventory() (mtgban.InventoryRecord, error) {
	if len(sz.inventory) > 0 {
		return sz.inventory, nil
	}

	err := sz.scrape(context.TODO(), "inventory")
	if err != nil {
		return nil, err
	}

	return sz.inventory, nil
}

func (sz *Strikezone) Buylist() (mtgban.BuylistRecord, error) {
	if len(sz.buylist) > 0 {
		return sz.buylist, nil
	}

	err := sz.scrape(context.TODO(), "buylist")
	if err != nil {
		return nil, err
	}

	return sz.buylist, nil
}

func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	info.InventoryTimestamp = &sz.inventoryDate
	info.BuylistTimestamp = &sz.buylistDate
	switch sz.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
