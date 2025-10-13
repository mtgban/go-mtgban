package hareruya

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
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

	modeInventory = "inventory"
	modeBuylist   = "buylist"

	buylistURL = "https://www.hareruyamtg.com/ja/purchase/search?"
)

type Hareruya struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	exchangeRate float64

	inventory     mtgban.InventoryRecord
	inventoryDate time.Time
	DisableRetail bool

	buylist        mtgban.BuylistRecord
	buylistDate    time.Time
	DisableBuylist bool

	client *http.Client
}

func NewScraper() *Hareruya {
	ha := Hareruya{}
	ha.inventory = mtgban.InventoryRecord{}
	ha.buylist = mtgban.BuylistRecord{}
	ha.MaxConcurrency = defaultConcurrency
	client := retryablehttp.NewClient()
	client.Logger = nil
	ha.client = client.StandardClient()
	return &ha
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (ha *Hareruya) printf(format string, a ...interface{}) {
	if ha.LogCallback != nil {
		ha.LogCallback("[HA] "+format, a...)
	}
}

func (ha *Hareruya) processBuylistSet(ctx context.Context, channel chan<- responseChan, cardSet string) error {
	var i int
	for {
		i++
		results, canExit, err := ha.processBuylistPage(ctx, channel, cardSet, i)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		ha.printf("cardSet %s page %d found %d results", cardSet, i, len(results))

		for _, result := range results {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case channel <- result:
				// done
			}
		}

		if canExit {
			return nil
		}
	}
}

func (ha *Hareruya) processBuylistPage(ctx context.Context, channel chan<- responseChan, cardSet string, page int) ([]responseChan, bool, error) {
	var canExit bool

	v := url.Values{}
	v.Set("sort", "price")
	v.Set("order", "DESC")
	v.Set("cardId", "")
	v.Set("product", "")
	v.Set("category", "")
	v.Set("cardset", cardSet)
	v.Set("colorsType", "0")
	v.Add("rarity[]", "1")
	v.Add("rarity[]", "2")
	v.Add("rarity[]", "3")
	v.Add("rarity[]", "4")
	v.Set("cardtypesType", "0")
	v.Set("subtype", "")
	v.Set("format", "")
	v.Set("illustrator", "")
	v.Add("foilFlg[]", "0")
	v.Add("foilFlg[]", "1")
	v.Set("language[]", "2") // English only
	v.Set("page", fmt.Sprint(page))

	link := buylistURL + v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, canExit, err
	}
	resp, err := ha.client.Do(req)
	if err != nil {
		return nil, canExit, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, canExit, err
	}

	var results []responseChan
	doc.Find(`.itemList`).EachWithBreak(func(i int, s *goquery.Selection) bool {
		link, _ := s.Find(`div.itemData a`).Attr("href")
		link = strings.TrimSpace(link)
		id := strings.Split(path.Base(link), "?")[0]

		title := s.Find(`div.itemData a`).Text()
		title = strings.TrimSpace(title)

		priceStr := s.Find(`p.itemDetail__price`).Text()
		priceStr = strings.TrimPrefix(priceStr, "¥ ")
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			ha.printf("page %d entry %d: %s", page, i, err.Error())
			return false
		}

		// Since we're sorting by price, as soon as we found an item that is not
		// being bought we can assume there are no more items in this set
		stock := s.Find(`span.itemUserAct__number__title`).Text()
		if stock != "個数" {
			canExit = true
			return false
		}

		theCard, err := preprocess(title)
		if err != nil {
			return true
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return true
		} else if err != nil {
			if theCard.IsBasicLand() {
				return true
			}
			ha.printf("%v in cardSet %s at page %d", err, cardSet, page)
			ha.printf("%q", theCard)
			ha.printf("%s", title)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ha.printf("- %s", card)
				}
			}
			return true
		}

		var priceRatio, sellPrice float64

		invCards := ha.inventory[cardId]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}

		deductions := []float64{1, 0.8, 0.5}
		for i, deduction := range deductions {
			out := responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions: mtgban.DefaultGradeTags[i],
					BuyPrice:   price * deduction * ha.exchangeRate,
					PriceRatio: priceRatio,
					URL:        "https://www.hareruyamtg.com" + link,
					OriginalId: id,
				},
			}

			results = append(results, out)
		}

		// Continue to next element
		return true
	})

	return results, canExit, nil
}

var condMap = map[string]string{
	"1": "NM",
	"2": "SP",
	"3": "MP",
	"4": "HP",
	"5": "PO",
}

