package mtgmatcher

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"
)

type cardFilterCallback func(inCard *InputCard, card *mtgjson.Card) bool

type promoTypeElement struct {
	// Name of the promo type to validate
	PromoType string

	// Validity date of the check, skipped if set is published before this
	ValidDate time.Time

	// Tag function
	TagFunc func(inCard *InputCard) bool

	// Simple tags to check, if TagFunc is not set
	Tags []string

	// Whether certain promos are not tagged, and are selected as wildcards
	CanBeWild bool
}

var promoTypeElements = []promoTypeElement{
	{
		PromoType: mtgjson.PromoTypePrerelease,
		Tags:      []string{"Prerelease", "Preview"},
	},
	{
		PromoType: mtgjson.PromoTypePlayPromo,
		Tags:      []string{"Play Promo"},
	},
	{
		PromoType: mtgjson.PromoTypePromoPack,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isPromoPack()
		},
	},
	{
		PromoType: mtgjson.PromoTypeSChineseAltArt,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isChineseAltArt()
		},
	},
	{
		PromoType: mtgjson.PromoTypeBuyABox,
		// After ZNR buy-a-box is also present in main set
		ValidDate: BuyABoxNotUniqueDate,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isBaB() || inCard.isRelease()
		},
		CanBeWild: true,
	},
	{
		PromoType: mtgjson.PromoTypeBundle,
		Tags:      []string{"Bundle"},
		CanBeWild: true,
	},
	{
		PromoType: mtgjson.PromoTypeGilded,
		Tags:      []string{"Gilded"},
	},
	{
		PromoType: mtgjson.PromoTypeTextured,
		Tags:      []string{"Textured"},
	},
	{
		PromoType: mtgjson.PromoTypeGalaxyFoil,
		TagFunc: func(inCard *InputCard) bool {
			// A lot of providers don't tag SLD cards as Galaxy, but just foil
			// (same for RainbowFoil), so this check essentially makes the test
			// pass, and let filtering continue elsewhere
			if inCard.isSecretLair() &&
				hasPrinting(inCard.Name, "promo_type", mtgjson.PromoTypeGalaxyFoil, "SLD") {
				return inCard.Foil || inCard.Contains("Glaxy")
			}
			return inCard.Contains("Galaxy")
		},
	},
	{
		PromoType: mtgjson.PromoTypeSurgeFoil,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isSurgeFoil()
		},
	},
	{
		PromoType: mtgjson.PromoTypeStepAndCompleat,
		Tags:      []string{"Compleat"},
	},
	{
		PromoType: mtgjson.PromoTypeConcept,
		Tags:      []string{"Concept"},
	},
	{
		PromoType: mtgjson.PromoTypeOilSlick,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isOilSlick()
		},
	},
	{
		PromoType: mtgjson.PromoTypeHaloFoil,
		Tags:      []string{"Halo"},
	},
	{
		PromoType: mtgjson.PromoTypeThickDisplay,
		ValidDate: SeparateFinishCollectorNumberDate,
		Tags:      []string{"Display", "Thick"},
	},
	{
		PromoType: mtgjson.PromoTypeSerialized,
		TagFunc: func(inCard *InputCard) bool {
			return inCard.isSerialized()
		},
	},
	{
		PromoType: mtgjson.PromoTypeConfettiFoil,
		Tags:      []string{"Confetti"},
	},
	{
		PromoType: mtgjson.PromoTypeEmbossed,
		Tags:      []string{"Ampersand", "Emblem", "Embossed"},
	},
	{
		PromoType: mtgjson.PromoTypeScroll,
		Tags:      []string{"Scroll", "Showcase Silver Foil"},
	},
	{
		PromoType: mtgjson.PromoTypePoster,
		Tags:      []string{"Poster", "Hand Drawn"},
	},
	{
		PromoType: mtgjson.PromoTypeInvisibleInk,
		Tags:      []string{"Invisible"},
	},
	{
		PromoType: mtgjson.PromoTypeRippleFoil,
		Tags:      []string{"Ripple"},
	},
}

