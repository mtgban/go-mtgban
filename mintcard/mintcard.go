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
	logCallback mtgban.LogCallbackFunc
	partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	skusData tcgplayer.SKUMap

	backend *mtgmatcher.Backend
}

// NewScraper returns a scraper.
func NewScraper(b *mtgmatcher.Backend) *MTGMintCard {
	mint := MTGMintCard{}
	mint.backend = b
	mint.inventory = mtgban.InventoryRecord{}
	mint.buylist = mtgban.BuylistRecord{}
	return &mint
}

func (mint *MTGMintCard) printf(format string, a ...any) {
	if mint.logCallback != nil {
		mint.logCallback("[MMC] "+format, a...)
	}
}

func (mint *MTGMintCard) processEntry(sku2uuid map[int]string, card Card, condition, finish, language, edition, setCode, editionID string) {
	cond := map[string]mtgban.Condition{
		"Mint": mtgban.NM,
		"SP":   mtgban.SP,
		"Used": mtgban.MP,
	}[condition]
	if cond == "" {
		mint.printf("Unknown condition tag %s", condition)
		return
	}
	if strings.Contains(card.Name, "(HP)") {
		cond = mtgban.HP
	}
	if strings.Contains(card.Name, "(DMG)") || strings.Contains(card.Name, "(Damaged)") {
		cond = mtgban.PO
	}

	link := "https://www.mtgmintcard.com/index.php?main_page=product_info&products_id=" + card.ID
	if mint.partner != "" {
		link += "&utm_source=" + url.QueryEscape(mint.partner) + "&utm_medium=referral&utm_campaign=" + url.QueryEscape(mint.partner)
	}

	cardID, found := sku2uuid[card.TCGplayerID]
	if !found {
		theCard, err := preprocess(mint.backend, card.Name, card.Number, finish, language, edition, setCode)
		if err != nil {
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				mint.printf("%v", err)
			}
			return
		}

		cardID, err = mint.backend.Match(theCard)
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
					card, _ := mint.backend.GetUUID(probe)
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
				OriginalID: card.ID,
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
		if mint.partner != "" {
			link += "&utm_source=" + url.QueryEscape(mint.partner) + "&utm_medium=referral&utm_campaign=" + url.QueryEscape(mint.partner)
		}

		gradeMap := grading(mint.backend, cardID, buyPrice)
		for _, grade := range mtgban.DefaultGradeTags {
			price := buyPrice * gradeMap[grade]
			if price > 0 {
				out := &mtgban.BuylistEntry{
					Conditions: grade,
					BuyPrice:   price,
					PriceRatio: priceRatio,
					URL:        link,
					OriginalID: card.ID,
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
	sku2uuid := mint.buildSku2UUID()
	mint.printf("Found %d skus", len(sku2uuid))

	for edition, product := range productList {
		for language, finishes := range product.Cards {
			for finish, conditions := range finishes {
				for cond, rarities := range conditions {
					for _, cards := range rarities {
						for _, card := range cards {
							mint.processEntry(sku2uuid, card, cond, finish, language, edition, product.Abbreviation, product.EditionID)
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

// skuFinish is the finish a sku's own printing/finish fields name, kept
// beside the id MatchID resolved for it so that id can later be checked
// against what it actually landed on.
type skuFinish struct {
	id           string
	foil, etched bool
}

// buildSku2UUID turns the raw TCGplayer sku catalog into a sku id -> uuid
// map, deterministically: ranging over the catalog to fill it directly
// would let whichever uuid Go's map iteration visits last win a sku two
// different uuids both claim, changing the mapping from run to run.
func (mint *MTGMintCard) buildSku2UUID() map[int]string {
	candidates := map[int][]skuFinish{}
	for uuid, skus := range mint.skusData {
		for _, sku := range skus {
			// Skip non-English printings
			if sku.Language != "ENGLISH" {
				continue
			}

			// Convert tcg sku ids into ban ids
			foil, etched := sku.Printing == "FOIL", sku.Finish == "ETCHED"
			id, err := mint.backend.MatchID(uuid, foil, etched)
			if err != nil {
				continue
			}
			candidates[sku.SkuID] = append(candidates[sku.SkuID], skuFinish{id, foil, etched})
		}
	}

	sku2uuid := map[int]string{}
	for skuID, matches := range candidates {
		// Dedup by id first: the catalog occasionally lists one uuid's own
		// sku twice, which is not an ambiguity even if that uuid's finish
		// does not match what the sku claims.
		byID := map[string]skuFinish{}
		for _, m := range matches {
			byID[m.id] = m
		}
		if len(byID) == 1 {
			for id := range byID {
				sku2uuid[skuID] = id
			}
			continue
		}
		// More than one uuid claims this sku: keep it only if exactly one
		// candidate actually resolved to the finish the sku itself names,
		// since MatchID falls back to a mismatched finish rather than
		// fail - and, like MatchID itself, never to foil for an etched
		// request.
		resolved := map[string]bool{}
		for id, m := range byID {
			co, err := mint.backend.GetUUID(id)
			isFoil := m.foil && !m.etched
			if err != nil || co.Foil != isFoil || co.Etched != m.etched {
				continue
			}
			resolved[id] = true
		}
		if len(resolved) == 1 {
			for id := range resolved {
				sku2uuid[skuID] = id
			}
		}
	}
	return sku2uuid
}

// Inventory returns what Load collected. See mtgban.Seller.
func (mint *MTGMintCard) Inventory() mtgban.InventoryRecord {
	return mint.inventory
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (mint *MTGMintCard) Buylist() mtgban.BuylistRecord {
	return mint.buylist
}

func grading(b *mtgmatcher.Backend, cardID string, price float64) map[mtgban.Condition]float64 {
	co, err := b.GetUUID(cardID)
	if err != nil {
		return nil
	}

	if co.Foil {
		return map[mtgban.Condition]float64{
			mtgban.NM: 1, mtgban.SP: 0.75, mtgban.MP: 0.5, mtgban.HP: 0.3,
		}
	}

	switch co.SetCode {
	case "LEA", "LEB", "2ED", "3ED":
		return map[mtgban.Condition]float64{
			mtgban.NM: 1,
		}
	}

	if price >= 30.25 {
		return map[mtgban.Condition]float64{
			mtgban.NM: 1, mtgban.SP: 0.85, mtgban.MP: 0.75, mtgban.HP: 0.65,
		}
	}
	if price >= 10.25 {
		return map[mtgban.Condition]float64{
			mtgban.NM: 1, mtgban.SP: 0.80, mtgban.MP: 0.7, mtgban.HP: 0.6,
		}
	}
	if price >= 0.25 {
		return map[mtgban.Condition]float64{
			mtgban.NM: 1, mtgban.SP: 0.75, mtgban.MP: 0.6, mtgban.HP: 0.35,
		}
	}
	return map[mtgban.Condition]float64{
		mtgban.NM: 1, mtgban.SP: 0.5, mtgban.MP: 0.5,
	}
}

// Info describes this scraper. See mtgban.Scraper.
func (mint *MTGMintCard) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTG Mint Card"
	info.Shorthand = "MMC"
	info.InventoryTimestamp = &mint.inventoryDate
	info.BuylistTimestamp = &mint.buylistDate
	info.CreditMultiplier = 1.1
	info.Game = mtgban.GameMagic
	return
}
