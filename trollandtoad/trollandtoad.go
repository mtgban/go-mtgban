package trollandtoad

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	tntOptions = "?Keywords=&hide-oos=on&min-price=&max-price=&items-pp=60&item-condition=&sort-order=&page-no=%d&view=list&subproduct=0&Rarity=&Ruleset=&minMana=&maxMana=&minPower=&maxPower=&minToughness=&maxToughness="
)

var tntAllPagesURL = []string{
	"https://www.trollandtoad.com/magic-the-gathering/all-singles/7085",
	"https://www.trollandtoad.com/magic-the-gathering/all-foil-singles/7880",
	"https://www.trollandtoad.com/magic-the-gathering/-non-english-sets-singles/6713",
	"https://www.trollandtoad.com/revised-black-border-italian-singles/12388",
	"https://www.trollandtoad.com/magic-the-gathering/the-dark-italian-/939",
	"https://www.trollandtoad.com/magic-the-gathering/-comic-idw-/7993",
	"https://www.trollandtoad.com/magic-the-gathering/-silver-stamped-singles/15513",
	"https://www.trollandtoad.com/magic-the-gathering/-from-the-vault/10963",
	"https://www.trollandtoad.com/magic-the-gathering/-guild-kits/14079",
	"https://www.trollandtoad.com/magic-the-gathering/-duel-decks/10962",
	"https://www.trollandtoad.com/magic-the-gathering/-planeswalker-deck-exclusives/11872",
}

type Trollandtoad struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *TNTClient
}

