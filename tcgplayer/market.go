package tcgplayer

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type TCGPlayerMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	Affiliate      string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	buylist     mtgban.BuylistRecord
	marketplace map[string]mtgban.InventoryRecord

	client *TCGClient
}

func (tcg *TCGPlayerMarket) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGMkt] "+format, a...)
	}
}

func NewScraperMarket(publicId, privateId string) *TCGPlayerMarket {
	tcg := TCGPlayerMarket{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerMarket) processEntry(channel chan<- responseChan, req requestChan) error {
	return nil
}

func (tcg *TCGPlayerMarket) scrape() error {
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

func (tcg *TCGPlayerMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerMarket) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
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

func (tcg *TCGPlayerMarket) IntializeInventory(reader io.Reader) error {
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

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = tcg.inventoryDate
	info.BuylistTimestamp = tcg.buylistDate
	info.MultiCondBuylist = true
	return
}
