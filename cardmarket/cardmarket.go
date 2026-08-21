// Package cardmarket scrapes Cardmarket, both the price-guide index and
// sealed product, across every game they carry.
package cardmarket

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type responseChan struct {
	ogID   int
	cardID string
	entry  mtgban.InventoryEntry
}

// CardMarketIndex prices singles from Cardmarket's price guide, the low and
// trend numbers rather than any one seller's listing.
type CardMarketIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int
	exchangeRate   float64

	// Optional field to select a single edition to go through
	TargetEdition string

	// TCGBridge maps a Cardmarket product id to the TCGplayer id of the
	// same single, for the keyless catalogs (yugioh, flesh and blood)
	// whose products carry no collector number and no version index, so
	// same-name products are told apart only by an exact id. bantool
	// builds it from cardtrader's blueprints, the one source linking the
	// two marketplaces; the scraper itself stays vendor-pure and receives
	// it as plain data.
	TCGBridge map[int]int

	inventory mtgban.InventoryRecord

	// priceGuide holds one game's published prices, indexed by the product
	// id they belong to: a run asks for one product's prices tens of
	// thousands of times, once per product in the catalog.
	priceGuide map[int]PriceGuide

	client *MKMClient
	gameID int
}

var availableIndexNames = []string{
	"MKM Low", "MKM Trend",
}

var name2shorthand = map[string]string{
	"MKM Low":   "MKMLow",
	"MKM Trend": "MKMTrend",
}

