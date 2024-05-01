package toamagic

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

	inventoryURL = "https://www.toamagic.com/catalog/magic_singles/8?layout=false"
	buylistURL   = "https://www.toamagic.com/buylist/magic_singles/8?layout=false"

	modeInventory = "inventory"
	modeBuylist   = "buylist"
)

type TOAMagic struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *retryablehttp.Client
}

func NewScraper() *TOAMagic {
	toa := TOAMagic{}
	toa.inventory = mtgban.InventoryRecord{}
	toa.buylist = mtgban.BuylistRecord{}
	toa.MaxConcurrency = defaultConcurrency
	toa.client = retryablehttp.NewClient()
	toa.client.Logger = nil
	return &toa
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (toa *TOAMagic) printf(format string, a ...interface{}) {
	if toa.LogCallback != nil {
		toa.LogCallback("[TOA] "+format, a...)
	}
}

func (toa *TOAMagic) processProduct(channel chan<- responseChan, productPath, mode string) error {
	resp, err := toa.client.Get("https://www.toamagic.com" + productPath + "?layout=false&filter_by_stock=in-stock")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	edition := doc.Find(`h1[class="page-title"]`).Text()
	edition = strings.TrimSuffix(edition, " Art Variants")

	// Inventory mode needs to might need to discover additional elements
	if strings.HasSuffix(edition, "Block") ||
		strings.HasSuffix(edition, "Sets") ||
		strings.HasSuffix(edition, "Editions") ||
		strings.HasSuffix(edition, "Edition") ||
		strings.HasSuffix(edition, "Decks") ||
		strings.HasSuffix(edition, "From the Vault") ||
		strings.HasSuffix(edition, "Collector Booster Era") ||
		strings.HasSuffix(edition, "Promos") {
		var links []string
		doc.Find(`a[class="clearfix"]`).Each(func(_ int, s *goquery.Selection) {
			link, _ := s.Attr("href")
			links = append(links, link)
		})

		for _, link := range links {
			err := toa.processProduct(channel, link, mode)
			if err != nil {
				toa.printf("%s", err.Error())
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

		creditStr := s.Find(`div[class="product-price"] span[class="store-credit"]`).Text()
		creditStr = strings.TrimSpace(creditStr)
		creditStr = strings.TrimPrefix(creditStr, "/ ")
		creditStr = strings.TrimSuffix(creditStr, "credit")
		creditStr = strings.TrimSpace(creditStr)

		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			return
		}

		price, _ := mtgmatcher.ParsePrice(priceStr)
		if price == 0 {
			return
		}
		credit, _ := mtgmatcher.ParsePrice(creditStr)

		conditions := ""
		if mode == modeInventory {
			cond := strings.Split(condLang, ", ")[0]
			cond = strings.TrimPrefix(cond, "Website Exclusive ")
			switch cond {
			case "NM-Mint", "NM":
				conditions = "NM"
			case "Light Play", "LP":
				conditions = "SP"
			case "Moderate Play":
				conditions = "MP"
			case "Heavy Play":
				conditions = "HP"
			case "Damaged":
				conditions = "PO"
			case "Graded":
				return
			default:
				toa.printf("Unsupported %s condition for %s", cond, title)
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
			switch edition {
			case "Homelands",
				"Fallen Empires":
				return
			case "Magic 2010 M10",
				"Mirage",
				"Portal",
				"Portal Second Age",
				"Tempest":
				if mtgmatcher.IsBasicLand(cardName) {
					return
				}
			}

			toa.printf("%v", err)
			toa.printf("%q", theCard)
			toa.printf("%s ~ %s ~ %s", cardName, edition, variant)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					toa.printf("- %s", card)
				}
			}
			return
		}

		if mode == modeInventory {
			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Price:      price,
					Conditions: conditions,
					Quantity:   qty,
					URL:        "https://www.toamagic.com" + link,
				},
			}
			channel <- out
		} else if mode == modeBuylist {
			var priceRatio, sellPrice float64

			invCards := toa.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			deductions := []float64{1, 1, 0.5, 0.3}
			for i, deduction := range deductions {
				var quantity int
				if i == 0 {
					quantity = qty
				}

				out := responseChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						Conditions: mtgban.DefaultGradeTags[i],
						BuyPrice:   price * deduction,
						TradePrice: credit * deduction,
						PriceRatio: priceRatio,
						Quantity:   quantity,
						URL:        "https://www.toamagic.com" + link,
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

	return toa.processProduct(channel, next, mode)
}

func (toa *TOAMagic) scrape(mode string) error {
	link := inventoryURL
	if mode == modeBuylist {
		link = buylistURL
	}
	resp, err := toa.client.Get(link)
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
	xpath := `ul[class="parent-category list small-12 column eat-both across-3  "] li`
	doc.Find(xpath).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Find("a").Attr("href")
		links = append(links, href)
		title := strings.TrimSpace(s.Find(`div[class="name"]`).Text())
		titles = append(titles, title)
	})

	toa.printf("Found %d categories", len(links))

	products := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < toa.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for productPath := range products {
				err := toa.processProduct(results, productPath, mode)
				if err != nil {
					toa.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i, link := range links {
			toa.printf("Processing %s", titles[i])
			products <- link
		}
		close(products)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		var err error
		if record.invEntry != nil {
			err = toa.inventory.AddRelaxed(record.cardId, record.invEntry)
		} else if record.buyEntry != nil {
			err = toa.buylist.Add(record.cardId, record.buyEntry)
		}
		if err != nil {
			toa.printf("%s", err.Error())
		}
	}

	if mode == modeInventory {
		toa.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		toa.buylistDate = time.Now()
	}

	return nil
}

func (toa *TOAMagic) Inventory() (mtgban.InventoryRecord, error) {
	if len(toa.inventory) > 0 {
		return toa.inventory, nil
	}

	err := toa.scrape(modeInventory)
	if err != nil {
		return nil, err
	}

	return toa.inventory, nil
}

func (toa *TOAMagic) Buylist() (mtgban.BuylistRecord, error) {
	if len(toa.buylist) > 0 {
		return toa.buylist, nil
	}

	err := toa.scrape(modeBuylist)
	if err != nil {
		return nil, err
	}

	return toa.buylist, nil
}

func (toa *TOAMagic) Info() (info mtgban.ScraperInfo) {
	info.Name = "Tales of Adventure"
	info.Shorthand = "TOA"
	info.InventoryTimestamp = &toa.inventoryDate
	info.BuylistTimestamp = &toa.buylistDate
	return
}
