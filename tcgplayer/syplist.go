package tcgplayer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// SYPGames are the categories Store Your Products is read for, one line per
// game. Magic is absent from SupportedGames because its singles are priced
// through scrapers of their own, but its SYP list is served the same way.
var SYPGames = map[string]int{
	mtgban.GameMagic:   tcgplayer.CategoryMagic,
	mtgban.GamePokemon: tcgplayer.CategoryPokemon,
}

// skuLanguageEnglish is the language id every English sku carries; Direct
// hosts no other.
const skuLanguageEnglish = 1

// SYPSku is what the catalog says about one sellable sku: the product it
// belongs to and the finish it is sold in. The list names skus, the datastore
// names products, and this is the step between them.
type SYPSku struct {
	ProductID int
	Finish    string
}

// SYPCatalog indexes a category's skus by id. It is built from the catalog
// dump published beside each game's datastore, which is the same file the
// datastore generators read, so the ids here are the ids stamped there.
type SYPCatalog map[int]SYPSku

// LoadSYPCatalog reads a category's catalog dump and keeps the English Near
// Mint skus of it, which is all Direct hosts. Everything else in the dump -
// and it is the larger part, five million skus for Magic - is dropped rather
// than carried through the run.
func LoadSYPCatalog(reader io.Reader) (SYPCatalog, error) {
	var dump tcgplayer.CatalogDump
	err := json.NewDecoder(reader).Decode(&dump)
	if err != nil {
		return nil, err
	}
	if len(dump.Products) == 0 {
		return nil, errors.New("empty catalog dump")
	}

	// PrintingNames is keyed by product; the printing a sku names is looked
	// up by its own id.
	printings := map[int]string{}
	for _, printing := range dump.Printings {
		printings[printing.PrintingID] = printing.Name
	}

	catalog := SYPCatalog{}
	for _, product := range dump.Products {
		// Read off the name once per product rather than once per sku: the
		// guard that meant to do that tested the result, which is empty for
		// every product not sold etched - almost all of them - so it asked
		// again for every sku of every one.
		finish := productFinish(product.Name)
		for _, sku := range product.Skus {
			if sku.LanguageID != skuLanguageEnglish || SKUConditionMap[sku.ConditionID] != "NM" {
				continue
			}
			name := printings[sku.PrintingID]
			if finish != "" {
				name = finish
			}
			catalog[sku.SKUID] = SYPSku{ProductID: product.ProductID, Finish: name}
		}
	}
	if len(catalog) == 0 {
		return nil, errors.New("catalog dump named no near mint skus")
	}
	return catalog, nil
}

// productFinish names a finish the catalog does not print. Magic's etched
// foil is no printing of TCGplayer's - it rides in the title of the product
// itself - and the printing list for the whole category names only Foil and
// Normal, so the title is the only place it is said.
func productFinish(productName string) string {
	if strings.Contains(productName, "(Foil Etched)") {
		return mtgmatcher.FinishEtched
	}
	return ""
}

// TCGSYPList reads TCGplayer's Store Your Products, the list of cards they
// can host for you and sell on Direct.
type TCGSYPList struct {
	LogCallback mtgban.LogCallbackFunc
	Affiliate   string

	// Catalog names the product and finish behind each sku the list refers
	// to. Without it there is nothing to resolve against: the list says a
	// sku id, the datastore knows product ids.
	Catalog SYPCatalog

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

// NewScraperSYP returns a SYP scraper for any game the list is read for. The
// list is served against an authorization ticket alone, so this needs no API
// credentials of its own.
func NewScraperSYP(game, auth string) (*TCGSYPList, error) {
	category, found := SYPGames[game]
	if !found {
		return nil, fmt.Errorf("unsupported SYP game %q", game)
	}

	tcg := TCGSYPList{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.auth = auth
	tcg.game = game
	tcg.category = category

	return &tcg, nil
}

// resolve names the printing a sku belongs to. The finish name is what a game
// with more than one foil needs - Pokemon tells holo from reverse holo by it -
// but a name is only answered where the card is sold in it, and the oldest
// Magic printings answer the plain flag alone. Ask by name, then by flag.
func (tcg *TCGSYPList) resolve(sku SYPSku) (string, error) {
	id := fmt.Sprint(sku.ProductID)
	cardID, err := mtgmatcher.MatchIDFinish(id, sku.Finish)
	if err == nil {
		return cardID, nil
	}
	canonical := mtgmatcher.CanonicalFinish(sku.Finish)
	if canonical == "" && sku.Finish != mtgmatcher.FinishEtched {
		// A finish only the game names, and it did not answer: there is no
		// flag that says it, so there is nothing left to ask.
		return "", err
	}
	return mtgmatcher.MatchID(id,
		canonical == mtgmatcher.FinishFoil || sku.Finish == mtgmatcher.FinishEtched,
		sku.Finish == mtgmatcher.FinishEtched)
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *TCGSYPList) Load(ctx context.Context) error {
	if tcg.Catalog == nil {
		return errors.New("catalog not loaded")
	}
	tcg.printf("Found %d near mint skus in the catalog", len(tcg.Catalog))

	sypList, err := LoadSYP(ctx, tcg.category, tcg.auth)
	if err != nil {
		return err
	}
	tcg.printf("Found syp list of %d entries", len(sypList))

	for _, syp := range sypList {
		// The list names a sku, the catalog names its product, and the
		// datastore stamps that product id on the printing it belongs to.
		sku, found := tcg.Catalog[syp.SkuID]
		if !found {
			continue
		}

		cardID, err := tcg.resolve(sku)
		if err != nil {
			continue
		}

		printing := "Normal"
		if sku.Finish != "" && !strings.EqualFold(sku.Finish, "Normal") {
			printing = "Foil"
		}
		entry := mtgban.BuylistEntry{
			BuyPrice:   syp.MarketPrice,
			Quantity:   syp.MaxQty,
			URL:        GenerateProductURL(sku.ProductID, printing, tcg.Affiliate, "", "English", true),
			OriginalID: fmt.Sprint(sku.ProductID),
			InstanceID: fmt.Sprint(syp.SkuID),
		}
		err = tcg.buylist.Add(cardID, &entry)
		if err != nil {
			tcg.printf("%s", err.Error())
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
