package tcgplayer

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

type TCGSYPList struct {
	LogCallback   mtgban.LogCallbackFunc
	inventoryDate time.Time

	inventory mtgban.InventoryRecord

	client *http.Client
}

const (
	sypExportCSVURL = "https://bancsvs.s3.us-east-2.amazonaws.com/SYP_Export.csv"
)

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
	sku2product := map[string]mtgjson.TCGSku{}
	for _, skus := range uuid2skusMap {
		for _, sku := range skus {
			sku2product[fmt.Sprint(sku.SkuId)] = sku
		}
	}

	resp, err := cleanhttp.DefaultClient().Get(sypExportCSVURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.LazyQuotes = true

	// Header
	// TCGplayer Id,Category,Product Name,Number,Rarity,Set,Condition,Market Price,Max QTY
	record, err := r.Read()
	if err != nil {
		return err
	}
	if len(record) < 8 {
		tcg.printf("%q", record)
		return errors.New("unexpected csv format")
	}

	for {
		record, err = r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		sku, found := sku2product[record[0]]
		if !found {
			continue
		}

		isFoil := sku.Printing == "FOIL"
		isEtched := sku.Finish == "FOIL ETCHED"
		cardId, err := mtgmatcher.MatchId(fmt.Sprint(sku.ProductId), isFoil, isEtched)
		if err != nil {
			// Skip errors for tokens and promos
			if record[4] != "T" && record[4] != "P" {
				tcg.printf("%d not found [%s %s]", sku.ProductId, record[2], record[5])
			}
			continue
		}

		qty, _ := strconv.Atoi(record[8])
		price, _ := strconv.ParseFloat(record[7], 64)

		cond, found := skuConditions[sku.Condition]
		if !found {
			continue
		}

		entry := mtgban.InventoryEntry{
			Conditions: cond,
			Price:      price,
			Quantity:   qty,
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
	info.InventoryTimestamp = tcg.inventoryDate
	info.MetadataOnly = true
	return
}
