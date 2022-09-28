package trollandtoad

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	tatOptions = "?Keywords=&hide-oos=on&min-price=&max-price=&items-pp=60&item-condition=&sort-order=&page-no=%d&view=list&subproduct=0&Rarity=&Ruleset=&minMana=&maxMana=&minPower=&maxPower=&minToughness=&maxToughness="
)

var tatAllPagesURL = []string{
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

	client *TATClient
}

func NewScraper() *Trollandtoad {
	tat := Trollandtoad{}
	tat.inventory = mtgban.InventoryRecord{}
	tat.buylist = mtgban.BuylistRecord{}
	tat.client = NewTATClient()
	tat.MaxConcurrency = defaultConcurrency
	return &tat
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (tat *Trollandtoad) printf(format string, a ...interface{}) {
	if tat.LogCallback != nil {
		tat.LogCallback("[TAT] "+format, a...)
	}
}

func (tat *Trollandtoad) parsePages(link string, lastPage int) error {
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
		Parallelism: tat.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		tat.printf("Visiting page %s", r.URL.Query().Get("page-no"))
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
				tat.printf("%v", err)
				tat.printf("%q", theCard)
				tat.printf("%s ~ %s", cardName, edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						tat.printf("- %s", card)
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
				tat.printf("Unsupported %s condition for %s %s", conditions, cardName, edition)
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
				tat.printf("%s: %s", theCard, err.Error())
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
		tat.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for i := 1; i <= lastPage; i++ {
		opts := fmt.Sprintf(tatOptions, i)
		q.AddURL(link + opts)
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	for res := range channel {
		err := tat.inventory.Add(res.cardId, res.invEntry)
		if err != nil {
			// Too many false positives
			//tat.printf("%v", err)
		}
	}

	tat.inventoryDate = time.Now()

	return nil
}

func (tat *Trollandtoad) scrapePages(link string) error {
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
	tat.printf("Parsing %d pages from %s", lastPage, link)
	return tat.parsePages(link, lastPage)
}

func (tat *Trollandtoad) scrape() error {
	for _, link := range tatAllPagesURL {
		err := tat.scrapePages(link)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tat *Trollandtoad) Inventory() (mtgban.InventoryRecord, error) {
	if len(tat.inventory) > 0 {
		return tat.inventory, nil
	}

	err := tat.scrape()
	if err != nil {
		return nil, err
	}

	return tat.inventory, nil
}

func (tat *Trollandtoad) processPage(channel chan<- responseChan, id, code string) error {
	products, err := tat.client.ProductsForId(id, code)
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
			case theCard.IsBasicLand():
			default:
				tat.printf("%v", err)
				tat.printf("%q", theCard)
				tat.printf("%s ~ %s", card.Name, card.Edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						tat.printf("- %s", card)
					}
				}
			}
			continue
		}

		price, err := mtgmatcher.ParsePrice(card.BuyPrice)
		if err != nil {
			tat.printf("%s %v", card.Name, err)
			continue
		}

		qty, err := strconv.Atoi(card.Quantity)
		if err != nil {
			tat.printf("%s %v", card.Name, err)
			continue
		}

		var priceRatio, sellPrice float64

		invCards := tat.inventory[cardId]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		link := "https://www2.trollandtoad.com/buylist/#!/search/All/" + url.QueryEscape(theCard.Name)
		deductions := []float64{1, 0.6, 0.6, 0.6}
		for i, deduction := range deductions {
			channel <- responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions: mtgban.DefaultGradeTags[i],
					BuyPrice:   price * deduction,
					TradePrice: price * deduction * 1.30,
					Quantity:   qty,
					PriceRatio: priceRatio,
					URL:        link,
				},
			}
		}
	}
	return nil
}

func (tat *Trollandtoad) parseBL() error {
	modern, err := tat.client.ListModernEditions()
	if err != nil {
		return err
	}
	vintage, err := tat.client.ListVintageEditions()
	if err != nil {
		return err
	}

	list := append(modern, vintage...)

	tat.printf("Processing %d editions", len(list))

	editions := make(chan TATEdition)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tat.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := tat.processPage(results, edition.CategoryId, edition.DeptId)
				if err != nil {
					tat.printf("%v", err)
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
			tat.printf("Processing %s", product.CategoryName)

			editions <- product
		}
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := tat.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			tat.printf("%s", err.Error())
			continue
		}
	}

	tat.buylistDate = time.Now()

	return nil
}

func (tat *Trollandtoad) Buylist() (mtgban.BuylistRecord, error) {
	if len(tat.buylist) > 0 {
		return tat.buylist, nil
	}

	err := tat.parseBL()
	if err != nil {
		return nil, err
	}

	return tat.buylist, nil
}

func (tat *Trollandtoad) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll and Toad"
	info.Shorthand = "TAT"
	info.InventoryTimestamp = &tat.inventoryDate
	info.BuylistTimestamp = &tat.buylistDate
	return
}
