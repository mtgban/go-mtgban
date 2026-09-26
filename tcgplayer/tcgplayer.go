// Package tcgplayer scrapes TCGplayer through both their partner API and
// their storefront: market and index pricing, sealed product, per-seller
// inventory, and the SKU catalog the other scrapers resolve against.
package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-tcgplayer"
)

// Market prices singles from TCGplayer's partner API, splitting the
// result into the sub-sellers their pricing endpoint reports and the buylist
// they publish alongside it.
type Market struct {
	logCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	affiliate      string
	maxConcurrency int
	skusData       SKUMap

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	backend *mtgmatcher.Backend
	client  *tcgplayer.Client
}

type marketChan struct {
	UUID      string
	Condition string
	Printing  string
	Finish    string
	ProductID int
	SkuID     int
	Language  string
}

type responseChan struct {
	cardID string
	entry  mtgban.InventoryEntry
	bl     *mtgban.BuylistEntry
}

var availableMarketNames = []string{
	"TCG Player", "TCG Direct",
}

var name2shorthand = map[string]string{
	"TCG Low":          "TCGLow",
	"TCG Market":       "TCGMarket",
	"TCG Mid":          "TCGMid",
	"TCG Direct Low":   "TCGDirectLow",
	"TCG Player":       "TCGPlayer",
	"TCG Direct":       "TCGDirect",
	"TCG Direct (net)": "TCGDirectNet",
	"TCGplayer":        "TCGPlayer",
	"TCGplayer Direct": "TCGDirect",
}

func (tcg *Market) printf(format string, a ...any) {
	if tcg.logCallback != nil {
		tcg.logCallback("[TCGMkt] "+format, a...)
	}
}

// NewScraperMarket returns a market scraper authenticated with a partner API
// key pair.
func NewScraperMarket(b *mtgmatcher.Backend, publicID, privateID string) (*Market, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Market{}
	tcg.backend = b
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.client = client
	tcg.maxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *Market) processEntry(ctx context.Context, channel chan<- responseChan, reqs []marketChan) error {
	ids := make([]int, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].SkuID
	}

	// Retrieve a list of skus with their prices
	results, err := tcg.client.GetMarketPricesBySKUs(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		var req marketChan
		for _, req = range reqs {
			if result.SKUID == req.SkuID {
				break
			}
		}

		isFoil := req.Printing == "FOIL"
		isEtched := req.Finish == "ETCHED"
		cardID, err := tcg.backend.MatchID(req.UUID, isFoil, isEtched)
		if err != nil {
			tcg.printf("%s - (tcgId:%d / uuid:%s)", err.Error(), req.ProductID, req.UUID)
			continue
		}

		// Skip impossible entries, such as listing mistakes that list a foil
		// price for a foil-only card
		co, _ := tcg.backend.GetUUID(cardID)
		if !co.Etched &&
			((co.Foil && req.Printing != "FOIL") ||
				(!co.Foil && req.Printing != "NON FOIL")) {
			continue
		}

		cond, err := mtgban.ParseCondition(req.Condition)
		if err != nil {
			tcg.printf("unknown condition %s for %d", req.Condition, req.SkuID)
			continue
		}

		// Sorted as in availableMarketNames
		prices := []float64{
			result.LowestListingPrice, getDirectPrice(result.DirectLowPrice),
		}
		printing := "Normal"
		if req.Printing == "FOIL" {
			printing = "Foil"
		}
		for i := range availableMarketNames {
			isDirect := i == 1
			link := GenerateProductURL(req.ProductID, printing, tcg.affiliate, cond, req.Language, isDirect)

			out := responseChan{
				cardID: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: cond,
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableMarketNames[i],
					Bundle:     isDirect,
					OriginalID: fmt.Sprint(req.ProductID),
					InstanceID: fmt.Sprint(result.SKUID),
				},
			}

			if isDirect {
				price := DirectPriceAfterFees(prices[i])
				if price > 0 {
					out.bl = &mtgban.BuylistEntry{
						Conditions: cond,
						BuyPrice:   price,
						URL:        link,
						VendorName: "TCG Direct (net)",
						OriginalID: fmt.Sprint(req.ProductID),
						InstanceID: fmt.Sprint(result.SKUID),
					}
				}
			}

			channel <- out
		}
	}

	return nil
}

