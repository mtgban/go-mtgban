package tcgplayer

import (
	"fmt"
	"io"
	"strings"
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
	// Retrieve all the SKUs for a productId, in order to parse later properties
	skus, err := tcg.client.SKUsForId(req.TCGProductId)
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

	co, err := mtgmatcher.GetUUID(req.UUID)
	if err != nil {
		return err
	}

	skuIds := []string{}
	allSkus := []skuType{}
	for _, result := range skus {
		switch result.LanguageId {
		case 1: // English
		case 7: // Japanese
			if !(co.Card.HasUniqueLanguage("Japanese") && strings.Contains(co.Card.Number, "★")) {
				continue
			}
		default:
			continue
		}

		// Untangle foiling status from single id (ie Unhinged, 10E etc)
		if result.PrintingId == 1 && !co.Card.HasNonFoil {
			continue
		} else if result.PrintingId == 2 && !co.Card.HasFoil {
			continue
		}

		s := skuType{
			SkuId: result.SkuId,
			Foil:  result.PrintingId == 2,
			Cond:  result.ConditionId,
		}
		allSkus = append(allSkus, s)
		skuIds = append(skuIds, fmt.Sprint(result.SkuId))
	}

	// Retrieve a list of skus with their prices
	results, err := tcg.client.PricesForSKU(strings.Join(skuIds, ","))
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
		var theSku skuType
		for _, target := range allSkus {
			if target.SkuId == result.SkuId {
				theSku = target
				break
			}
		}
		if theSku.SkuId == 0 {
			continue
		}

		theCard := &mtgmatcher.Card{
			Id:   req.UUID,
			Foil: theSku.Foil,
		}
		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			return err
		}

		cond := ""
		switch theSku.Cond {
		case 1:
			cond = "NM"
		case 2:
			cond = "SP"
		case 3:
			cond = "MP"
		case 4:
			cond = "HP"
		case 5:
			cond = "PO"
		default:
			tcg.printf("unknown condition %d for %d", theSku.Cond, theSku.SkuId)
			continue
		}

		prices := []float64{
			result.LowestListingPrice, result.DirectLowPrice,
		}
		names := []string{
			"TCG Player", "TCG Direct",
		}

		link := "https://shop.tcgplayer.com/product/productsearch?id=" + req.TCGProductId
		if tcg.Affiliate != "" {
			link += fmt.Sprintf("&utm_campaign=affiliate&utm_medium=%s&utm_source=%s&partner=%s", tcg.Affiliate, tcg.Affiliate, tcg.Affiliate)
		}

		for i := range names {
			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: cond,
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
					card, _ := mtgmatcher.GetUUID(page.UUID)
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
			tcg.printf("%s", err.Error())
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

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = tcg.inventoryDate
	info.BuylistTimestamp = tcg.buylistDate
	info.MultiCondBuylist = true
	info.NoQuantityInventory = true
	info.NoCredit = true
	return
}
