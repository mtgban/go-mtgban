package starcitygames

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
	defaultConcurrency = 3

	buylistBookmark = "https://sellyourcards.starcitygames.com/"
)

type Starcitygames struct {
	LogCallback   mtgban.LogCallbackFunc
	inventoryDate time.Time
	buylistDate   time.Time

	Affiliate string

	TargetEdition string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *SCGClient
	game   int
}

func NewScraper(game int, guid, apiKey string) *Starcitygames {
	scg := Starcitygames{}
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.client = NewSCGClient(guid, apiKey)
	scg.game = game
	return &scg
}

type responseChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
	pageURL  string

	ignoreErr bool
}

func (scg *Starcitygames) printf(format string, a ...interface{}) {
	if scg.LogCallback != nil {
		scg.LogCallback("[SCG] "+format, a...)
	}
}

func (scg *Starcitygames) processProduct(p CatalogProduct) {
	// A single malformed product must never abort the whole catalog stream;
	// recover, log, and skip it.
	defer func() {
		if r := recover(); r != nil {
			scg.printf("recovered from panic on %q (sku=%s): %v", p.Name, p.SKU, r)
		}
	}()

	// This scraper handles singles only; sealed has its own scraper.
	if !strings.HasPrefix(p.SKU, "SGL-") {
		return
	}
	if gameFromCatalog(p.Game) != scg.game {
		return
	}
	if scg.TargetEdition != "" && scg.TargetEdition != p.Set {
		return
	}

	cardId, err := resolveProduct(scg.game, p)
	if err != nil {
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		}
		// Skip tokens and similar
		if strings.Contains(p.Name, "Token") || strings.HasPrefix(p.Name, "{") {
			return
		}
		scg.printf("%v for %q [%s %s #%s] sku=%s scryfall=%s", err, p.Name, p.Set, p.Finish, p.CollectorNumber, p.SKU, p.ScryfallID)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			for _, probe := range alias.Probe() {
				co, _ := mtgmatcher.GetUUID(probe)
				scg.printf("- %s", co)
			}
		}
		return
	}

	link := SCGProductURL([]string{p.URL}, nil, scg.Affiliate)

	customFields := map[string]string{
		"SCGName":     p.Name,
		"SCGEdition":  p.Set,
		"SCGLanguage": p.Language,
		"SCGFinish":   p.Finish,
		"scgNumber":   p.CollectorNumber,
		"scgSKU":      p.SKU,
		"SCGID":       fmt.Sprint(p.ID),
	}

	ignore := strings.Contains(p.Set, "World Championship") || strings.Contains(p.Name, "Token")

	for _, v := range p.Variants {
		condition, err := catalogCondition(v.Condition)
		if err != nil {
			scg.printf("%v for %q", err, p.Name)
			continue
		}

		// A single catalog download carries both retail and buylist data, so
		// both records are populated in the same pass.
		retailPrice, _ := mtgmatcher.ParsePrice(v.Price)

		if retailPrice > 0 && v.Qty > 0 {
			entry := &mtgban.InventoryEntry{
				Price:      retailPrice,
				Conditions: condition,
				Quantity:   v.Qty,
				OriginalId: p.SKU,
				InstanceId: v.SKU,
				URL:        SCGProductURL([]string{p.URL}, []string{v.SKU}, scg.Affiliate),
				CustomFields: map[string]string{
					"SCGID": fmt.Sprint(p.ID),
				},
			}
			if condition == "NM" {
				entry.CustomFields = customFields
			}
			if err := scg.inventory.AddStrict(cardId, entry); err != nil && !ignore {
				scg.printf("%s", err.Error())
				scg.printf("-> %s", link)
			}
		}

		if buyPrice, err := mtgmatcher.ParsePrice(v.SellListPrice); err == nil && buyPrice > 0 {
			var priceRatio float64
			if retailPrice > 0 {
				priceRatio = buyPrice / retailPrice * 100
			}

			var blFields map[string]string
			if condition == "NM" {
				blFields = customFields
			}

			entry := &mtgban.BuylistEntry{
				Conditions:   condition,
				BuyPrice:     buyPrice,
				PriceRatio:   priceRatio,
				URL:          link,
				OriginalId:   v.SKU,
				CustomFields: blFields,
			}
			if err := scg.buylist.Add(cardId, entry); err != nil && !ignore {
				scg.printf("%s", err.Error())
			}
		}
	}
}

// loadCatalog streams the single catalog export, which carries both retail
// (price/qty) and buylist (sell_list_price) data per variant, and fills the
// inventory and buylist in one pass.
func (scg *Starcitygames) loadCatalog(ctx context.Context) error {
	body, err := scg.client.DownloadCatalog(ctx)
	if err != nil {
		return err
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

func (scg *Starcitygames) Load(ctx context.Context) error {
	if err := scg.loadCatalog(ctx); err != nil {
		return fmt.Errorf("catalog load failed: %w", err)
	}
	return nil
}

func (scg *Starcitygames) Inventory() mtgban.InventoryRecord {
	return scg.inventory
}

func (scg *Starcitygames) Buylist() mtgban.BuylistRecord {
	return scg.buylist
}

func (scg *Starcitygames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCG"
	info.InventoryTimestamp = &scg.inventoryDate
	info.BuylistTimestamp = &scg.buylistDate
	info.CreditMultiplier = 1.3
	switch scg.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
