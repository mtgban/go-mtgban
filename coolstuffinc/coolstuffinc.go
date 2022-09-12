package coolstuffinc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
	queue "github.com/gocolly/colly/v2/queue"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	http "github.com/hashicorp/go-retryablehttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	csiInventoryURL = "https://www.coolstuffinc.com/magic+the+gathering/"
	csiBuylistURL   = "https://www.coolstuffinc.com/ajax_buylist.php"
)

type Coolstuffinc struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	httpclient *http.Client
}

func NewScraper() *Coolstuffinc {
	csi := Coolstuffinc{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.httpclient = http.NewClient()
	csi.httpclient.Logger = nil
	csi.MaxConcurrency = defaultConcurrency
	return &csi
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (csi *Coolstuffinc) printf(format string, a ...interface{}) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSI] "+format, a...)
	}
}

func (csi *Coolstuffinc) scrape() error {
	channel := make(chan responseChan)

	allowedPages := []string{}

	c := colly.NewCollector(
		colly.AllowedDomains("www.coolstuffinc.com"),

		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),

		colly.Async(true),
	)

	c.SetClient(cleanhttp.DefaultClient())

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
		Parallelism: csi.MaxConcurrency,
	})

	c.OnRequest(func(r *colly.Request) {
		//csi.printf("Visiting %s", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		class := e.Attr("class")
		// Skip "commons" "uncommons" and other categories containing the same data
		if class == "nav-singles" || strings.Contains(e.Text, "Japanese") {
			return
		}

		u, err := url.Parse(link)
		if err != nil {
			return
		}

		// Only visit a real mtg url
		found := false
		for _, page := range allowedPages {
			if u.Path == page {
				found = true
				break
			}
		}
		if !found {
			return
		}

		// Ignore url with incorrect encodes (missing the '?'
		// to separate query from path)
		if strings.HasPrefix(u.Path, "&") {
			return
		}

		q := u.Query()
		page := q.Get("page")
		num, _ := strconv.Atoi(page)
		if num < 1 {
			return
		}
		singles := q.Get("sh")
		if singles != "" && singles != "1" {
			return
		}
		res := q.Get("resultsperpage")
		if res != "" && res != "25" {
			return
		}
		set := q.Get("s")
		if set != "" && set != "mtg" {
			return
		}
		sort := q.Get("sb")
		if sort != "" && sort != "price|asc" {
			return
		}

		u.RawQuery = fmt.Sprintf("resultsperpage=25&sb=&sh=1&s=mtg&page=%s", page)
		err = c.Visit(u.String())
		if err != nil {
			if err != colly.ErrAlreadyVisited {
				//csi.printf("error while linking %s: %s", e.Request.AbsoluteURL(link), err.Error())
			}
		}
	})

	// Callback for when a scraped page contains a form element
	c.OnHTML(`div[class="row product-search-row main-container"]`, func(e *colly.HTMLElement) {
		// Skip the "on sale" pages, these will appear elsewhere
		pageTitle := e.DOM.ParentsUntil("~").Find("title").Text()
		if strings.Contains(pageTitle, " Sale") || strings.Contains(pageTitle, " Signed") {
			return
		}

		cardName := e.ChildText(`span[itemprop="name"]`)
		fullEdition := e.ChildText(`div[class="breadcrumb-trail"]`)
		// Strip mtg away, skip anything that is not prefixed like it
		if !strings.HasPrefix(fullEdition, "Magic: The Gathering") {
			return
		}
		edition := strings.TrimPrefix(fullEdition, "Magic: The Gathering » ")

		// These special cards are mixed in the normal edition
		if strings.HasPrefix(edition, "Magic: The Gathering » Masterpiece") {
			if !strings.Contains(pageTitle, "Inventions") &&
				!strings.Contains(pageTitle, "Expeditions") &&
				!strings.Contains(pageTitle, "Invocations") {
				return
			}
		}

		notes := e.ChildText(`div[class="large-8 medium-12 small- 12 product-notes"]`)

		// Extract number or set information
		imgURL := e.ChildAttr(`div[class="large-12 medium-12 small-12"] div[class="large-2 medium-2 small-4 columns search-photo-cell"] img[itemprop="image"]`, "data-src")
		altImgURL := e.ChildAttr(`div[class="large-12 medium-12 small-12"] div[class="productLink"] img[itemprop="image"]`, "src")
		imgName := strings.TrimSuffix(path.Base(imgURL), filepath.Ext(imgURL))
		altImgName := strings.TrimSuffix(path.Base(altImgURL), filepath.Ext(altImgURL))
		if len(altImgName) > len(imgName) {
			imgName = altImgName
		}
		maybeNum := strings.TrimPrefix(imgName, strings.Replace(cardName, " ", "", -1))

		e.ForEach(`div[itemprop="offers"]`, func(_ int, elem *colly.HTMLElement) {
			theCard, err := preprocess(cardName, edition, notes, maybeNum)
			if err != nil {
				return
			}

			soon := elem.ChildText(`div[class="large-12 medium-12 small-12"]`)
			if strings.Contains(soon, "Estimated") {
				return
			}

			qtyStr := elem.ChildText(`div[class="row"]:first-child`)
			switch {
			case qtyStr == "",
				qtyStr == "Out of Stock",
				strings.Contains(qtyStr, "Estimated"),
				strings.HasPrefix(qtyStr, "Near"),
				strings.HasPrefix(qtyStr, "Foil"):
				return
			}
			fields := strings.Split(qtyStr, " ")
			qtyStr = strings.Replace(fields[0], "+", "", 1)

			isFoil := len(fields) > 1 && fields[1] == "Foil"
			idx := 1
			if isFoil {
				idx = 2
			}

			conditions := strings.Join(fields[idx:], " ")
			switch conditions {
			case "Mint",
				"Near Mint":
				conditions = "NM"
			case "Played":
				conditions = "MP"
			default:
				switch {
				case strings.Contains(conditions, "BGS"),
					strings.Contains(conditions, "New"),
					strings.Contains(conditions, "signature"),
					strings.Contains(conditions, "Summer Magic"),
					strings.Contains(conditions, "Unique"):
				default:
					csi.printf("Unsupported %s condition for %s", conditions, theCard)
				}
				return
			}
			if strings.Contains(cardName, "Signed by") {
				conditions = "HP"
			}

			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				csi.printf("%s %s %v", cardName, edition, err)
				return
			}

			priceStr := elem.ChildText(`b[itemprop="price"]`)
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				csi.printf("%v", err)
				return
			}

			if price == 0.0 || qty == 0 {
				return
			}

			// preprocess() might return something that derived foil status
			// from one of the fields (cardName in particular)
			theCard.Foil = theCard.Foil || isFoil
			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				return
			} else if err != nil {
				switch {
				case theCard.IsBasicLand(),
					strings.HasSuffix(theCard.Name, "Guildgate"),
					strings.HasSuffix(theCard.Name, "Signet"),
					strings.Contains(cardName, "Token"):
				default:
					csi.printf("%v", err)
					csi.printf("%v", theCard)
					csi.printf("'%s' '%s' '%s' '%s'", cardName, fullEdition, notes, maybeNum)

					var alias *mtgmatcher.AliasingError
					if errors.As(err, &alias) {
						probes := alias.Probe()
						for _, probe := range probes {
							card, _ := mtgmatcher.GetUUID(probe)
							csi.printf("- %s", card)
						}
					}
				}
				return
			}

			link := elem.ChildAttr(`div[class="large-3 medium-3 small-2 columns text-right"] link[itemprop="url"]`, "content")
			if csi.Partner != "" {
				link += "?utm_referrer=mtgban"
			}

			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: conditions,
					Price:      price,
					Quantity:   qty,
					URL:        link,
				},
			}

			channel <- out
		})
	})

	resp, err := csi.httpclient.Get(csiInventoryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	q, _ := queue.New(
		csi.MaxConcurrency,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	doc.Find(`div[class="set-wrapper"]`).Find("li").Each(func(i int, s *goquery.Selection) {
		editionUrl, _ := s.Find("a").Attr("href")
		editionName := s.Find("a").Text()

		if editionName == "" {
			return
		}
		if strings.HasSuffix(editionUrl, "/") {
			return
		}

		allowedPages = append(allowedPages, editionUrl)

		q.AddURL("https://www.coolstuffinc.com" + editionUrl + "?resultsperpage=25&sb=&sh=1&s=mtg&page=1")
	})

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

		err := csi.inventory.Add(res.cardId, res.invEntry)
		if err != nil {
			csi.printf("%v", err)
		}
	}

	csi.inventoryDate = time.Now()

	return nil
}

