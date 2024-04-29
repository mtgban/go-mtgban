package cardshark

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"golang.org/x/exp/slices"
)

const (
	defaultConcurrency = 8
	baseURL            = "http://www.cardshark.com"
	inventoryURL       = "http://www.cardshark.com/Buy/Magic-the-Gathering/Find-Cards/"
)

type Cardshark struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Referral       string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord
}

func NewScraper() *Cardshark {
	cs := Cardshark{}
	cs.inventory = mtgban.InventoryRecord{}
	cs.marketplace = map[string]mtgban.InventoryRecord{}
	cs.MaxConcurrency = defaultConcurrency
	return &cs
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (cs *Cardshark) printf(format string, a ...interface{}) {
	if cs.LogCallback != nil {
		cs.LogCallback("[CShark] "+format, a...)
	}
}

func (cs *Cardshark) scrape() error {
	results := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.cardshark.com"),

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

	// Callback for the list of editions
	c.OnHTML(`table[id="ctl00_ContentPlaceHolder1_gvCardSets"]`, func(e *colly.HTMLElement) {
		e.ForEach(`td`, func(_ int, el *colly.HTMLElement) {
			editionText := el.ChildText("a")
			switch editionText {
			case "Booster Boxes",
				"Booster Packs",
				"Box Topper Cards",
				"Emblems",
				"Oversize Cards",
				"Special Occasion",
				"Tempest Remastered",
				"Tokens",
				"World Championship Decks":
				return
			// Too many duplicates
			case "Alliances",
				"Fallen Empires",
				"Homelands":
				return
			}

			c.Visit(e.Request.AbsoluteURL(el.ChildAttr("a", "href")))
		})
	})

	// Callback for the list of cards in a single edition
	c.OnHTML(`table[id="ctl00_ContentPlaceHolder1_GridView1"]`, func(e *colly.HTMLElement) {
		e.ForEach(`tr`, func(_ int, el *colly.HTMLElement) {
			cardPath := el.ChildAttr("a", "href")
			if cardPath == "" {
				return
			}
			price := el.ChildText("td:nth-child(5)")
			if !strings.HasPrefix(price, "$") {
				return
			}

			c.Visit(e.Request.AbsoluteURL(cardPath))
		})
	})

	// Callback for the list of offers of a single card
	c.OnHTML(`table[id="ctl00_ContentPlaceHolder1_ctl01_gvItems2"]`, func(e *colly.HTMLElement) {
		pageTitle := e.DOM.ParentsUntil("~").Find("title").Text()
		titles := strings.Split(pageTitle, "|")

		cardName := strings.TrimSpace(titles[0])
		edition := ""
		if len(titles) > 1 {
			edition = strings.TrimSpace(titles[1])
		}
		number := e.DOM.ParentsUntil("~").Find(`span[id="ctl00_ContentPlaceHolder1_ctl00_lblCardNumber"]`).Text()

		_, ogId := path.Split(e.Request.AbsoluteURL(""))

		link := "http://www.cardshark.com/CardDetail.aspx?id=" + ogId + "&Game=Magic"
		if cs.Referral != "" {
			link += "&ref=" + cs.Referral
		}

		theCard, err := preprocess(cardName, edition, number, "")
		if err != nil {
			return
		}

		var altCardId string
		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			// Ignore errors from basic lands
			if mtgmatcher.IsBasicLand(cardName) {
				return
			}
			cs.printf("%v", err)
			cs.printf("%v", theCard)
			cs.printf("%s | %s | %s", cardName, edition, number)

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

		e.ForEach(`tr`, func(_ int, el *colly.HTMLElement) {
			id := cardId

			priceStr := el.ChildText("td:nth-child(1)")
			if priceStr == "" || priceStr == "No Sellers Currently" {
				return
			}

			qtyStr := el.ChildText("td:nth-child(2)")
			seller := el.ChildText("td:nth-child(5) a:first-child")
			conditions := el.ChildText("td:nth-child(6)")
			notes := el.ChildText("td:nth-child(7)")

			if strings.Contains(notes, "Foreign") {
				return
			}

			// Redo a match in case we have new information
			if notes != "" {
				altCard, err := preprocess(cardName, edition, number, notes)
				if err != nil {
					return
				}
				altCardId, err = mtgmatcher.Match(altCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					return
				} else if err != nil {
					// Ignore aliasing errors for this match
					var alias *mtgmatcher.AliasingError
					if errors.As(err, &alias) {
						return
					}

					cs.printf("%v", err)
					cs.printf("%v", theCard)
					cs.printf("%s | %s | %s | %s", cardName, edition, number, notes)
				}
			}

			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				cs.printf("%s", err.Error())
				return
			}
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				cs.printf("%s", err.Error())
				return
			}

			if price == 0.0 || qty == 0 {
				return
			}

			switch conditions {
			case "Mint", "Near Mint":
				conditions = "NM"
			case "Lightly Played":
				conditions = "SP"
			case "Moderately Played":
				conditions = "MP"
			case "Heavy Play":
				conditions = "HP"
			default:
				cs.printf("Unsupported %s condition for %s", conditions, theCard)
			}

			if notes != "" && altCardId != "" {
				id = altCardId
			}
			out := responseChan{
				cardId: id,
				invEntry: &mtgban.InventoryEntry{
					Conditions: conditions,
					Price:      price,
					Quantity:   qty,
					URL:        link,
					SellerName: seller,
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
		err := cs.inventory.AddRelaxed(result.cardId, result.invEntry)
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

	cs.inventoryDate = time.Now()

	return nil
}

func (cs *Cardshark) Inventory() (mtgban.InventoryRecord, error) {
	if len(cs.inventory) > 0 {
		return cs.inventory, nil
	}

	err := cs.scrape()
	if err != nil {
		return nil, err
	}

	return cs.inventory, nil

}

func (cs *Cardshark) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(cs.inventory) == 0 {
		_, err := cs.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := cs.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range cs.inventory {
		for i := range cs.inventory[card] {
			if cs.inventory[card][i].SellerName == sellerName {
				if cs.inventory[card][i].Price == 0 {
					continue
				}
				if cs.marketplace[sellerName] == nil {
					cs.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				cs.marketplace[sellerName][card] = append(cs.marketplace[sellerName][card], cs.inventory[card][i])
			}
		}
	}

	if len(cs.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return cs.marketplace[sellerName], nil
}

func (cs *Cardshark) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	cs.marketplace = market
	cs.inventory = inventory

	cs.printf("Loaded inventory from file")

	cs.printf("Loaded inventory from file")

	return nil
}

func (cs *Cardshark) MarketNames() []string {
	var out []string

	for card := range cs.inventory {
		for i := range cs.inventory[card] {
			sellerName := cs.inventory[card][i].SellerName
			if slices.Contains(out, sellerName) {
				continue
			}

			out = append(out, sellerName)
		}
	}

	return out
}

func (cs *Cardshark) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Shark"
	info.Shorthand = "CShark"
	info.InventoryTimestamp = &cs.inventoryDate
	return
}
