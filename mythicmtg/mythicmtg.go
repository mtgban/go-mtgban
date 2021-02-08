package mythicmtg

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	mmtgBuylistURL = "https://mythicmtg.com/public-buylist"

	defaultConcurrency = 8
)

type Mythicmtg struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time

	buylistDate time.Time
	buylist     mtgban.BuylistRecord

	client *MythicClient
}

func NewScraper() *Mythicmtg {
	mmtg := Mythicmtg{}
	mmtg.inventory = mtgban.InventoryRecord{}
	mmtg.buylist = mtgban.BuylistRecord{}
	mmtg.MaxConcurrency = defaultConcurrency
	mmtg.client = NewMythicClient()
	return &mmtg
}

type respChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (mmtg *Mythicmtg) printf(format string, a ...interface{}) {
	if mmtg.LogCallback != nil {
		mmtg.LogCallback("[MMTG] "+format, a...)
	}
}

func (mmtg *Mythicmtg) processPage(channel chan<- respChan, start int) error {
	reader, err := mmtg.client.Products(start)
	if err != nil {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	doc.Find(`div[class="product-forms-tabs-wrapper"]`).Each(func(i int, s *goquery.Selection) {
		dataId, _ := s.Find(`ul li:nth-child(1)`).Attr("data-id")
		dataIdFoil, _ := s.Find(`ul li:nth-child(2)`).Attr("data-id")

		for _, id := range []string{dataId, dataIdFoil} {
			if id == "" {
				continue
			}
			sub := s.Find(`div[id="` + id + `"]`)

			cardName := sub.Find(`div[class="ty-grid-list__item-name"] bdi`).Text()
			link, _ := sub.Find(`div[class="ty-grid-list__item-name"] bdi a`).Attr("href")
			cardName = strings.TrimSuffix(cardName, "(foil)")
			cardName = strings.TrimSpace(cardName)

			fields := strings.Split(link, "/")
			edition := ""
			if len(fields) > 4 {
				edition = fields[4]
				edition = strings.Title(strings.Replace(edition, "-", " ", -1))
			}

			priceStr := sub.Find(`span[class="ty-price-num"]`).Text()
			price, _ := mtgmatcher.ParsePrice(priceStr)
			if price == 0 {
				continue
			}

			items := sub.Find(`span[class="ty-qty-in-stock ty-control-group__item"]`).Text()
			items = strings.TrimSpace(items)
			items = strings.TrimSuffix(items, "\u00a0item(s)")
			qty, _ := strconv.Atoi(items)
			if qty == 0 {
				continue
			}

			theCard, err := preprocess(cardName, edition)
			if err != nil {
				continue
			}
			if dataIdFoil == id {
				theCard.Foil = true
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				switch edition {
				case "Homelands", "Alliances", "Fallen Empires":
				default:
					mmtg.printf("%v", err)
					mmtg.printf("%q", theCard)
					mmtg.printf("%q ~ %q", cardName, edition)
					alias, ok := err.(*mtgmatcher.AliasingError)
					if ok {
						probes := alias.Probe()
						for _, probe := range probes {
							card, _ := mtgmatcher.GetUUID(probe)
							mmtg.printf("- %s", card)
						}
					}
				}
				continue
			}

			channel <- respChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      price,
					Quantity:   qty,
					URL:        link,
				},
			}
		}
	})

	return nil
}

func (mmtg *Mythicmtg) scrape() error {
	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	totalProducts, err := mmtg.client.TotalItems()
	if err != nil {
		return err
	}
	allPages := totalProducts/DefaultResultsPerPage + 1
	mmtg.printf("Parsing %d items, for a total of %d requests", totalProducts, allPages)

	for i := 0; i < mmtg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for start := range pages {
				err = mmtg.processPage(channel, start)
				if err != nil {
					mmtg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= allPages; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := mmtg.inventory.Add(record.cardId, record.invEntry)
		if err != nil {
			mmtg.printf("%v", err)
			continue
		}
	}

	mmtg.inventoryDate = time.Now()

	return nil
}

func (mmtg *Mythicmtg) Inventory() (mtgban.InventoryRecord, error) {
	if len(mmtg.inventory) > 0 {
		return mmtg.inventory, nil
	}

	err := mmtg.scrape()
	if err != nil {
		return nil, err
	}

	return mmtg.inventory, nil
}

func (mmtg *Mythicmtg) parseBL() error {
	resp, err := http.Get(mmtgBuylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(`table[class="ty-table"]`).Each(func(i int, s *goquery.Selection) {
		editionName := s.Find("thead").Find("tr:nth-child(1)").Find("th").Text()

		s.Find("tbody").Find("tr").Each(func(_ int, rows *goquery.Selection) {
			cardName := rows.Find("td:nth-child(1)").Text()
			priceStr := rows.Find("td:nth-child(3)").Text()
			creditStr := rows.Find("td:nth-child(4)").Text()
			qtyStr := rows.Find("td:nth-child(5)").Text()

			qty, err := strconv.Atoi(qtyStr)
			if err != nil || qty == 0 {
				return
			}

			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil || price == 0 {
				return
			}
			credit, err := strconv.ParseFloat(creditStr, 64)
			if err != nil {
				return
			}

			theCard, err := preprocess(cardName, editionName)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				mmtg.printf("%v", err)
				mmtg.printf("%q", theCard)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						mmtg.printf("- %s", card)
					}
				}
				return
			}

			var priceRatio, sellPrice float64

			invCards := mmtg.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			out := &mtgban.BuylistEntry{
				BuyPrice:   price,
				PriceRatio: priceRatio,
				TradePrice: credit,
				Quantity:   qty,
				URL:        mmtgBuylistURL,
			}
			err = mmtg.buylist.Add(cardId, out)
			if err != nil {
				mmtg.printf("%v", err)
			}
		})
	})

	return nil
}

func (mmtg *Mythicmtg) Buylist() (mtgban.BuylistRecord, error) {
	if len(mmtg.buylist) > 0 {
		return mmtg.buylist, nil
	}

	err := mmtg.parseBL()
	if err != nil {
		return nil, err
	}

	return mmtg.buylist, nil
}

func grading(cardId string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	if entry.BuyPrice < 30 {
		return nil
	}

	grade = map[string]float64{
		"SP": 0.8, "MP": 0.6, "HP": 0,
	}

	return
}

func (mmtg *Mythicmtg) Info() (info mtgban.ScraperInfo) {
	info.Name = "Mythic MTG"
	info.Shorthand = "MMTG"
	info.InventoryTimestamp = mmtg.buylistDate
	info.BuylistTimestamp = mmtg.buylistDate
	info.Grading = grading
	return
}
