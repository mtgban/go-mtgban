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

// ExportStock returns your own listings as an InventoryRecord, using the
// Simple API token rather than the full one.
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
		blueprint, found := blueprints[product.BlueprintID]
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
			OriginalID: fmt.Sprint(product.BlueprintID),
			InstanceID: fmt.Sprint(product.ID),
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
		bp, found := blueprints[product.BlueprintID]
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
			OriginalID:   fmt.Sprint(product.BlueprintID),
			InstanceID:   fmt.Sprint(product.ID),
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
func BlueprintsForGame(ctx context.Context, client *CTAuthClient, gameID int, targetEdition string, logf func(string, ...any)) ([]Blueprint, []Expansion, error) {
	expansions, err := client.Expansions(ctx)
	if err != nil {
		return nil, nil, err
	}

	var blueprints []Blueprint
	for _, exp := range expansions {
		if exp.GameID != gameID {
			continue
		}
		// The OCG expansions are whole separate Japanese catalogs (bach-jp
		// beside bach) whose prices must not land on the TCG printings the
		// datastore carries.
		if gameID == GameYuGiOh && (strings.HasSuffix(exp.Code, "-jp") || strings.Contains(exp.Name, "OCG")) {
			continue
		}
		if targetEdition != "" && exp.Name != targetEdition && exp.Code != strings.ToLower(targetEdition) {
			continue
		}
		bp, err := client.Blueprints(ctx, exp.ID)
		if err != nil {
			if logf != nil {
				logf("skipping %d %s due to %s", exp.ID, exp.Name, err.Error())
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
	case GameMagic:
		return product.Properties.MTGLanguage
	case GameLorcana:
		return product.Properties.LorcanaLanguage
	case GameRiftbound:
		return product.Properties.RiftboundLanguage
	case GameOnePiece:
		return product.Properties.OnePieceLanguage
	case GameYuGiOh:
		return product.Properties.YuGiOhLanguage
	case GameFleshAndBlood:
		return product.Properties.FabLanguage
	case GamePokemon:
		return product.Properties.PokemonLanguage
	}
	return ""
}

// collectorNumberRe matches the collector number shapes One Piece's matcher
// reads out of a variation before it falls back to the first digit-leading
// word. A number of this shape wins that read whatever else the variation
// carries.
var collectorNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*$`)

// gameVariation spells the printing a blueprint names. One Piece, Riftbound
// and Yu-Gi-Oh all file several printings under one collector number, so the
// number alone aliases them; the blueprint's Version carries the very wording
// the datastore labels them with ("OP16 Release Event", "Summoner Skirmish |
// Champion", and for Yu-Gi-Oh the rarity itself), which is what tells them
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
	case GameOnePiece:
		// One Piece numbers come in shapes the matcher cannot read - "P-L",
		// "OP07-047P2", the alpha-suffixed pre-errata codes - and behind one
		// of those the version's own digits answer in their place.
		if !collectorNumberRe.MatchString(number) {
			return number
		}
	case GameRiftbound:
		// Riftbound numbers are bare ("202", "058c", "T05"), always readable.
	case GameYuGiOh:
		// Yu-Gi-Oh's Version is the rarity, which is the game's whole
		// identity beside the number, and the matcher narrows on nothing
		// else. The one wording to leave behind is the literal "Token": it
		// names the game's token products, and the shared token guard reads
		// that word out of the variation and refuses the listing outright.
		if bp.Version == "Token" {
			return number
		}
	case GameFleshAndBlood:
		// A blueprint's version crosses the treatment with the wording
		// that picks between same-numbered printings, "Extended Art |
		// Rainbow Foil". The treatment half is already spoken for: the
		// listing states it as its own finish, off FabFoilNew, and saying
		// it twice narrows nothing. Only what the finish cannot say is
		// worth passing on.
		number, classic := fabPlainNumber(number)
		version := fabWording(fabVersions.Replace(bp.Version), number)
		if classic && !mtgmatcher.Contains(version, fabCCTag) {
			version = strings.TrimSpace(version + " " + fabCCTag)
		}
		if version == "" {
			return number
		}
		return number + " " + version
	case GamePokemon:
		// Pokemon's Version carries what the bare number cannot: the real
		// collector number with its set total ("Holo Promo | 013/025" where
		// the number field says "013h"), the treatment that picks between
		// same-numbered printings ("Non-Holo", "Cosmos Holo", "Reverse
		// Holo"), and the label the promo sets tell their reprints apart by.
		// The matcher reads its numbers from the back, so the Version's
		// full number is asked before the decorated blueprint field.
	default:
		return number
	}
	return number + " " + bp.Version
}

// fabCCTag is the catalog's word for the Classic Constructed printing, which
// CardTrader spells "CC Label".
const fabCCTag = "CC Tag"

// fabVersions spells a Flesh and Blood version the way the catalog does.
var fabVersions = strings.NewReplacer("CC Label", fabCCTag)

// fabPlainNumber takes the Classic Constructed marker off a collector number
// and says whether it was there. The catalog files that printing at the plain
// number and tells it from its twin by the tag alone, so a number wearing the
// marker names nothing, and the listing settles for whichever other set
// prints the card.
func fabPlainNumber(number string) (string, bool) {
	trimmed, found := strings.CutSuffix(number, "cc")
	if !found || trimmed == "" {
		return number, false
	}
	// Only a number wears the marker; a name ending in those letters does not.
	last := trimmed[len(trimmed)-1]
	if last < '0' || last > '9' {
		return number, false
	}
	return trimmed, true
}

// fabTreatments are the values a Flesh and Blood listing states as its own
// finish. A version naming one of them adds nothing the finish has not
// already said.
var fabTreatments = map[string]bool{
	"rainbow foil": true,
	"cold foil":    true,
	"gold foil":    true,
	"regular":      true,
	"normal":       true,
}

// fabWording keeps the half of a blueprint's version that names a printing
// rather than a finish. A version restating the collector number - "DYN115"
// against the number DYN115 - names nothing either, and would only stutter.
func fabWording(version, number string) string {
	var kept []string
	for _, part := range strings.Split(version, "|") {
		part = strings.TrimSpace(part)
		if part == "" || fabTreatments[strings.ToLower(part)] || strings.EqualFold(part, number) {
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, " ")
}

// gameName spells the card a blueprint names, which is not always its Name
// field. Yu-Gi-Oh's token blueprints say "Token" in the Version and leave the
// bare art in the name - "Dual Avatar Spirit", "Generaider" - while the
// catalog files those printings as "Token: Dual Avatar Spirit". The word
// cannot ride in the variation, where the shared token guard reads it and
// refuses the listing outright, so it moves into the name instead. A name
// already saying it is left alone: "Grinder Token" must not become
// "Token: Grinder Token".
func gameName(gameID int, bp *Blueprint) string {
	if gameID == GameYuGiOh && bp.Version == "Token" &&
		!mtgmatcher.Contains(bp.Name, "token") {
		return "Token: " + bp.Name
	}
	return bp.Name
}

func gameFoil(gameID int, product Product) bool {
	switch gameID {
	case GameMagic:
		return product.Properties.MTGFoil
	case GameLorcana:
		return product.Properties.LorcanaFoil
	case GameRiftbound:
		return product.Properties.RiftboundFoil
	case GameOnePiece:
		return product.Properties.OnePieceFoil
	case GameFleshAndBlood:
		// Every treatment the listing names is a foil, and the print run
		// beside it is not one, so the treatment alone answers the flag.
		// Yu-Gi-Oh deliberately has no arm: the rarity is its treatment,
		// so every listing reads nonfoil through the default below.
		switch product.Properties.FabFoilNew {
		case "", "Regular", "false":
			return false
		}
		return true
	case GamePokemon:
		// Only the reverse holo is a foil treatment the flag can stand for;
		// a first-edition listing is whatever its rarity makes it, and the
		// named finish above is what actually resolves either one. This
		// answers the fallback that matches on text when the blueprint
		// carries no TCGplayer id.
		return product.Properties.PokemonReverse
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
	switch gameID {
	case GameFleshAndBlood:
		// The treatment and the print run cross, and the datastore gives
		// each crossing its own printing, so a first-edition listing has
		// to name both. Only the run is worth naming on its own: the
		// unlimited one is what a product answers with by default.
		treatment := product.Properties.FabFoilNew
		switch treatment {
		case "", "Regular", "false":
			treatment = "Normal"
		}
		if product.Properties.FirstEdition {
			return "1st Edition " + treatment
		}
		if treatment == "Normal" {
			return ""
		}
		return treatment
	case GameYuGiOh:
		// The rarity is Yu-Gi-Oh's treatment and the print run is its
		// finish, so the run is the whole of what a listing names. Only
		// the first edition is worth naming: a product answers a lowered
		// flag with its unlimited printing already.
		if product.Properties.FirstEdition {
			return "1st Edition"
		}
	case GamePokemon:
		// Both flags name a printing the foilness cannot reach. The reverse
		// holo sits beside a holo rare's own printing, which is foil too, so
		// the flag lands on the wrong one of the pair; the first-edition run
		// sits beside the unlimited one, which the flag cannot tell it from
		// at all.
		if product.Properties.PokemonReverse {
			return "Reverse Holofoil"
		}
		if product.Properties.FirstEdition {
			return "1st Edition"
		}
	}
	return ""
}

// FormatBlueprints indexes blueprints by id and expansions by name, keeping
// either the sealed products or the singles.
func FormatBlueprints(blueprints []Blueprint, inExpansions []Expansion, sealed bool) (map[int]*Blueprint, map[int]string) {
	// Create a map to be able to retrieve edition name in the blueprint
	formatted := map[int]*Blueprint{}
	expansions := map[int]string{}
	for i := range blueprints {
		// Sealed is selected by exclusion, so a category cardtrader adds
		// later lands on the sealed side rather than silently vanishing;
		// accessories arrive with it and are dropped downstream by the
		// product-map resolution, which only names real sealed products.
		// A token and an oversized card are singles, whichever game
		// prints them: the sealed side would otherwise be asked about a
		// card by a name it shares with the set it came in.
		singles := false
		switch blueprints[i].CategoryID {
		case CategoryMagicSingles, CategoryMagicTokens, CategoryMagicOversized,
			CategoryLorcanaSingles, CategoryLorcanaOversized, CategoryLorcanaTokens,
			CategoryRiftboundSingles, CategoryRiftboundOversized,
			CategoryOnePieceSingles, CategoryOnePieceOversized, CategoryOnePieceDon,
			CategoryYuGiOhSingles, CategoryYuGiOhOversized,
			CategoryFleshAndBloodSingles, CategoryFleshAndBloodArtCardTokens,
			CategoryPokemonSingles, CategoryPokemonOversized:
			singles = true
		}
		if singles == sealed {
			continue
		}

		// Keep track of blueprints as they are more accurate that the
		// information found in product
		formatted[blueprints[i].ID] = &blueprints[i]

		// Load expansions array
		_, found := expansions[blueprints[i].ExpansionID]
		if !found {
			for j := range inExpansions {
				if inExpansions[j].ID == blueprints[i].ExpansionID {
					expansions[blueprints[i].ExpansionID] = inExpansions[j].Name
				}
			}
		}

		// The name is missing from the blueprints endpoint, fill it with data
		// retrieved from the expansions endpoint
		formatted[blueprints[i].ID].Expansion.Name = expansions[blueprints[i].ExpansionID]

		// Move the blueprint properties from the custom structure from blueprints
		// to the place as expected by Preprocess()
		formatted[blueprints[i].ID].Properties = formatted[blueprints[i].ID].FixedProperties
	}

	return formatted, expansions
}
