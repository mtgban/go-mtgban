package tcgplayer

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-tcgplayer"
)

// Index prices Magic singles from the partner API's price guide, the
// low and market numbers rather than any one seller's listing.
type Index struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory mtgban.InventoryRecord

	client *tcgplayer.Client
}

var availableIndexNames = []string{
	"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
}

type indexChan struct {
	TCGProductId string
	UUID         string
	Etched       bool
}

func (tcg *Index) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper authenticated with a partner API
// key pair.
func NewScraperIndex(publicID, privateID string) (*Index, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Index{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *Index) processEntry(ctx context.Context, channel chan<- responseChan, reqs []indexChan) error {
	var ids []int
	for i := range reqs {
		id, err := strconv.Atoi(reqs[i].TCGProductId)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	results, err := tcg.client.GetMarketPricesByProducts(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		// Skip empty entries
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		productID := fmt.Sprint(result.ProductID)

		uuid := ""
		isFoil := result.SubTypeName == "Foil"
		isEtched := false
		for _, req := range reqs {
			if req.TCGProductId == productID {
				uuid = req.UUID
				isEtched = req.Etched
				break
			}
		}

		cardID, err := mtgmatcher.MatchID(uuid, isFoil, isEtched)
		if err != nil {
			tcg.printf("(%d / %s) - %s", result.ProductID, uuid, err)
			continue
		}

		// Skip impossible entries, such as listing mistakes that list a foil
		// price for a foil-only card
		co, _ := mtgmatcher.GetUUID(cardID)
		if !co.Etched &&
			((co.Foil && result.SubTypeName != "Foil") ||
				(!co.Foil && result.SubTypeName != "Normal")) {
			continue
		}

		// These are sorted as in availableIndexNames
		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, getDirectPrice(result.DirectLowPrice),
		}

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}

			// Certain sets are marked as English on the site despite not being as such
			// Override here, so that links don't point at filters that provide no results
			lang := co.Language
			switch co.SetCode {
			case "STA", "SOA":
				lang = "English"
			}

			isDirect := availableIndexNames[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.Affiliate, "", lang, isDirect)

			out := responseChan{
				cardID: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableIndexNames[i],
					Bundle:     isDirect,
				},
			}

			channel <- out
		}
	}

	return nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Index) Load(ctx context.Context) error {
	pages := make(chan indexChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			dupes := map[string]struct{}{}
			buffer := make([]indexChan, 0, tcgplayer.MaxIDsInRequest)

			for page := range pages {
				// Skip dupes
				_, found := dupes[page.TCGProductId]
				if found {
					continue
				}
				dupes[page.TCGProductId] = struct{}{}

				// Add our pair to the buffer
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
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetAllSets()
		i := 1
		for _, code := range sets {
			set, _ := mtgmatcher.GetSet(code)

			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			for _, card := range set.Cards {
				tcgID, found := card.Identifiers["tcgplayerProductId"]
				if found {
					pages <- indexChan{
						TCGProductId: tcgID,
						UUID:         card.UUID,
					}
				}

				// Sometimes etched-only cards have two tcgIds by mistake, skip one
				tcgEtchedID, found := card.Identifiers["tcgplayerEtchedProductId"]
				if found && tcgEtchedID != tcgID {
					pages <- indexChan{
						TCGProductId: tcgEtchedID,
						UUID:         card.UUID,
						Etched:       true,
					}
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		// Relaxed because sometimes we get duplicates due to how the ids
		// get buffered, but there is really no harm
		err := tcg.inventory.AddRelaxed(result.cardID, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *Index) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (tcg *Index) MarketNames() []string {
	return availableIndexNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (tcg *Index) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *Index) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.Family = "TCG"
	return
}
