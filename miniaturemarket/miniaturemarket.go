package miniaturemarket

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

type Miniaturemarket struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	BuylistDate    time.Time
	MaxConcurrency int

	client    *MMClient
	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Miniaturemarket {
	mm := Miniaturemarket{}
	mm.client = NewMMClient()
	mm.inventory = mtgban.InventoryRecord{}
	mm.buylist = mtgban.BuylistRecord{}
	mm.MaxConcurrency = defaultConcurrency
	return &mm
}

const (
	defaultConcurrency = 6

	firstPage = 1
	lastPage  = 9
)

type respChan struct {
	card     *mtgdb.Card
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (mm *Miniaturemarket) printf(format string, a ...interface{}) {
	if mm.LogCallback != nil {
		mm.LogCallback("[MM] "+format, a...)
	}
}

func (mm *Miniaturemarket) processPage(channel chan<- respChan, page int, secondHalf bool) error {
	spring, err := mm.client.SearchSpringPage(page, secondHalf)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(spring.Results))
	if err != nil {
		return err
	}

	doc.Find("div.grouped-product").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h3").Text())
		theCard, err := preprocess(title)
		if err != nil {
			return
		}

		var infoGroups []MMPrivateInfoGroup
		data := strings.Replace(s.Find("div.grouped-product-info").Text(), "|", ",", -1)
		// Adjust raw data to be compatible with MMPrivateInfoGroup
		data = strings.Replace(data, "\"image\":false", "\"image\":\"\"", -1)
		data = strings.Replace(data, "\"instock\":0", "\"instock\":\"0\"", -1)
		data = strings.Replace(data, "\"price\":\"", "\"price\":", -1)
		data = strings.Replace(data, "\",\"regular_price", ",\"regular_price", -1)
		err = json.Unmarshal([]byte(data), &infoGroups)
		if err != nil {
			mm.printf("%s - %s", data, err.Error())
			return
		}

		for _, group := range infoGroups {
			if group.Price <= 0 || group.Stock <= 0 {
				continue
			}

			// Needed to discern duplicates of this particular card
			if theCard.Name == "Sorcerous Spyglass" {
				switch group.SKU {
				case "M-660-012-1NM", "M-660-012-3F", "M-650-124-3F":
					theCard.Variation += "XLN"
				case "M-660-016-1NM", "M-660-016-3F", "M-650-176-3F":
					theCard.Variation += "ELD"
				}
			}

			theCard.Foil = strings.HasPrefix(group.Name, "Foil")

			cond := group.Name
			switch cond {
			case "Near Mint", "Foil Near Mint", "Foil Near MInt":
				cond = "NM"
			case "Played", "Foil Played":
				cond = "MP"
			default:
				mm.printf("Unsupported %s condition", cond)
				return
			}

			cc, err := theCard.Match()
			if err != nil {
				mm.printf("%q", theCard)
				mm.printf("%s", title)
				mm.printf("%v", err)
				return
			}

			fields := strings.Split(group.SKU, "-")
			urlPage := strings.Join(fields[:len(fields)-1], "-") + ".html"

			channel <- respChan{
				card: cc,
				invEntry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      group.Price,
					Quantity:   group.Stock,
					URL:        "http://www.miniaturemarket.com/" + urlPage,
				},
			}
		}
	})

	return nil
}

