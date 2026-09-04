// Package magiccorner scrapes Magic Corner.
package magiccorner

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

const (
	defaultConcurrency = 8
)

// Magiccorner prices Magic Corner's singles, both what they sell and what they
// buy.
type Magiccorner struct {
	VerboseLog     bool
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	MaxConcurrency int

	DisableRetail  bool
	DisableBuylist bool

	exchangeRate float64

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
	client    *MCClient
}

// NewScraper returns a scraper, failing if the edition list cannot be read.
func NewScraper() (*Magiccorner, error) {
	mc := Magiccorner{}
	mc.inventory = mtgban.InventoryRecord{}
	mc.buylist = mtgban.BuylistRecord{}
	mc.client = NewMCClient()
	mc.MaxConcurrency = defaultConcurrency
	return &mc, nil
}

type resultChan struct {
	cardID   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (mc *Magiccorner) printf(format string, a ...any) {
	if mc.LogCallback != nil {
		mc.LogCallback("[MC] "+format, a...)
	}
}

// blindAttraction reports whether the printings an aliasing error names are
// all attractions. Their printings are told apart by which of their lights
// are lit and by nothing else, and this store publishes a print-run index
// instead - a number that names no light, and that no other seller repeats
// - so a listing of one names them all equally. There is nothing to choose
// between, and nothing worth reporting.
func blindAttraction(probes []string) bool {
	for _, probe := range probes {
		co, err := mtgmatcher.GetUUID(probe)
		if err != nil || magic.AttractionLights(&co.Card) == "" {
			return false
		}
	}
	return len(probes) > 0
}

func (mc *Magiccorner) processEntry(ctx context.Context, channel chan<- resultChan, edition MCEdition) error {
	cards, err := mc.client.GetInventoryForEdition(ctx, edition)
	if err != nil {
		return err
	}

	printed := false

	// Keep track of the processed ids, and don't add duplicates
	duplicate := map[int]bool{}

	for _, card := range cards {
		if !printed && mc.VerboseLog {
			mc.printf("Processing id %d - %s (%s, code: %s)", edition.ID, edition.Name, card.Extra, card.Code)
			printed = true
		}

		for i, v := range card.Variants {
			// Skip duplicate cards
			if duplicate[v.ID] {
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

			cardID, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// The basic lands need custom handling for each edition if they
				// aren't found with other methods, ignore errors until they are
				// added to the variants table.
				if mtgmatcher.IsBasicLand(card.Name) {
					continue
				}
				var alias *mtgmatcher.AliasingError
				aliased := errors.As(err, &alias)
				if aliased && blindAttraction(alias.Probe()) {
					continue
				}

				mc.printf("%v", err)
				mc.printf("%q", theCard)
				mc.printf("%q", card)

				if aliased {
					for _, probe := range alias.Probe() {
						card, _ := mtgmatcher.GetUUID(probe)
						mc.printf("- %s", card)
					}
				}
				continue
			}

			channel <- resultChan{
				cardID: cardID,
				invEntry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      v.Price * mc.exchangeRate,
					Quantity:   v.Quantity,
					URL:        "https://www.magiccorner.it" + card.URL,
					OriginalID: fmt.Sprint(card.ID),
					InstanceID: fmt.Sprint(v.ID),
				},
			}

			duplicate[v.ID] = true
		}
	}

	return nil
}

// Scrape returns an array of Entry, containing pricing and card information
func (mc *Magiccorner) scrape(ctx context.Context) error {
	editionList, err := mc.client.GetEditionList(ctx, true)
	if err != nil {
		return err
	}

	mtgban.WorkerPool(ctx, mc.MaxConcurrency, editionList,
		func(ctx context.Context, edition MCEdition, results chan<- resultChan) error {
			return mc.processEntry(ctx, results, edition)
		},
		func(result resultChan) {
			err := mc.inventory.AddRelaxed(result.cardID, result.invEntry)
			if err != nil {
				mc.printf("%s", err.Error())
			}
		},
		mc.printf,
	)

	mc.inventoryDate = time.Now()

	return nil
}

// SetConfig applies options after the scraper was built. See
// mtgban.ScraperConfig.
func (mc *Magiccorner) SetConfig(opt mtgban.ScraperOptions) {
	mc.DisableRetail = opt.DisableRetail
	mc.DisableBuylist = opt.DisableBuylist
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mc *Magiccorner) Load(ctx context.Context) error {
	var errs []error

	// Both sides price in euro, so the rate has to be in place before either
	// runs: fetched from the retail path alone it stayed zero whenever retail
	// was disabled, and every buy price came out at zero with it
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mc.exchangeRate = rate

	if !mc.DisableRetail {
		err := mc.scrape(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inventory load failed: %w", err))
		}
	}

	if !mc.DisableBuylist {
		err := mc.scrapeBL(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("buylist load failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mc *Magiccorner) Inventory() mtgban.InventoryRecord {
	return mc.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (mc *Magiccorner) Buylist() mtgban.BuylistRecord {
	return mc.buylist
}

func (mc *Magiccorner) parseBL(ctx context.Context, channel chan<- resultChan, edition MCExpansion) error {
	i := 1
	totals := 0
	for {
		mc.printf("Querying %s page %d", edition.Name, i)
		result, err := mc.client.GetBuylistForEdition(ctx, edition.ID, i)
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
			qty := min(product.Quantity, 4)

			if price == 0 {
				continue
			}

			theCard, err := preprocessBL(cardName, edition, product.ID, product.SerialNumber)
			if err != nil {
				continue
			}

			cardID, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				var alias *mtgmatcher.AliasingError
				aliased := errors.As(err, &alias)
				if aliased && blindAttraction(alias.Probe()) {
					continue
				}

				mc.printf("%v", err)
				mc.printf("%q", theCard)

				if aliased {
					for _, probe := range alias.Probe() {
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

				var quantity int
				if grade == "NM" {
					quantity = qty
				}

				channel <- resultChan{
					cardID: cardID,
					buyEntry: &mtgban.BuylistEntry{
						Quantity:   quantity,
						Conditions: grade,
						BuyPrice:   price * mc.exchangeRate * factor,
						URL:        link,
						OriginalID: product.ID,
					},
				}
			}
		}

		i++
		totals += len(result.Products)
		if totals >= result.Total {
			break
		}

		// The exit above is a fixpoint: a successful page carrying no products
		// leaves totals where it was while Total stays positive, so the same
		// request repeats forever
		if len(result.Products) == 0 {
			return fmt.Errorf("%s: page %d was empty after %d of %d products", edition.Name, i, totals, result.Total)
		}
	}

	return nil
}

func (mc *Magiccorner) scrapeBL(ctx context.Context) error {
	editions, err := mc.client.GetBuylistEditions(ctx)
	if err != nil {
		return err
	}
	mc.printf("Found %d editions", len(editions))

	mtgban.WorkerPool(ctx, mc.MaxConcurrency, editions,
		func(ctx context.Context, edition MCExpansion, results chan<- resultChan) error {
			return mc.parseBL(ctx, results, edition)
		},
		func(record resultChan) {
			err := mc.buylist.AddRelaxed(record.cardID, record.buyEntry)
			if err != nil {
				mc.printf("%s", err.Error())
			}
		},
		mc.printf,
	)

	mc.buylistDate = time.Now()

	return nil
}

// Info describes this scraper. See mtgban.Scraper.
func (mc *Magiccorner) Info() (info mtgban.ScraperInfo) {
	info.Name = "Magic Corner"
	info.Shorthand = "MC"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mc.inventoryDate
	info.BuylistTimestamp = &mc.buylistDate
	return
}
