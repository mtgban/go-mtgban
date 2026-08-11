package cardtrader

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// nonUSDCurrencies are the currencies listings are quoted in besides dollars.
// The quoting currency follows the expansion rather than the seller - the same
// seller lists Origins in dollars and Vendetta in pounds - so a missing rate
// costs the whole expansion, not a handful of listings.
var nonUSDCurrencies = []string{"EUR", "GBP"}

// exchangeRates retrieves one rate per non-dollar currency, up front, so that
// converting a listing needs no request of its own.
func exchangeRates(ctx context.Context) (map[string]float64, error) {
	rates := make(map[string]float64, len(nonUSDCurrencies))
	for _, currency := range nonUSDCurrencies {
		rate, err := mtgban.GetExchangeRate(ctx, currency)
		if err != nil {
			return nil, err
		}
		rates[currency] = rate
	}
	return rates, nil
}

// priceToUSD converts a CardTrader price to dollars. A currency with no rate
// is reported rather than being silently multiplied by the wrong one.
func priceToUSD(cents int, currency string, rates map[string]float64) (float64, error) {
	price := float64(cents) / 100
	if currency == "USD" {
		return price, nil
	}
	rate, found := rates[currency]
	if !found {
		return 0, fmt.Errorf("unsupported currency %q", currency)
	}
	return price * rate, nil
}

