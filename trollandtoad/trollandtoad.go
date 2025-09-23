package trollandtoad

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

type Trollandtoad struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
}

func NewScraper() *Trollandtoad {
	tnt := Trollandtoad{}
	tnt.inventory = mtgban.InventoryRecord{}
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

func (tnt *Trollandtoad) parsePages(ctx context.Context, link string, lastPage int) error {
	channel := make(chan responseChan)

	c := colly.NewCollector(
		colly.AllowedDomains("www.trollandtoad.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),

		colly.StdlibContext(ctx),
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
			case strings.Contains(edition, "The List"):
			case strings.Contains(edition, "Mystery Booster"):
			case strings.Contains(theCard.Variation, "Token"):
			case mtgmatcher.IsToken(theCard.Name):
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

	for i := 0; i < lastPage; i++ {
		opts := fmt.Sprintf(tntOptions, i+1)
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

	return nil
}

func (tnt *Trollandtoad) scrapePages(ctx context.Context, link, edition string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
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
	tnt.printf("Parsing %d pages from %s", lastPage, edition)
	return tnt.parsePages(ctx, link, lastPage)
}

const (
	categoryPage = "https://www.trollandtoad.com/magic-the-gathering/1041"
)

func (tnt *Trollandtoad) scrape(ctx context.Context) error {
	link := categoryPage
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	var pages []string
	var titles []string
	doc.Find(`div[class="card-body d-none d-sm-block py-0"] ul li a`).Each(func(_ int, s *goquery.Selection) {
		page, found := s.Attr("href")
		if !found {
			return
		}

		title := strings.Replace(strings.TrimSpace(s.Text()), "  ", " ", -1)
		switch title {
		case "All Foil Singles",
			"All Singles",
			"Arena Codes",
			"Artist Alters",
			"Artist's Proofs",
			"Booster Boxes",
			"Complete Sets",
			"Creature Forge",
			"Fat Packs/Bundles",
			"Magic: The Gathering Lots & Bundles",
			"Memorabilia",
			"Misprints",
			"Modern Legal Sets",
			"PSA Graded Magic Cards",
			"Sealed Product",
			"Standard Legal Sets",
			"Supplies",
			"Token Cards",
			"Toy Figures":
			return
		}
		pages = append(pages, page)
		titles = append(titles, title)
	})

	for i, page := range pages {
		err := tnt.scrapePages(ctx, "https://www.trollandtoad.com"+page+tntOptions, titles[i])
		if err != nil {
			tnt.printf("%s error: %s", titles[i], err.Error())
		}
	}

	tnt.inventoryDate = time.Now()

	return nil
}

func (tnt *Trollandtoad) Inventory() (mtgban.InventoryRecord, error) {
	if len(tnt.inventory) > 0 {
		return tnt.inventory, nil
	}

	err := tnt.scrape(context.TODO())
	if err != nil {
		return nil, err
	}

	return tnt.inventory, nil
}

func (tnt *Trollandtoad) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll and Toad"
	info.Shorthand = "TNT"
	info.InventoryTimestamp = &tnt.inventoryDate
	return
}
