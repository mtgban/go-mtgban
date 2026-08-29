package tcgplayer

import (
	"context"
	"errors"
	"fmt"

	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// TCGSYPList reads TCGplayer's Store Your Products, the list of cards they
// can host for you and sell on Direct.
type TCGSYPList struct {
	LogCallback mtgban.LogCallbackFunc
	Affiliate   string
	SKUsData    SKUMap

	game        string
	category    int
	auth        string
	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

func (tcg *TCGSYPList) printf(format string, a ...any) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGSYPList] "+format, a...)
	}
}

// NewScraperSYP returns a SYP scraper using the given authorization token for
// any supported game.
func NewScraperSYP(game, auth string) (*TCGSYPList, error) {
	tcg := TCGSYPList{}

	switch game {
	case mtgban.GameMagic:
		tcg.category = tcgplayer.CategoryMagic
	default:
		return nil, errors.New("unsupported SYP game")
	}

	tcg.buylist = mtgban.BuylistRecord{}
	tcg.auth = auth
	tcg.game = game

	return &tcg, nil
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *TCGSYPList) Load(ctx context.Context) error {
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
			sku2product[sku.SkuID] = sku
		}
	}

	sypList, err := LoadSYP(ctx, tcg.category, tcg.auth)
	if err != nil {
		return err
	}
	tcg.printf("Found syp list of %d entries", len(sypList))

	for _, syp := range sypList {
		sku, found := sku2product[syp.SkuID]
		if !found {
			continue
		}

		isFoil := sku.Printing == "FOIL"
		isEtched := sku.Finish == "FOIL ETCHED"
		cardID, err := mtgmatcher.MatchID(fmt.Sprint(sku.ProductID), isFoil, isEtched)
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
		link := GenerateProductURL(sku.ProductID, printing, tcg.Affiliate, "", "English", true)

		entry := mtgban.BuylistEntry{
			BuyPrice: syp.MarketPrice,
			Quantity: syp.MaxQty,
			URL:      link,
		}

		err = tcg.buylist.Add(cardID, &entry)
		if err != nil {
			tcg.printf("%s", err.Error())
			continue
		}
	}

	tcg.buylistDate = time.Now()

	return nil
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (tcg *TCGSYPList) Buylist() mtgban.BuylistRecord {
	return tcg.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *TCGSYPList) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCGplayer SYP"
	info.Shorthand = "SYP"
	info.BuylistTimestamp = &tcg.buylistDate
	info.MetadataOnly = true
	info.QuantityPriority = true
	info.Game = tcg.game
	return
}
