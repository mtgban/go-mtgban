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

	auth        string
	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

func (tcg *TCGSYPList) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSYPList] "+format, a...)
	}
}

func NewScraperSYP(auth string) *TCGSYPList {
	tcg := TCGSYPList{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.auth = auth
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

	sypList, err := LoadSyp(tcg.auth)
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

		entry := mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   syp.MarketPrice,
			Quantity:   syp.MaxQty,
			URL:        link,
		}

		err = tcg.buylist.Add(cardId, &entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.buylistDate = time.Now()

	return nil
}

func (tcg *TCGSYPList) Buylist() (mtgban.BuylistRecord, error) {
	if len(tcg.buylist) > 0 {
		return tcg.buylist, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.buylist, nil
}

func (tcg *TCGSYPList) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCGplayer SYP"
	info.Shorthand = "SYP"
	info.BuylistTimestamp = &tcg.buylistDate
	info.MetadataOnly = true
	return
}