var simpleFilterCallbacks = map[string]cardFilterCallback{
	"ARN": lightDarkManaCost,

	"FEM":  femVariantInArtist,
	"ALL":  variantInArtistOrFlavor,
	"HML":  variantInArtistOrFlavor,
	"PALP": variantInArtistOrFlavor,
	"PELP": variantInArtistOrFlavor,
	"CM2":  variantInArtistOrFlavor,

	"PMPS": variantInWatermark,
	"GK1":  variantInWatermark,
	"GK2":  variantInWatermark,

	"PLS": altArtCheck,

	"BBD": foilCheck,
	"CN2": foilCheck,
	"8ED": foilCheck,
	"FRF": foilCheck,
	"9ED": foilCheck,
	"ONS": foilCheck,
	"7ED": foilCheck,
	"10E": foilCheck,
	"UNH": foilCheck,
	"40K": foilCheck,

	"DKM": deckmastersVariant,
	"UST": singleLetterVariant,

	"POR": portalDemoGame,

	"2XM": launchPromoInSet,
	"2X2": launchPromoInSet,
	"CMM": launchPromoInSet,

	"CLB": variantBeforePlainCard,

	"KLD": starterDeckCheck,
	"AER": starterDeckCheck,

	"DD2": japaneseCheck,
	"STA": japaneseCheck,
	"WAR": japaneseCheck,

	"GRN": guildgateVariant,
	"RNA": guildgateVariant,

	"UNF": attractionVariant,

	"BOT": shatteredCheck,

	"MAT":  retroCheck,
	"MH2":  retroCheck,
	"P30H": retroCheck,
	"30A":  retroCheck,
	"PW23": retroCheck,
	"RVR":  retroCheck,
	"MH3":  retroCheck,

	"BRO": babOrBuyaboxRetroCheck,

	"THB": foilMisprint,
	"STX": foilMisprint,
	"SHM": foilMisprint,

	"PAL99": nodateMisprint,
	"PULG":  nodateMisprint,
	"HHO":   nodateMisprint,
	"PTOR":  laquatusMisprint,

	"PTC":  wcdNumberCompare,
	"WC97": wcdNumberCompare,
	"WC98": wcdNumberCompare,
	"WC99": wcdNumberCompare,
	"WC00": wcdNumberCompare,
	"WC01": wcdNumberCompare,
	"WC02": wcdNumberCompare,
	"WC03": wcdNumberCompare,
	"WC04": wcdNumberCompare,

	"PPTK": lubuPrereleaseVariant,

	"ALA": showcaseCheck,

	"IKO": reskinGodzillaCheck,

	"LTR":  lotrTripleFiltering,
	"PLTR": lotrTripleFiltering,

	"BFZ": fullartCheckForBasicLands,
	"ZEN": fullartCheckForBasicLands,

	// This is needed only for sets with multiple printings of the same card
	"KHM": phyrexianCheck,
	"NEO": phyrexianCheck,
	"SNC": phyrexianCheck,
	"DMU": phyrexianCheck,
	"ONE": phyrexianCheck,

	"TSR": releaseRetroCheck,
	"CLU": releaseRetroCheck,

	"PHOU": draftweekendCheck,
	"PXLN": draftweekendCheck,
	"PUST": draftweekendCheck,
	"PRIX": draftweekendCheck,
	"PDOM": draftweekendCheck,
	"PM19": draftweekendCheck,
	"PGRN": draftweekendCheck,
	"PRNA": draftweekendCheck,

	"G14": judgeLandCheck,
	"P23": judgeLandCheck,
}

var complexFilterCallbacks = map[string][]cardFilterCallback{
	"JMP": {launchPromoInSet, phyrexianCheck},
	"J22": {launchPromoInSet, animeCheck},
	"BRR": {schematicCheck},
	"DMR": {launchPromoInSet, releaseRetroCheck},
	"VOW": {wpnCheck, reskinDraculaCheck},
	"SLD": {sldVariant, etchedCheck, thickDisplayCheck, phyrexianCheck, reskinRenameCheck},
	"CMR": {variantInCommanderDeck, etchedCheck, thickDisplayCheck},
	"M3C": {foilCheck, thickDisplayCheck},

	"PWAR": {japaneseCheck, draftweekendCheck},

	// These two checks need to be separate in case two cards have the same number
	// but are originally from two different editions
	"PLST": {listNumberCompare, listEditionCheck},
}

func judgeLandCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if (inCard.Contains("14") && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)) ||
		inCard.Contains("23") && strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	} else if !inCard.Contains("14") && strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) ||
		!inCard.Contains("23") && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	}
	return false
}

