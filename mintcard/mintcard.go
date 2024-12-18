package mintcard

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type MTGMintCard struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *MTGMintCard {
	mint := MTGMintCard{}
	mint.inventory = mtgban.InventoryRecord{}
	mint.buylist = mtgban.BuylistRecord{}
	return &mint
}

func (mint *MTGMintCard) printf(format string, a ...interface{}) {
	if mint.LogCallback != nil {
		mint.LogCallback("[MMC] "+format, a...)
	}
}

func (mint *MTGMintCard) processEntry(card Card, condition, finish, langauge, edition, setCode string) {
	cond := map[string]string{
		"Mint": "NM",
		"SP":   "SP",
		"Used": "MP",
	}[condition]
	if cond == "" {
		mint.printf("Unknown condition tag", condition)
		return
	}
	if strings.Contains(card.Name, "(HP)") {
		cond = "HP"
	}
	if strings.Contains(card.Name, "(DMG)") || strings.Contains(card.Name, "(Damaged)") {
		cond = "PO"
	}

	link := "https://www.mtgmintcard.com/index.php?main_page=product_info&products_id=" + card.ID

	theCard, err := preprocess(card.Name, card.Number, finish, langauge, edition, setCode)
	if err != nil {
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			mint.printf("%v", err)
		}
		return
	}

	cardId, err := mtgmatcher.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return
	} else if err != nil {
		// Skip errors on tokens
		if strings.Contains(card.Name, "Token") {
			return
		}
		mint.printf("%v", err)
		mint.printf("%q", theCard)
		mint.printf("%s|%s|%s|%s|%s|%s", card.Name, card.Number, finish, langauge, edition, setCode)
		mint.printf("%s", link)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := mtgmatcher.GetUUID(probe)
				mint.printf("- %s", card)
			}
		}
		return
	}

	var sellPrice float64
	if card.Price != "" {
		sellPrice, err = strconv.ParseFloat(card.Price, 64)
		if err != nil {
			mint.printf("%v", err)
		}

		if sellPrice > 0 {
			out := &mtgban.InventoryEntry{
				Conditions: cond,
				Price:      sellPrice,
				Quantity:   card.Quantity,
				URL:        link,
				OriginalId: card.ID,
			}
			err = mint.inventory.Add(cardId, out)
			if err != nil {
				mint.printf("%v", err)
			}
		}
	}

	if card.BuyPrice != "" {
		buyPrice, err := strconv.ParseFloat(card.BuyPrice, 64)
		if err != nil {
			mint.printf("%v", err)
		}

		var priceRatio float64
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		link := "https://www.mtgmintcard.com/buylist?action=advanced_search&mo_1=1&mo_2=1&card_name=" + url.QueryEscape(card.Name)
		if buyPrice > 0 {
			out := &mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   buyPrice,
				PriceRatio: priceRatio,
				URL:        link,
				OriginalId: card.ID,
			}
			err = mint.buylist.Add(cardId, out)
			if err != nil {
				mint.printf("%v", err)
			}
		}
	}
}

func (mint *MTGMintCard) scrape() error {
	mintClient, err := NewMintClient()
	if err != nil {
		return err
	}
	productList, err := mintClient.GetProductList()
	if err != nil {
		return err
	}

	mint.printf("Found %d editions", len(productList))

	for edition, product := range productList {
		for langauge, finishes := range product.Cards {
			for finish, conditions := range finishes {
				for cond, rarities := range conditions {
					for _, cards := range rarities {
						for _, card := range cards {
							mint.processEntry(card, cond, finish, langauge, edition, product.Abbreviation)
						}
					}
				}
			}
		}
	}

	mint.inventoryDate = time.Now()
	mint.buylistDate = time.Now()

	return nil
}

func (mint *MTGMintCard) Inventory() (mtgban.InventoryRecord, error) {
	if len(mint.inventory) > 0 {
		return mint.inventory, nil
	}

	err := mint.scrape()
	if err != nil {
		return nil, err
	}

	return mint.inventory, nil

}

func (mint *MTGMintCard) Buylist() (mtgban.BuylistRecord, error) {
	if len(mint.buylist) > 0 {
		return mint.buylist, nil
	}

	err := mint.scrape()
	if err != nil {
		return nil, err
	}

	return mint.buylist, nil
}

func (mint *MTGMintCard) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTG Mint Card"
	info.Shorthand = "MMC"
	info.InventoryTimestamp = &mint.inventoryDate
	info.BuylistTimestamp = &mint.buylistDate
	info.CreditMultiplier = 1.1
	return
}