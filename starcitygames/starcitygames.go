package starcitygames

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency  = 8
	defaultRequestLimit = 200

	buylistBookmark = "https://sellyourcards.starcitygames.com/bookmark/"
)

type Starcitygames struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	Affiliate string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *SCGClient
}

func NewScraper(guid, bearer string) *Starcitygames {
	scg := Starcitygames{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(guid, bearer)
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
	results, err := scg.client.GetPage(page)
	if err != nil {
		return err
	}

	for _, result := range results {
		if len(result.Document.ProductType) == 0 {
			return errors.New("malformed product_type")
		}
		if result.Document.ProductType[0] != "Singles" {
			scg.printf("Skipping product_type %s", result.Document.ProductType[0])
			continue
		}

		if len(result.Document.CardName) == 0 {
			return errors.New("malformed card_name")
		}
		if len(result.Document.Set) == 0 {
			return errors.New("malformed set")
		}
		if len(result.Document.Finish) == 0 {
			return errors.New("malformed finish")
		}
		if len(result.Document.Language) == 0 {
			return errors.New("malformed language")
		}
		if len(result.Document.UniqueID) == 0 {
			return errors.New("malformed unique_id")
		}
		cardName := result.Document.CardName[0]
		edition := result.Document.Set[0]
		finish := result.Document.Finish[0]
		language := result.Document.Language[0]
		id := result.Document.UniqueID[0]

		var number string
		if len(result.Document.CollectorNumber) > 0 {
			number = result.Document.CollectorNumber[0]
		}
		var variant string
		if len(result.Document.Subtitle) > 0 {
			variant += result.Document.Subtitle[0]
		}

		cc := SCGCardVariant{
			Name:     cardName,
			Subtitle: variant,
		}
		theCard, err := preprocess(&cc, edition, language, finish == "Foil", number)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil && strings.Contains(edition, "Planeswalker Symbol Reprints") {
			continue
		} else if err != nil {
			scg.printf("%v", err)
			scg.printf("%q", theCard)
			scg.printf("%v ~ %s ~ %s", cc, edition, number)

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

		var link string
		if len(result.Document.URLDetail) > 0 {
			link = "https://starcitygames.com" + result.Document.URLDetail[0]
			if scg.Affiliate != "" {
				link += "?aff=" + scg.Affiliate
			}
		}

		for _, attribute := range result.Document.HawkChildAttributes {
			if len(attribute.VariantLanguage) == 0 {
				return errors.New("malformed variant_language")
			}

			if attribute.VariantLanguage[0] != language {
				continue
			}

			if len(attribute.Price) == 0 {
				return errors.New("malformed price")
			}
			if len(attribute.Qty) == 0 {
				return errors.New("malformed qty")
			}
			if len(attribute.Condition) == 0 {
				return errors.New("malformed condition")
			}
			priceStr := attribute.Price[0]
			qtyStr := attribute.Qty[0]
			condition := attribute.Condition[0]

			switch condition {
			case "Near Mint":
				condition = "NM"
			case "Played":
				condition = "SP"
			case "Heavily Played":
				condition = "HP"
			default:
				scg.printf("unknown condition %s for %s", condition, cardName)
				continue
			}

			price, err := mtgmatcher.ParsePrice(priceStr)
			if err != nil {
				scg.printf("invalid price for %s: %s", cardName, err.Error())
				continue
			}

			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				scg.printf("invalid price for %s: %s", cardName, err.Error())
				continue
			}

			if qty == 0 || price == 0 {
				continue
			}

			out := responseChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Price:      price,
					Conditions: condition,
					Quantity:   qty,
					OriginalId: id,
					URL:        link,
				},
				ignoreErr: strings.Contains(edition, "World Championship") && theCard.IsBasicLand(),
			}
			channel <- out
		}
	}

	return nil
}

func (scg *Starcitygames) scrape() error {
	totalPages, err := scg.client.NumberOfPages()
	if err != nil {
		return err
	}
	scg.printf("Found %d pages", totalPages)

	pages := make(chan int)
	results := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < scg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				scg.printf("Processing page %d", page)
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
	search, err := scg.client.SearchAll(page, defaultRequestLimit)
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

			theCard, err := preprocess(&result, hit.SetName, hit.Language, hit.Finish == "foil", "")
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
		for j := 0; j < search.EstimatedTotalHits; j += defaultRequestLimit {
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