func listNumberCompare(inCard *InputCard, card *mtgjson.Card) bool {
	number := ExtractNumber(inCard.Variation)

	// If a number is found, check that it's matching the card number
	if number != "" {
		// Compare the number portion of the field only
		cardNumbers := strings.Split(card.Number, "-")
		listNumbers := strings.Split(number, "-")
		cardNumber := cardNumbers[len(cardNumbers)-1]
		listNumber := listNumbers[len(listNumbers)-1]
		if cardNumber == listNumber {
			return false
		}

		// Skip anything else, the number needs to be correct,
		// unless there is actually an edition name (ie Masters 25)
		// that will be processed later
		maybeEdition := inCard.Variation
		maybeEdition = strings.Replace(maybeEdition, "Non-Foil", "", 1)
		maybeEdition = strings.Replace(maybeEdition, "Foil", "", 1)
		maybeEdition = strings.TrimLeft(maybeEdition, " -")
		_, err := GetSetByName(maybeEdition)
		if err != nil {
			return true
		}
	}

	return false
}

var allPlayerRewardsSet = []string{
	"P03", "P04", "P05", "P06", "P07", "P08", "P09", "P10", "P11",
}

func listEditionCheck(inCard *InputCard, card *mtgjson.Card) bool {
	code := strings.Split(card.Number, "-")[0]
	set, err := GetSet(code)
	if err != nil {
		return true
	}

	switch inCard.Name {
	case "Phantom Centaur":
		return misprintCheck(inCard, card)
	case "Laboratory Maniac":
		// Only card with same number, so the chunk below trips the check
		if !inCard.Contains(code) && !inCard.Contains(set.Name) && EditionTable[inCard.Variation] != set.Name {
			return true
		}
	default:
		if inCard.Contains("Player Rewards") && slices.Contains(allPlayerRewardsSet, code) {
			return false
		}

		edition := set.Name
		if strings.Contains(inCard.Variation, "vs.") {
			edition = strings.TrimPrefix(edition, "Duel Decks: ")
		}

		if !inCard.Contains(code) && !inCard.Contains(edition) && EditionTable[inCard.Variation] != set.Name {
			// This chunk is needed in case there was a plain number already
			// processed in the previous step
			number := ExtractNumber(inCard.Variation)

			cardNumbers := strings.Split(card.Number, "-")
			listNumbers := strings.Split(number, "-")
			cardNumber := cardNumbers[len(cardNumbers)-1]
			listNumber := listNumbers[len(listNumbers)-1]
			if cardNumber == listNumber {
				return false
			}

			return true
		}
	}

	return false
}

func phyrexianCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isPhyrexian() && card.Language != mtgjson.LanguagePhyrexian {
		return true
	} else if !inCard.isPhyrexian() && card.Language == mtgjson.LanguagePhyrexian {
		return true
	}
	return false
}

// Handle full vs nonfull art basic land
func fullartCheckForBasicLands(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isBasicFullArt() && !card.IsFullArt {
		return true
	} else if inCard.isBasicNonFullArt() && card.IsFullArt {
		return true
	}
	return false
}

func lotrTripleFiltering(inCard *InputCard, card *mtgjson.Card) bool {
	switch card.Name {
	case "Delighted Halfling",
		"Lobelia Sackville-Baggins",
		"Frodo Baggins",
		"Bilbo, Retired Burglar",
		"Gandalf, Friend of the Shire",
		"Wizard's Rockets":
		num := ExtractNumber(inCard.Variation)
		if num != "" && (Contains(inCard.Edition, "Prerelease") || Contains(inCard.Edition, "Promo")) {
			return card.SetCode != "PLTR"
		}
		if inCard.Contains("Stamp") {
			if !inCard.Contains("No") && card.SetCode != "PLTR" {
				return true
			} else if inCard.Contains("No") && card.SetCode == "PLTR" {
				return true
			}
		}
	case "Saruman of Many Colors":
		if inCard.Contains("Store Champ") && !card.HasPromoType(mtgjson.PromoTypeStoreChampionship) {
			return true
		} else if !inCard.Contains("Store Champ") && card.HasPromoType(mtgjson.PromoTypeStoreChampionship) {
			return true
		}
	}
	return false
}

func lightDarkManaCost(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isARNLightMana() && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
		return true
	} else if (inCard.isARNDarkMana() || inCard.Variation == "") && strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
		return true
	}
	return false
}

