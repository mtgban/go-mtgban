package secretdeskorrigans

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultConcurrency = 8

	inventoryURL = "https://www.lesecretdeskorrigans.com/catalog/magic_singles/8?layout=false"
)

type SecretDesKorrigans struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord

	exchangeRate float64

	client *http.Client
}

func NewScraper() (*SecretDesKorrigans, error) {
	sdk := SecretDesKorrigans{}
	sdk.inventory = mtgban.InventoryRecord{}
	sdk.MaxConcurrency = defaultConcurrency
	client := retryablehttp.NewClient()
	client.Logger = nil
	sdk.client = client.StandardClient()
	return &sdk, nil
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
}

func (sdk *SecretDesKorrigans) printf(format string, a ...interface{}) {
	if sdk.LogCallback != nil {
		sdk.LogCallback("[SDK] "+format, a...)
	}
}

func (sdk *SecretDesKorrigans) processProduct(ctx context.Context, channel chan<- responseChan, productPath string) error {
	link := "https://www.lesecretdeskorrigans.com" + productPath + "?layout=false&filter_by_stock=in-stock"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := sdk.client.Do(req)
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
	edition = strings.TrimSuffix(edition, " Singles")
	edition = strings.TrimSuffix(edition, " singles")

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
			err := sdk.processProduct(ctx, channel, link)
			if err != nil {
				sdk.printf("%s", err.Error())
			}
		}
		return nil
	}

	xpath := `ul[class="products"] li[class="product"] div[class="inner"] div[class="meta"]`
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
		condLang := s.Find(container + ` span[class="variant-short-info variant-description"]`).Text()

		qtyStr := s.Find(container + ` span[class="variant-short-info variant-qty"]`).Text()
		qtyStr = strings.TrimPrefix(qtyStr, "Limit ")
		qtyStr = strings.TrimSuffix(qtyStr, " En stock")
		qtyStr = strings.TrimSuffix(qtyStr, " En Stock")
		qtyStr = strings.TrimSuffix(qtyStr, " In Stock")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			return
		}

		priceStr := s.Find(`div[class="product-price"] span[class="regular price"]`).Text()
		priceStr = strings.TrimPrefix(priceStr, "CAD")
		price, _ := mtgmatcher.ParsePrice(priceStr)
		if price == 0 {
			sdk.printf("price error '%s': %s", priceStr, err.Error())
			return
		}

		conditions := ""
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
			sdk.printf("Unsupported %s condition for %s", cond, title)
			return
		}

		//log.Println(cardName, edition, variant, cond, price)

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

			sdk.printf("%v", err)
			sdk.printf("%q", theCard)
			sdk.printf("%s | %s | %s", cardName, edition, variant)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					sdk.printf("- %s", card)
				}
			}
			return
		}

		out := responseChan{
			cardId: cardId,
			invEntry: &mtgban.InventoryEntry{
				Price:      price * sdk.exchangeRate,
				Conditions: conditions,
				Quantity:   qty,
				URL:        "https://www.lesecretdeskorrigans.com" + link,
			},
		}
		channel <- out
	})

	// Search for the next page, if not found we processed them all
	next, found := doc.Find(`a[class="next_page"]`).Attr("href")
	if !found {
		return nil
	}

	return sdk.processProduct(ctx, channel, next)
}

func (sdk *SecretDesKorrigans) scrape(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate("CAD")
	if err != nil {
		return err
	}
	sdk.exchangeRate = rate

	link := inventoryURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := sdk.client.Do(req)
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
	xpath := `ul.parent-category li`
	doc.Find(xpath).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Find("a").Attr("href")
		links = append(links, href)
		title := strings.TrimSpace(s.Find(`div[class="name"]`).Text())
		titles = append(titles, title)
	})

	sdk.printf("Found %d categories", len(links))

	products := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < sdk.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for productPath := range products {
				err := sdk.processProduct(ctx, results, productPath)
				if err != nil {
					sdk.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i, link := range links {
			sdk.printf("Processing %s", titles[i])
			products <- link
		}
		close(products)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := sdk.inventory.AddRelaxed(record.cardId, record.invEntry)
		if err != nil {
			sdk.printf("%s", err.Error())
		}
	}

	sdk.inventoryDate = time.Now()

	return nil
}

func (sdk *SecretDesKorrigans) Inventory() (mtgban.InventoryRecord, error) {
	if len(sdk.inventory) > 0 {
		return sdk.inventory, nil
	}

	err := sdk.scrape(context.TODO())
	if err != nil {
		return nil, err
	}

	return sdk.inventory, nil
}

func (sdk *SecretDesKorrigans) Info() (info mtgban.ScraperInfo) {
	info.Name = "Le Secret des Korrigans"
	info.Shorthand = "SK"
	info.InventoryTimestamp = &sdk.inventoryDate
	return
}
