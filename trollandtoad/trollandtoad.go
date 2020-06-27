package trollandtoad

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

const (
	defaultConcurrency = 8

	tatPagesURL = "https://www.trollandtoad.com/magic-the-gathering/all-singles/7085"
	tatOptions  = "?Keywords=&min-price=&max-price=&items-pp=60&item-condition=&selected-cat=7085&sort-order=&page-no=%d&view=list&subproduct=0&Rarity=&Ruleset=&minMana=&maxMana=&minPower=&maxPower=&minToughness=&maxToughness="
)

type Trollandtoad struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	BuylistDate    time.Time
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
	return &tat
}

type responseChan struct {
	card     *mtgdb.Card
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (tat *Trollandtoad) printf(format string, a ...interface{}) {
	if tat.LogCallback != nil {
		tat.LogCallback("[TaT] "+format, a...)
	}
}

func (tat *Trollandtoad) parsePages(lastPage int) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.trollandtoad.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: tat.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//tat.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML(`div[class='product-info card-body col pl-0 pl-sm-3']`, func(e *colly.HTMLElement) {
		link := e.ChildAttr(`a[class='card-text']`, "href")
		id := path.Base(link)
		cardName := e.ChildText(`a[class='card-text']`)
		edition := e.ChildText(`div[class='row mb-2'] div[class='col-12 prod-cat']`)

		theCard, err := preprocess(cardName, edition)
		if err != nil {
			return
		}
		cc, err := theCard.Match()
		if err != nil {
			switch {
			case strings.Contains(edition, "World Championships"):
			default:
				tat.printf("%q", theCard)
				tat.printf("%s ~ %s", cardName, edition)
				tat.printf("%v", err)
			}
			return
		}

		options, err := tat.client.GetProductOptions(id)
		if err != nil {
			tat.printf(err.Error())
			return
		}

		for _, option := range options {
			if option.Price == 0.0 || option.Quantity == 0 {
				continue
			}

			conditions := option.Conditions
			switch conditions {
			case "NM":
			case "LP":
				conditions = "SP"
			case "PL":
				conditions = "MP"
			default:
				tat.printf("Unsupported %s condition for %s %s", conditions, cardName, edition)
				continue
			}

			var out responseChan
			out = responseChan{
				card: cc,
				invEntry: &mtgban.InventoryEntry{
					Conditions: conditions,
					Price:      option.Price,
					Quantity:   option.Quantity,
					URL:        e.Request.AbsoluteURL(link),
				},
			}
			channel <- out
		}
	})

	q, _ := queue.New(
		tat.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	for i := 1; i <= lastPage; i++ {
		opts := fmt.Sprintf(tatOptions, i)
		q.AddURL(tatPagesURL + opts)
	}

	q.Run(c)

	go func() {
		c.Wait()
		close(channel)
	}()

	for res := range channel {
		err := tat.inventory.Add(res.card, res.invEntry)
		if err != nil {
			// Too many false positives
			//tat.printf("%v", err)
		}
	}

	tat.InventoryDate = time.Now()

	return nil
}

func (tat *Trollandtoad) scrape() error {
	resp, err := http.Get(tatPagesURL)
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

	tat.printf("Parsing %d pages", lastPage)
	return tat.parsePages(lastPage)
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

func (tat *Trollandtoad) processPage(channel chan<- responseChan, id string) error {
	products, err := tat.client.ProductsForId(id)
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

		cc, err := theCard.Match()
		if err != nil {
			switch {
			case strings.Contains(card.Edition, "World Championships"):
			default:
				tat.printf("%q", theCard)
				tat.printf("%s ~ %s", card.Name, card.Edition)
				tat.printf("%v", err)
			}
			continue
		}

		price, err := strconv.ParseFloat(card.BuyPrice, 64)
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

		invCards := tat.inventory[*cc]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		channel <- responseChan{
			card: cc,
			buyEntry: &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: price * 1.30,
				Quantity:   qty,
				PriceRatio: priceRatio,
			},
		}
	}
	return nil
}

func (tat *Trollandtoad) parseBL() error {
	list, err := tat.client.ListEditions()
	if err != nil {
		return err
	}

	tat.printf("Processing %d editions", len(list))

	editions := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tat.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := tat.processPage(results, edition)
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
			editions <- product.CategoryId
		}
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := tat.buylist.Add(record.card, record.buyEntry)
		if err != nil {
			tat.printf(err.Error())
			continue
		}
	}

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
	info.Shorthand = "TaT"
	info.InventoryTimestamp = tat.InventoryDate
	info.BuylistTimestamp = tat.BuylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
