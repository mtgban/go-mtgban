package coolstuffinc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	http "github.com/hashicorp/go-retryablehttp"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	csiInventoryURL = "https://www.coolstuffinc.com/sq/?s=mtg"

	defaultBuylistPage = "https://www.coolstuffinc.com/main_buylist_display.php"
)

var deductions = []float64{1, 1, 0.75}

type Coolstuffinc struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	// If set to true scrape will skip all entries without a nonfoil NM price
	// but will be almost twice as fast
	FastMode bool

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

func (csi *Coolstuffinc) processSearch(results chan<- responseChan, itemName string) error {
	result, err := Search(itemName, csi.FastMode)
	if err != nil {
		return err
	}

	// result.PageId may be empty if the results have only one page
	for page := 1; ; page++ {
		data := result.Data

		if page > 1 {
			link := "https://www.coolstuffinc.com/sq/" + result.PageId + "?page=" + fmt.Sprint(page)

			resp, err := cleanhttp.DefaultClient().Get(link)
			if err != nil {
				continue
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			csi.printf("newDoc - %s", err.Error())
			continue
		}

		doc.Find(`div[class="row product-search-row main-container"]`).Each(func(i int, s *goquery.Selection) {
			cardName := s.Find(`span[itemprop="name"]`).Text()

			pid, _ := s.Find(`span[class="rating-display "]`).Attr("data-pid")
			edition := itemName
			notes := s.Find(`div[class="large-8 medium-12 small- 12 product-notes"]`).Text()
			notes = strings.TrimPrefix(notes, "Notes: ")

			imgURL, _ := s.Find(`a[class="productLink"]`).Find("img").Attr("data-src")
			if imgURL == "" {
				imgURL, _ = s.Find(`a[class="productLink"]`).Find("img").Attr("src")
				if imgURL == "" {
					log.Println("img not found", cardName, edition)
				}
			}

			theCard, err := preprocess(cardName, edition, notes, imgURL)
			if err != nil {
				return
			}

			s.Find(`div[itemprop="offers"]`).Each(func(i int, se *goquery.Selection) {
				fullRow := strings.TrimSpace(se.Text())
				switch {
				case strings.Contains(fullRow, "Out of Stock"),
					strings.Contains(fullRow, "not currently available"):
					return
				}

				qtyStr := se.Find(`span[class="card-qty"]`).Text()
				qtyStr = strings.TrimSpace(strings.TrimSuffix(qtyStr, "+"))
				// If preorder has no quantity,, set max allowed
				if qtyStr == "" && strings.Contains(notes, "Preorder") {
					qtyStr = "20"
				}

				qty, err := strconv.Atoi(qtyStr)
				if err != nil {
					log.Println(fullRow)
					csi.printf("%s %s %v", cardName, edition, err)
					return
				}

				bundleStr := se.Find(`div[class="b1-gx-free"]`).Text()
				bundle := bundleStr == "Buy 1 get 3 free!"

				if !bundle && bundleStr != "" {
					log.Println(bundleStr)
				}

				// Derive the condition portion
				conditions := strings.TrimLeft(fullRow, qtyStr+"+ ")
				conditions = strings.Split(conditions, "$")[0]
				conditions = strings.TrimSuffix(conditions, bundleStr)
				// From the sale text, there is a weird space
				conditions = strings.TrimSuffix(conditions, "Was ")

				isFoil := strings.HasPrefix(conditions, "Foil")

				switch conditions {
				case "Near Mint", "Foil Near Mint":
					conditions = "NM"
				case "Played", "Foil Played":
					conditions = "MP"
				default:
					switch {
					case strings.Contains(conditions, "BGS"),
						strings.Contains(conditions, "Unique"):
					default:
						csi.printf("Unsupported '%s' condition for %s", conditions, theCard)
					}
					return
				}
				if strings.Contains(cardName, "Signed by") {
					conditions = "HP"
				}

				priceStr := se.Find(`b[itemprop="price"]`).Text()
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					csi.printf("%v", err)
					return
				}
				if bundle {
					price /= 4
				}

				if price == 0.0 || qty == 0 {
					return
				}

				link := "https://www.coolstuffinc.com/p/" + pid
				if csi.Partner != "" {
					link += "?utm_referrer=mtgban"
				}

				// preprocess() might return something that derived foil status
				// from one of the fields (cardName in particular)
				theCard.Foil = theCard.Foil || isFoil
				cardId, err := mtgmatcher.Match(theCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					return
				} else if err != nil {
					switch {
					// Ignore errors
					case theCard.IsBasicLand(),
						notes == "" && strings.Contains(edition, "The List"),
						strings.Contains(notes, "Preorder"):
					default:
						csi.printf("%v", err)
						csi.printf("%v", theCard)
						csi.printf("'%s' '%s' '%s'", cardName, edition, notes)
						csi.printf("- %s", link)

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

				out := responseChan{
					cardId: cardId,
					invEntry: &mtgban.InventoryEntry{
						Conditions: conditions,
						Price:      price,
						Quantity:   qty,
						URL:        link,
						OriginalId: pid,
					},
				}

				results <- out
			})
		})

		next, _ := doc.Find(`span[id="nextLink"]`).Find("a").Attr("href")
		if next == "" {
			break
		}
	}

	return nil
}

func (csi *Coolstuffinc) scrape() error {
	resp, err := csi.httpclient.Get(csiInventoryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	var itemNames []string
	doc.Find(`fieldset`).Each(func(i int, s *goquery.Selection) {
		title := s.Find(`h2[class="mb10"] b`).Text()
		if title != "Item Set" {
			return
		}
		s.Find(`div[class="toggleTable"]`).Find("li").Each(func(j int, se *goquery.Selection) {
			itemName, _ := se.Find(`input[type="checkbox"]`).Attr("value")
			switch {
			case strings.Contains(itemName, "Bulk"),
				strings.Contains(itemName, "Relic Token"),
				itemName == "Magic":
				return
			}

			itemNames = append(itemNames, itemName)
		})
	})
	// Sort for predictable results
	sort.Strings(itemNames)

	csi.printf("Found %d items", len(itemNames))

	start := time.Now()

	items := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < csi.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for itemName := range items {
				csi.printf("Processing %s", itemName)
				err := csi.processSearch(results, itemName)
				if err != nil {
					csi.printf("%v for %s", err, itemName)
				}
			}
			wg.Done()
		}()
	}
	go func() {
		for _, item := range itemNames {
			items <- item
		}
		close(items)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := csi.inventory.Add(record.cardId, record.invEntry)
		if err != nil {
			csi.printf("%s", err.Error())
		}
	}

	log.Println("This operation took", time.Since(start))

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

	data, err := io.ReadAll(resp.Body)
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

		for i, deduction := range deductions {
			channel <- responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions: mtgban.DefaultGradeTags[i],
					BuyPrice:   price * deduction,
					PriceRatio: priceRatio,
					URL:        defaultBuylistPage,
				},
			}
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

	data, err := io.ReadAll(resp.Body)
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
	info.CreditMultiplier = 1.25
	return
}
