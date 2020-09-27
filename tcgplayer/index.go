package tcgplayer

import (
	"fmt"
	"io"
	"sync"

	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
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

func (tcg *TCGPlayerIndex) processEntry(channel chan<- responseChan, req requestChan) error {
	results, err := tcg.client.PricesForId(req.TCGProductId)
	if err != nil {
		if err.Error() == "403 Forbidden" && req.retry < defaultAPIRetry {
			req.retry++
			tcg.printf("API returned 403 in a response with status code 200")
			tcg.printf("Retrying %d/%d", req.retry, defaultAPIRetry)
			time.Sleep(time.Duration(req.retry) * 2 * time.Second)
			err = tcg.processEntry(channel, req)
		}
		return err
	}

	for _, result := range results {
		theCard := &mtgmatcher.Card{
			Id:   req.UUID,
			Foil: result.SubTypeName == "Foil",
		}
		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			return err
		}

		co, _ := mtgmatcher.GetUUID(cardId)

		// This avoids duplicates for foil-only or nonfoil-only cards
		// in particular Tenth Edition and Unhinged
		if (co.Foil && result.SubTypeName != "Foil") ||
			(!co.Foil && result.SubTypeName != "Normal") {
			continue
		}

		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, result.DirectLowPrice,
		}
		names := []string{
			"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
		}

		link := "https://shop.tcgplayer.com/product/productsearch?id=" + req.TCGProductId
		if tcg.Affiliate != "" {
			link += fmt.Sprintf("&utm_campaign=affiliate&utm_medium=%s&utm_source=%s&partner=%s", tcg.Affiliate, tcg.Affiliate, tcg.Affiliate)
		}

		for i := range names {
			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: names[i],
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGPlayerIndex) scrape() error {
	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := tcg.processEntry(channel, page)
				if err != nil {
					card, _ := mtgmatcher.Unmatch(page.UUID)
					tcg.printf("%s (%s / %s) - %s", card, page.TCGProductId, page.UUID, err)
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
				if !found {
					continue
				}

				pages <- requestChan{
					TCGProductId: tcgId,
					UUID:         card.UUID,
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			tcg.printf(err.Error())
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

func (tcg *TCGPlayerIndex) IntializeInventory(reader io.Reader) error {
	inventory, err := mtgban.LoadInventoryFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	tcg.inventory = inventory

	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerIndex) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = tcg.inventoryDate
	info.MetadataOnly = true
	return
}