func femVariantInArtist(inCard *InputCard, card *mtgjson.Card) bool {
	// Since the check is field by field Foglio may alias Phil or Kaja
	if strings.Contains(inCard.Variation, "Foglio") {
		inCard.Variation = strings.Replace(inCard.Variation, "Phil Foglio", "PhilFoglio", 1)
		inCard.Variation = strings.Replace(inCard.Variation, "Kaja Foglio", "KajaFoglio", 1)
	}
	return variantInArtistOrFlavor(inCard, card)
}

func variantInArtistOrFlavor(inCard *InputCard, card *mtgjson.Card) bool {
	// Skip the check if this tag is empty, so that users can notice
	// there is an aliasing problem
	if inCard.Variation == "" {
		return true
	}

	fields := strings.Fields(inCard.Variation)
	found := false

	// Keep flavor text author only
	flavor := card.FlavorText
	if strings.Contains(flavor, "—") {
		fields := strings.Split(flavor, "—")
		flavor = fields[len(fields)-1]
	}

	// Check field by field, it's usually enough for just two elements
	for _, field := range fields {
		// Skip short text like 'jr.' since they are often missing
		// Skip Land too for High and Low lands alias
		// Skip Sass due to the fact that 's' are ignored
		if len(field) < 4 || strings.HasPrefix(field, "Land") || field == "Sass" {
			continue
		}
		if Contains(flavor, field) || Contains(card.Artist, field) {
			found = true
			break
		}
	}

	if !found {
		// If not found double check if variation contains the same number suffix
		numberSuffix := inCard.possibleNumberSuffix()
		if numberSuffix == "" || (numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix)) {
			return true
		}
	}

	return false
}

// Check watermark when variation has no number information
func variantInWatermark(inCard *InputCard, card *mtgjson.Card) bool {
	// Skip the check if this tag is empty, so that users can notice there is an aliasing problem
	if inCard.Variation == "" {
		return true
	}
	if !Contains(inCard.Variation, card.Watermark) {
		return true
	}
	return false
}

// Foil-only-booster cards, non-special version has both foil and non-foil
func altArtCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isGenericAltArt() && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	} else if !inCard.isGenericAltArt() && strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	}
	return false
}

// Foil-only-booster cards, non-special version only have non-foil
// (only works if card has no other duplicates within the same edition)
func foilCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.Foil && card.HasFinish(mtgjson.FinishNonfoil) {
		return true
	} else if !inCard.Foil && card.HasFinish(mtgjson.FinishFoil) {
		return true
	}
	return false
}

func etchedCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isEtched() && !card.HasFinish(mtgjson.FinishEtched) {
		return true
		// Some thick display cards are not marked as etched
	} else if !inCard.isEtched() && !inCard.isThickDisplay() && card.HasFinish(mtgjson.FinishEtched) {
		return true
	}
	return false
}

func thickDisplayCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isThickDisplay() && !card.HasPromoType(mtgjson.PromoTypeThickDisplay) {
		return true
	} else if !inCard.isThickDisplay() && card.HasPromoType(mtgjson.PromoTypeThickDisplay) {
		return true
	}
	return false
}

// Single letter variants
func singleLetterVariant(inCard *InputCard, card *mtgjson.Card) bool {
	numberSuffix := inCard.possibleNumberSuffix()
	if len(card.Variations) > 0 && numberSuffix == "" {
		numberSuffix = "a"
	}
	if numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix) {
		return true
	}
	return false
}

func deckmastersVariant(inCard *InputCard, card *mtgjson.Card) bool {
	numberSuffix := inCard.possibleNumberSuffix()
	switch card.Name {
	case "Incinerate", "Icy Manipulator":
		inCard.Foil = inCard.Foil || inCard.Contains("Promo")
		return foilCheck(inCard, card)
	default:
		// Pick the first of the two if not specified
		if len(card.Variations) > 0 && numberSuffix == "" {
			numberSuffix = "a"
		}
		// Reset for lands
		if inCard.isBasicLand() {
			numberSuffix = ""
		}
	}

	if numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix) {
		return true
	}
	return false
}

// Variants related to flavor text presence
func portalDemoGame(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isPortalAlt() && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) && !strings.HasSuffix(card.Number, "d") {
		return true
	} else if !inCard.isPortalAlt() && (strings.HasSuffix(card.Number, mtgjson.SuffixVariant) || strings.HasSuffix(card.Number, "d")) {
		return true
	}
	return false
}

