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

// TCGGame is the retail scraper for any single-game TCGplayer category whose
// cards the matcher identifies by name + collector number + finish (Lorcana,
// Riftbound, ...); Magic has its own SKU-driven scrapers. SupportedGames
// below maps each game tag to the category it is served from.
type TCGGame struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	editions map[int]tcgplayer.Group

	printings map[int]string

	category            int
	categoryName        string
	categoryDisplayName string
	game                string

	productTypes []string

	// sealed selects the sealed mode: products are resolved through the
	// sealed product map by their product id instead of the matcher, and
	// only Unopened skus are priced.
	sealed    bool
	sealedMap map[int][]string

	client *tcgplayer.Client
}

func (tcg *TCGGame) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tag := "[TCG](" + tcg.categoryName + ") "
		if !slices.Contains(tcg.productTypes, tcgplayer.ProductTypesSingles[0]) {
			tag += "{" + strings.Join(tcg.productTypes, ",") + "} "
		}
		tcg.LogCallback(tag+format, a...)
	}
}

// SupportedGames maps every game tag TCGGame and TCGGameIndex can be built
// for to the TCGplayer category carrying it. Magic is deliberately absent: it
// is identified by SKU and has its own scrapers. Supporting one more game is
// one entry here, provided the matcher has a datastore for it.
var SupportedGames = map[string]int{
	mtgban.GameLorcana:   tcgplayer.CategoryLorcana,
	mtgban.GameRiftbound: tcgplayer.CategoryRiftbound,
	mtgban.GameOnePiece:  tcgplayer.CategoryOnePiece,
}

func NewScraperGame(game, publicID, privateID string) (*TCGGame, error) {
	category, found := SupportedGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported game %q", game)
	}

	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := TCGGame{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency

	tcg.category = category
	tcg.game = game
	tcg.productTypes = tcgplayer.ProductTypesSingles

	tcg.printings = map[int]string{}

	return &tcg, nil
}

// NewScraperGameSealed prices a game's sealed products: everything the
// category files outside the singles type, so a product type TCGplayer
// adds later is picked up rather than silently skipped. Products resolve
// through the sealed product map by their product id, the identity the
// datastore stamps on every sealed entry.
func NewScraperGameSealed(game, publicID, privateID string) (*TCGGame, error) {
	tcg, err := NewScraperGame(game, publicID, privateID)
	if err != nil {
		return nil, err
	}
	tcg.sealed = true
	tcg.productTypes = tcgplayer.ProductTypesSealed
	return tcg, nil
}

