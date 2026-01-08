package vegassingles

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

var conditionMap = map[string]string{
	"Near Mint":         "NM",
	"Lightly Played":    "SP",
	"Moderately Played": "MP",
	"Heavily Played":    "HP",
	"Damaged":           "PO",
}

func buildProductSlug(displayName string) string {
	slug := strings.ToLower(displayName)
	slug = strings.ReplaceAll(slug, "(", "")
	slug = strings.ReplaceAll(slug, ")", "")
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, " - ", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

type Vegassingles struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	client *VSClient

	inventoryDate time.Time
	buylistDate   time.Time
	inventory     mtgban.InventoryRecord
	buylist       mtgban.BuylistRecord
}

func NewScraper() *Vegassingles {
	vs := Vegassingles{}
	vs.inventory = mtgban.InventoryRecord{}
	vs.buylist = mtgban.BuylistRecord{}
	vs.client = NewVSClient()
	vs.MaxConcurrency = defaultConcurrency
	return &vs
}

func (vs *Vegassingles) printf(format string, a ...interface{}) {
	if vs.LogCallback != nil {
		vs.LogCallback("[VS] "+format, a...)
	}
}

func (vs *Vegassingles) processProduct(product VSProduct) error {
	theCard, err := preprocess(product)
	if err != nil {
		return err
	}

	cardId, err := mtgmatcher.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		vs.printf("%v", err)
		vs.printf("%s: %q", product.ID, product.DisplayName)
		return nil
	}

	// Build buylist URL
	u, _ := url.Parse("https://buylist.vegas.singles/retailer/buylist")
	q := u.Query()
	q.Set("product_line", "Magic: the Gathering")
	q.Set("q", product.DisplayName)
	q.Set("sort", "Relevance")
	u.RawQuery = q.Encode()
	buylistLink := u.String()

	// Build retail product URL
	retailLink := "https://vegas.singles/products/" + buildProductSlug(product.DisplayName)

	// Process buylist variants (from store_pass_variant_info)
	for _, variant := range product.VariantInfo {
		if variant.OfferPrice == 0 {
			continue
		}

		cond, found := conditionMap[variant.Title]
		if !found {
			vs.printf("unknown condition: %s", variant.Title)
			continue
		}

		var priceRatio float64
		if product.Price > 0 {
			priceRatio = variant.OfferPrice / product.Price * 100
		}

		err = vs.buylist.Add(cardId, &mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   variant.OfferPrice,
			PriceRatio: priceRatio,
			URL:        buylistLink,
			OriginalId: strconv.FormatInt(product.ProductID, 10),
			InstanceId: strconv.FormatInt(variant.ID, 10),
		})
		if err != nil {
			vs.printf("%d: %s", product.ProductID, err.Error())
		}
	}

	// Process retail variants (from variant_info)
	for _, variant := range product.RetailVariantInfo {
		if variant.Price == 0 {
			continue
		}

		cond, found := conditionMap[variant.Title]
		if !found {
			continue
		}

		err = vs.inventory.Add(cardId, &mtgban.InventoryEntry{
			Conditions: cond,
			Price:      variant.Price,
			Quantity:   variant.InventoryQuantity,
			URL:        retailLink,
			OriginalId: strconv.FormatInt(product.ProductID, 10),
			InstanceId: variant.SKU,
		})
		if err != nil {
			vs.printf("%d: %s", product.ProductID, err.Error())
		}
	}

	return nil
}

func (vs *Vegassingles) scrape(ctx context.Context) error {
	totalPages, err := vs.client.getCount(ctx)
	if err != nil {
		return err
	}
	vs.printf("Total pages: %d", totalPages)

	pages := make(chan int)
	results := make(chan []VSProduct)
	var wg sync.WaitGroup

	for i := 0; i < vs.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range pages {
				products, err := vs.client.getPage(ctx, page)
				if err != nil {
					vs.printf("page %d: %s", page, err.Error())
					continue
				}
				results <- products
			}
		}()
	}

	go func() {
		for page := 1; page <= totalPages; page++ {
			pages <- page
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	var productCount int
	for products := range results {
		for _, product := range products {
			err := vs.processProduct(product)
			if err != nil {
				vs.printf("process error: %s", err.Error())
			}
			productCount++
		}
	}

	vs.printf("Processed %d products", productCount)
	vs.inventoryDate = time.Now()
	vs.buylistDate = time.Now()

	return nil
}

func (vs *Vegassingles) Load(ctx context.Context) error {
	return vs.scrape(ctx)
}

func (vs *Vegassingles) Inventory() mtgban.InventoryRecord {
	return vs.inventory
}

func (vs *Vegassingles) Buylist() mtgban.BuylistRecord {
	return vs.buylist
}

func (vs *Vegassingles) Info() (info mtgban.ScraperInfo) {
	info.Name = "Vegas Singles"
	info.Shorthand = "VS"
	info.InventoryTimestamp = &vs.inventoryDate
	info.BuylistTimestamp = &vs.buylistDate
	info.Game = mtgban.GameMagic
	return
}
