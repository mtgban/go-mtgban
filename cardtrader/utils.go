package cardtrader

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// priceToUSD converts a CardTrader price to dollars, through a rate table
// fetched once per run by mtgban.GetExchangeRates. A currency the feed does
// not quote is reported rather than being silently multiplied by the wrong
// rate.
//
// The whole table is held rather than a chosen few rates. The quoting currency
// belongs to the authenticating app and not to the listing - the same
// expansion comes back in dollars to one token and euros to another, and an
// app's setting can change between runs - so there is no list of currencies to
// be right about in advance, and being wrong about it costs an expansion at a
// time: one request fetches a whole expansion and the response quotes all of
// it in the one currency.
func priceToUSD(cents int, currency string, rates map[string]float64) (float64, error) {
	price := float64(cents) / 100
	if currency == "USD" {
		return price, nil
	}
	rate, found := rates[strings.ToLower(currency)]
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

	rates, err := mtgban.GetExchangeRates(ctx)
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

// listingPrice reads whichever of the three price fields a listing carries.
// The endpoints disagree on which one they fill - an export quotes
// price_cents, the marketplace quotes price, and an order quotes buyer_price -
// and each carries the currency it was quoted in, so the two travel together.
func listingPrice(product Product) (int, string) {
	if product.PriceCents != 0 {
		return product.PriceCents, product.PriceCurrency
	}
	if product.Price.Cents != 0 {
		return product.Price.Cents, product.Price.Currency
	}
	return product.BuyerPrice.Cents, product.BuyerPrice.Currency
}

// ConvertProducts turns listings into an InventoryRecord, resolving each
// through the blueprint it was sold against and pricing it in dollars by the
// rate table that mtgban.GetExchangeRates fills. A listing quoted in a
// currency the table does not cover is left out rather than priced by the
// wrong rate.
func ConvertProducts(blueprints map[int]*Blueprint, products []Product, rates map[string]float64) mtgban.InventoryRecord {
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

		cents, currency := listingPrice(product)
		price, err := priceToUSD(cents, currency, rates)
		if err != nil {
			continue
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
		// Every treatment the listing names is a foil, so the named finish
		// answers the flag too. Yu-Gi-Oh deliberately has no arm: the rarity
		// is its treatment, so every listing reads nonfoil through the
		// default below.
		return gameFinish(gameID, product) != ""
	}
	return false
}

// gameFinish names the finish a listing prices, for the games whose
// properties name one rather than flagging it. Flesh and Blood is the only
// one so far: its treatment is a string, and the name reaches the printing's
// own Cold Foil sibling, which the boolean cannot - it has one bit for a
// game selling three treatments. The plain values name no treatment and are
// left to the flag; the stringly-false is what old listings carry.
func gameFinish(gameID int, product Product) string {
	if gameID != GameIdFleshAndBlood {
		return ""
	}
	switch product.Properties.FabFoilNew {
	case "", "Regular", "false":
		return ""
	}
	return product.Properties.FabFoilNew
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