func (tcg *TCGGame) processPage(ctx context.Context, channel chan<- genericChan, page int) error {
	products, err := tcg.client.ListAllProducts(ctx, tcg.category, tcg.productTypes, true, page)
	if err != nil {
		return err
	}

	productMap := map[int]tcgplayer.Product{}
	skuMap := map[int]tcgplayer.SKU{}
	var skuIds []int
	for _, product := range products {
		productMap[product.ProductID] = product

		for _, sku := range product.Skus {
			if tcg.sealed {
				if sku.ConditionID != SKUConditionUnopened {
					continue
				}
			} else {
				_, found := SKUConditionMap[sku.ConditionID]
				if !found {
					continue
				}
			}
			// Only English
			if sku.LanguageID != 1 {
				continue
			}

			skuIds = append(skuIds, sku.SKUID)
			skuMap[sku.SKUID] = sku
		}
	}

	for i := 0; i < len(skuIds); i += tcgplayer.MaxIDsInRequest {
		start := i
		end := i + tcgplayer.MaxIDsInRequest
		if end > len(skuIds) {
			end = len(skuIds)
		}

		results, err := tcg.client.GetMarketPricesBySKUs(ctx, skuIds[start:end])
		if err != nil {
			return err
		}

		for _, result := range results {
			price := result.LowestListingPrice
			if price == 0 {
				continue
			}

			sku := skuMap[result.SKUID]
			product, found := productMap[sku.ProductID]
			if !found {
				continue
			}

			if tcg.sealed {
				// The product id is the sealed entry's whole identity;
				// anything the map does not name is a product the
				// datastore does not carry
				uuids := tcg.sealedMap[sku.ProductID]
				if len(uuids) != 1 {
					continue
				}
				channel <- genericChan{
					key: uuids[0],
					entry: mtgban.InventoryEntry{
						Conditions: "NM",
						Price:      price,
						Quantity:   1,
						URL:        GenerateProductURL(sku.ProductID, "", tcg.Affiliate, "", "", false),
						OriginalId: fmt.Sprint(sku.ProductID),
						InstanceId: fmt.Sprint(sku.SKUID),
					},
				}
				continue
			}

			cardName := product.Name
			number := RawProductNumber(&product)
			// The printing name rides along in the variation so the game
			// rules can tell foil sub-types apart (SelectFinish).
			printing := tcg.printings[sku.PrintingID]
			theCard := &mtgmatcher.InputCard{
				Name:      cardName,
				Edition:   tcg.editions[product.GroupID].Name,
				Variation: strings.TrimSpace(number + " " + printing),
				Foil:      printing != "Normal",
			}
			// Every game datastore stamps the TCGplayer product id on the
			// printing it names, so the id identifies the card outright and
			// Match tries it before the fields above. It names a product
			// though, and the foil flag beside it names one of two
			// finishes: a sku printed in anything else is a choice neither
			// can express, so the wording answers for those alone rather
			// than the id resolving them to the plain foil.
			if printing == "Normal" || printing == "Foil" {
				theCard.Id = fmt.Sprint(sku.ProductID)
			}
			cardID, err := mtgmatcher.Match(theCard)
			if errors.Is(err, mtgmatcher.ErrUnsupported) {
				continue
			} else if err != nil {
				// Name the card, not just the price row: a sku id alone
				// says nothing about which product failed to match.
				tcg.printf("%v for %q (product %d)", err, theCard, sku.ProductID)
				tcg.printf("%+v", result)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					tcg.printf("%d %s got ids: %s", sku.ProductID, cardName, probes)
					for _, probe := range probes {
						co, _ := mtgmatcher.GetUUID(probe)
						tcg.printf("%s: %s", probe, co)
					}
				}
				continue
			}

			condition := SKUConditionMap[sku.ConditionID]

			link := GenerateProductURL(sku.ProductID, printing, tcg.Affiliate, condition, "", false)

			out := genericChan{
				key: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: condition,
					Price:      price,
					Quantity:   1,
					URL:        link,
					OriginalId: fmt.Sprint(sku.ProductID),
					InstanceId: fmt.Sprint(sku.SKUID),
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGGame) Load(ctx context.Context) error {
	// Initialize data for debug logs
	var err error
	tcg.categoryName, tcg.categoryDisplayName, err = GetCategoryNames(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}

	printings, err := tcg.client.ListCategoryPrintings(ctx, tcg.category)
	if err != nil {
		return err
	}
	tcg.printf("Found %d printings for category %d", len(printings), tcg.category)
	for _, printing := range printings {
		tcg.printf("%d - %s", printing.PrintingID, printing.Name)
		tcg.printings[printing.PrintingID] = printing.Name
	}

	editions, err := EditionMap(ctx, tcg.client, tcg.category)
	if err != nil {
		return err
	}
	tcg.editions = editions
	tcg.printf("Found %d editions", len(editions))

	// The totals must count the same product types the pages list, or the
	// page offsets walk a different result set than the count promised
	totals, err := tcg.client.TotalProducts(ctx, tcg.category, tcg.productTypes)
	if err != nil {
		return err
	}
	tcg.printf("Found %d products", totals)

	if tcg.sealed {
		tcg.sealedMap = mtgmatcher.BuildSealedProductMap("tcgplayerProductId")
		tcg.printf("Loaded %d sealed products", len(tcg.sealedMap))
	}

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

func (tcg *TCGGame) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

func (tcg *TCGGame) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCGplayer"
	info.Shorthand = "TCGPlayer"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.NoQuantityInventory = true
	info.Game = tcg.game
	if tcg.sealed {
		info.Shorthand = "TCGSealed"
		info.SealedMode = true
	}
	return
}
