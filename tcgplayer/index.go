package tcgplayer

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-tcgplayer"
)

// Index prices Magic singles from the partner API's price guide, the
// low and market numbers rather than any one seller's listing.
type Index struct {
	logCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	affiliate      string
	maxConcurrency int

	inventory mtgban.InventoryRecord

	backend *mtgmatcher.Backend
	client  *tcgplayer.Client

	// printings maps a product id and the subtype of its price rows to
	// the printing that product sells in that finish.
	printings map[string]map[string]string
}

var availableIndexNames = []string{
	"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
}

func (tcg *Index) printf(format string, a ...any) {
	if tcg.logCallback != nil {
		tcg.logCallback("[TCGIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper authenticated with a partner API
// key pair.
func NewScraperIndex(b *mtgmatcher.Backend, publicID, privateID string) (*Index, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Index{}
	tcg.backend = b
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.maxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *Index) processEntry(ctx context.Context, channel chan<- responseChan, reqs []string) error {
	var ids []int
	for _, req := range reqs {
		id, err := strconv.Atoi(req)
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

		cardID, found := tcg.printings[fmt.Sprint(result.ProductID)][result.SubTypeName]
		if !found {
			continue
		}
		co, _ := tcg.backend.GetUUID(cardID)

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
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.affiliate, "", lang, isDirect)

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

// crossSetProductIDs reports every TCGplayer product id (regular or etched)
// claimed by cards from more than one set, keyed to the sets that claim it.
// A double-faced card's two faces, or a plain misprint beside the copy it
// doubles, always share a set - a same-named oversized sibling does not, and
// neither does an id MTGJSON has simply copied onto the wrong printing (the
// Dungeon of the Mad Mage report this guards against: AFR's ordinary card
// and OAFR's oversized one both carry 245106, which belongs only to the
// oversized product). Spanning more than one set means the id is wrong on
// at least one of them, and pricing from it would be a guess.
func crossSetProductIDs(b *mtgmatcher.Backend) map[string][]string {
	setsByID := map[string]map[string]bool{}
	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			for _, key := range [...]string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
				id, found := card.Identifiers[key]
				if !found {
					continue
				}
				if setsByID[id] == nil {
					setsByID[id] = map[string]bool{}
				}
				setsByID[id][code] = true
			}
		}
	}

	collisions := map[string][]string{}
	for id, sets := range setsByID {
		if len(sets) <= 1 {
			continue
		}
		for code := range sets {
			collisions[id] = append(collisions[id], code)
		}
		sort.Strings(collisions[id])
	}
	return collisions
}

// productPrintings maps each product id, and the subtype its price rows
// carry, to the printing the datastore says that product sells in that
// finish: "Normal" for nonfoil, "Foil" for foil, and for etched on an etched
// product. A product claimed by more than one card keeps the first, the same
// way requesting each id once always did.
func productPrintings(b *mtgmatcher.Backend, collisions map[string][]string) map[string]map[string]string {
	printings := map[string]map[string]string{}
	add := func(id, subtype, cardID string) {
		if id == "" || cardID == "" || collisions[id] != nil {
			return
		}
		if printings[id] == nil {
			printings[id] = map[string]string{}
		}
		if _, found := printings[id][subtype]; !found {
			printings[id][subtype] = cardID
		}
	}

	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		byNumber := map[string]*mtgmatcher.CardObject{}
		for _, card := range set.Cards {
			co, found := b.UUIDs[card.UUID]
			if !found {
				continue
			}
			byNumber[co.Number+"|"+co.Language] = co
			id := co.Identifiers["tcgplayerProductId"]
			etchedID := co.Identifiers["tcgplayerEtchedProductId"]
			if etchedID == "" {
				etchedID = id
			}
			add(id, "Normal", co.FoilUUIDs[mtgmatcher.FinishNonfoil])
			add(id, "Foil", co.FoilUUIDs[mtgmatcher.FinishFoil])
			add(etchedID, "Foil", co.FoilUUIDs[mtgmatcher.FinishEtched])
		}

		// A star printing with no product of its own is the other finish
		// of its base card's product (FRF 65★ is the foil of 95037).
		for _, co := range byNumber {
			base, found := byNumber[strings.TrimSuffix(co.Number, "★")+"|"+co.Language]
			if !found || base == co || co.Identifiers["tcgplayerProductId"] != "" {
				continue
			}
			id := base.Identifiers["tcgplayerProductId"]
			add(id, "Normal", co.FoilUUIDs[mtgmatcher.FinishNonfoil])
			add(id, "Foil", co.FoilUUIDs[mtgmatcher.FinishFoil])
		}
	}

	// A two-sided token sheet's combined entity is in no set's card list
	// (see mtgmatcher/magic/tokenpairs.go), one object per finish.
	for uuid, co := range b.UUIDs {
		if co.Identifiers["derivedTokenPair"] != "true" {
			continue
		}
		subtype := "Normal"
		if co.Foil || co.Etched {
			subtype = "Foil"
		}
		add(co.Identifiers["tcgplayerProductId"], subtype, uuid)
	}
	return printings
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Index) Load(ctx context.Context) error {
	collisions := crossSetProductIDs(tcg.backend)
	for id, sets := range collisions {
		tcg.printf("skipping id %s, claimed by more than one set: %v", id, sets)
	}
	tcg.printings = productPrintings(tcg.backend, collisions)

	pages := make(chan string)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.maxConcurrency; i++ {
		wg.Go(func() {
			buffer := make([]string, 0, tcgplayer.MaxIDsInRequest)

			for page := range pages {
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
		for id := range tcg.printings {
			pages <- id
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