func (ha *Hareruya) processSet(ctx context.Context, channel chan<- responseChan, cardSet string) error {
	var i int
	for {
		i++
		products, err := SearchCardSet(ctx, ha.client, cardSet, i)
		if err != nil {
			return err
		}

		if len(products) == 0 {
			return nil
		}

		for _, product := range products {
			theCard, err := Preprocess(product)
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// Skip errors from lands, "misc" promos, and tokens
				if theCard.IsBasicLand() ||
					strings.Contains(theCard.Edition, "The List") || // lots at set 280
					strings.Contains(theCard.Edition, "Mystery Booster") || // lots at set 280
					strings.Contains(product.ProductName, "Token") {
					continue
				}
				ha.printf("%v at set %s (page %d)", err, cardSet, i)
				ha.printf("%q", theCard)
				ha.printf("%v", product)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						ha.printf("- %s", card)
					}
				}
				continue
			}

			cond, found := condMap[product.CardCondition]
			if !found {
				ha.printf("%v condition '%s' not found", product, product.CardCondition)
				continue
			}

			link := "https://www.hareruyamtg.com/en/products/detail/" + product.Product + "?lang=EN&class=" + product.ProductClass
			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Price:      product.Price * ha.exchangeRate,
					Conditions: cond,
					URL:        link,
					OriginalId: product.Product,
				},
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case channel <- out:
				// done
			}
		}
	}
}

func (ha *Hareruya) getCardSets(ctx context.Context) ([]string, error) {
	link := "https://www.hareruyamtg.com/en/products/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := ha.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []string
	doc.Find("select#front_product_search_cardset option").Each(func(i int, s *goquery.Selection) {
		value, _ := s.Attr("value")
		if value == "" {
			return
		}
		results = append(results, value)
	})

	// Sort to keep results predictable and find min/max
	slices.SortFunc(results, func(a, b string) int {
		if len(a) > len(b) {
			return 1
		}
		if len(a) < len(b) {
			return -1
		}
		return strings.Compare(a, b)
	})

	return results, nil
}

func (ha *Hareruya) scrape(ctx context.Context, mode string) error {
	rate, err := mtgban.GetExchangeRate(ctx, "JPY")
	if err != nil {
		return err
	}
	ha.exchangeRate = rate

	cardSets, err := ha.getCardSets(ctx)
	if err != nil {
		return err
	}
	ha.printf("Found %d card sets", len(cardSets))

	sets := make(chan string)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < ha.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for cardSet := range sets {
				if mode == modeInventory {
					err = ha.processSet(ctx, results, cardSet)
				} else if mode == modeBuylist {
					err = ha.processBuylistSet(ctx, results, cardSet)
				}
				if err != nil {
					ha.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i, cardSet := range cardSets {
			ha.printf("Processing card set %s (%d/%d)", cardSet, i+1, len(cardSets))
			sets <- cardSet
		}
		close(sets)

		wg.Wait()
		close(results)
	}()

	if mode == modeInventory {
		for record := range results {
			err := ha.inventory.Add(record.cardId, record.invEntry)
			if err != nil {
				ha.printf("%s", err.Error())
			}
		}

		ha.inventoryDate = time.Now()
	} else if mode == modeBuylist {
		for record := range results {
			co, _ := mtgmatcher.GetUUID(record.cardId)
			// This store tracks the two different EU/US printings as separate entries
			if co.SetCode == "MPS" {
				err = ha.buylist.AddRelaxed(record.cardId, record.buyEntry)
			} else {
				err = ha.buylist.Add(record.cardId, record.buyEntry)
			}
			if err != nil {
				ha.printf("%s", err.Error())
			}
		}

		ha.buylistDate = time.Now()
	}

	return nil
}

func (ha *Hareruya) SetConfig(opt mtgban.ScraperOptions) {
	ha.DisableRetail = opt.DisableRetail
	ha.DisableBuylist = opt.DisableBuylist
}

func (ha *Hareruya) Load(ctx context.Context) error {
	var errs []error

	if !ha.DisableRetail {
		err := ha.scrape(ctx, modeInventory)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !ha.DisableBuylist {
		err := ha.scrape(ctx, modeBuylist)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (ha *Hareruya) Inventory() mtgban.InventoryRecord {
	return ha.inventory
}

func (ha *Hareruya) Buylist() mtgban.BuylistRecord {
	return ha.buylist
}

func (ha *Hareruya) Info() (info mtgban.ScraperInfo) {
	info.Name = "Hareruya"
	info.Shorthand = "HA"
	info.CountryFlag = "JP"
	info.NoQuantityInventory = true
	info.InventoryTimestamp = &ha.inventoryDate
	info.BuylistTimestamp = &ha.buylistDate
	return
}
