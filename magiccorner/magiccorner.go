package magiccorner

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	excelize "github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8

	buylistURL = "https://www.magiccorner.it/it/vendi-letue-carte"
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
	cardId string
	entry  *mtgban.InventoryEntry
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
				entry: &mtgban.InventoryEntry{
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
		err = mc.inventory.AddRelaxed(result.cardId, result.entry)
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

func getSpreadsheetURL() (string, error) {
	resp, err := cleanhttp.DefaultClient().Get(buylistURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	link, found := doc.Find(`div[class="panel-body"] ul li a`).First().Attr("href")
	if !found {
		return "", errors.New("spreadsheet anchor tag not found")
	}

	return "https://www.magiccorner.it/" + link, nil
}

func (mc *Magiccorner) parseBL() error {
	blURL, err := getSpreadsheetURL()
	if err != nil {
		return err
	}
	mc.printf("Using %s as input spreadsheet", blURL)

	resp, err := cleanhttp.DefaultClient().Get(blURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		return err
	}

	// Get all the rows in the Sheet1.
	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		return err
	}
	for i, row := range rows {
		if i < 5 || len(row) < 8 {
			continue
		}
		cardName := row[1]
		edition := row[3]
		price, _ := strconv.ParseFloat(row[4], 64)
		priceFoil, _ := strconv.ParseFloat(row[8], 64)

		if cardName == "" || edition == "" || price == 0 {
			continue
		}

		theCard, err := preprocessBL(cardName, edition)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			mc.printf("%v", err)
			mc.printf("%q", theCard)
			mc.printf("%q", row)

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

		out := &mtgban.BuylistEntry{
			BuyPrice: price * mc.exchangeRate,
			URL:      "https://www.magiccorner.it/it/vendi-letue-carte",
		}
		err = mc.buylist.Add(cardId, out)
		if err != nil {
			mc.printf("%v", err)
		}

		// Repeat for foils, or skip
		if priceFoil == 0 {
			continue
		}

		theCard.Foil = true

		cardId, err = mtgmatcher.Match(theCard)
		if err != nil {
			mc.printf("%v", err)
			mc.printf("%q", theCard)
			mc.printf("%q", row)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					mc.printf("- %s", card)
				}
			}
			continue
		}

		out = &mtgban.BuylistEntry{
			BuyPrice: price * mc.exchangeRate,
			URL:      buylistURL,
		}
		err = mc.buylist.Add(cardId, out)
		if err != nil {
			mc.printf("%v", err)
		}
	}

	mc.buylistDate = time.Now()

	return nil
}

func (mc *Magiccorner) Buylist() (mtgban.BuylistRecord, error) {
	if len(mc.buylist) > 0 {
		return mc.buylist, nil
	}

	err := mc.parseBL()
	if err != nil {
		return nil, err
	}

	return mc.buylist, nil
}

var eighthEditionDate = time.Date(2003, time.July, 1, 0, 0, 0, 0, time.UTC)

func grading(cardId string, entry mtgban.BuylistEntry) map[string]float64 {
	set, err := mtgmatcher.GetSetUUID(cardId)
	if err != nil {
		return nil
	}

	setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
	if err != nil {
		return nil
	}

	if setDate.After(eighthEditionDate) {
		return map[string]float64{
			"SP": 0.8, "MP": 0.7, "HP": 0.5,
		}
	}

	return map[string]float64{
		"SP": 0.7, "MP": 0.5, "HP": 0.3,
	}
}

func (mc *Magiccorner) Info() (info mtgban.ScraperInfo) {
	info.Name = "Magic Corner"
	info.Shorthand = "MC"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = mc.inventoryDate
	info.BuylistTimestamp = mc.buylistDate
	info.Grading = grading
	info.NoCredit = true
	return
}