// Launch promos within the set itself
func launchPromoInSet(inCard *InputCard, card *mtgjson.Card) bool {
	anyAlternative := card.IsAlternative ||
		card.BorderColor == mtgjson.BorderColorBorderless ||
		card.HasFrameEffect(mtgjson.FrameEffectExtendedArt)
	if (inCard.isRelease() || inCard.isBaB()) && !anyAlternative {
		return true
	} else if !(inCard.isRelease() || inCard.isBaB()) && anyAlternative && !card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
		return true
	}
	return false
}

// Identical cards
func variantInCommanderDeck(inCard *InputCard, card *mtgjson.Card) bool {
	// Filter only cards that may have the flag set
	hasAlternate := card.IsAlternative
	for _, id := range card.Variations {
		alt := backend.UUIDs[id]
		if alt.IsAlternative {
			hasAlternate = true
			break
		}
	}
	// Only check when cards do have alts, as some vendors use the
	// Variation field for unnecessary info for unrelated cards
	// Skip EA because it does not need this deduplication
	if !inCard.isExtendedArt() && !inCard.isEtched() && hasAlternate {
		if inCard.Variation == "" && card.IsAlternative {
			return true
		} else if inCard.Variation != "" && !card.IsAlternative {
			return true
		}
	}
	return false
}

// EA cards from commander decks appear before the normal prints, beyondBaseSet needs help
func variantBeforePlainCard(inCard *InputCard, card *mtgjson.Card) bool {
	cn, _ := strconv.Atoi(card.Number)
	if cn > 607 && cn < 930 {
		return extendedartCheck(inCard, card)
	}
	return false
}

// Intro/Starter deck
func starterDeckCheck(inCard *InputCard, card *mtgjson.Card) bool {
	isStarter := Contains(inCard.Variation, "Starter") || Contains(inCard.Variation, "Intro")
	if !isStarter && (card.HasPromoType(mtgjson.PromoTypeStarterDeck) || card.IsAlternative) {
		return true
	} else if isStarter && !card.HasPromoType(mtgjson.PromoTypeStarterDeck) && !card.IsAlternative {
		return true
	}
	return false
}

// Japanese Planeswalkers
func japaneseCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if (inCard.isJPN() || inCard.isGenericAltArt()) && card.Language != mtgjson.LanguageJapanese {
		return true
	} else if !inCard.isJPN() && !inCard.isGenericAltArt() && card.Language == mtgjson.LanguageJapanese {
		return true
	}
	return false
}

// Pick one of the printings in case they are not specified
func guildgateVariant(inCard *InputCard, card *mtgjson.Card) bool {
	if strings.Contains(card.Name, "Guildgate") && inCard.Variation == "" {
		cn, _ := strconv.Atoi(card.Number)
		if cn%2 == 0 {
			return true
		}
	}
	return false
}

// Due to the WPN lands
func wpnCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isWPNGateway() && !card.HasPromoType(mtgjson.PromoTypeWPN) {
		return true
	} else if !inCard.isWPNGateway() && card.HasPromoType(mtgjson.PromoTypeWPN) {
		return true
	}
	return false
}

// Handle the different Attractions
func attractionVariant(inCard *InputCard, card *mtgjson.Card) bool {
	if card.AttractionLights != nil && (strings.Contains(inCard.Variation, "/") || strings.Contains(inCard.Variation, "-")) {
		lights := make([]string, 0, len(card.AttractionLights))
		for _, light := range card.AttractionLights {
			lights = append(lights, strconv.Itoa(light))
		}
		tag := strings.Join(lights, "/")
		variation := strings.Replace(inCard.Variation, " ", "", -1)
		variation = strings.Replace(variation, "-", "/", -1)
		if variation != tag {
			return true
		}
	}
	switch card.Name {
	case "Space Beleren",
		"Comet, Stellar Pup":
		if inCard.isBorderless() && !inCard.isGalaxyFoil() {
			if card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) {
				return true
			}
		} else if inCard.isGalaxyFoil() && !inCard.isBorderless() {
			if card.BorderColor == mtgjson.BorderColorBorderless {
				return true
			}
		}
	default:
		if !inCard.isBorderless() && !inCard.isGalaxyFoil() &&
			slices.Contains(card.Types, "Land") &&
			card.BorderColor == mtgjson.BorderColorBorderless &&
			card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) {
			return true
		}
	}
	return false
}

