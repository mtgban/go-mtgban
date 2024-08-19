package mtgseattle

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/PuerkitoBio/goquery"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	defaultConcurrency = 8

	inventoryURL = "https://www.mtgseattle.com/catalog/magic_singles/8"
	buylistURL   = "https://www.mtgseattle.com/buylist"

	modeInventory = "inventory"
	modeBuylist   = "buylist"
)

type MTGSeattle struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *retryablehttp.Client
}

func NewScraper() *MTGSeattle {
	ms := MTGSeattle{}
	ms.inventory = mtgban.InventoryRecord{}
	ms.buylist = mtgban.BuylistRecord{}
	ms.MaxConcurrency = defaultConcurrency
	ms.client = retryablehttp.NewClient()
	ms.client.Logger = nil
	return &ms
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (ms *MTGSeattle) printf(format string, a ...interface{}) {
	if ms.LogCallback != nil {
		ms.LogCallback("[MS] "+format, a...)
	}
}

func (ms *MTGSeattle) processProduct(channel chan<- responseChan, product, mode string) error {
	resp, err := ms.client.Get("https://www.mtgseattle.com" + product + "?layout=false")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	edition := doc.Find(`h1[class="page-title"]`).Text()

	// Inventory mode needs to might need to discover additional elements
	if mode == modeInventory &&
		(strings.HasSuffix(edition, "Block") ||
			strings.HasSuffix(edition, "Sets") ||
			strings.HasSuffix(edition, "Editions") ||
			strings.HasSuffix(edition, "Edition") ||
			strings.HasSuffix(edition, "Decks") ||
			strings.HasSuffix(edition, "From the Vault") ||
			strings.HasSuffix(edition, "Promos")) {
		var links []string
		doc.Find(`a[class="clearfix"]`).Each(func(_ int, s *goquery.Selection) {
			link, _ := s.Attr("href")
			links = append(links, link)
		})

		for _, link := range links {
			err := ms.processProduct(channel, link, mode)
			if err != nil {
				ms.printf("%s", err.Error())
			}
		}
		return nil
	}

	xpath := `ul[class="products"] li[class="product"] div[class="inner"] div[class="meta"]`
	if mode == modeBuylist {
		xpath = `ul[class="products"] li[class="product"] div[class="inner"] div[class="meta credit"]`
	}
	doc.Find(xpath).Each(func(_ int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")

		title := strings.TrimSpace(s.Find("h4").Text())
		fields := strings.Split(title, " - ")
		variant := ""
		cardName := fields[0]
		if len(fields) > 1 {
			variant = strings.Join(fields[1:], " ")
		}
		if strings.HasSuffix(cardName, "- Foil") {
			cardName = strings.TrimSuffix(cardName, "- Foil")
			variant = "Foil"
		}

		container := `div[class="list-variants grid small-12 medium-8"] div[class="variant-row in-stock"] span[class="variant-main-info small-12 medium-4 large-5 column eat-both"]`
		if mode == modeBuylist {
			container = `span[class="variant-main-info small-12 medium-5 large-5 column eat-both"]`
		}
		// This will skip the variants in the search page
		condLang := s.Find(container + ` span[class="variant-short-info variant-description"]`).Text()

		qtyStr := s.Find(container + ` span[class="variant-short-info variant-qty"]`).Text()
		qtyStr = strings.TrimPrefix(qtyStr, "Limit ")
		qtyStr = strings.TrimSuffix(qtyStr, " In Stock")

		priceStr := s.Find(`div[class="product-price"] span[class="regular price"]`).Text()

		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			return
		}

		price, _ := mtgmatcher.ParsePrice(priceStr)
		if price == 0 {
			return
		}

		// Adjust price for their discount when bought from the website
		if mode == modeInventory {
			price *= 0.95
		}

		conditions := ""
		if mode == modeInventory {
			cond := strings.Split(condLang, ", ")[0]
			switch cond {
			case "NM-Mint":
				conditions = "NM"
			case "Light Play":
				conditions = "SP"
			case "Moderate Play":
				conditions = "MP"
			case "Heavy Play":
				conditions = "HP"
			case "Graded":
				return
			default:
				ms.printf("Unsupported %s condition for %s", cond, title)
				return
			}
		} else if mode == modeBuylist {
			// Early exit to avoid catching sealed and similar
			if condLang != "NM-Mint, English" {
				return
			}
		}

		theCard, err := preprocess(cardName, edition, variant)
		if err != nil {
			return
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			// Skip reporting an error for known failures (invalid variant number)
			if mtgmatcher.IsBasicLand(cardName) {
				switch edition {
				case "5th Edition",
					"Collectors Edition",
					"Ice Age",
					"International Collectors Edition",
					"Mirage",
					"Portal 1",
					"Portal Second Age",
					"Summer Magic (Edgar)",
					"Tempest":
					return
				}
			}
			switch edition {
			case "Homelands":
				return
			}

			ms.printf("%v", err)
			ms.printf("%q", theCard)
			ms.printf("%s ~ %s ~ %s", cardName, edition, variant)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ms.printf("- %s", card)
				}
			}
			return
		}

		// Sanity check, a bunch of EA cards are market as foil when they
		// actually don't have a foil printing, just skip them
		if strings.Contains(title, "Foil - Extended Art") {
			co, err := mtgmatcher.GetUUID(cardId)
			if err != nil || !co.Foil {
				return
			}
		}

		if mode == modeInventory {
			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Price:      price,
					Conditions: conditions,
					Quantity:   qty,
					URL:        "https://www.mtgseattle.com" + link,
				},
			}
			channel <- out
		} else if mode == modeBuylist {
			var priceRatio, sellPrice float64

			invCards := ms.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			gradeMap := grading(cardId, price)
			for _, grade := range mtgban.DefaultGradeTags {
				var quantity int
				if grade == "NM" {
					quantity = qty
				}

				factor := gradeMap[grade]
				out := responseChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						Conditions: grade,
						BuyPrice:   price * factor,
						PriceRatio: priceRatio,
						Quantity:   quantity,
						URL:        "https://www.mtgseattle.com" + link,
					},
				}
				channel <- out
			}
		}
	})

	// Search for the next page, if not found we processed them all
	next, found := doc.Find(`a[class="next_page"]`).Attr("href")
	if !found {
		return nil
	}

	return ms.processProduct(channel, next, mode)
}