// Use the Simple API Token to convert your own inventory to a standard InventoryRecord
func (ct *CTAuthClient) ExportStock(ctx context.Context, blueprints map[int]*Blueprint) (mtgban.InventoryRecord, error) {
	products, err := ct.ProductsExport(ctx)
	if err != nil {
		return nil, err
	}

	rates, err := exchangeRates(ctx)
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

		cardID, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		price, err := priceToUSD(product.PriceCents, product.PriceCurrency, rates)
		if err != nil {
			continue
		}

		quantity := product.Quantity

		condition, found := condMap[product.Properties.Condition]
		if !found {
			continue
		}

		inventory.AddRelaxed(cardID, &mtgban.InventoryEntry{
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

		cardID, err := mtgmatcher.Match(theCard)
		if err != nil {
			continue
		}

		priceCents := product.PriceCents
		currency := product.PriceCurrency
		if priceCents == 0 {
			priceCents = product.Price.Cents
			currency = product.Price.Currency
		}
		if priceCents == 0 {
			priceCents = product.BuyerPrice.Cents
			currency = product.BuyerPrice.Currency
		}

		price := float64(priceCents) / 100.0
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

		inventory.AddRelaxed(cardID, &mtgban.InventoryEntry{
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

// BlueprintsForGame fetches every blueprint of a game, one expansion at a
// time, skipping the expansions that fail to fetch rather than aborting
// the lot. A non-empty targetEdition narrows the fetch to that expansion
// by name or code; logf, when given, reports the skips. The expansions
// are returned too, since callers key edition names off them.
func BlueprintsForGame(ctx context.Context, client *CTAuthClient, gameID int, targetEdition string, logf func(string, ...interface{})) ([]Blueprint, []Expansion, error) {
	expansions, err := client.Expansions(ctx)
	if err != nil {
		return nil, nil, err
	}

	var blueprints []Blueprint
	for _, exp := range expansions {
		if exp.GameId != gameID {
			continue
		}
		// The OCG expansions are whole separate Japanese catalogs (bach-jp
		// beside bach) whose prices must not land on the TCG printings the
		// datastore carries.
		if gameID == GameIdYuGiOh && (strings.HasSuffix(exp.Code, "-jp") || strings.Contains(exp.Name, "OCG")) {
			continue
		}
		if targetEdition != "" && exp.Name != targetEdition && exp.Code != strings.ToLower(targetEdition) {
			continue
		}
		bp, err := client.Blueprints(ctx, exp.Id)
		if err != nil {
			if logf != nil {
				logf("skipping %d %s due to %s", exp.Id, exp.Name, err.Error())
			}
			continue
		}
		blueprints = append(blueprints, bp...)
	}
	return blueprints, expansions, nil
}

// gameLanguage and gameFoil read a listing's language and foil flag for the
// given game: cardtrader keys the properties per game, the Magic fields
// decoding empty for every other one.
func gameLanguage(gameID int, product Product) string {
	switch gameID {
	case GameIdMagic:
		return product.Properties.MTGLanguage
	case GameIdLorcana:
		return product.Properties.LorcanaLanguage
	case GameIdRiftbound:
		return product.Properties.RiftboundLanguage
	case GameIdOnePiece:
		return product.Properties.OnePieceLanguage
	case GameIdYuGiOh:
		return product.Properties.YuGiOhLanguage
	case GameIdFleshAndBlood:
		return product.Properties.FabLanguage
	}
	return ""
}

// collectorNumberRe matches the collector number shapes One Piece's matcher
// reads out of a variation before it falls back to the first digit-leading
// word. A number of this shape wins that read whatever else the variation
// carries.
var collectorNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*$`)

// gameVariation spells the printing a blueprint names. One Piece and
// Riftbound both file their event and parallel printings under the base
// card's collector number, so the number alone aliases them; the blueprint's
// Version carries the very wording the datastore labels them with ("OP16
// Release Event", "Summoner Skirmish | Champion"), which is what tells them
// apart.
//
// The Version rides behind a number, and only then: its wording is full of
// years and volume numbers, and with nothing ahead of them one of those would
// answer as the collector number and select nothing at all.
func gameVariation(gameID int, bp *Blueprint, number string) string {
	if bp.Version == "" || number == "" {
		return number
	}
	switch gameID {
	case GameIdOnePiece:
		// One Piece numbers come in shapes the matcher cannot read - "P-L",
		// "OP07-047P2", the alpha-suffixed pre-errata codes - and behind one
		// of those the version's own digits answer in their place.
		if !collectorNumberRe.MatchString(number) {
			return number
		}
	case GameIdRiftbound:
		// Riftbound numbers are bare ("202", "058c", "T05"), always readable.
	default:
		return number
	}
	return number + " " + bp.Version
}

func gameFoil(gameID int, product Product) bool {
	switch gameID {
	case GameIdMagic:
		return product.Properties.MTGFoil
	case GameIdLorcana:
		return product.Properties.LorcanaFoil
	case GameIdRiftbound:
		return product.Properties.RiftboundFoil
	case GameIdOnePiece:
		return product.Properties.OnePieceFoil
	case GameIdFleshAndBlood:
		// Anything beyond the plain treatment is a foil; the empty and
		// stringly-false values old listings carry are plain too. Yu-Gi-Oh
		// deliberately has no arm: the rarity is the finish, so every
		// listing reads nonfoil through the default below.
		switch product.Properties.FabFoilNew {
		case "", "Regular", "false":
			return false
		}
		return true
	}
	return false
}

// coldFoilID hops a resolved Flesh and Blood id onto its product's Cold
// Foil entry, preferring the plainest print run exactly as the loader's
// foil defaults do; a product without a cold entry keeps the resolved id.
func coldFoilID(cardID string) string {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return cardID
	}
	for _, key := range []string{"coldfoil", "unlimitededitioncoldfoil", "1steditioncoldfoil"} {
		id, found := co.FoilUUIDs[key]
		if found {
			return id
		}
	}
	return cardID
}

func FormatBlueprints(blueprints []Blueprint, inExpansions []Expansion, sealed bool) (map[int]*Blueprint, map[int]string) {
	// Create a map to be able to retrieve edition name in the blueprint
	formatted := map[int]*Blueprint{}
	expansions := map[int]string{}
	for i := range blueprints {
		// Sealed is selected by exclusion, so a category cardtrader adds
		// later lands on the sealed side rather than silently vanishing;
		// accessories arrive with it and are dropped downstream by the
		// product-map resolution, which only names real sealed products.
		singles := false
		switch blueprints[i].CategoryId {
		case CategoryMagicSingles, CategoryMagicTokens, CategoryMagicOversized,
			CategoryLorcanaSingles, CategoryLorcanaOversized,
			CategoryRiftboundSingles, CategoryRiftboundOversized,
			CategoryOnePieceSingles,
			CategoryYuGiOhSingles,
			CategoryFleshAndBloodSingles:
			singles = true
		}
		if singles == sealed {
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