func shatteredCheck(inCard *InputCard, card *mtgjson.Card) bool {
	isShattered := inCard.Contains("Shattered") || inCard.Contains("Borderless")
	if isShattered && !card.HasFrameEffect(mtgjson.FrameEffectShattered) {
		return true
	} else if !isShattered && card.HasFrameEffect(mtgjson.FrameEffectShattered) {
		return true
	}
	return false
}

// This check skips serialized cards as their collector numbers would not match
func schematicCheck(inCard *InputCard, card *mtgjson.Card) bool {
	cn, err := strconv.Atoi(card.Number)
	if err != nil {
		return false
	}
	isSchematic := inCard.Contains("Schematic") || inCard.Contains("Blueprint")
	if isSchematic && cn < 64 {
		return true
	} else if !isSchematic && cn >= 64 {
		return true
	}
	return false
}

func animeCheck(inCard *InputCard, card *mtgjson.Card) bool {
	switch card.Name {
	case "Valorous Stance",
		"Dragon Fodder",
		"Stitcher's Supplier",
		"Tragic Slip",
		"Thermo-Alchemist":
		cn, _ := strconv.Atoi(card.Number)
		isAnime := inCard.Contains("Anime")
		if isAnime && (cn < 52 || cn > 97) {
			return true
		} else if !isAnime && cn >= 52 && cn <= 97 {
			return true
		}
	}
	return false
}

func retroCheckInternal(isRetro bool, cardFrameVersion string) bool {
	if isRetro && cardFrameVersion != "1997" {
		return true
	} else if !isRetro && cardFrameVersion == "1997" {
		return true
	}
	return false
}

func retroCheck(inCard *InputCard, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isRetro() || inCard.beyondBaseSet, card.FrameVersion)
}

// This edition has retro-only promotional cards, but most
// providers only tag the promo type, instead of the frame
func babOrBuyaboxRetroCheck(inCard *InputCard, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isBundle() || inCard.isBaB(), card.FrameVersion)
}

func releaseRetroCheck(inCard *InputCard, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isRetro() || inCard.isRelease(), card.FrameVersion)
}