func (csi *Coolstuffinc) Inventory() (mtgban.InventoryRecord, error) {
	if len(csi.inventory) > 0 {
		return csi.inventory, nil
	}

	err := csi.scrape()
	if err != nil {
		return nil, err
	}

	return csi.inventory, nil

}

func (csi *Coolstuffinc) processPage(channel chan<- responseChan, edition string) error {
	resp, err := csi.httpclient.PostForm(csiBuylistURL, url.Values{
		"ajaxtype": {"selectProductSetName2"},
		"ajaxdata": {edition},
		"gamename": {"mtg"},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var blob struct {
		HTML string `json:"html"`
	}
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(blob.HTML))
	if err != nil {
		return err
	}

	doc.Find(".main-container").Each(func(i int, s *goquery.Selection) {
		cardName, _ := s.Attr("data-name")
		priceStr, _ := s.Attr("data-price")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			csi.printf("%v", err)
			return
		}

		foil, _ := s.Attr("data-foil")
		extra := s.Find(".search-info-cell").Find(".mini-print").Not(".breadcrumb-trail").Text()
		extra = strings.Replace(extra, "\n", " ", -1)
		info := s.Find(".search-info-cell").Find(".breadcrumb-trail").Text()
		if strings.Contains(info, "Sealed") {
			return
		}
		if strings.Contains(info, "Game Night") {
			edition = "Game Night 2018"
			if strings.Contains(info, "2019") {
				edition = "Game Night 2019"
			}
		}

		theCard, err := preprocess(cardName, edition, extra, "")
		if err != nil {
			return
		}
		theCard.Foil = foil == "yes" || strings.Contains(info, "Foil")

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			switch {
			case theCard.IsBasicLand(),
				strings.Contains(cardName, "Token"):
			default:
				csi.printf("%v", err)
				csi.printf("%q", theCard)
				csi.printf("'%s' '%s' '%s'", cardName, edition, extra)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						csi.printf("- %s", card)
					}
				}
			}
			return
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[cardId]
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
				BuyPrice:   price,
				TradePrice: price * 1.3,
				Quantity:   0,
				PriceRatio: priceRatio,
				URL:        "https://www.coolstuffinc.com/main_buylist_display.php",
			},
		}
	})
	return nil
}

func (csi *Coolstuffinc) parseBL() error {
	resp, err := csi.httpclient.PostForm(csiBuylistURL, url.Values{
		"ajaxtype": {"selectsearchgamename2"},
		"ajaxdata": {"mtg"},
	})
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var blob struct {
		HTML string `json:"html"`
	}
	err = json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(blob.HTML))
	if err != nil {
		return err
	}

	editions := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editions {
				err := csi.processPage(results, edition)
				if err != nil {
					csi.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		doc.Find("option").Each(func(_ int, s *goquery.Selection) {
			edition := s.Text()
			if edition == "Bulk Magic" {
				return
			}
			editions <- edition
		})
		close(editions)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := csi.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			csi.printf("%s", err.Error())
			continue
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

func (csi *Coolstuffinc) Buylist() (mtgban.BuylistRecord, error) {
	if len(csi.buylist) > 0 {
		return csi.buylist, nil
	}

	err := csi.parseBL()
	if err != nil {
		return nil, err
	}

	return csi.buylist, nil
}

func (csi *Coolstuffinc) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cool Stuff Inc"
	info.Shorthand = "CSI"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
