// Package starcitygames scrapes Star City Games, for singles and sealed
// product, across every game they carry.
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

// Starcitygames prices SCG's singles, both what they sell and what they buy.
type Starcitygames struct {
	LogCallback   mtgban.LogCallbackFunc
	inventoryDate time.Time
	buylistDate   time.Time

	Affiliate string

	TargetEdition string

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	// buckets holds the listings a second stock record has already been seen
	// for, so the pair folds together in either stream order.
	buckets map[string]struct{}

	setIDs map[string]int
	client *SCGClient
	game   int
}

// NewScraper returns a singles scraper for one game, using the given API key.
func NewScraper(game int, apiKey string) *Starcitygames {
	scg := Starcitygames{}
	scg.reset()
	scg.client = NewSCGClient(apiKey)
	scg.game = game
	return &scg
}

// reset drops everything a run has collected, for a stream that broke and has
// to be read again from the top. What a listing's records have already been
// seen goes with the records themselves: they are all coming again.
func (scg *Starcitygames) reset() {
	scg.inventory = mtgban.InventoryRecord{}
	scg.buylist = mtgban.BuylistRecord{}
	scg.buckets = map[string]struct{}{}
}

func (scg *Starcitygames) printf(format string, a ...any) {
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

	// This scraper handles singles only; sealed has its own scraper, and
	// the catalog also carries supplies and bulk lots that are not cards
	// at all. product_type says which is which, so nothing else has to
	// infer it from the name or the sku.
	if p.ProductType != ProductTypeSingles {
		return
	}
	if !strings.HasPrefix(p.SKU, "SGL-") {
		return
	}
	if gameFromCatalog(p.Game) != scg.game {
		return
	}
	if scg.TargetEdition != "" && scg.TargetEdition != p.Set {
		return
	}

	cardID, err := resolveProduct(scg.game, p)
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

	link := SCGProductURL(p.URL, "", scg.Affiliate)

	// The buylist link points at the sell-your-cards page for this printing,
	// not the retail page; fall back to retail if the set can't be matched.
	buyURL := link
	ids := setIDsForProduct(scg.setIDs, p.Set, p.SKU)
	if len(ids) > 0 {
		buyURL = SCGBuylistURL(scg.game, p.Name, p.Language, ids)
	}

	customFields := map[string]string{
		"SCGName":     p.Name,
		"SCGEdition":  p.Set,
		"SCGLanguage": p.Language,
		"SCGFinish":   p.Finish,
		"scgNumber":   p.CollectorNumber,
		"scgSKU":      p.SKU,
	}

	ignore := strings.Contains(p.Set, "World Championship") || strings.Contains(p.Name, "Token")

	// A second stock bucket of a listing the catalog also carries plainly.
	// Star City Games splits some Armory Deck singles across two product
	// records - "SGL-FAB-AGB-014-ENN" beside "SGL-FAB-AGB-014_CC-ENN" - which
	// agree on the card, the set, the number, the treatment and the price of
	// every grade they share, and differ only in how many copies sit in each.
	// They are one listing's stock written twice over, so the pair's copies
	// are added together rather than reported as the duplicate they would
	// otherwise look like, which discarded them.
	//
	// Which of the two comes down the stream first is not the catalog's to
	// promise, and it does not hold: most Armory Deck: Pleiades listings lead
	// with the marked record. So it is the one that arrives second that is
	// folded in, whether or not it wears the marker - the first is the one
	// there is nothing yet to fold it into.
	second := scg.secondStock(p.SKU)

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
				OriginalID: p.SKU,
				InstanceID: v.SKU,
				URL:        SCGProductURL(p.URL, v.SKU, scg.Affiliate),
			}
			if condition == "NM" {
				entry.CustomFields = customFields
			}
			if err := scg.addInventoryStock(second, cardID, entry); err != nil && !ignore {
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
				URL:          buyURL,
				OriginalID:   p.SKU,
				InstanceID:   v.SKU,
				CustomFields: blFields,
			}
			if err := scg.addBuylistStock(second, cardID, entry); err != nil && !ignore {
				scg.printf("%s", err.Error())
			}
		}
	}
}

// loadCatalog streams the single catalog export, which carries both retail
// (price/qty) and buylist (sell_list_price) data per variant, and fills the
// inventory and buylist in one pass.
func (scg *Starcitygames) loadCatalog(ctx context.Context) error {
	setIDs, err := scg.client.SetIDs(ctx, scg.game)
	if err != nil {
		scg.printf("could not load set ids for buylist links: %v", err)
	}
	scg.setIDs = setIDs

	count := 0
	err = scg.client.StreamCatalog(ctx, func() {
		scg.printf("Catalog stream broke after %d products, downloading it again", count)
		scg.reset()
		count = 0
	}, func(p CatalogProduct) error {
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

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (scg *Starcitygames) Load(ctx context.Context) error {
	if err := scg.loadCatalog(ctx); err != nil {
		return fmt.Errorf("catalog load failed: %w", err)
	}
	return nil
}

// secondStock reports whether this sku is the second stock bucket of a
// listing already seen this run: by the marker it wears, or by its plain
// twin arriving after the marked record registered the pair. Which of the
// two the catalog streams first is not its to promise, and it does not
// hold, so whichever arrives second is the one that folds.
func (scg *Starcitygames) secondStock(sku string) bool {
	key := bucketKey(sku)
	if secondBucket(sku) {
		scg.buckets[key] = struct{}{}
		return true
	}
	_, paired := scg.buckets[key]
	return paired
}

// addInventoryStock files a variant's retail record with the strictness the
// bucket decision picked: a second bucket merges into the first rather than
// reading as the duplicate it would otherwise be.
func (scg *Starcitygames) addInventoryStock(second bool, cardID string, entry *mtgban.InventoryEntry) error {
	if second {
		return scg.inventory.Add(cardID, entry)
	}
	return scg.inventory.AddStrict(cardID, entry)
}

// addBuylistStock is addInventoryStock for the buylist side of the record.
func (scg *Starcitygames) addBuylistStock(second bool, cardID string, entry *mtgban.BuylistEntry) error {
	if second {
		return scg.buylist.AddRelaxed(cardID, entry)
	}
	return scg.buylist.Add(cardID, entry)
}

// Inventory returns what Load collected. See mtgban.Seller.
func (scg *Starcitygames) Inventory() mtgban.InventoryRecord {
	return scg.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (scg *Starcitygames) Buylist() mtgban.BuylistRecord {
	return scg.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (scg *Starcitygames) Info() (info mtgban.ScraperInfo) {
	info.Name = "Star City Games"
	info.Shorthand = "SCG"
	info.InventoryTimestamp = &scg.inventoryDate
	info.BuylistTimestamp = &scg.buylistDate
	info.CreditMultiplier = 1.3
	switch scg.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameFleshAndBlood:
		info.Game = mtgban.GameFleshAndBlood
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	case GameRiftbound:
		info.Game = mtgban.GameRiftbound
	}
	return
}
