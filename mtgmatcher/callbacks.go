package mtgmatcher

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

type cardFilterCallback func(inCard *Card, card *mtgjson.Card) bool

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

	"DKM": deckmastersVariant,
	"UST": singleLetterVariant,

	"POR": portalDemoGame,

	"2XM": launchPromoInSet,
	"2X2": launchPromoInSet,
	"JMP": launchPromoInSet,
	"J22": animeCheck,

	"CMR": variantInCommanderDeck,
	"CLB": variantBeforePlainCard,

	"KLD": starterDeckCheck,
	"AER": starterDeckCheck,

	"DD2":  japaneseCheck,
	"STA":  japaneseCheck,
	"WAR":  japaneseCheck,
	"PWAR": japaneseCheck,

	"GRN": guildgateVariant,
	"RNA": guildgateVariant,

	"VOW": wpnCheck,

	"UNF": attractionVariant,

	"BOT": shatteredCheck,
	"MUL": serializedCheck,

	"MH2":     retroCheck,
	"P30H":    retroCheck,
	"P30HJAP": retroCheck,
	"30A":     retroCheck,

	"BRO": babOrBuyaboxRetroCheck,

	"SLD": sldVariant,

	"THB": foilMisprint,
	"STX": foilMisprint,
	"SHM": foilMisprint,
	"GPT": foilMisprint,

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
}

var complexFilterCallbacks = map[string][]cardFilterCallback{
	"BRR": []cardFilterCallback{serializedCheck, schematicCheck},
	"DMR": []cardFilterCallback{launchPromoInSet, releaseRetroCheck},
}

func lightDarkManaCost(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isARNLightMana() && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
		return true
	} else if (inCard.isARNDarkMana() || inCard.Variation == "") && strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
		return true
	}
	return false
}

func femVariantInArtist(inCard *Card, card *mtgjson.Card) bool {
	// Since the check is field by field Foglio may alias Phil or Kaja
	if strings.Contains(inCard.Variation, "Foglio") {
		inCard.Variation = strings.Replace(inCard.Variation, "Phil Foglio", "PhilFoglio", 1)
		inCard.Variation = strings.Replace(inCard.Variation, "Kaja Foglio", "KajaFoglio", 1)
	}
	return variantInArtistOrFlavor(inCard, card)
}

func variantInArtistOrFlavor(inCard *Card, card *mtgjson.Card) bool {
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
func variantInWatermark(inCard *Card, card *mtgjson.Card) bool {
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
func altArtCheck(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isGenericAltArt() && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	} else if !inCard.isGenericAltArt() && strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
		return true
	}
	return false
}

// Foil-only-booster cards, non-special version only have non-foil
// (only works if card has no other duplicates within the same edition)
func foilCheck(inCard *Card, card *mtgjson.Card) bool {
	if inCard.Foil && card.HasFinish(mtgjson.FinishNonfoil) {
		return true
	} else if !inCard.Foil && card.HasFinish(mtgjson.FinishFoil) {
		return true
	}
	return false
}

// Single letter variants
func singleLetterVariant(inCard *Card, card *mtgjson.Card) bool {
	numberSuffix := inCard.possibleNumberSuffix()
	if len(card.Variations) > 0 && numberSuffix == "" {
		numberSuffix = "a"
	}
	if numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix) {
		return true
	}
	return false
}

func deckmastersVariant(inCard *Card, card *mtgjson.Card) bool {
	numberSuffix := inCard.possibleNumberSuffix()
	if len(card.Variations) > 0 && numberSuffix == "" {
		numberSuffix = "a"
		if inCard.Foil || inCard.Contains("Promo") {
			numberSuffix = mtgjson.SuffixSpecial
		} else if card.HasFinish(mtgjson.FinishNonfoil) &&
			(card.Name == "Incinerate" || card.Name == "Icy Manipulator") {
			numberSuffix = ""
		}
	}
	if numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix) {
		return true
	}
	return false
}

// Variants related to flavor text presence
func portalDemoGame(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isPortalAlt() && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) && !strings.HasSuffix(card.Number, "d") {
		return true
	} else if !inCard.isPortalAlt() && (strings.HasSuffix(card.Number, mtgjson.SuffixVariant) || strings.HasSuffix(card.Number, "d")) {
		return true
	}
	return false
}

// Launch promos within the set itself
func launchPromoInSet(inCard *Card, card *mtgjson.Card) bool {
	if (inCard.isRelease() || inCard.isBaB()) && !card.IsAlternative {
		return true
	} else if !(inCard.isRelease() || inCard.isBaB()) && card.IsAlternative && !card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
		return true
	}
	return false
}

// Identical cards
func variantInCommanderDeck(inCard *Card, card *mtgjson.Card) bool {
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
func variantBeforePlainCard(inCard *Card, card *mtgjson.Card) bool {
	cn, _ := strconv.Atoi(card.Number)
	if cn > 607 && cn < 930 {
		if inCard.isExtendedArt() && !card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
			return true
		} else if !inCard.isExtendedArt() && card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
			return true
		}
	}
	return false
}

// Intro/Starter deck
func starterDeckCheck(inCard *Card, card *mtgjson.Card) bool {
	isStarter := Contains(inCard.Variation, "Starter") || Contains(inCard.Variation, "Intro")
	if !isStarter && card.HasPromoType(mtgjson.PromoTypeStarterDeck) {
		return true
	} else if isStarter && !card.HasPromoType(mtgjson.PromoTypeStarterDeck) {
		return true
	}
	return false
}

