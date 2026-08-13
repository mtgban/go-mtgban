package starcitygames

import (
	"context"
	"fmt"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type StarcitygamesSealed struct {
	LogCallback   mtgban.LogCallbackFunc
	inventoryDate time.Time
	buylistDate   time.Time

	Affiliate string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	productMap map[string]string
	setIDs     map[string]int
	client     *SCGClient
	game       int
}

func NewScraperSealed(game int, apiKey string) *StarcitygamesSealed {
	scg := StarcitygamesSealed{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(apiKey)
	scg.game = game
	return &scg
}

func (scg *StarcitygamesSealed) printf(format string, a ...interface{}) {
	if scg.LogCallback != nil {
		scg.LogCallback("[SCGSealed] "+format, a...)
	}
}

// buildProductMap indexes the sealed products by their SCG id (the catalog SKU)
// so a catalog product can be resolved to its mtgban uuid directly.
func buildProductMap() map[string]string {
	out := map[string]string{}
	for _, uuid := range mtgmatcher.GetSealedUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		scgID, found := co.Identifiers["scgId"]
		if !found {
			continue
		}
		out[scgID] = uuid
	}
	return out
}

func (scg *StarcitygamesSealed) processProduct(p CatalogProduct) {
	// A single malformed product must never abort the whole catalog stream;
	// recover, log, and skip it.
	defer func() {
		if r := recover(); r != nil {
			scg.printf("recovered from panic on %q (sku=%s): %v", p.Name, p.SKU, r)
		}
	}()

	// This scraper handles sealed only; singles have their own scraper,
	// and the catalog also carries supplies and bulk lots. product_type
	// states it outright, where the sku prefix only implied it.
	if p.ProductType != ProductTypeSealed {
		return
	}
	if gameFromCatalog(p.Game) != scg.game {
		return
	}

	// Sealed products are keyed by their SKU, which mtgban stores as the
	// scgId; the games whose datastore does not catalog it (riftbound,
	// lorcana) resolve by name instead, English only, unique or nothing.
	uuid, found := scg.productMap[p.SKU]
	if !found {
		if scg.game == GameMagic {
			return
		}
		if p.Language != "" && p.Language != "English" {
			return
		}
		if mtgmatcher.SealedIsLanguageVariant(p.Name) {
			return
		}
		resolved, err := mtgmatcher.ResolveSealed(p.Name)
		if err != nil {
			return
		}
		uuid = resolved
	}

	link := SCGProductURL(p.URL, "", scg.Affiliate)

	// The buylist link points at the sell-your-cards page for this product.
	// Sealed products carry no catalog set, so match the set off the product
	// name; fall back to retail if nothing matches.
	buyURL := link
	ids := setIDsForProduct(scg.setIDs, p.Name, p.SKU)
	if len(ids) > 0 {
		buyURL = SCGBuylistURL(scg.game, p.Name, p.Language, ids)
	}

	for _, v := range p.Variants {
		retailPrice, _ := mtgmatcher.ParsePrice(v.Price)

		if retailPrice > 0 && v.Qty > 0 {
			entry := &mtgban.InventoryEntry{
				Price:      retailPrice,
				Quantity:   v.Qty,
				OriginalId: p.SKU,
				InstanceId: v.SKU,
				URL:        SCGProductURL(p.URL, v.SKU, scg.Affiliate),
			}
			if err := scg.inventory.Add(uuid, entry); err != nil {
				scg.printf("%s", err.Error())
			}
		}

		if buyPrice, err := mtgmatcher.ParsePrice(v.SellListPrice); err == nil && buyPrice > 0 {
			var priceRatio float64
			if retailPrice > 0 {
				priceRatio = buyPrice / retailPrice * 100
			}

			entry := &mtgban.BuylistEntry{
				BuyPrice:   buyPrice,
				PriceRatio: priceRatio,
				URL:        buyURL,
				OriginalId: v.SKU,
			}
			if err := scg.buylist.Add(uuid, entry); err != nil {
				scg.printf("%s", err.Error())
			}
		}
	}
}

// Load streams the single catalog export (authenticated with the API key) and
// fills the sealed inventory and buylist in one pass.
func (scg *StarcitygamesSealed) Load(ctx context.Context) error {
	scg.productMap = buildProductMap()

	setIDs, err := scg.client.SetIDs(ctx, scg.game)
	if err != nil {
		scg.printf("could not load set ids for buylist links: %v", err)
	}
	scg.setIDs = setIDs

	body, err := scg.client.DownloadCatalog(ctx)
	if err != nil {
		return fmt.Errorf("catalog load failed: %w", err)
	}
	defer body.Close()

	count := 0
	err = decodeCatalog(body, func(p CatalogProduct) error {
		scg.processProduct(p)
		count++
		if count%5000 == 0 {
			scg.printf("Processed %d products", count)
		}
		return nil
	})
	if err != nil {
		return err
	}
	scg.printf("Processed %d products total", count)

	now := time.Now()
	scg.inventoryDate = now
	scg.buylistDate = now
	return nil
}

func (scg *StarcitygamesSealed) Inventory() mtgban.InventoryRecord {
	return scg.inventory
}

func (scg *StarcitygamesSealed) Buylist() mtgban.BuylistRecord {
	return scg.buylist
}

func (scg *StarcitygamesSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCGSealed"
	info.InventoryTimestamp = &scg.inventoryDate
	info.BuylistTimestamp = &scg.buylistDate
	info.SealedMode = true
	switch scg.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	}
	return
}
