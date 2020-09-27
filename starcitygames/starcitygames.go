package starcitygames

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type Starcitygames struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	products []productList

	client *SCGClient
}

type productList struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"abbr"`
}

func NewScraper(buylistCategories io.Reader) (*Starcitygames, error) {
	scg := Starcitygames{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient()
	scg.MaxConcurrency = defaultConcurrency

	if buylistCategories != nil {
		d := json.NewDecoder(buylistCategories)
		err := d.Decode(&scg.products)
		if err != nil {
			return nil, err
		}
	}

	return &scg, nil
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (scg *Starcitygames) printf(format string, a ...interface{}) {
	if scg.LogCallback != nil {
		scg.LogCallback("[SCG] "+format, a...)
	}
}

var scgEndpoints = []string{
	"https://starcitygames.com/shop/singles/english/",
	"https://starcitygames.com/shop/singles/foil-english/",
}

func (scg *Starcitygames) scrape() error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("starcitygames.com"),
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),
		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: scg.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//scg.printf("Visiting %s", r.URL.String())
	})

	// Callback for links on scraped pages (edition names)
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		u, err := url.Parse(link)
		if err != nil {
			return
		}

		q := u.Query()
		if q.Get("Color") != "" {
			return
		}
		if q.Get("_bc_fsnf") != "" {
			return
		}
		if q.Get("Rarity") != "" {
			return
		}

		if q.Get("page") == "" {
			q.Set("page", "1")
		}
		q.Set("Language", "English")
		q.Set("sort", "alphaasc")
		u.RawQuery = q.Encode()
		link = u.String()

		if (strings.Contains(link, "/shop/singles/english/") ||
			strings.Contains(link, "/shop/singles/foil-english/")) &&
			strings.Count(link, "/") == 5 {
			err := c.Visit(e.Request.AbsoluteURL(link))
			if err != nil {
				if err != colly.ErrAlreadyVisited {
					scg.printf("error while linking: %s", err.Error())
				}
			}
		}
	})

	c.OnHTML(`section[class="category-products"]`, func(e *colly.HTMLElement) {
		fullEdition := e.Attr("data-category")
		fields := strings.Split(fullEdition, " (")
		edition := fields[0]
		isFoil := len(fields) > 1 && strings.HasPrefix(fields[1], "Foil")

		switch edition {
		case "Alpha-cut 4th Edition",
			"Alternate 4th Edition",
			"Misprints",
			"Pro Player Cards",
			"Rarities and Misprints",
			"Summer Magic",
			"Wyvern-backed Fallen Empires":
			return
		default:
			if strings.Contains(edition, "Oversized") {
				return
			}
		}

		e.ForEach(`table tr.product`, func(_ int, elem *colly.HTMLElement) {
			dataId := elem.Attr("data-id")
			fullName := elem.Attr("data-name")
			subtitle := elem.ChildText(`div[class="listItem-details"] p[class="category-Subtitle"]`)

			card, err := convert(fullName, subtitle, edition)
			if err != nil {
				return
			}
			card.Id = dataId
			card.Foil = isFoil

			theCard, err := preprocess(card)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				scg.printf("%v", err)
				scg.printf("%q", theCard)
				scg.printf("'%q' (%s) [%s]", fullName, subtitle, edition)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.Unmatch(probe)
						scg.printf("- %s", card)
					}
				}
				return
			}

			entries, err := scg.client.SearchData(dataId)
			if err != nil {
				scg.printf("%s: %s", fullName, err)
				return
			}

			for _, entry := range entries {
				price := entry.Price
				qty := entry.InventoryLevel

				if price <= 0.0 || qty <= 0 || entry.PurchasingDisabled {
					continue
				}

				conditions := "N/A"
				for _, option := range entry.OptionValues {
					if option.OptionDisplayName == "Condition" {
						conditions = option.Label
						break
					}
				}

				switch conditions {
				case "Near Mint":
					conditions = "NM"
				case "Played":
					conditions = "SP"
				case "Heavily Played":
					conditions = "MP"
				case "N/A":
					continue
				default:
					scg.printf("unknown condition %s for ", conditions, card.Name)
					continue
				}

				channel <- responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
						Price:      price,
						Quantity:   qty,
					},
				}
			}
		})
	})

	q, _ := queue.New(
		scg.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for _, endpoint := range scgEndpoints {
		resp, err := scg.client.List(endpoint)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			scg.printf("%s", err)
			continue
		}

		doc.Find(`div[id="ajax-conetnt"]`).Find("li").Each(func(i int, s *goquery.Selection) {
			editionUrl, ok := s.Find("a").Attr("href")
			if !ok {
				return
			}

			q.AddURL(editionUrl + "?Language=English&page=1&sort=alphaasc")
		})
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	dupes := map[string]bool{}
	for res := range channel {
		key := res.cardId + res.invEntry.Conditions
		if dupes[key] {
			continue
		}
		dupes[key] = true

		err := scg.inventory.Add(res.cardId, res.invEntry)
		if err != nil {
			scg.printf("%v", err)
		}
	}
	scg.inventoryDate = time.Now()

	return nil
}

func (scg *Starcitygames) Inventory() (mtgban.InventoryRecord, error) {
	if len(scg.inventory) > 0 {
		return scg.inventory, nil
	}

	err := scg.scrape()
	if err != nil {
		return nil, err
	}

	return scg.inventory, nil

}

func (scg *Starcitygames) processProduct(channel chan<- responseChan, product string) error {
	search, err := scg.client.SearchProduct(product)
	if err != nil {
		return err
	}

	for _, results := range search.Results {
		if len(results) == 0 {
			continue
		}
		switch search.Edition {
		case "3rd Edition BB",
			"Promotional Cards: Oversized":
			continue
		}

		for _, result := range results {
			conditions := result.Condition
			switch conditions {
			case "NM/M":
				conditions = "NM"
			case "PL":
				conditions = "SP"
			case "HP":
				conditions = "MP"
			default:
				scg.printf("unknown condition %s for %v", conditions, result)
				continue
			}
			if result.Language != "English" {
				if !(result.Language == "Japanese" && search.Edition == "War of the Spark" && result.Subtitle != "") {
					continue
				}
			}

			result.edition = search.Edition
			theCard, err := preprocess(result)
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				scg.printf("%v", err)
				scg.printf("%q", theCard)
				scg.printf("'%q' (%s)", result, search.Edition)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.Unmatch(probe)
						scg.printf("- %s", card)
					}
				}
				continue
			}

			price, err := strconv.ParseFloat(result.Price, 64)
			if err != nil {
				scg.printf("%s %s", theCard.Name, err)
				continue
			}

			var priceRatio, sellPrice float64

			invCards := scg.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			channel <- responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions: conditions,
					BuyPrice:   price,
					TradePrice: price * 1.30,
					Quantity:   0,
					PriceRatio: priceRatio,
					URL:        "https://old.starcitygames.com/buylist",
				},
			}
		}
	}
	return nil
}

func (scg *Starcitygames) parseBL() error {
	products := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < scg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for product := range products {
				err := scg.processProduct(results, product)
				if err != nil {
					scg.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, product := range scg.products {
			products <- product.Id
		}
		close(products)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := scg.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			scg.printf(err.Error())
			continue
		}
	}

	scg.buylistDate = time.Now()

	return nil
}

func (scg *Starcitygames) Buylist() (mtgban.BuylistRecord, error) {
	if len(scg.buylist) > 0 {
		return scg.buylist, nil
	}

	err := scg.parseBL()
	if err != nil {
		return nil, err
	}

	return scg.buylist, nil
}

func (scg *Starcitygames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCG"
	info.InventoryTimestamp = scg.inventoryDate
	info.BuylistTimestamp = scg.buylistDate
	info.MultiCondBuylist = true
	return
}
