package trollandtoad

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	buylistURL     = "https://www2.trollandtoad.com/buylist/ajax_scripts/csv-download.php?deptCode="
	buylistLinkURL = "https://www2.trollandtoad.com/buylist/#!/search/All/"

	GameLorcana = "6"
)

type TrollAndToadGeneric struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	inventoryDate time.Time
	buylistDate   time.Time

	DisableRetail  bool
	DisableBuylist bool

	game string
}

func NewGenericScraper(game string) *TrollAndToadGeneric {
	tnt := TrollAndToadGeneric{}
	tnt.inventory = mtgban.InventoryRecord{}
	tnt.buylist = mtgban.BuylistRecord{}
	tnt.game = game

	tnt.MaxConcurrency = defaultConcurrency
	return &tnt
}

func (tnt *TrollAndToadGeneric) printf(format string, a ...interface{}) {
	if tnt.LogCallback != nil {
		tnt.LogCallback("[TNT] "+format, a...)
	}
}

func (tnt *TrollAndToadGeneric) parsePages(ctx context.Context, link string, lastPage int) error {
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
		title := e.ChildText(`a[class='card-text']`)
		edition := e.ChildText(`div[class='row mb-2'] div[class='col-12 prod-cat']`)

		oos := e.ChildText(`div[class='row mb-2 '] div[class='col-12'] div[class='font-weight-bold font-smaller text-muted']`)
		if oos == "Out of Stock" {
			return
		}

		// Workaound certain cards not being formatted correctly
		if strings.Contains(title, ")") && !strings.Contains(title, ") - ") {
			title = strings.Replace(title, ")", ") -", 1)
			pos := strings.Index(title, ") -")
			title += " - " + title[pos+3:]
		}

		chunks := strings.Split(title, " - ")
		if len(chunks) < 2 {
			tnt.printf("invalid name format for: %s", title)
			return
		}
		cardName := strings.Join(chunks[:len(chunks)-2], " - ")
		number := chunks[len(chunks)-2]
		foil := strings.Contains(strings.ToLower(chunks[len(chunks)-1]), "foil")

		cardId, err := mtgmatcher.SimpleSearch(cardName, number, foil)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			tnt.printf("%v", err)
			tnt.printf("%s %s %s", cardName, edition, number)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				tnt.printf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					tnt.printf("%s: %s", probe, co)
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
			tnt.printf("%v", err)
		}
	}

	tnt.inventoryDate = time.Now()

	return nil
}

func (tnt *TrollAndToadGeneric) scrapePages(ctx context.Context, link string) error {
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
	tnt.printf("Parsing %d pages from %s", lastPage, link)
	return tnt.parsePages(ctx, link, lastPage)
}

func (tnt *TrollAndToadGeneric) scrape(ctx context.Context) error {
	var link string
	if tnt.game == GameLorcana {
		link = "https://www.trollandtoad.com/disney-lorcana/19773"
	}

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
	doc.Find(`ul[id="subCatList"] li a`).Each(func(_ int, s *goquery.Selection) {
		page, found := s.Attr("href")
		if !found {
			return
		}
		pages = append(pages, page)
	})

	for _, page := range pages {
		err := tnt.scrapePages(ctx, "https://www.trollandtoad.com"+page+tntOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tnt *TrollAndToadGeneric) scrapeBuylist(ctx context.Context) error {
	link := buylistURL + tnt.game

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	csvReader := csv.NewReader(resp.Body)
	csvReader.ReuseRecord = true

	// Drop header
	_, err = csvReader.Read()
	if err == io.EOF {
		return errors.New("empty csv")
	}
	if err != nil {
		return err
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if record[1] == "Bulk" {
			continue
		}

		title := record[2]
		// Workaound certain cards not being formatted correctly
		if strings.Contains(title, ")") && !strings.Contains(title, ") - ") {
			title = strings.Replace(title, ")", ") -", 1)
			pos := strings.Index(title, ") -")
			title += " - " + title[pos+3:]
		}

		chunks := strings.Split(title, " - ")
		if len(chunks) < 2 {
			tnt.printf("invalid name format for: %s", title)
			continue
		}
		cardName := strings.Join(chunks[:len(chunks)-2], " - ")
		number := chunks[len(chunks)-2]
		foil := strings.Contains(strings.ToLower(chunks[len(chunks)-1]), "foil")
		link := buylistLinkURL + cardName

		cardId, err := mtgmatcher.SimpleSearch(cardName, number, foil)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			tnt.printf("%v", err)
			tnt.printf("%q", record)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				tnt.printf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					tnt.printf("%s: %s", probe, co)
				}
			}
			continue
		}

		price, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			continue
		}

		qty, err := strconv.Atoi(record[5])
		if err != nil {
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

		entry := &mtgban.BuylistEntry{
			BuyPrice:   price,
			Quantity:   qty,
			PriceRatio: priceRatio,
			URL:        link,
			OriginalId: record[0],
		}

		err = tnt.buylist.Add(cardId, entry)
		if err != nil {
			tnt.printf("%s", err.Error())
		}
	}

	tnt.buylistDate = time.Now()

	return nil
}

func (tnt *TrollAndToadGeneric) SetConfig(opt mtgban.ScraperOptions) {
	tnt.DisableRetail = opt.DisableRetail
	tnt.DisableBuylist = opt.DisableBuylist
}

func (tnt *TrollAndToadGeneric) Load(ctx context.Context) error {
	var errs []error

	if !tnt.DisableRetail {
		err := tnt.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !tnt.DisableBuylist {
		err := tnt.scrapeBuylist(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (tnt *TrollAndToadGeneric) Inventory() (mtgban.InventoryRecord, error) {
	return tnt.inventory, nil
}

func (tnt *TrollAndToadGeneric) Buylist() (mtgban.BuylistRecord, error) {
	return tnt.buylist, nil
}

func (tnt *TrollAndToadGeneric) Info() (info mtgban.ScraperInfo) {
	info.Name = "Troll and Toad"
	info.Shorthand = "TNT"
	info.InventoryTimestamp = &tnt.inventoryDate
	info.BuylistTimestamp = &tnt.buylistDate
	switch tnt.game {
	case "6":
		info.Game = mtgban.GameLorcana
	}
	return
}