func (ms *MTGSeattle) scrape(mode string) error {
	link := inventoryURL
	if mode == modeBuylist {
		link = buylistURL
	}
	resp, err := ms.client.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	var links []string
	var titles []string
	xpath := `div[class="content inner clearfix"] ul[class="parent-category list small-12 columns eat-both fancy-row across-1  "] li a[class="clearfix"]`
	if mode == modeBuylist {
		xpath = `div[class="hidden-buylist-tree"] ul[id="category_tree"] li[class="depth_1"] ul[class="category_tree"] li a`
	}
	doc.Find(xpath).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		links = append(links, href)
		titles = append(titles, strings.TrimSpace(s.Text()))
	})

	ms.printf("Found %d categories", len(links))

	products := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < ms.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for product := range products {
				err := ms.processProduct(results, product, mode)
				if err != nil {
					ms.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i, link := range links {
			ms.printf("Processing %s", titles[i])
			products <- link
		}
		close(products)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		var err error
		if record.invEntry != nil {
			err = ms.inventory.Add(record.cardId, record.invEntry)
		} else if record.buyEntry != nil {
			err = ms.buylist.Add(record.cardId, record.buyEntry)
		}
		if err != nil {
			ms.printf("%s", err.Error())
		}
	}

	if mode == modeInventory {
		ms.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		ms.buylistDate = time.Now()
	}

	return nil
}

func (ms *MTGSeattle) Inventory() (mtgban.InventoryRecord, error) {
	if len(ms.inventory) > 0 {
		return ms.inventory, nil
	}

	err := ms.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return ms.inventory, nil
}

func (ms *MTGSeattle) Buylist() (mtgban.BuylistRecord, error) {
	if len(ms.buylist) > 0 {
		return ms.buylist, nil
	}

	err := ms.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return ms.buylist, nil
}

func grading(cardId string, price float64) map[string]float64 {
	co, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return nil
	}

	if co.Foil {
		if price >= 50 {
			return map[string]float64{
				"NM": 1, "SP": 0.8, "MP": 0.6, "HP": 0.4,
			}
		}
		if price >= 5 {
			return map[string]float64{
				"NM": 1, "SP": 0.75, "MP": 0.5, "HP": 0.3,
			}
		}
		return map[string]float64{
			"NM": 1, "SP": 0.7, "MP": 0.4, "HP": 0.25,
		}
	}

	switch co.SetCode {
	case "LEA", "LEB", "2ED":
		if price >= 50 {
			return map[string]float64{
				"NM": 1, "SP": 0.8, "MP": 0.6, "HP": 0.4,
			}
		}
		if price >= 5 {
			return map[string]float64{
				"NM": 1, "SP": 0.75, "MP": 0.55, "HP": 0.35,
			}
		}
		return map[string]float64{
			"NM": 1, "SP": 0.7, "MP": 0.5, "HP": 0.3,
		}
	}

	if price >= 50 {
		return map[string]float64{
			"NM": 1, "SP": 0.85, "MP": 0.75, "HP": 0.65,
		}
	}
	if price >= 5 {
		return map[string]float64{
			"NM": 1, "SP": 0.80, "MP": 0.7, "HP": 0.6,
		}
	}
	return map[string]float64{
		"NM": 1, "SP": 0.75, "MP": 0.6, "HP": 0.5,
	}
}

func (ms *MTGSeattle) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGSeattle"
	info.Shorthand = "MS"
	info.InventoryTimestamp = &ms.inventoryDate
	info.BuylistTimestamp = &ms.buylistDate
	info.CreditMultiplier = 1.33
	return
}
