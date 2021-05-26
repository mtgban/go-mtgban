package cardtrader

import (
	"compress/gzip"
	"encoding/csv"
	"io"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	ctStockExportURL = "https://www.cardtrader.com/api/simple/v1/products/download_csv?token="
)

// Use the Simple API Token to convert your own inventory to a standard InventoryRecord
func ExportStock(blueprints map[int]*Blueprint, token string, rates ...float64) (mtgban.InventoryRecord, error) {
	resp, err := cleanhttp.DefaultClient().Get(ctStockExportURL + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	csvReader := csv.NewReader(gzipReader)
	// id,blueprint_id,bundle,category_id,description,name_en,expansion_id,game_id,graded,price_cents,price_currency,quantity,mtg_rarity,condition,mtg_language,mtg_foil,signed,altered
	_, err = csvReader.Read()
	if err != nil {
		return nil, err
	}

	inventory := mtgban.InventoryRecord{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		blueprintId, _ := strconv.Atoi(record[1])

		theCard, err := Preprocess(blueprints[blueprintId])
		if err != nil {
			continue
		}
		theCard.Foil, _ = strconv.ParseBool(record[15])
		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		currency := record[10]
		priceCents, _ := strconv.Atoi(record[9])
		price := float64(priceCents) / 100.0
		if currency == "EUR" && len(rates) > 0 && rates[0] != 0 {
			price *= rates[0]
		}

		quantity, _ := strconv.Atoi(record[11])
		conditions := record[13]
		conds, found := condMap[conditions]
		if !found {
			continue
		}

		err = inventory.AddRelaxed(cardId, &mtgban.InventoryEntry{
			Price:      price,
			Quantity:   quantity,
			Conditions: conds,
			SellerName: "mtgban",
			OriginalId: record[1],
			InstanceId: record[0],
		})
	}

	return inventory, nil
}
