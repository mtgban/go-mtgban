package starcitygames

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency  = 3
	defaultRequestLimit = 200

	buylistBookmark = "https://sellyourcards.starcitygames.com/mtg/bookmark/"
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
	game   int
}

func NewScraper(game int, guid, bearer string) *Starcitygames {
	scg := Starcitygames{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(guid, bearer)
	scg.MaxConcurrency = defaultConcurrency
	scg.game = game
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
	results, err := scg.client.GetPage(scg.game, page)
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
		id := fmt.Sprint(result.Document.UniqueID[0])

		var number string
		if len(result.Document.CollectorNumber) > 0 {
			number = result.Document.CollectorNumber[0]
		}
		var variant string
		if len(result.Document.Subtitle) > 0 {
			variant += result.Document.Subtitle[0]
		}
		var sku string
		if len(result.Document.HawkChildAttributes) > 0 &&
			len(result.Document.HawkChildAttributes[0].VariantSKU) > 0 {
			sku = result.Document.HawkChildAttributes[0].VariantSKU[0]
		}

		var cardId string
		switch scg.game {
		case GameMagic:
			cc := SCGCardVariant{
				Name:     cardName,
				Subtitle: variant,
				Sku:      sku,
			}
			theCard, err := preprocess(&cc, edition, language, finish == "Foil", number)
			if err != nil {
				continue
			}

			cardId, err = mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// Skip errors from tokens and similar
				if strings.Contains(cardName, "Token") ||
					strings.Contains(variant, "Token") ||
					strings.HasPrefix(cardName, "{") {
					continue
				}
				scg.printf("%v", err)
				scg.printf("%q", theCard)
				scg.printf("%v ~ %s ~ %s ~ %s", cc, edition, finish, number)

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
		default:
			return errors.New("unsupported game")
		}

		customFields := map[string]string{
			"SCGName":     cardName,
			"SCGEdition":  edition,
			"SCGLanguage": language,
			"SCGFinish":   finish,
			"scgSubtitle": variant,
			"scgNumber":   number,
			"scgSKU":      sku,
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
			qty := attribute.Qty[0]
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
					URL:        SCGProductURL(result.Document.URLDetail, attribute.VariantSKU, scg.Affiliate),
				},
				ignoreErr: strings.Contains(edition, "World Championship"),
			}
			if condition == "NM" {
				out.invEntry.CustomFields = customFields
			}
			channel <- out
		}
	}

	return nil
}

func (scg *Starcitygames) scrape() error {
	totalPages, err := scg.client.NumberOfPages(scg.game)
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
		err := scg.inventory.AddStrict(record.cardId, record.invEntry)
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
	search, err := scg.client.SearchAll(scg.game, page, defaultRequestLimit)
	if err != nil {
		return err
	}

	for _, hit := range search.Hits {
		link, _ := url.JoinPath(
			buylistBookmark,
			url.QueryEscape(hit.Name),
			"0/1/0/0", // various faucets (bulk, hotlist, etc)
			fmt.Sprint(hit.SetID),
			hit.Language,
			",",           // rarity
			"0/999999.99", // min/max price range
			hit.Finish,    // N F or N,F
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
				if hit.Finish == "F" {
					conditions = "MP"
				}
			case "HP":
				conditions = "MP"
				// Stricter grading for foils
				if hit.Finish == "F" {
					conditions = "HP"
				}
			default:
				scg.printf("unknown condition %s for %v", conditions, result)
				continue
			}

			var cardId string
			if scg.game == GameMagic {
				theCard, err := preprocess(&result, hit.SetName, hit.Language, hit.Finish == "F", hit.CollectorNumber)
				if err != nil {
					break
				}

				cardId, err = mtgmatcher.Match(theCard)
				if errors.Is(err, mtgmatcher.ErrUnsupported) {
					break
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
					break
				}
			} else {
				return errors.New("unsupported game")
			}

			var priceRatio, sellPrice float64
			price := result.BuyPrice

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
					// custom, helps debugging
					"scgSubtitle": hit.Subtitle,
					"scgNumber":   hit.CollectorNumber,
					"scgSKU":      result.Sku,
				}
			}

			channel <- responseChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions:   conditions,
					BuyPrice:     price,
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
	search, err := scg.client.SearchAll(scg.game, 0, 1)
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
			co, _ := mtgmatcher.GetUUID(record.cardId)
			if co.Layout == "token" {
				continue
			}
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
	info.CreditMultiplier = 1.3
	switch scg.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	}
	return
}
