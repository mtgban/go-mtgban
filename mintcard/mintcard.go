// Package mintcard scrapes MTG Mint Card.
package mintcard

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

// MTGMintCard prices MTG Mint Card's singles, both what they sell and what
// they buy.
type MTGMintCard struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	SKUsData tcgplayer.SKUMap
}

// NewScraper returns a scraper.
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

func (mint *MTGMintCard) processEntry(sku2uuid map[int]string, card Card, condition, finish, language, edition, setCode, editionID string) {
	cond := map[string]string{
		"Mint": "NM",
		"SP":   "SP",
		"Used": "MP",
	}[condition]
	if cond == "" {
		mint.printf("Unknown condition tag %s", condition)
		return
	}
	if strings.Contains(card.Name, "(HP)") {
		cond = "HP"
	}
	if strings.Contains(card.Name, "(DMG)") || strings.Contains(card.Name, "(Damaged)") {
		cond = "PO"
	}

	link := "https://www.mtgmintcard.com/index.php?main_page=product_info&products_id=" + card.ID
	if mint.Partner != "" {
		link += "&utm_source=" + url.QueryEscape(mint.Partner) + "&utm_medium=referral&utm_campaign=" + url.QueryEscape(mint.Partner)
	}

	cardID, found := sku2uuid[card.TCGplayerID]
	if !found {
		theCard, err := preprocess(card.Name, card.Number, finish, language, edition, setCode)
		if err != nil {
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				mint.printf("%v", err)
			}
			return
		}

		cardID, err = mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			// Skip errors on tokens
			if strings.Contains(card.Name, "Token") {
				return
			}
			mint.printf("%v", err)
			mint.printf("%q", theCard)
			mint.printf("%s|%s|%s|%s|%s|%s|%s", card.Name, card.Number, finish, language, edition, setCode, card.TCGplayerID)
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
	}

	var err error
	var sellPrice float64
	if card.Price != "" && card.Quantity > 0 {
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
			err = mint.inventory.Add(cardID, out)
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

		link := "https://www.mtgmintcard.com/buylist?action=advanced_search&ed=" + editionID + "&mo_1=1&mo_2=1&card_name=" + url.QueryEscape(card.Name)
		if mint.Partner != "" {
			link += "&utm_source=" + url.QueryEscape(mint.Partner) + "&utm_medium=referral&utm_campaign=" + url.QueryEscape(mint.Partner)
		}

		gradeMap := grading(cardID, buyPrice)
		for _, grade := range mtgban.DefaultGradeTags {
			price := buyPrice * gradeMap[grade]
			if price > 0 {
				out := &mtgban.BuylistEntry{
					Conditions: grade,
					BuyPrice:   price,
					PriceRatio: priceRatio,
					URL:        link,
					OriginalId: card.ID,
				}
				err = mint.buylist.Add(cardID, out)
				if err != nil {
					mint.printf("%v", err)
				}
			}
		}
	}
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mint *MTGMintCard) Load(ctx context.Context) error {
	mint.printf("Loading MTGMintCard data")
	mintClient, err := NewMintClient(ctx)
	if err != nil {
		return err
	}
	productList, err := mintClient.GetProductList(ctx)
	if err != nil {
		return err
	}
	mint.printf("Found %d editions", len(productList))

	mint.printf("Converting TCGSKU into reusable format")
	sku2uuid := map[int]string{}
	for uuid, skus := range mint.SKUsData {
		for _, sku := range skus {
			// Skip non-English printings
			if sku.Language != "ENGLISH" {
				continue
			}

			// Convert tcg sku ids into ban ids
			id, err := mtgmatcher.MatchId(uuid, sku.Printing == "FOIL", sku.Finish == "ETCHED")
			if err != nil {
				continue
			}
			sku2uuid[sku.SkuId] = id
		}
	}
	mint.printf("Found %d skus", len(sku2uuid))

	for edition, product := range productList {
		for language, finishes := range product.Cards {
			for finish, conditions := range finishes {
				for cond, rarities := range conditions {
					for _, cards := range rarities {
						for _, card := range cards {
							mint.processEntry(sku2uuid, card, cond, finish, language, edition, product.Abbreviation, product.EditionId)
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

// Inventory returns what Load collected. See mtgban.Seller.
func (mint *MTGMintCard) Inventory() mtgban.InventoryRecord {
	return mint.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (mint *MTGMintCard) Buylist() mtgban.BuylistRecord {
	return mint.buylist
}

func grading(cardID string, price float64) map[string]float64 {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return nil
	}

	if co.Foil {
		return map[string]float64{
			"NM": 1, "SP": 0.75, "MP": 0.5, "HP": 0.3,
		}
	}

	switch co.SetCode {
	case "LEA", "LEB", "2ED", "3ED":
		return map[string]float64{
			"NM": 1,
		}
	}

	if price >= 30.25 {
		return map[string]float64{
			"NM": 1, "SP": 0.85, "MP": 0.75, "HP": 0.65,
		}
	}
	if price >= 10.25 {
		return map[string]float64{
			"NM": 1, "SP": 0.80, "MP": 0.7, "HP": 0.6,
		}
	}
	if price >= 0.25 {
		return map[string]float64{
			"NM": 1, "SP": 0.75, "MP": 0.6, "HP": 0.35,
		}
	}
	return map[string]float64{
		"NM": 1, "SP": 0.5, "MP": 0.5,
	}
}

// Info describes this scraper. See mtgban.Scraper.
func (mint *MTGMintCard) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTG Mint Card"
	info.Shorthand = "MMC"
	info.InventoryTimestamp = &mint.inventoryDate
	info.BuylistTimestamp = &mint.buylistDate
	info.CreditMultiplier = 1.1
	return
}
