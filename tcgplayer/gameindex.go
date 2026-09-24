package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-tcgplayer"
)

// TCGGameIndex is the market-price index counterpart of TCGGame, serving the
// same single-game categories through the matcher's name + collector number +
// finish identification.
type TCGGameIndex struct {
	logCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	affiliate      string
	maxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]tcgplayer.Group

	category            int
	categoryName        string
	categoryDisplayName string
	game                mtgban.Game

	productTypes []string

	backend *mtgmatcher.Backend
	client  *tcgplayer.Client
}

func (tcg *TCGGameIndex) printf(format string, a ...any) {
	if tcg.logCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Equal(tcg.productTypes, tcgplayer.SinglesProductTypes(tcg.category)) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.logCallback(tag+format, a...)
	}
}

// NewScraperGameIndex returns an index scraper for one game, authenticated
// with a partner API key pair.
func NewScraperGameIndex(b *mtgmatcher.Backend, publicID, privateID string) (*TCGGameIndex, error) {
	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}
	category, found := tcgGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", game)
	}

	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := TCGGameIndex{}
	tcg.backend = b
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.maxConcurrency = defaultConcurrency

	tcg.category = category
	tcg.game = game
	tcg.productTypes = tcgplayer.SinglesProductTypes(category)

	return &tcg, nil
}

func (tcg *TCGGameIndex) processPage(ctx context.Context, channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(ctx, tcg.category, tcg.productTypes, false, page)
	if err != nil {
		return err
	}

	productMap := map[int]tcgplayer.Product{}
	ids := make([]int, len(products))
	for i, product := range products {
		ids[i] = product.ProductID
		productMap[product.ProductID] = product
	}

	results, err := tcg.client.GetMarketPricesByProducts(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		product, found := productMap[result.ProductID]
		if !found {
			continue
		}
		if isUnsupportedProduct(&product) {
			continue
		}

		cardName := productMap[result.ProductID].Name
		// See TCGGame.processPage: the product id and the finish identify
		// the price row.
		theCard := &mtgmatcher.InputCard{
			ID:     fmt.Sprint(result.ProductID),
			Finish: result.SubTypeName,
			Foil:   result.SubTypeName != "Normal",
		}
		cardID, err := tcg.backend.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			// A row quoting a market price and nothing else is a pricing-side
			// relic: the catalog has split or retired the printing its subtype
			// names, and the surviving printing is priced by its own row
			// alongside this one. Refusing it is right - the stale number can
			// be many times the real price - but complaining every run is
			// noise, so only the finish is let through quietly. Any other
			// failure on such a row still speaks up.
			marketOnly := result.LowPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0
			if marketOnly && errors.Is(err, mtgmatcher.ErrCardWrongFinish) {
				continue
			}

			// Name the card, not just the price row: a product id alone
			// says nothing about which product failed to match.
			tcg.printf("%v for %q %s (product %d)", err, cardName, result.SubTypeName, result.ProductID)
			tcg.printf("%+v", result)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				tcg.printf("%d %s got ids: %s", product.ProductID, cardName, probes)
				for _, probe := range probes {
					co, _ := tcg.backend.GetUUID(probe)
					tcg.printf("%s: %s", probe, co)
				}
			}
			continue
		}

		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, result.DirectLowPrice,
		}

		for i := range prices {
			if prices[i] == 0 {
				continue
			}

			isDirect := availableIndexNames[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.affiliate, "", "", isDirect)

			out := genericChan{
				key: cardID,
				entry: mtgban.InventoryEntry{
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableIndexNames[i],
					Bundle:     isDirect,
					OriginalID: fmt.Sprint(result.ProductID),
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *TCGGameIndex) Load(ctx context.Context) error {
	// Initialize data for debug logs
	var err error
	tcg.categoryName, tcg.categoryDisplayName, err = GetCategoryNames(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}

	editions, err := EditionMap(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	totals, err := tcg.client.TotalProducts(ctx, tcg.category, []string{"Cards"})
	if err != nil {
		return err
	}
	tcg.printf("Found %d products", totals)

	pageNums := make([]int, 0, totals/tcgplayer.MaxItemsInResponse+1)
	for i := 0; i < totals; i += tcgplayer.MaxItemsInResponse {
		pageNums = append(pageNums, i)
	}

	mtgban.WorkerPool(ctx, tcg.maxConcurrency, pageNums,
		func(ctx context.Context, page int, channel chan<- genericChan) error {
			return tcg.processPage(ctx, channel, page)
		},
		func(result genericChan) {
			err := tcg.inventory.Add(result.key, &result.entry)
			if err != nil {
				tcg.printf("%s", err.Error())
			}
		},
		tcg.printf,
	)

	tcg.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *TCGGameIndex) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (tcg *TCGGameIndex) MarketNames() []string {
	return availableIndexNames[:len(availableIndexNames)-1]
}

// InfoForScraper describes one of the sub-scrapers named above.
func (tcg *TCGGameIndex) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *TCGGameIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.Game = tcg.game
	info.Family = "TCG"
	return
}