// derivedSkuMatches reports whether sku prices the two-sided token sheet's
// combined entity in the finish it was minted for - one of ownIDs (the
// pairing's own product ids, distinct from the single face co's own sku
// list is keyed by), not unopened/etched, and in the requested language or
// English. The sku catalog spells a nonfoil printing "NON FOIL", not
// "NORMAL" - confirmed against the real file, and against this same
// package's own req.Printing checks a few lines up - a one-word typo here
// silently zeroed every nonfoil two-sided token sheet's price rather than
// erroring, since an empty sku list is indistinguishable from "priced
// elsewhere."
func derivedSkuMatches(sku TCGSku, ownIDs map[string]bool, wantFoil bool, wantLanguage string) bool {
	if !ownIDs[strconv.Itoa(sku.ProductID)] {
		return false
	}
	if sku.Condition == "UNOPENED" || sku.Finish == "ETCHED" {
		return false
	}
	wantPrinting := "NON FOIL"
	if wantFoil {
		wantPrinting = "FOIL"
	}
	if sku.Printing != wantPrinting {
		return false
	}
	return mtgmatcher.Equals(sku.Language, wantLanguage) || sku.Language == "ENGLISH"
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Market) Load(ctx context.Context) error {
	skusMap := tcg.skusData
	if skusMap == nil {
		return errors.New("sku map not loaded")
	}
	tcg.printf("Found skus for %d entries", len(skusMap))

	start := time.Now()

	// A two-sided token sheet's product is filed under both of its single-
	// faced tokens' sku lists in the datastore's own sku file - one physical
	// card, one price, but two unrelated uuids both listing it - so priced
	// the ordinary way it lands on both, and a token that pairs with several
	// partners absorbs every one of their prices. mtgmatcher now mints a
	// combined entity for it; derivedProductIDs is every id one of those
	// entities claims, read once so the per-card walk below can skip it
	// rather than mis-price the single-faced token, and derivedBySet is
	// where each entity's own combined walk (below) prices it instead.
	derivedProductIDs := map[int]bool{}
	derivedBySet := map[string][]*mtgmatcher.CardObject{}
	for _, co := range tcg.backend.UUIDs {
		if co.Identifiers["derivedTokenPair"] != "true" {
			continue
		}
		derivedBySet[co.SetCode] = append(derivedBySet[co.SetCode], co)
		for _, id := range strings.Split(co.Identifiers["tcgplayerProductIds"], ",") {
			n, err := strconv.Atoi(id)
			if err == nil {
				derivedProductIDs[n] = true
			}
		}
	}

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.maxConcurrency; i++ {
		wg.Go(func() {
			buffer := make([]marketChan, 0, tcgplayer.MaxIDsInRequest)

			for page := range pages {
				// Add our data to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == cap(buffer) {
					err := tcg.processEntry(ctx, channel, buffer)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntry(ctx, channel, buffer)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
		})
	}

	go func() {
		sets := tcg.backend.GetAllSets()
		total := len(sets) - 1
		i := 1

		idsFound := map[int]struct{}{}
		for _, code := range sets {
			set, _ := tcg.backend.GetSet(code)

			switch set.Code {
			case "4EDALT":
				continue
			}

			tcg.printf("Scraping %s (%d/%d)", set.Name, i, total)
			i++

			for _, card := range set.Cards {
				uuid := card.Identifiers["mtgjsonId"]
				skus, found := skusMap[uuid]
				if !found {
					continue
				}

				_, found = card.Identifiers["needsNewTCGSKUs"]
				if found {
					tcgID := card.Identifiers["tcgplayerProductId"]
					id, err := strconv.Atoi(tcgID)
					if err != nil {
						continue
					}

					altSkus, err := tcg.client.ListProductSKUs(ctx, id)
					if err != nil {
						tcg.printf("Error retrieving alternative SKUs: %s", err.Error())
						continue
					}

					skus = skus[:0]
					for _, sku := range altSkus {
						lang, found := map[int]string{
							1:  "ENGLISH",
							2:  "CHINESE SIMPLIFIED",
							3:  "CHINESE TRADITIONAL",
							4:  "FRENCH",
							5:  "GERMAN",
							6:  "ITALIAN",
							7:  "JAPANESE",
							8:  "KOREAN",
							9:  "PORTUGUESE BRAZIL",
							10: "RUSSIAN",
							11: "SPANISH",
						}[sku.LanguageID]
						if !found {
							continue
						}

						// Check for language early because we cannot have
						// duplicated sku ids, while the card may very well do
						if !mtgmatcher.Equals(lang, card.Language) {
							continue
						}

						printing := "NON FOIL"
						if sku.PrintingID == 2 {
							printing = "FOIL"
						}

						cond, found := map[int]string{
							1: "NEAR MINT",
							2: "LIGHTLY PLAYED",
							3: "MODERATELY PLAYED",
							4: "HEAVILY PLAYED",
							5: "DAMAGED",
						}[sku.ConditionID]
						if !found {
							continue
						}

						skus = append(skus, TCGSku{
							Condition: cond,
							Language:  lang,
							Printing:  printing,
							ProductID: id,
							SkuID:     sku.SKUID,
						})
					}
				}

				hasNonfoil := card.HasFinish(mtgmatcher.FinishNonfoil)
				hasFoil := card.HasFinish(mtgmatcher.FinishFoil)
				hasEtched := card.HasFinish(mtgmatcher.FinishEtched)

				for _, sku := range skus {
					// Skip sealed products
					if sku.Condition == "UNOPENED" {
						continue
					}
					// Skip non-main languages
					if !mtgmatcher.Equals(sku.Language, card.Language) {
						// These two sets contain English sku, skip them
						switch set.Code {
						case "LEGITA", "DRKITA":
							continue
						}
						// Otherwise many Japanese and special cards are listed as English, skip anything else
						if sku.Language != "ENGLISH" {
							continue
						}
					}
					// Extra validation for incorrect data
					if !hasNonfoil && sku.Printing == "NON FOIL" {
						continue
					}
					if !hasFoil && !hasEtched && (sku.Printing == "FOIL" || sku.Finish == "ETCHED") {
						continue
					}
					if !hasEtched && sku.Finish == "ETCHED" {
						continue
					}
					// This product is a two-sided token sheet's pairing, not
					// this single-faced token alone; the walk below prices
					// it once, on the entity that is actually that product.
					if derivedProductIDs[sku.ProductID] {
						continue
					}
					// Make sure the right id is parsed
					// Check for tcgplayerProductId due to non-English cards from duplicated sets
					if sku.Finish != "ETCHED" && card.Identifiers["tcgplayerProductId"] != "" && fmt.Sprint(sku.ProductID) != card.Identifiers["tcgplayerProductId"] {
						continue
					}
					// Skip dupes
					_, found := idsFound[sku.SkuID]
					if found {
						continue
					}
					idsFound[sku.SkuID] = struct{}{}

					pages <- marketChan{
						UUID:      card.UUID,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductID: sku.ProductID,
						SkuID:     sku.SkuID,
						Language:  sku.Language,
					}
				}
			}

			// Price each two-sided token sheet's combined entity once, from
			// whichever single-faced side's sku list carries its own ids -
			// mtgjson files one product's skus under both faces, which is
			// the collision derivedProductIDs above steers away from them.
			for _, co := range derivedBySet[set.Code] {
				skus, found := skusMap[co.Identifiers["tokenPairPartA"]]
				if !found {
					skus, found = skusMap[co.Identifiers["tokenPairPartB"]]
				}
				if !found {
					continue
				}

				ownIDs := map[string]bool{}
				for _, id := range strings.Split(co.Identifiers["tcgplayerProductIds"], ",") {
					ownIDs[id] = true
				}

				for _, sku := range skus {
					if !derivedSkuMatches(sku, ownIDs, co.Foil, co.Language) {
						continue
					}
					_, dupe := idsFound[sku.SkuID]
					if dupe {
						continue
					}
					idsFound[sku.SkuID] = struct{}{}

					pages <- marketChan{
						UUID:      co.UUID,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductID: sku.ProductID,
						SkuID:     sku.SkuID,
						Language:  sku.Language,
					}
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.inventory.AddStrict(result.cardID, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
		}
		if result.bl != nil {
			err := tcg.buylist.Add(result.cardID, result.bl)
			if err != nil {
				tcg.printf("%s", err.Error())
			}
		}
	}
	tcg.inventoryDate = time.Now()
	tcg.buylistDate = time.Now()

	tcg.printf("Took %v", time.Since(start))

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *Market) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (tcg *Market) Buylist() mtgban.BuylistRecord {
	return tcg.buylist
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (tcg *Market) MarketNames() []string {
	return availableMarketNames
}

// TraderNames names the sub-vendors this trader splits into. See
// mtgban.Trader.
func (tcg *Market) TraderNames() []string {
	return []string{"TCG Direct (net)"}
}

// InfoForScraper describes one of the sub-scrapers named above.
func (tcg *Market) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *Market) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.BuylistTimestamp = &tcg.buylistDate
	info.NoQuantityInventory = true
	return
}
