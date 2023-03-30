package cardtrader

import (
	"fmt"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Use the Simple API Token to convert your own inventory to a standard InventoryRecord
func (ct *CTAuthClient) ExportStock(blueprints map[int]*Blueprint) (mtgban.InventoryRecord, error) {
	products, err := ct.ProductsExport()
	if err != nil {
		return nil, err
	}

	currencyRate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}

	inventory := mtgban.InventoryRecord{}
	for _, product := range products {
		blueprint, found := blueprints[product.BlueprintId]
		if !found {
			continue
		}

		theCard, err := Preprocess(blueprint)
		if err != nil {
			continue
		}
		theCard.Foil = product.Properties.Foil

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		price := currencyRate * float64(product.PriceCents) / 100.0

		quantity := product.Quantity

		condition, found := condMap[product.Properties.Condition]
		if !found {
			continue
		}

		inventory.AddRelaxed(cardId, &mtgban.InventoryEntry{
			Price:      price,
			Quantity:   quantity,
			Conditions: condition,
			SellerName: "mtgban",
			OriginalId: fmt.Sprint(product.BlueprintId),
			InstanceId: fmt.Sprint(product.Id),
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

		inventory.AddRelaxed(cardId, &mtgban.InventoryEntry{
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