// Foil cards which exist *only* as misprints
func foilMisprint(inCard *InputCard, card *mtgjson.Card) bool {
	if !inCard.Foil {
		return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
	}

	// Get number in case there is no EA information available
	maybeNumber := ExtractNumber(inCard.Variation)

	switch card.Name {
	case "Temple of Abandon":
		if inCard.isExtendedArt() || strings.HasPrefix(maybeNumber, "347") {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Strict Proctor":
		if !inCard.isExtendedArt() || strings.HasPrefix(maybeNumber, "33") {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Reflecting Pool":
		return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
	}
	return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
}

func nodateMisprint(inCard *InputCard, card *mtgjson.Card) bool {
	switch card.Name {
	case "Beast of Burden",
		"Island",
		"Stocking Tiger":
		if inCard.Contains("Misprint") ||
			inCard.Contains("No Expansion Symbol") ||
			inCard.Contains("No Date") ||
			inCard.Contains("No Stamp") ||
			inCard.Contains("No Symbol") {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixVariant)
		}
	}
	return strings.HasSuffix(card.Number, mtgjson.SuffixVariant)
}

func laquatusMisprint(inCard *InputCard, card *mtgjson.Card) bool {
	switch card.Name {
	case "Laquatus's Champion":
		if Contains(inCard.Variation, "dark") {
			return card.Number != "67†a"
		}
		if Contains(inCard.Variation, "misprint") {
			return card.Number != "67†"
		}
		return card.Number != "67"
	}
	return false
}

func sldVariant(inCard *InputCard, card *mtgjson.Card) bool {
	var result bool
	switch card.Name {
	case "Demonlord Belzenlok",
		"Griselbrand",
		"Liliana's Contract",
		"Kothophed, Soul Hoarder",
		"Razaketh, the Foulblooded":
		result = strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		if inCard.isEtched() {
			result = !result
		}
	case "Plague Sliver",
		"Shadowborn Apostle",
		"Toxin Sliver",
		"Virulent Sliver":
		result = strings.HasSuffix(card.Number, "Φ")
		if inCard.isStepAndCompleat() {
			result = !result
		}
	case "Mogis, God of Slaughter":
		if card.BorderColor != mtgjson.BorderColorBorderless && !inCard.Foil {
			return true
		}
	}
	// Prevent flipping the result below if multiple printings are present (ie Griselbrand)
	if inCard.isEtched() || inCard.isStepAndCompleat() {
		return result
	}

	// All the Rainbow SLD cards not serialized are tagged as two different entries
	// ie Goblin Lackey or Aminatou, the Fateshifter
	if hasPrinting(card.Name, "promo_type", mtgjson.PromoTypeRainbowFoil, "SLD") ||
		hasPrinting(card.Name, "promo_type", mtgjson.PromoTypeGalaxyFoil, "SLD") {
		result = strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		if inCard.Foil {
			result = !result
		}
	}
	return result
}

func wcdNumberCompare(inCard *InputCard, card *mtgjson.Card) bool {
	prefix, sideboard := inCard.worldChampPrefix()
	wcdNum := extractWCDNumber(inCard.Variation, prefix, sideboard)

	// If a wcdNum is found, check that it's matching the card number
	if wcdNum != "" {
		if wcdNum == card.Number {
			return false
		}
		// Skip anything else, the number needs to be correct
		return true
	}

	// Else rebuild the number manually using prefix, sideboard, and num as hints
	if prefix != "" {
		// Copy this field so we can discard portions that have
		// already been used for deduplication
		cn := card.Number
		if sideboard && !strings.HasSuffix(cn, "sb") {
			return true
		} else if !sideboard && strings.HasSuffix(cn, "sb") {
			return true
		}
		cn = strings.Replace(cn, "sb", "", 1)

		// ML and MLP conflict with HasPrefix, so strip away
		// the numeric part and do a straight equal
		idx := strings.IndexFunc(cn, func(c rune) bool {
			return unicode.IsDigit(c)
		})
		if idx < 1 || prefix != cn[:idx] {
			return true
		}
		cn = strings.Replace(cn, prefix, "", 1)

		num := ExtractNumber(inCard.Variation)
		if num != "" {
			cnn := cn
			// Strip last character if it's a letter
			if unicode.IsLetter(rune(cn[len(cn)-1])) {
				cnn = cn[:len(cn)-1]
			}
			// Try both simple number and original collector number
			if num != cnn && num != cn {
				return true
			}
			cn = strings.Replace(cn, num, "", 1)
		}

		if len(cn) > 0 && unicode.IsLetter(rune(cn[len(cn)-1])) {
			suffix := inCard.possibleNumberSuffix()
			if suffix != "" && !strings.HasSuffix(cn, suffix) {
				return true
			}
		}
	}
	return false
}

func lubuPrereleaseVariant(inCard *InputCard, card *mtgjson.Card) bool {
	if (strings.Contains(inCard.Variation, "April") || strings.Contains(inCard.Variation, "4/29")) && card.OriginalReleaseDate != "1999-04-29" {
		return true
	} else if (strings.Contains(inCard.Variation, "July") || strings.Contains(inCard.Variation, "7/4")) && card.OriginalReleaseDate != "1999-07-04" {
		return true
	}
	return false
}

func borderlessCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isBorderless() && card.BorderColor != mtgjson.BorderColorBorderless {
		return true
	} else if !inCard.isBorderless() && card.BorderColor == mtgjson.BorderColorBorderless && !card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
		return true
	}
	return false
}

func showcaseCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isShowcase() && !card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
		return true
	} else if !inCard.isShowcase() && card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
		return true
	}
	return false
}

func extendedartCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isExtendedArt() && !card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
		return true
		// BaB are allowed to have extendedart
	} else if !inCard.isExtendedArt() && card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) && !card.HasPromoType(mtgjson.PromoTypeBuyABox) {
		return true
	}
	return false
}

// IKO-Style cards with different names
func reskinGodzillaCheck(inCard *InputCard, card *mtgjson.Card) bool {
	// Also some providers do not tag Japanese-only Godzilla cards as such
	if inCard.isReskin() && !card.HasPromoType(mtgjson.PromoTypeGodzilla) {
		return true
	} else if !inCard.isReskin() && !inCard.beyondBaseSet && card.HasPromoType(mtgjson.PromoTypeGodzilla) {
		return true
	}
	return false
}

func reskinDraculaCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if inCard.isReskin() && !card.HasPromoType(mtgjson.PromoTypeDracula) {
		return true
	} else if !inCard.isReskin() && !inCard.beyondBaseSet && card.HasPromoType(mtgjson.PromoTypeDracula) {
		return true
	}
	return false
}

// In case there is no number information and the card may known with other names
func reskinRenameCheck(inCard *InputCard, card *mtgjson.Card) bool {
	if ExtractNumber(inCard.Variation) != "" || card.FlavorName == "" {
		return false
	}
	if inCard.isReskin() && !Contains(inCard.originalName, card.FlavorName) {
		return true
	} else if !inCard.isReskin() && Contains(inCard.originalName, card.FlavorName) {
		return true
	}
	return false
}

func misprintCheck(inCard *InputCard, card *mtgjson.Card) bool {
	// These cards are allowed to have the star at the end
	if inCard.isBasicLand() && inCard.isJudge() {
		return false
	}

	hasSuffix := strings.HasSuffix(card.Number, mtgjson.SuffixVariant) || strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
	if inCard.Contains("Misprint") && !hasSuffix {
		return true
	} else if !inCard.Contains("Misprint") && hasSuffix {
		return true
	}
	return false
}

func draftweekendCheck(inCard *InputCard, card *mtgjson.Card) bool {
	releaseOrDraft := inCard.Contains("Draft Weekend") || (inCard.Contains("Release") && !inCard.isPrerelease())
	if releaseOrDraft && !card.HasPromoType(mtgjson.PromoTypeDraftWeekend) {
		return true
	} else if !releaseOrDraft && card.HasPromoType(mtgjson.PromoTypeDraftWeekend) {
		return true
	}
	return false
}

type numberFilterCallback func(inCard *InputCard) []string

var numberFilterCallbacks = map[string]numberFilterCallback{
	// Some editions duplicate foil and nonfoil in the same set
	"7ED": duplicateEveryFoil,
	"8ED": duplicateEveryFoil,
	"9ED": duplicateEveryFoil,

	// These editions duplicate foil and nonfoil for some cards only
	"10E": duplicateSomeFoil,
	"UNH": duplicateSomeFoil,
	"FRF": duplicateSomeFoil,
	"ONS": duplicateSomeFoil,
	"THB": duplicateSomeFoil,
	"STX": duplicateSomeFoil,
	"SHM": duplicateSomeFoil,
	"M3C": duplicateSomeFoil,
	"DKM": duplicateSomeFoil,

	// Intro lands from these sets when non-fullart always have this
	"ZEN": duplicateBasicLands,
	"BFZ": duplicateBasicLands,
	"OGW": duplicateBasicLands,

	// JPN planeswalkers
	"WAR":  duplicateJPNPlaneswalkers,
	"PWAR": duplicateJPNPlaneswalkers,
	"DD2":  duplicateJPNPlaneswalkers,

	// 40K could have numbers reported alongside the surge tag
	"40K": duplicateSomeFoil,

	// This is a mess
	"SLD": duplicateSLD,
}

func duplicateEveryFoil(inCard *InputCard) []string {
	if inCard.Foil {
		return []string{mtgjson.SuffixSpecial}
	}
	return nil
}

func duplicateSomeFoil(inCard *InputCard) []string {
	if inCard.Foil {
		return []string{mtgjson.SuffixSpecial, ""}
	}
	return nil
}

func duplicateBasicLands(inCard *InputCard) []string {
	if inCard.isBasicNonFullArt() {
		return []string{"a"}
	}
	return nil
}

func duplicateJPNPlaneswalkers(inCard *InputCard) []string {
	if inCard.isJPN() {
		return []string{mtgjson.SuffixSpecial, "s" + mtgjson.SuffixSpecial}
	}
	return nil
}

func duplicateSLD(inCard *InputCard) []string {
	if inCard.isStepAndCompleat() {
		return []string{mtgjson.SuffixPhiLow, ""}
	}

	if inCard.isEtched() || inCard.isThickDisplay() {
		return []string{mtgjson.SuffixSpecial, ""}
	}

	if inCard.Foil &&
		(hasPrinting(inCard.Name, "promo_type", mtgjson.PromoTypeRainbowFoil, "SLD") ||
			hasPrinting(inCard.Name, "promo_type", mtgjson.PromoTypeGalaxyFoil, "SLD")) {
		return []string{mtgjson.SuffixSpecial, ""}
	}

	return nil
}
