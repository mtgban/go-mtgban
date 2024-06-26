package tcgplayer

import (
	"fmt"

	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

type TCGSYPList struct {
	LogCallback mtgban.LogCallbackFunc
	Affiliate   string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
}

func (tcg *TCGSYPList) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSYPList] "+format, a...)
	}
}

func NewScraperSYP() *TCGSYPList {
	tcg := TCGSYPList{}
	tcg.inventory = mtgban.InventoryRecord{}
	return &tcg
}

func (tcg *TCGSYPList) scrape() error {
	tcg.printf("Retrieving skus")
	uuid2skusMap, err := getAllSKUs()
	if err != nil {
		return err
	}
	tcg.printf("Found skus for %d entries", len(uuid2skusMap))

	// Convert to a map of id:sku, we'll regenerate the uuid differently
	sku2product := map[int]mtgjson.TCGSku{}
	for _, skus := range uuid2skusMap {
		for _, sku := range skus {
			sku2product[sku.SkuId] = sku
		}
	}

	sypList, err := LoadSyp()
	if err != nil {
		return err
	}
	tcg.printf("Found syp list of %d entries", len(sypList))

	for _, syp := range sypList {
		sku, found := sku2product[syp.SkuId]
		if !found {
			continue
		}

		isFoil := sku.Printing == "FOIL"
		isEtched := sku.Finish == "FOIL ETCHED"
		cardId, err := mtgmatcher.MatchId(fmt.Sprint(sku.ProductId), isFoil, isEtched)
		if err != nil {
			continue
		}

		cond, found := skuConditions[sku.Condition]
		if !found {
			continue
		}

		printing := "Normal"
		if sku.Printing == "FOIL" {
			printing = "Foil"
		}
		link := TCGPlayerProductURL(sku.ProductId, printing, tcg.Affiliate, cond, "English")

		entry := mtgban.InventoryEntry{
			Conditions: cond,
			Price:      syp.MarketPrice,
			Quantity:   syp.MaxQty,
			URL:        link,
		}

		err = tcg.inventory.Add(cardId, &entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGSYPList) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGSYPList) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player SYP List"
	info.Shorthand = "TCGSYPList"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	return
}