func (mkm *CardMarketIndex) printf(format string, a ...any) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper for one game, authenticated with an
// app token and secret.
func NewScraperIndex(gameID int, appToken, appSecret string) (*CardMarketIndex, error) {
	mkm := CardMarketIndex{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	mkm.gameID = gameID
	return &mkm, nil
}

func (mkm *CardMarketIndex) processEdition(ctx context.Context, channel chan<- responseChan, idExpansion int) error {
	products, err := mkm.client.MKMProductsInExpansion(ctx, idExpansion)
	if err != nil {
		return err
	}

	for _, product := range products {
		err := mkm.processProduct(channel, &product)
		if err != nil {
			mkm.printf("product id %d returned %s", product.IdProduct, err)
		}
	}
	return nil
}

func (mkm *CardMarketIndex) processProduct(channel chan<- responseChan, product *MKMProduct) error {
	var cardID string
	var cardIDFoil string
	var err error

	switch mkm.gameID {
	case GameIdMagic:
		// An exact mcmId match ties the product to its printings more
		// reliably than name/number matching, which cannot tell apart
		// products sharing a collector number (e.g. RVR 312 vs 312z,
		// both "312" upstream); preprocess only when no id is known.
		cardID, cardIDFoil = Fallback(product)
		if cardID != "" {
			break
		}

		theCard, err := Preprocess(product.Name, product.Number, product.ExpansionName)
		if err != nil {
			_, ok := err.(*PreprocessError)
			if ok {
				return err
			}
			return nil
		}

		cardID, err = mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil {
			if mtgmatcher.IsToken(theCard.Name) ||
				theCard.Edition == "Pro Tour Collector Set" ||
				strings.HasPrefix(theCard.Edition, "World Championship Decks") {
				return nil
			}

			mkm.printf("%v", err)
			mkm.printf("%q", theCard)
			mkm.printf("%v | %v | %v ", product.Name, product.ExpansionName, product.Number)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("- %s", card)
				}
			}
			return err
		}

		cardIDFoil, _ = mtgmatcher.MatchId(cardID, true)
	case GameIdLorcana, GameIdRiftbound, GameIdOnePiece:
		fields := strings.SplitN(product.Name, " (V.", 2)
		cardName := fields[0]
		number := product.Number
		// The V-index cardmarket synthesizes for same-number siblings is
		// how One Piece tells a base art from its variants (V.1 the base,
		// the rest its alternates), and says nothing for the finish-driven
		// games; hand it to the matcher's own rules either way. The foil
		// probes are inert for One Piece too - both flags resolve to the
		// same printing.
		if len(fields) > 1 {
			number = strings.TrimSpace(number + " V." + strings.TrimSuffix(fields[1], ")"))
		}

		cardID, err = mtgmatcher.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: false})
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return nil
		} else if err != nil && !errors.Is(err, mtgmatcher.ErrCardWrongVariant) {
			mkm.printf("%v", err)
			mkm.printf("%+v", product)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				mkm.printf("%s got ids: %s", cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
					mkm.printf("%s: %s", probe, co)
				}
			}
			return err
		}
		// A wrong-variant miss above may just mean the card has no nonfoil
		// printing (Match validates the finish); adopt the foil id then.
		var errFoil error
		cardIDFoil, errFoil = mtgmatcher.Match(&mtgmatcher.InputCard{Name: cardName, Edition: product.ExpansionName, Variation: number, Foil: true})
		if cardID == "" {
			cardID = cardIDFoil
		}

		if cardID == "" {
			// Neither finish matched, so the miss was genuine; the foil
			// probe's error may carry the more informative verdict
			if errFoil != nil {
				err = errFoil
			}
			mkm.printf("%v", err)
			mkm.printf("%+v", product)
			return err
		}
	case GameIdYugioh, GameIdFleshAndBlood:
		// These catalogs carry no collector number and no version index,
		// and same-name products abound, so a product resolves through the
		// TCGplayer id the cardtrader bridge knows it by or not at all -
		// name matching has nothing to distinguish on.
		tcgID, found := mkm.TCGBridge[product.IdProduct]
		if !found {
			return nil
		}
		cardID, err = mtgmatcher.MatchId(fmt.Sprint(tcgID), false)
		if err != nil {
			return nil
		}
		cardIDFoil = cardID
		if mkm.gameID == GameIdFleshAndBlood {
			// Cardmarket sells each Flesh and Blood treatment as its own
			// product and each print run as its own expansion, so a
			// product is one printing and says which: the run in the
			// expansion name, the treatment in a parenthetical after the
			// card's. The flag names neither, and answered every one of
			// them with the unlimited plain printing.
			if finish := fabFinish(product.ExpansionName, product.Name); finish != "" {
				if id, ferr := mtgmatcher.MatchIdFinish(fmt.Sprint(tcgID), finish); ferr == nil {
					cardID = id
				}
			}
			cardIDFoil = cardID
		}
		if mkm.gameID == GameIdYugioh {
			// Yu-Gi-Oh's second column is the first edition's, which is a
			// print run rather than a foil, so the flag cannot name it -
			// both flags answer with the unlimited printing and the column
			// was dropped for having nowhere to attach. Naming the run
			// reaches it, and errors into an empty id for the products
			// sold in no first edition, which the guard below drops.
			cardIDFoil, _ = mtgmatcher.MatchIdFinish(fmt.Sprint(tcgID), "1st Edition")
		}
	default:
		return errors.New("unsupported game")
	}

	// Look for the price presence
	guide, found := mkm.priceGuide[product.IdProduct]
	if !found {
		return fmt.Errorf("IdProduct %d not found in PriceGuide", product.IdProduct)
	}

	// Sorted as availableIndexNames
	prices := []float64{guide.LowPrice, guide.TrendPrice}
	foilprices := []float64{guide.FoilLowPrice, guide.FoilTrendPrice}

	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return err
	}

	// A catalog that gives each treatment its own product prices one
	// printing per product, and the product's own columns are that
	// printing's whatever its foilness - there is no second column for
	// them to be in. Every other catalog keeps the foil beside the plain
	// card and splits the two across the columns.
	perTreatment := mkm.gameID == GameIdFleshAndBlood

	// If card is not foil, add prices from the prices array, then check
	// if there is a foil printing, and add prices from the foilprices array.
	// If a card is foil-only or is etched, then we just use foilprices data.
	if perTreatment || (!co.Foil && !co.Etched) {
		link := BuildURL(product.IdProduct, mkm.gameID, mkm.Affiliate, false)

		quantity := product.CountArticles - product.CountFoils
		if perTreatment {
			quantity = product.CountArticles
		}

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}

			out := responseChan{
				ogID:   product.IdProduct,
				cardID: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i] * mkm.exchangeRate,
					Quantity:   quantity,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}

		if !perTreatment && (foilprices[0] != 0 || foilprices[1] != 0) {
			link := BuildURL(product.IdProduct, mkm.gameID, mkm.Affiliate, true)

			// An empty foil id means the card has no foil printing (Match
			// errored on the foil probe), so residual foil prices in the
			// guide have nothing to attach to
			if cardIDFoil != "" && cardID != cardIDFoil {
				for i := range availableIndexNames {
					if foilprices[i] == 0 {
						continue
					}
					out := responseChan{
						ogID:   product.IdProduct,
						cardID: cardIDFoil,
						entry: mtgban.InventoryEntry{
							Conditions: "NM",
							Price:      foilprices[i] * mkm.exchangeRate,
							Quantity:   product.CountFoils,
							URL:        link,
							SellerName: availableIndexNames[i],
							OriginalId: fmt.Sprint(product.IdProduct),
						},
					}

					channel <- out
				}
			}
		}
	} else {
		link := BuildURL(product.IdProduct, mkm.gameID, mkm.Affiliate, true)

		for i := range availableIndexNames {
			if foilprices[i] == 0 || product.CountFoils == 0 {
				continue
			}
			out := responseChan{
				ogID:   product.IdProduct,
				cardID: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      foilprices[i] * mkm.exchangeRate,
					Quantity:   product.CountFoils,
					URL:        link,
					SellerName: availableIndexNames[i],
					OriginalId: fmt.Sprint(product.IdProduct),
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mkm *CardMarketIndex) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}
	mkm.exchangeRate = rate

	priceGuide, err := GetPriceGuide(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	mkm.priceGuide = make(map[int]PriceGuide, len(priceGuide))
	for _, entry := range priceGuide {
		mkm.priceGuide[entry.IdProduct] = entry
	}

	mkm.printf("Obtained today's price guide with %d prices", len(priceGuide))

	list, err := mkm.client.Expansions(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	list = FilterAndSortExpansions(list)

	// The Japanese-program expansions are whole separate catalogs (OP01-JP
	// beside OP01) whose prices must not land on the English printings the
	// datastore carries.
	if mkm.gameID == GameIdOnePiece {
		kept := list[:0]
		for _, exp := range list {
			if strings.HasSuffix(exp.SetCode, "-JP") || strings.Contains(exp.Name, "(Japanese)") {
				continue
			}
			kept = append(kept, exp)
		}
		list = kept
	}

	mkm.printf("Parsing %d expansion ids", len(list))

	// Pre-filter items if a target edition is set
	items := list
	if mkm.TargetEdition != "" {
		items = nil
		for _, exp := range list {
			if exp.Name == mkm.TargetEdition {
				items = append(items, exp)
			}
		}
	}

	// The bridge is keyed by the Cardmarket id and valued by the TCGplayer
	// one, and a cardtrader blueprint names every Cardmarket product it
	// sells as, so nothing stops two products from resolving to one
	// printing. An index wants a single price per name per uuid, and a
	// second one is worth the log line the callback already prints rather
	// than a second row no consumer can choose between.
	add := mkm.inventory.AddStrict
	switch mkm.gameID {
	case GameIdYugioh, GameIdFleshAndBlood:
		add = mkm.inventory.AddUnique
	}

	mtgban.WorkerPool(ctx, mkm.MaxConcurrency, items,
		func(ctx context.Context, exp MKMExpansion, channel chan<- responseChan) error {
			mkm.printf("Processing %s (%d)", exp.Name, exp.IdExpansion)
			err := mkm.processEdition(ctx, channel, exp.IdExpansion)
			if err != nil {
				return fmt.Errorf("expansion %s (id %d) returned %s", exp.Name, exp.IdExpansion, err.Error())
			}
			return nil
		},
		func(result responseChan) {
			err := add(result.cardID, &result.entry)
			if err != nil {
				card, cerr := mtgmatcher.GetUUID(result.cardID)
				if cerr != nil {
					mkm.printf("%d - %s: %s", result.ogID, cerr.Error(), result.cardID)
					return
				}
				// Skip too many errors
				if mtgmatcher.IsToken(card.Name) ||
					card.Edition == "Pro Tour Collector Set" ||
					strings.HasPrefix(card.Edition, "World Championship Decks") {
					return
				}
				mkm.printf("%d - %s", result.ogID, err.Error())
			}
		},
		mkm.printf,
	)

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mkm *CardMarketIndex) Inventory() mtgban.InventoryRecord {
	return mkm.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (mkm *CardMarketIndex) MarketNames() []string {
	return availableIndexNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (mkm *CardMarketIndex) InfoForScraper(name string) mtgban.ScraperInfo {
	info := mkm.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (mkm *CardMarketIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Market Index"
	info.Shorthand = "MKMIndex"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.MetadataOnly = true
	info.Family = "MKM"
	switch mkm.gameID {
	case GameIdMagic:
		info.Game = mtgban.GameMagic
	case GameIdLorcana:
		info.Game = mtgban.GameLorcana
	case GameIdRiftbound:
		info.Game = mtgban.GameRiftbound
	case GameIdOnePiece:
		info.Game = mtgban.GameOnePiece
	case GameIdYugioh:
		info.Game = mtgban.GameYuGiOh
	case GameIdFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	}
	return
}
