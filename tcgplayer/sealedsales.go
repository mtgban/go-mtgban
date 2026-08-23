package tcgplayer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// SealedSales prices the sealed products the listing walk leaves behind.
// TCGplayer stops publishing a listing price the moment the last listing
// goes away, which for old sealed product is most of the catalog - but the
// storefront keeps a record of what the product actually sold for. This
// scraper asks for those sales, for exactly the products that have no
// listing today, and publishes the recent median as a statistic: the
// purchase price alone, with the shipping the buyer paid left out.
type SealedSales struct {
	LogCallback    mtgban.LogCallbackFunc
	Affiliate      string
	MaxConcurrency int

	// How long to pause between two sales lookups. The sales endpoint is
	// the storefront's own, outside the partner API and its keys, so the
	// pace stays polite rather than parallel.
	RequestPace time.Duration

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	game          string

	client *tcgplayer.Client
}

// NewScraperSealedSales returns a sealed sales scraper for one game,
// authenticated with a partner API key pair. The keys serve the listing
// check that selects the products; the sales themselves need none.
func NewScraperSealedSales(game, publicID, privateID string) (*SealedSales, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}
	tcg := SealedSales{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.game = game
	tcg.MaxConcurrency = defaultConcurrency
	tcg.RequestPace = time.Second
	return &tcg, nil
}

func (tcg *SealedSales) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSealedSales] "+format, a...)
	}
}

// salesWindow is how far back a sale may lie and still speak for the
// product. Sealed prices drift; a transaction older than this says more
// about the market then than the product now.
const salesWindow = 365 * 24 * time.Hour

// salesPrice condenses a product's recent sales into one price: the median
// purchase price of the single-unit sales inside the window, shipping
// excluded. Lot sales are dropped because the recorded price cannot say
// what one unit went for. Zero means the sales say nothing.
func salesPrice(sales []LatestSalesData, now time.Time) float64 {
	var prices []float64
	for _, sale := range sales {
		if sale.Quantity != 1 || sale.PurchasePrice == 0 {
			continue
		}
		if now.Sub(sale.OrderDate) > salesWindow {
			continue
		}
		prices = append(prices, sale.PurchasePrice)
	}
	if len(prices) == 0 {
		return 0
	}
	sort.Float64s(prices)
	mid := len(prices) / 2
	if len(prices)%2 == 1 {
		return prices[mid]
	}
	return (prices[mid-1] + prices[mid]) / 2
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *SealedSales) Load(ctx context.Context) error {
	sealedMap := mtgmatcher.BuildSealedProductMap("tcgplayerProductId")
	tcg.printf("Loaded %d sealed products", len(sealedMap))

	productIDs := make([]int, 0, len(sealedMap))
	for id := range sealedMap {
		productIDs = append(productIDs, id)
	}
	sort.Ints(productIDs)

	// The listing check: a product any seller has listed is already priced
	// by the sealed scraper, and a listing price beats a sales memory.
	var unlisted []int
	for i := 0; i < len(productIDs); i += tcgplayer.MaxIDsInRequest {
		end := min(i+tcgplayer.MaxIDsInRequest, len(productIDs))
		results, err := tcg.client.GetMarketPricesByProducts(ctx, productIDs[i:end])
		if err != nil {
			return err
		}
		listed := map[int]bool{}
		for _, result := range results {
			if result.LowPrice != 0 {
				listed[result.ProductID] = true
			}
		}
		for _, id := range productIDs[i:end] {
			if !listed[id] {
				unlisted = append(unlisted, id)
			}
		}
	}
	tcg.printf("%d of %d products have no listing, asking their sales", len(unlisted), len(productIDs))

	var priced int
	for _, id := range unlisted {
		if err := ctx.Err(); err != nil {
			return err
		}
		sales, err := LatestSales(ctx, fmt.Sprint(id))
		if err != nil {
			tcg.printf("sales for %d: %v", id, err)
			time.Sleep(tcg.RequestPace)
			continue
		}
		price := salesPrice(sales, time.Now())
		if price == 0 {
			time.Sleep(tcg.RequestPace)
			continue
		}

		uuids := sealedMap[id]
		if len(uuids) != 1 {
			time.Sleep(tcg.RequestPace)
			continue
		}
		err = tcg.inventory.Add(uuids[0], &mtgban.InventoryEntry{
			Conditions: "NM",
			Price:      price,
			Quantity:   1,
			URL:        GenerateProductURL(id, "", tcg.Affiliate, "", "", false),
			OriginalID: fmt.Sprint(id),
		})
		if err != nil {
			tcg.printf("%s", err.Error())
		} else {
			priced++
		}
		time.Sleep(tcg.RequestPace)
	}
	tcg.printf("Priced %d products off their sales", priced)

	tcg.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *SealedSales) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *SealedSales) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Last Sold"
	info.Shorthand = "TCGSealedSales"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.SealedMode = true
	info.Game = tcg.game
	return
}