// Scrape returns an array of Entry, containing pricing and card information
func (mm *Miniaturemarket) scrape() error {
	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	// The normal API roughly returns half of the elements, so we query it
	// twice, sorting by name in two different ways.
	// In order to reduce the number of false duplicates, check how many
	// elements are left over, and add an appopriate number of fake pages
	// that will be queried in reverse order.
	pagination, err := mm.client.GetPagination(MMDefaultResultsPerPage)
	if err != nil {
		return err
	}
	lastPage, err := mm.client.SearchSpringPage(pagination.TotalPages, false)
	if err != nil {
		return err
	}
	leftover := lastPage.Pagination.TotalResults - lastPage.Pagination.End
	extraPages := leftover/MMDefaultResultsPerPage + 2
	totalPages := pagination.TotalPages + extraPages

	mm.printf("Parsing %d pages with %d extra (%d total)", pagination.TotalPages, extraPages, totalPages)

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				// Restore normal numbering if we need to sort Z-A
				secondHalf := false
				if page > pagination.TotalPages {
					secondHalf = true
					page = page - pagination.TotalPages
				}
				mm.processPage(channel, page, secondHalf)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= totalPages; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := mm.inventory.Add(record.card, record.invEntry)
		// Do not print an error if we expect a duplicate due to the sorting
		if err != nil && mm.inventory[*record.card][0].URL != record.invEntry.URL {
			mm.printf("%v", err)
			continue
		}
	}

	mm.InventoryDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) processEntry(channel chan<- respChan, page int) error {
	buyback, err := mm.client.BuyBackPage(MMCategoryMtgSingles, page)
	if err != nil {
		return err
	}

	for _, card := range buyback {
		if card.MtgCondition == "" ||
			card.MtgSet == "Bulk MTG" ||
			card.MtgRarity == "Sealed Product" {
			continue
		}

		switch card.MtgCondition {
		case "Near Mint", "Foil Near Mint", "Foil Near MInt":
		default:
			mm.printf("Unsupported %s condition", card.MtgCondition)
			continue
		}

		price, err := strconv.ParseFloat(card.Price, 64)
		if err != nil {
			return err
		}

		if price <= 0 {
			continue
		}

		theCard, err := preprocess(card.Name)
		if err != nil {
			continue
		}

		theCard.Foil = card.IsFoil

		// Needed to discern duplicates of this particular card
		if theCard.Name == "Sorcerous Spyglass" {
			switch card.SKU {
			case "M-660-012-1NM", "M-660-012-3F", "M-650-124-3F":
				theCard.Variation += "XLN"
			case "M-660-016-1NM", "M-660-016-3F", "M-650-176-3F":
				theCard.Variation += "ELD"
			}
		}

		cc, err := theCard.Match()
		if err != nil {
			mm.printf("%q", theCard)
			mm.printf("%s", card)
			mm.printf("%v", err)
			continue
		}

		var priceRatio, sellPrice float64

		invCards := mm.inventory[*cc]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		channel <- respChan{
			card: cc,
			buyEntry: &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: price * 1.3,
				Quantity:   0,
				PriceRatio: priceRatio,
			},
		}
	}

	mm.BuylistDate = time.Now()

	return nil
}

func (mm *Miniaturemarket) parseBL() error {
	pages := make(chan int)
	results := make(chan respChan)
	var wg sync.WaitGroup

	for i := 0; i < mm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := mm.processEntry(results, page)
				if err != nil {
					mm.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := firstPage; i <= lastPage; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		err := mm.buylist.Add(result.card, result.buyEntry)
		if err != nil {
			mm.printf(err.Error())
			continue
		}
	}

	return nil
}

func (mm *Miniaturemarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(mm.inventory) > 0 {
		return mm.inventory, nil
	}

	start := time.Now()
	mm.printf("Inventory scraping started at %s", start)

	err := mm.scrape()
	if err != nil {
		return nil, err
	}
	mm.printf("Inventory scraping took %s", time.Since(start))

	return mm.inventory, nil
}

func (mm *Miniaturemarket) Buylist() (mtgban.BuylistRecord, error) {
	if len(mm.buylist) > 0 {
		return mm.buylist, nil
	}

	start := time.Now()
	mm.printf("Buylist scraping started at %s", start)

	err := mm.parseBL()
	if err != nil {
		return nil, err
	}
	mm.printf("Buylist scraping took %s", time.Since(start))

	return mm.buylist, nil
}

func (mm *Miniaturemarket) Grading(card mtgdb.Card, entry mtgban.BuylistEntry) (grade map[string]float64) {
	grade = map[string]float64{
		"SP": 0.75, "MP": 0.75, "HP": 0,
	}
	if entry.BuyPrice <= 0.08 {
		grade = map[string]float64{
			"SP": 0.4, "MP": 0.4, "HP": 0,
		}
	} else if entry.BuyPrice <= 0.1 {
		grade = map[string]float64{
			"SP": 0.5, "MP": 0.5, "HP": 0,
		}
	} else if entry.BuyPrice <= 0.15 {
		grade = map[string]float64{
			"SP": 0.66, "MP": 0.66, "HP": 0,
		}
	}
	return
}

func (mm *Miniaturemarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "Miniature Market"
	info.Shorthand = "MM"
	info.InventoryTimestamp = mm.InventoryDate
	info.BuylistTimestamp = mm.BuylistDate
	return
}
