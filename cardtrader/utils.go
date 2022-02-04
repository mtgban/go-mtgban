package cardtrader

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
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
	// id,blueprint_id,bundle,category_id,description,name_en,expansion_id,expansion_name,expansion_code,game_id,graded,price_cents,price_currency,quantity,mtg_rarity,condition,mtg_language,mtg_foil,signed,altered
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
		blueprintId, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, err
		}

		theCard, err := Preprocess(blueprints[blueprintId])
		if err != nil {
			continue
		}

		theCard.Foil, err = strconv.ParseBool(record[17])
		if err != nil {
			return nil, err
		}

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		priceCents, err := strconv.Atoi(record[11])
		if err != nil {
			return nil, err
		}
		price := float64(priceCents) / 100.0

		currency := record[12]
		if currency == "EUR" && len(rates) > 0 && rates[0] != 0 {
			price *= rates[0]
		}

		quantity, err := strconv.Atoi(record[13])
		if err != nil {
			return nil, err
		}

		conditions := record[15]
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

func ConvertProducts(blueprints map[int]*Blueprint, products []Product, rates ...float64) mtgban.InventoryRecord {
	inventory := mtgban.InventoryRecord{}
	for _, product := range products {
		theCard, err := Preprocess(blueprints[product.BlueprintId])
		if err != nil {
			continue
		}
		theCard.Foil = product.Properties.Foil

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		price := float64(product.PriceCents) / 100.0

		currency := product.PriceCurrency
		if currency == "EUR" && len(rates) > 0 && rates[0] != 0 {
			price *= rates[0]
		}

		quantity := product.Quantity

		conds, found := condMap[product.Properties.Condition]
		if !found {
			continue
		}

		var customFields map[string]string
		if product.Description != "" || product.UserDataField != "" {
			customFields = map[string]string{}
			if product.Description != "" {
				customFields["description"] = product.Description
			}
			if product.UserDataField != "" {
				customFields["user_data_field"] = product.UserDataField
			}
		}

		err = inventory.AddRelaxed(cardId, &mtgban.InventoryEntry{
			Price:        price,
			Quantity:     quantity,
			Conditions:   conds,
			SellerName:   "mtgban",
			OriginalId:   fmt.Sprint(product.BlueprintId),
			InstanceId:   fmt.Sprint(product.Id),
			CustomFields: customFields,
		})
	}

	return inventory
}