// Japanese Planeswalkers
func japaneseCheck(inCard *Card, card *mtgjson.Card) bool {
	if (inCard.isJPN() || inCard.isGenericAltArt()) && card.Language != mtgjson.LanguageJapanese {
		return true
	} else if !inCard.isJPN() && !inCard.isGenericAltArt() && card.Language == mtgjson.LanguageJapanese {
		return true
	}
	return false
}

// Pick one of the printings in case they are not specified
func guildgateVariant(inCard *Card, card *mtgjson.Card) bool {
	if strings.Contains(card.Name, "Guildgate") && inCard.Variation == "" {
		cn, _ := strconv.Atoi(card.Number)
		if cn%2 == 0 {
			return true
		}
	}
	return false
}

// Due to the WPN lands
func wpnCheck(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isWPNGateway() && !card.HasPromoType(mtgjson.PromoTypeWPN) {
		return true
	} else if !inCard.isWPNGateway() && card.HasFinish(mtgjson.PromoTypeWPN) {
		return true
	}
	return false
}

// Handle the different Attractions
func attractionVariant(inCard *Card, card *mtgjson.Card) bool {
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
			sliceStringHas(card.Types, "Land") &&
			card.BorderColor == mtgjson.BorderColorBorderless &&
			card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) {
			return true
		}
	}
	return false
}

func shatteredCheck(inCard *Card, card *mtgjson.Card) bool {
	isShattered := inCard.Contains("Shattered") || inCard.Contains("Borderless")
	if isShattered && !card.HasFrameEffect(mtgjson.FrameEffectShattered) {
		return true
	} else if !isShattered && card.HasFrameEffect(mtgjson.FrameEffectShattered) {
		return true
	}
	return false
}

func serializedCheck(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isSerialized() && !card.HasPromoType(mtgjson.PromoTypeSerialized) {
		return true
	} else if !inCard.isSerialized() && card.HasPromoType(mtgjson.PromoTypeSerialized) {
		return true
	}
	return false
}

func schematicCheck(inCard *Card, card *mtgjson.Card) bool {
	// Skip check for serialized cards as collector numbers would not match
	if inCard.isSerialized() {
		return false
	}

	cn, _ := strconv.Atoi(card.Number)
	isSchematic := inCard.Contains("Schematic") || inCard.Contains("Blueprint")
	if isSchematic && cn < 64 {
		return true
	} else if !isSchematic && cn >= 64 {
		return true
	}
	return false
}

func animeCheck(inCard *Card, card *mtgjson.Card) bool {
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

func retroCheck(inCard *Card, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isRetro() || inCard.beyondBaseSet, card.FrameVersion)
}

// This edition has retro-only promotional cards, but most
// providers only tag the promo type, instead of the frame
func babOrBuyaboxRetroCheck(inCard *Card, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isBundle() || inCard.isBaB(), card.FrameVersion)
}

func releaseRetroCheck(inCard *Card, card *mtgjson.Card) bool {
	return retroCheckInternal(inCard.isRetro() || inCard.isRelease(), card.FrameVersion)
}

func foilMisprint(inCard *Card, card *mtgjson.Card) bool {
	if !inCard.Foil {
		return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
	}
	switch card.Name {
	case "Temple of Abandon":
		if inCard.isExtendedArt() {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Strict Proctor":
		if !inCard.isExtendedArt() {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Reflecting Pool":
		return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
	case "Shadow Lance":
		if inCard.Contains("Misprint") {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	}
	return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
}

func nodateMisprint(inCard *Card, card *mtgjson.Card) bool {
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

func laquatusMisprint(inCard *Card, card *mtgjson.Card) bool {
	switch card.Name {
	case "Laquatus's Champion":
		if Contains(inCard.Variation, "dark") {
			if card.Number != "67†a" {
				return true
			}
		} else if Contains(inCard.Variation, "misprint") {
			if card.Number != "67†" {
				return true
			}
		} else {
			if card.Number != "67" {
				return true
			}
		}
	}
	return false
}

func sldVariant(inCard *Card, card *mtgjson.Card) bool {
	switch card.Name {
	case "Demonlord Belzenlok",
		"Griselbrand",
		"Liliana's Contract",
		"Kothophed, Soul Hoarder",
		"Razaketh, the Foulblooded":
		if inCard.isEtched() {
			return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Plague Sliver",
		"Shadowborn Apostle",
		"Toxin Sliver",
		"Virulent Sliver":
		if inCard.isStepAndCompleat() {
			return !strings.HasSuffix(card.Number, "Φ")
		}
	}
	return false
}

func wcdNumberCompare(inCard *Card, card *mtgjson.Card) bool {
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

func lubuPrereleaseVariant(inCard *Card, card *mtgjson.Card) bool {
	if (strings.Contains(inCard.Variation, "April") || strings.Contains(inCard.Variation, "4/29/1999")) && card.OriginalReleaseDate != "1999-04-29" {
		return true
	} else if (strings.Contains(inCard.Variation, "July") || strings.Contains(inCard.Variation, "7/4/1999")) && card.OriginalReleaseDate != "1999-07-04" {
		return true
	}
	return false
}

func showcaseCheck(inCard *Card, card *mtgjson.Card) bool {
	if inCard.isShowcase() && !card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
		return true
	} else if !inCard.isShowcase() && card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
		return true
	}
	return false
}
