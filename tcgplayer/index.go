package tcgplayer

import (
	"fmt"
	"io"
	"sync"

	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TCGPlayerIndex struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	client *TCGClient
}

var availableIndexNames = []string{
	"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
}

type indexChan struct {
	TCGProductId string
	UUID         string
	Etched       bool
}

func (tcg *TCGPlayerIndex) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGIndex] "+format, a...)
	}
}

func NewScraperIndex(publicId, privateId string) *TCGPlayerIndex {
	tcg := TCGPlayerIndex{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerIndex) processEntry(channel chan<- responseChan, reqs []indexChan) error {
	ids := make([]string, len(reqs))
	for i := range reqs {
		ids[i] = reqs[i].TCGProductId
	}

	results, err := tcg.client.TCGPricesForIds(ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		// Skip empty entries
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		productId := fmt.Sprint(result.ProductId)

		uuid := ""
		isFoil := result.SubTypeName == "Foil"
		isEtched := false
		for _, req := range reqs {
			if req.TCGProductId == productId {
				uuid = req.UUID
				isEtched = req.Etched
				break
			}
		}

		cardId, err := mtgmatcher.MatchId(uuid, isFoil, isEtched)
		if err != nil {
			tcg.printf("(%d / %s) - %s", result.ProductId, uuid, err)
			continue
		}

		// Skip impossible entries, such as listing mistakes that list a foil
		// price for a foil-only card
		co, _ := mtgmatcher.GetUUID(cardId)
		if !co.Etched &&
			((co.Foil && result.SubTypeName != "Foil") ||
				(!co.Foil && result.SubTypeName != "Normal")) {
			continue
		}

		// These are sorted as in availableIndexNames
		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, result.DirectLowPrice,
		}
		link := TCGPlayerProductURL(result.ProductId, result.SubTypeName, tcg.Affiliate, co.Language)

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}
			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableIndexNames[i],
					Bundle:     i == 3,
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGPlayerIndex) scrape() error {
	pages := make(chan indexChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			idFound := map[string]string{}
			buffer := make([]indexChan, 0, maxIdsInRequest)

			for page := range pages {
				// Skip dupes
				_, found := idFound[page.TCGProductId]
				if found {
					continue
				}
				idFound[page.TCGProductId] = ""

				// Add our pair to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == maxIdsInRequest {
					err := tcg.processEntry(channel, buffer)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntry(channel, buffer)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetSets()
		i := 1
		for _, set := range sets {
			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			for _, card := range set.Cards {
				tcgId, found := card.Identifiers["tcgplayerProductId"]
				if found {
					pages <- indexChan{
						TCGProductId: tcgId,
						UUID:         card.UUID,
					}
				}

				// Sometimes etched-only cards have two tcgIds by mistake, skip one
				tcgEtchedId, found := card.Identifiers["tcgplayerEtchedProductId"]
				if found && tcgEtchedId != tcgId {
					pages <- indexChan{
						TCGProductId: tcgEtchedId,
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
		err := tcg.inventory.AddRelaxed(result.cardId, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGPlayerIndex) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerIndex) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) == 0 {
		_, err := tcg.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := tcg.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range tcg.inventory {
		for i := range tcg.inventory[card] {
			if tcg.inventory[card][i].SellerName == sellerName {
				if tcg.inventory[card][i].Price == 0 {
					continue
				}
				if tcg.marketplace[sellerName] == nil {
					tcg.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				tcg.marketplace[sellerName][card] = append(tcg.marketplace[sellerName][card], tcg.inventory[card][i])
			}
		}
	}

	if len(tcg.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return tcg.marketplace[sellerName], nil
}

func (tcg *TCGPlayerIndex) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	tcg.marketplace = market
	tcg.inventory = inventory

	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerIndex) MarketNames() []string {
	return availableIndexNames
}

func (tcg *TCGPlayerIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	return
}
