package magiccorner

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	buylistURL = "https://www.cardgamecorner.com/it/vendi-letue-carte"
)

type Magiccorner struct {
	VerboseLog     bool
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	exchangeRate float64

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
	client    *MCClient
}

func NewScraper() (*Magiccorner, error) {
	mc := Magiccorner{}
	mc.inventory = mtgban.InventoryRecord{}
	mc.buylist = mtgban.BuylistRecord{}
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mc.exchangeRate = rate
	mc.client = NewMCClient()
	mc.MaxConcurrency = defaultConcurrency
	return &mc, nil
}

type resultChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (mc *Magiccorner) printf(format string, a ...interface{}) {
	if mc.LogCallback != nil {
		mc.LogCallback("[MC] "+format, a...)
	}
}

func (mc *Magiccorner) processEntry(channel chan<- resultChan, edition MCEdition) error {
	cards, err := mc.client.GetInventoryForEdition(edition)
	if err != nil {
		return err
	}

	printed := false

	// Keep track of the processed ids, and don't add duplicates
	duplicate := map[int]bool{}

	for _, card := range cards {
		if !printed && mc.VerboseLog {
			mc.printf("Processing id %d - %s (%s, code: %s)", edition.Id, edition.Name, card.Extra, card.Code)
			printed = true
		}

		for i, v := range card.Variants {
			// Skip duplicate cards
			if duplicate[v.Id] {
				if mc.VerboseLog {
					mc.printf("Skipping duplicate card: %s (%s %s)", card.Name, card.Edition, v.Foil)
				}
				continue
			}

			// Only keep English cards and a few other exceptions
			switch v.Language {
			case "EN":
			case "JP":
				switch edition.Name {
				case "War of the Spark: Japanese Alternate-Art Planeswalkers":
				default:
					continue
				}
			case "IT":
				switch edition.Name {
				case "Revised EU FBB":
				case "Rinascimento":
				case "L'Oscurità":
				case "Leggende":
				default:
					continue
				}
			default:
				continue
			}

			if v.Quantity < 1 {
				continue
			}

			cond := v.Condition
			switch cond {
			case "NM/M":
				cond = "NM"
			case "SP", "HP":
			case "GD":
				cond = "MP"
			case "D":
				cond = "PO"
			default:
				mc.printf("Unknown '%s' condition", cond)
				continue
			}

			theCard, err := preprocess(&card, i)
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// The basic lands need custom handling for each edition if they
				// aren't found with other methods, ignore errors until they are
				// added to the variants table.
				if mtgmatcher.IsBasicLand(card.Name) {
					continue
				}
				mc.printf("%v", err)
				mc.printf("%q", theCard)
				mc.printf("%q", card)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						mc.printf("- %s", card)
					}
				}
				continue
			}

			channel <- resultChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      v.Price * mc.exchangeRate,
					Quantity:   v.Quantity,
					URL:        "https://www.magiccorner.it" + card.URL,
					OriginalId: fmt.Sprint(card.Id),
					InstanceId: fmt.Sprint(v.Id),
				},
			}

			duplicate[v.Id] = true
		}
	}

	return nil
}

// Scrape returns an array of Entry, containing pricing and card information
func (mc *Magiccorner) scrape() error {
	editionList, err := mc.client.GetEditionList(true)
	if err != nil {
		return err
	}

	pages := make(chan MCEdition)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < mc.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := mc.processEntry(results, page)
				if err != nil {
					mc.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, edition := range editionList {
			pages <- edition
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		err = mc.inventory.AddRelaxed(result.cardId, result.invEntry)
		if err != nil {
			mc.printf("%s", err.Error())
			continue
		}
	}

	mc.inventoryDate = time.Now()

	return nil
}

func (mc *Magiccorner) Inventory() (mtgban.InventoryRecord, error) {
	if len(mc.inventory) > 0 {
		return mc.inventory, nil
	}

	err := mc.scrape()
	if err != nil {
		return nil, err
	}

	return mc.inventory, nil
}
func (mc *Magiccorner) parseBL(channel chan<- resultChan, edition MCExpansion) error {
	i := 1
	totals := 0
	for {
		mc.printf("Querying %s page %d", edition.Name, i)
		result, err := mc.client.GetBuylistForEdition(edition.Id, i)
		if err != nil {
			return err
		}

		for _, product := range result.Products {
			// Product is not being bought
			if product.SerialNumber == 99999 {
				continue
			}

			cardName := product.ModelEn
			edition := product.Category
			price := product.MinAcquisto
			credit := product.MaxAcquisto
			qty := product.Quantity
			if qty > 4 {
				qty = 4
			}

			if price == 0 {
				continue
			}

			theCard, err := preprocessBL(cardName, edition, product.ID)
			if err != nil {
				continue
			}

			cardId, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				mc.printf("%v", err)
				mc.printf("%q", theCard)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						mc.printf("- %s", card)
					}
				}
				continue
			}

			link := fmt.Sprintf("https://www.cardgamecorner.com/it/buylist?q=%s&game=magic", url.QueryEscape(product.ModelEn))

			gradeMap := map[string]float64{
				"NM": 1, "SP": 0.77, "MP": 0, "HP": 0.36,
			}
			for _, grade := range mtgban.DefaultGradeTags {
				factor := gradeMap[grade]
				if factor == 0 {
					continue
				}

				channel <- resultChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						Quantity:   qty,
						Conditions: grade,
						BuyPrice:   price * mc.exchangeRate * factor,
						TradePrice: credit * mc.exchangeRate * factor,
						URL:        link,
						OriginalId: product.ID,
					},
				}
			}
		}

		i++
		totals += len(result.Products)
		if totals >= result.Total {
			break
		}
	}

	return nil
}

func (mc *Magiccorner) Buylist() (mtgban.BuylistRecord, error) {
	if len(mc.buylist) > 0 {
		return mc.buylist, nil
	}

	editions, err := mc.client.GetBuylistEditions()
	if err != nil {
		return nil, err
	}
	mc.printf("Found %d editions", len(editions))

	editionsChan := make(chan MCExpansion)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < mc.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for edition := range editionsChan {
				err := mc.parseBL(results, edition)
				if err != nil {
					mc.printf("%v", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, edition := range editions {
			editionsChan <- edition
		}
		close(editionsChan)

		wg.Wait()
		close(results)
	}()

	for record := range results {
		err := mc.buylist.AddRelaxed(record.cardId, record.buyEntry)
		if err != nil {
			mc.printf("%s", err.Error())
			continue
		}
	}

	mc.buylistDate = time.Now()

	return mc.buylist, nil
}

func (mc *Magiccorner) Info() (info mtgban.ScraperInfo) {
	info.Name = "Magic Corner"
	info.Shorthand = "MC"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mc.inventoryDate
	info.BuylistTimestamp = &mc.buylistDate
	return
}
