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
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]tcgplayer.Group

	category            int
	categoryName        string
	categoryDisplayName string
	game                string

	productTypes []string

	client *tcgplayer.Client
}

func (tcg *TCGGameIndex) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Equal(tcg.productTypes, tcgplayer.SinglesProductTypes(tcg.category)) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

// NewScraperGameIndex returns an index scraper for one game, authenticated
// with a partner API key pair.
func NewScraperGameIndex(game, publicID, privateID string) (*TCGGameIndex, error) {
	category, found := SupportedGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", game)
	}

	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := TCGGameIndex{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency

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

		cardName := productMap[result.ProductID].Name
		number := RawProductNumber(&product)
		theCard := &mtgmatcher.InputCard{
			// See TCGGame.processPage: the product id and the finish beside
			// it identify the sku, the text fields are the fallback.
			ID:        fmt.Sprint(result.ProductID),
			Name:      cardName,
			Edition:   tcg.editions[product.GroupID].Name,
			Variation: strings.TrimSpace(number + " " + result.SubTypeName),
			Finish:    result.SubTypeName,
			Foil:      result.SubTypeName != "Normal",
		}
		cardID, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			// Name the card, not just the price row: a product id alone
			// says nothing about which product failed to match.
			tcg.printf("%v for %q (product %d)", err, theCard, result.ProductID)
			tcg.printf("%+v", result)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				tcg.printf("%d %s got ids: %s", product.ProductID, cardName, probes)
				for _, probe := range probes {
					co, _ := mtgmatcher.GetUUID(probe)
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
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.Affiliate, "", "", isDirect)

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

	mtgban.WorkerPool(ctx, tcg.MaxConcurrency, pageNums,
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
	return
}
