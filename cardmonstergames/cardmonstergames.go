// Package cardmonstergames scrapes Card Monster Games, a US store on
// Shopify whose singles carry TCGplayer sku ids.
package cardmonstergames

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/tcgplayer"

	api "github.com/mtgban/go-tcgplayer"
)

// storefrontPace is how often the storefront is asked for a page.
const storefrontPace = 1100 * time.Millisecond

// CardMonsterGames prices the Magic singles Card Monster Games sells online.
type CardMonsterGames struct {
	logCallback mtgban.LogCallbackFunc

	backend *matcher.Backend

	// catalog is TCGplayer's catalog dump, which names the printing and
	// grade of the sku each listing carries.
	catalog *api.CatalogDump

	client *cmgClient

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

// NewScraper returns a scraper matching against b.
func NewScraper(b *matcher.Backend) *CardMonsterGames {
	cmg := CardMonsterGames{backend: b}
	cmg.inventory = mtgban.InventoryRecord{}
	cmg.client = newClient()
	return &cmg
}

func (cmg *CardMonsterGames) printf(format string, a ...any) {
	if cmg.logCallback != nil {
		cmg.logCallback("[CMG] "+format, a...)
	}
}

// fetchProducts walks every product the storefront lists, a few thousand
// of them, sealed and other games beside the singles.
func (cmg *CardMonsterGames) fetchProducts(ctx context.Context) ([]shopifyProduct, error) {
	var out []shopifyProduct
	for page := 1; ; page++ {
		if page*productsPerPage > walkLimit {
			return nil, fmt.Errorf("storefront lists more than the %d products Shopify walks", walkLimit)
		}
		if page > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(storefrontPace):
			}
		}
		products, err := cmg.client.products(ctx, page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}
		out = append(out, products...)
		if len(products) < productsPerPage {
			return out, nil
		}
	}
}

// skuGrades reads the grade of each sku in wanted off the catalog.
func skuGrades(dump *api.CatalogDump, wanted map[int]bool) map[int]mtgban.Condition {
	out := map[int]mtgban.Condition{}
	for _, product := range dump.Products {
		for _, sku := range product.Skus {
			if !wanted[sku.SKUID] {
				continue
			}
			cond, found := tcgplayer.SKUConditionMap[sku.ConditionID]
			if found {
				out[sku.SKUID] = cond
			}
		}
	}
	return out
}

// processProducts prices every available variant whose SKU the catalog
// resolves to a Magic single. Sealed product, other games' singles and the
// few listings carrying no sku resolve to nothing and are left out.
func (cmg *CardMonsterGames) processProducts(products []shopifyProduct) error {
	sku2uuid, err := tcgplayer.SKUCards(cmg.backend, cmg.catalog)
	if err != nil {
		return err
	}
	wanted := map[int]bool{}
	for _, product := range products {
		for _, variant := range product.Variants {
			id, err := strconv.Atoi(variant.SKU)
			if err == nil && sku2uuid[id] != "" {
				wanted[id] = true
			}
		}
	}
	grades := skuGrades(cmg.catalog, wanted)

	for _, product := range products {
		link := baseURL + "/products/" + product.Handle
		for _, variant := range product.Variants {
			if !variant.Available {
				continue
			}
			id, err := strconv.Atoi(variant.SKU)
			if err != nil {
				continue
			}
			cardID, found := sku2uuid[id]
			if !found {
				continue
			}
			price, err := strconv.ParseFloat(variant.Price, 64)
			if err != nil || price <= 0 {
				continue
			}
			err = cmg.inventory.Add(cardID, &mtgban.InventoryEntry{
				Conditions: grades[id],
				Price:      price,
				Quantity:   1,
				URL:        link,
				OriginalID: strconv.FormatInt(product.ID, 10),
				InstanceID: strconv.FormatInt(variant.ID, 10),
			})
			if err != nil {
				cmg.printf("%s: %v", product.Title, err)
			}
		}
	}
	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (cmg *CardMonsterGames) Load(ctx context.Context) error {
	products, err := cmg.fetchProducts(ctx)
	if err == nil {
		cmg.printf("Found %d products", len(products))
		err = cmg.processProducts(products)
	}
	if err == nil && len(cmg.inventory) == 0 {
		err = errors.New("no product carries a Magic single's sku")
	}
	if err != nil {
		return fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, err)
	}
	cmg.inventoryDate = time.Now()
	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (cmg *CardMonsterGames) Inventory() mtgban.InventoryRecord {
	return cmg.inventory
}

// Info describes this scraper. See mtgban.Scraper.
func (cmg *CardMonsterGames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Monster Games"
	info.Shorthand = "CMG"
	info.Game = cmg.backend.Game
	info.NoQuantityInventory = true
	info.InventoryTimestamp = &cmg.inventoryDate
	return
}