func NewScraper() *Trollandtoad {
	tnt := Trollandtoad{}
	tnt.inventory = mtgban.InventoryRecord{}
	tnt.buylist = mtgban.BuylistRecord{}
	tnt.client = NewTNTClient()
	tnt.MaxConcurrency = defaultConcurrency
	return &tnt
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (tnt *Trollandtoad) printf(format string, a ...interface{}) {
	if tnt.LogCallback != nil {
		tnt.LogCallback("[TNT] "+format, a...)
	}
}

func (tnt *Trollandtoad) parsePages(link string, lastPage int) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.trollandtoad.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 2 * time.Second,
		Parallelism: tnt.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		tnt.printf("Visiting page %s", r.URL.Query().Get("page-no"))
	})

	c.OnHTML(`div[class="product-col col-12 p-0 my-1 mx-sm-1 mw-100"]`, func(e *colly.HTMLElement) {
		link := e.ChildAttr(`a[class='card-text']`, "href")
		cardName := e.ChildText(`a[class='card-text']`)
		edition := e.ChildText(`div[class='row mb-2'] div[class='col-12 prod-cat']`)

		oos := e.ChildText(`div[class='row mb-2 '] div[class='col-12'] div[class='font-weight-bold font-smaller text-muted']`)
		if oos == "Out of Stock" {
			return
		}

		theCard, err := preprocess(cardName, edition)
		if err != nil {
			return
		}
		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			switch {
			case strings.Contains(edition, "World Championships"):
			case theCard.IsBasicLand():
			default:
				tnt.printf("%v", err)
				tnt.printf("%q", theCard)
				tnt.printf("%s ~ %s", cardName, edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						tnt.printf("- %s", card)
					}
				}
			}
			return
		}

		e.ForEach(`div[class="row position-relative align-center py-2 m-auto"]`, func(_ int, el *colly.HTMLElement) {
			conditions := el.ChildText(`div[class='col-3 text-center p-1']`)
			switch {
			case strings.Contains(conditions, "Near Mint"):
				conditions = "NM"
			case strings.Contains(conditions, "Lightly Played"):
				conditions = "SP"
			case strings.Contains(conditions, "Played"): // includes Moderately
				conditions = "MP"
			case strings.Contains(conditions, "See Image for Condition"):
				return
			default:
				tnt.printf("Unsupported %s condition for %s %s", conditions, cardName, edition)
				return
			}

			qtys := el.ChildTexts(`option`)
			if len(qtys) == 0 {
				return
			}
			qtyStr := qtys[len(qtys)-1]
			qty, _ := strconv.Atoi(qtyStr)
			if qty == 0 {
				return
			}

			priceStr := el.ChildText(`div[class='col-2 text-center p-1']`)
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				tnt.printf("%s: %s", theCard, err.Error())
				return
			}
			if price == 0 {
				return
			}

			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: conditions,
					Price:      price,
					Quantity:   qty,
					URL:        e.Request.AbsoluteURL(link),
				},
			}
			channel <- out
		})
	})

	q, _ := queue.New(
		tnt.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for i := 1; i <= lastPage; i++ {
		opts := fmt.Sprintf(tntOptions, i)
		q.AddURL(link + opts)
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	for res := range channel {
		err := tnt.inventory.Add(res.cardId, res.invEntry)
		if err != nil {
			// Too many false positives
			//tnt.printf("%v", err)
		}
	}

	tnt.inventoryDate = time.Now()

	return nil
}

func (tnt *Trollandtoad) scrapePages(link string) error {
	resp, err := cleanhttp.DefaultClient().Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	lastPage := 0
	doc.Find(`div[class="lastPage pageLink d-flex font-weight-bold"]`).Each(func(_ int, s *goquery.Selection) {
		page, _ := s.Attr("data-page")
		lastPage, err = strconv.Atoi(page)
	})
	if err != nil {
		return err
	}

	if lastPage == 0 {
		lastPage = 1
	}
	tnt.printf("Parsing %d pages from %s", lastPage, link)
	return tnt.parsePages(link, lastPage)
}

func (tnt *Trollandtoad) scrape() error {
	for _, link := range tntAllPagesURL {
		err := tnt.scrapePages(link)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tnt *Trollandtoad) Inventory() (mtgban.InventoryRecord, error) {
	if len(tnt.inventory) > 0 {
		return tnt.inventory, nil
	}

	err := tnt.scrape()
	if err != nil {
		return nil, err
	}

	return tnt.inventory, nil
}

func (tnt *Trollandtoad) processPage(channel chan<- responseChan, id, code string) error {
	products, err := tnt.client.ProductsForId(id, code)
	if err != nil {
		return err
	}

	for _, card := range products.Product {
		if !strings.Contains(card.Condition, "Near Mint") {
			continue
		}

		theCard, err := preprocess(card.Name, card.Edition)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			switch {
			case strings.Contains(card.Edition, "World Championships"):
			case strings.Contains(card.Name, "Token"):
			case strings.Contains(card.Name, "Silver Stamped"):
			case theCard.IsBasicLand():
			default:
				tnt.printf("%v", err)
				tnt.printf("%q", theCard)
				tnt.printf("%s ~ %s", card.Name, card.Edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						tnt.printf("- %s", card)
					}
				}
			}
			continue
		}

		price, err := mtgmatcher.ParsePrice(card.BuyPrice)
		if err != nil {
			tnt.printf("%s %v", card.Name, err)
			continue
		}

		qty, err := strconv.Atoi(card.Quantity)
		if err != nil {
			tnt.printf("%s %v", card.Name, err)
			continue
		}

		var priceRatio, sellPrice float64

		invCards := tnt.inventory[cardId]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		// TnT buylist gets confused with an apostrophe in the card name
		cleanName := strings.Replace(theCard.Name, "'", "", -1)
		link := "https://www2.trollandtoad.com/buylist/#!/search/All/" + cleanName
		channel <- responseChan{
			cardId: cardId,
			buyEntry: &mtgban.BuylistEntry{
				Conditions: "NM",
				BuyPrice:   price,
				Quantity:   qty,
				PriceRatio: priceRatio,
				URL:        link,
			},
		}
	}
	return nil
}

func (tnt *Trollandtoad) parseBL() error {
	modern, err := tnt.client.ListModernEditions()
	if err != nil {
		return err
	}
	vintage, err := tnt.client.ListVintageEditions()
	if err != nil {
		return err
	}

	list := append(modern, vintage...)

	tnt.printf("Processing %d editions", len(list))

	editions := make(chan TNTEdition)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tnt.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := tnt.processPage(results, edition.CategoryId, edition.DeptId)
				if err != nil {
					tnt.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, product := range list {
			// Bulk cards
			if product.CategoryId == "" {
				continue
			}
			tnt.printf("Processing %s", product.CategoryName)

			editions <- product
		}
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := tnt.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			tnt.printf("%s", err.Error())
			continue
		}
	}

	tnt.buylistDate = time.Now()

	return nil
}

func (tnt *Trollandtoad) Buylist() (mtgban.BuylistRecord, error) {
	if len(tnt.buylist) > 0 {
		return tnt.buylist, nil
	}

	err := tnt.parseBL()
	if err != nil {
		return nil, err
	}

	return tnt.buylist, nil
}

func (tnt *Trollandtoad) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll and Toad"
	info.Shorthand = "TNT"
	info.InventoryTimestamp = &tnt.inventoryDate
	info.BuylistTimestamp = &tnt.buylistDate
	info.CreditMultiplier = 1.25
	return
}
