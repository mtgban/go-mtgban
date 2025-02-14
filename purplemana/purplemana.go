package purplemana

import (
	"fmt"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	hotlistURL = "https://www.purplemana.com/hotlist"

	GameMagic   = "MTG"
	GameLorcana = "LRC"
)

type Purplemana struct {
	LogCallback mtgban.LogCallbackFunc

	game string

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

func NewScraper(game string) *Purplemana {
	pm := Purplemana{}
	pm.buylist = mtgban.BuylistRecord{}
	pm.game = game
	return &pm
}

func (pm *Purplemana) printf(format string, a ...interface{}) {
	if pm.LogCallback != nil {
		pm.LogCallback("[PM] "+format, a...)
	}
}

func (pm *Purplemana) scrape() error {
	products, err := GetHotList()
	if err != nil {
		return err
	}

	for _, product := range products {
		if product.SellerPayout == 0 {
			continue
		}

		tcgID := fmt.Sprint(product.CatalogProducts.TcgplayerID)
		uuid := mtgmatcher.Tcg2UUID(tcgID)
		if uuid == "" {
			pm.printf("tcg id %s not found", tcgID)
			continue
		}

		foil := product.CatalogProducts.Variant == "Foil"

		cardId, err := mtgmatcher.MatchId(uuid, foil)
		if err != nil {
			pm.printf("uuid %s (from tcg id %s) as %s not found", uuid, tcgID, product.CatalogProducts.Variant)
			continue
		}

		var cond string
		switch product.Condition {
		case "Near Mint":
			cond = "NM"
		case "Lightly Played":
			cond = "SP"
		case "Moderately Played":
			cond = "MP"
		case "Heavily Played":
			cond = "HP"
		default:
			pm.printf("unknown condition: %s", product.Condition)
			continue
		}

		err = pm.buylist.AddRelaxed(cardId, &mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   product.SellerPayout,
			URL:        hotlistURL,
			OriginalId: fmt.Sprint(product.CatalogProducts.ID),
		})
		if err != nil {
			pm.printf("%s", err.Error())
		}
	}

	pm.buylistDate = time.Now()

	return nil
}

func (pm *Purplemana) Buylist() (mtgban.BuylistRecord, error) {
	if len(pm.buylist) > 0 {
		return pm.buylist, nil
	}

	err := pm.scrape()
	if err != nil {
		return nil, err
	}

	return pm.buylist, nil
}

func (pm *Purplemana) Info() (info mtgban.ScraperInfo) {
	info.Name = "Purplemana"
	info.Shorthand = "PM"
	info.BuylistTimestamp = &pm.buylistDate
	switch pm.game {
	case GameMagic:
		info.Game = mtgban.GameMagic
	case GameLorcana:
		info.Game = mtgban.GameLorcana
	}
	return
}
