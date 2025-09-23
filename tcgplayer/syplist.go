package tcgplayer

import (
	"context"
	"errors"
	"fmt"

	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type TCGSYPList struct {
	LogCallback mtgban.LogCallbackFunc
	Affiliate   string
	SKUsData    SKUMap

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

func (tcg *TCGSYPList) scrape(ctx context.Context) error {
	tcg.printf("Retrieving skus")
	uuid2skusMap := tcg.SKUsData
	if uuid2skusMap == nil {
		return errors.New("sku map not loaded")
	}
	tcg.printf("Found skus for %d entries", len(uuid2skusMap))

	// Convert to a map of id:sku, we'll regenerate the uuid differently
	sku2product := map[int]TCGSku{}
	for _, skus := range uuid2skusMap {
		for _, sku := range skus {
			sku2product[sku.SkuId] = sku
		}
	}

	sypList, err := LoadSyp(ctx, tcg.auth)
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

		if sku.Condition != "NEAR MINT" {
			continue
		}

		printing := "Normal"
		if sku.Printing == "FOIL" {
			printing = "Foil"
		}
		link := GenerateProductURL(sku.ProductId, printing, tcg.Affiliate, "", "English", true)

		entry := mtgban.BuylistEntry{
			BuyPrice: syp.MarketPrice,
			Quantity: syp.MaxQty,
			URL:      link,
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

	err := tcg.scrape(context.TODO())
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
