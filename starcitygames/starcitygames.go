package starcitygames

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	buylistBookmark = "https://sellyourcards.starcitygames.com/bookmark/"
)

type Starcitygames struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *SCGClient
}

func NewScraper(bearer string) *Starcitygames {
	scg := Starcitygames{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(bearer)
	scg.MaxConcurrency = defaultConcurrency
	return &scg
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry

	ignoreErr bool
}

func (scg *Starcitygames) printf(format string, a ...interface{}) {
	if scg.LogCallback != nil {
		scg.LogCallback("[SCG] "+format, a...)
	}
}

func (scg *Starcitygames) processPage(channel chan<- responseChan, page int) error {
	resp, err := scg.client.GetPage(page)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// Iterate on each result
	doc.Find(`div[class="hawk-results-item"]`).Each(func(_ int, s *goquery.Selection) {
		cardName := s.Find(`h2[class="hawk-results-item__title"]`).Text()
		edition := s.Find(`p[class="hawk-results-item__category"]`).Text()
		link, _ := s.Find(`h2[class="hawk-results-item__title"] a`).Attr("href")
		link = "http://starcitygames.com" + link

		// Iterate on each condition
		s.Find(`div[class="hawk-results-item__options-table-row"]`).Each(func(_ int, se *goquery.Selection) {
			condLang := se.Find(`div[class="hawk-results-item__options-table-cell hawk-results-item__options-table-cell--name childCondition"]`).Text()
			fields := strings.Split(condLang, " - ")
			if len(fields) < 2 {
				scg.printf("invalid condLang format: %s", condLang)
				return
			}
			conditions := strings.TrimSpace(fields[0])
			language := strings.TrimSpace(fields[1])

			cc := SCGCardVariant{
				Name: cardName,
			}
			theCard, err := preprocess(&cc, edition, language, false)
			if err != nil {
				return
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				return
			} else if err != nil {
				scg.printf("%v", err)
				scg.printf("%q", theCard)
				scg.printf("%v ~ %s", cc, edition)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						scg.printf("- %s", card)
					}
				}
				return
			}

			switch conditions {
			case "Near Mint":
				conditions = "NM"
			case "Played":
				conditions = "SP"
			case "Heavily Played":
				conditions = "HP"
			default:
				scg.printf("unknown condition %s for %s", conditions, cardName)
				return
			}

			priceStr := se.Find(`span[class="hawkSalePrice"]`).Text()
			if priceStr == "" {
				priceStr = se.Find(`span[class="hawk-old-price"]`).Text()
				if priceStr == "" {
					priceStr = se.Find(`div[class="hawk-results-item__options-table-cell hawk-results-item__options-table-cell--price childAttributes"]`).Text()
				}
			}
			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				scg.printf("invalid price for %s: %s", cardName, err.Error())
				return
			}

			qtyStr := se.Find(`div[class="hawk-results-item__options-table-cell hawk-results-item__options-table-cell--qty childAttributes"]`).Text()
			qtyStr = strings.TrimPrefix(qtyStr, "QTY: ")
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				scg.printf("invalid price for %s: %s", cardName, err.Error())
				return
			}

			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Price:      price,
					Conditions: conditions,
					Quantity:   qty,
					URL:        link,
				},
				ignoreErr: strings.Contains(edition, "World Championship") && theCard.IsBasicLand(),
			}
			channel <- out
		})
	})

	return nil
}

func (scg *Starcitygames) scrape() error {
	items, err := scg.client.NumberOfItems()
	if err != nil {
		return err
	}
	totalPages := items/DefaultRequestLimit + 1
	scg.printf("Found %d items for %d pages", items, totalPages)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < scg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := scg.processPage(results, page)
				if err != nil {
					scg.printf("%v", err)
				}
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
		close(results)
	}()

	for record := range results {
		err := scg.inventory.Add(record.cardId, record.invEntry)
		if err != nil && !record.ignoreErr {
			scg.printf("%s", err.Error())
		}
	}

	scg.inventoryDate = time.Now()

	return nil
}

func (scg *Starcitygames) Inventory() (mtgban.InventoryRecord, error) {
	if len(scg.inventory) > 0 {
		return scg.inventory, nil
	}

	err := scg.scrape()
	if err != nil {
		return nil, err
	}

	return scg.inventory, nil

}

func (scg *Starcitygames) processBLPage(channel chan<- responseChan, page int) error {
	search, err := scg.client.SearchAll(page, DefaultRequestLimit)
	if err != nil {
		return err
	}

	for _, hit := range search.Hits {
		foil := ","
		if hit.Finish == "foil" {
			foil = "f"
		}
		link, _ := url.JoinPath(
			buylistBookmark,
			url.QueryEscape(hit.Name),
			",/0/0/0", // various faucets (hot list, rarity, bulk etc)
			fmt.Sprint(hit.SetID),
			",", // unclear
			hit.Language,
			"0/999999.99", // min/max price range
			foil,
			"default",
		)

		for _, result := range hit.Variants {
			conditions := result.VariantValue
			switch conditions {
			case "NM", "NM/M":
				conditions = "NM"
			case "PL":
				conditions = "SP"
				// Stricter grading for foils
				if hit.Finish == "foil" {
					conditions = "MP"
				}
			case "HP":
				conditions = "MP"
				// Stricter grading for foils
				if hit.Finish == "foil" {
					conditions = "HP"
				}
			default:
				scg.printf("unknown condition %s for %v", conditions, result)
				continue
			}

			theCard, err := preprocess(&result, hit.SetName, hit.Language, hit.Finish == "foil")
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				scg.printf("%v", err)
				scg.printf("%q", theCard)
				scg.printf("'%q' (%s, %s, %s)", result, hit.SetName, hit.Language, hit.Finish)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						scg.printf("- %s", card)
					}
				}
				continue
			}

			var priceRatio, sellPrice float64
			price := result.BuyPrice
			trade := result.TradePrice

			invCards := scg.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}
			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			// Add the line entry as needed by the csv import
			var customFields map[string]string
			if conditions == "NM" {
				customFields = map[string]string{
					"SCGName":     hit.Name,
					"SCGEdition":  hit.SetName,
					"SCGLanguage": hit.Language,
					"SCGFinish":   hit.Finish,
				}
			}

			channel <- responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions:   conditions,
					BuyPrice:     price,
					TradePrice:   trade,
					Quantity:     0,
					PriceRatio:   priceRatio,
					URL:          link,
					CustomFields: customFields,
				},
			}
		}
	}
	return nil
}

func (scg *Starcitygames) parseBL() error {
	search, err := scg.client.SearchAll(0, 1)
	if err != nil {
		return err
	}
	scg.printf("Parsing %d cards", search.EstimatedTotalHits)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < scg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				scg.printf("Processing page %d", page)
				err := scg.processBLPage(results, page)
				if err != nil {
					scg.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for j := 0; j < search.EstimatedTotalHits; j += DefaultRequestLimit {
			pages <- j
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := scg.buylist.Add(record.cardId, record.buyEntry)
		if err != nil {
			scg.printf("%s", err.Error())
			continue
		}
	}

	scg.buylistDate = time.Now()

	return nil
}

func (scg *Starcitygames) Buylist() (mtgban.BuylistRecord, error) {
	if len(scg.buylist) > 0 {
		return scg.buylist, nil
	}

	err := scg.parseBL()
	if err != nil {
		return nil, err
	}

	return scg.buylist, nil
}

func (scg *Starcitygames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCG"
	info.InventoryTimestamp = &scg.inventoryDate
	info.BuylistTimestamp = &scg.buylistDate
	return
}
