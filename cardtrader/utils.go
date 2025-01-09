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
		theCard.Foil = product.Properties.MTGFoil

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
		bp, found := blueprints[product.BlueprintId]
		if !found {
			continue
		}
		theCard, err := Preprocess(bp)
		if err != nil {
			continue
		}
		theCard.Foil = product.Properties.MTGFoil

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
		if product.Description != "" || product.UserDataField != "" || product.Tag != "" {
			customFields = map[string]string{}
			if product.Description != "" {
				customFields["description"] = product.Description
			}
			if product.UserDataField != "" {
				customFields["user_data_field"] = product.UserDataField
			}
			if product.Tag != "" {
				customFields["tag"] = product.Tag
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

func FormatBlueprints(blueprints []Blueprint, inExpansions []Expansion, sealed bool) (map[int]*Blueprint, map[int]string) {
	// Create a map to be able to retrieve edition name in the blueprint
	formatted := map[int]*Blueprint{}
	expansions := map[int]string{}
	for i := range blueprints {
		switch blueprints[i].CategoryId {
		case CategoryMagicSingles, CategoryMagicTokens, CategoryMagicOversized,
			CategoryLorcanaSingles, CategoryLorcanaOversized:
			if sealed {
				continue
			}
		case CategoryMagicBoosterBoxes, CategoryMagicBoosters, CategoryMagicStarterDecks,
			CategoryMagicBoxDisplays, CategoryMagicBoxedSet, CategoryMagicPreconstructedDecks,
			CategoryMagicBundles, CategoryMagicTournamentPrereleasePacks:
			if !sealed {
				continue
			}
		default:
			continue
		}

		// Keep track of blueprints as they are more accurate that the
		// information found in product
		formatted[blueprints[i].Id] = &blueprints[i]

		// Load expansions array
		_, found := expansions[blueprints[i].ExpansionId]
		if !found {
			for j := range inExpansions {
				if inExpansions[j].Id == blueprints[i].ExpansionId {
					expansions[blueprints[i].ExpansionId] = inExpansions[j].Name
				}
			}
		}

		// The name is missing from the blueprints endpoint, fill it with data
		// retrieved from the expansions endpoint
		formatted[blueprints[i].Id].Expansion.Name = expansions[blueprints[i].ExpansionId]

		// Move the blueprint properties from the custom structure from blueprints
		// to the place as expected by Preprocess()
		formatted[blueprints[i].Id].Properties = formatted[blueprints[i].Id].FixedProperties
	}

	return formatted, expansions
}
