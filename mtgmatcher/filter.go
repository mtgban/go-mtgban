package mtgmatcher

import (
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

// Remove any unrelated edition from the input array.
func filterPrintings(inCard *Card, editions []string) (printings []string) {
	maybeYear := ExtractYear(inCard.Variation)
	if maybeYear == "" {
		maybeYear = ExtractYear(inCard.Edition)
	}

	for _, setCode := range editions {
		set, found := backend.Sets[setCode]
		if !found || set.IsOnlineOnly || setCode == "PRED" {
			continue
		}

		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		switch {
		// If the edition matches, use it as is
		case Equals(inCard.Edition, set.Name):
			// pass-through

		case inCard.isPrerelease():
			switch set.Name {
			case "Duels of the Planeswalkers 2012 Promos",
				"Grand Prix Promos",
				"Pro Tour Promos",
				"Resale Promos",
				"World Championship Promos":
				continue
			case "Prerelease Events":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isPromoPack():
			switch set.Name {
			case "Dragon's Maze Promos", // due to Plains
				"Grand Prix Promos":
				continue
			case "M20 Promo Packs":
			default:
				switch {
				case strings.HasSuffix(set.Name, "Promos"):
				case setDate.After(PromosForEverybodyYay) && set.Type == "expansion":
					skip := true
					foundCards := matchInSet(inCard, setCode)
					for _, card := range foundCards {
						if card.HasPromoType(mtgjson.PromoTypePromoPack) {
							skip = false
							break
						}
					}
					if skip {
						continue
					}
				default:
					continue
				}
			}

		case inCard.isRelease():
			switch set.Name {
			case "Launch Parties",
				"Promotional Planes",
				"Release Events":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isBaB():
			switch set.Name {
			case "Launch Parties",
				"Modern Horizons":
			case "Pro Tour Promos":
				continue
			default:
				switch {
				case setDate.After(BuyABoxInExpansionSetsDate) &&
					(set.Type == "expansion" || set.Type == "core"):
				case strings.HasSuffix(set.Name, "Promos"):
				default:
					continue
				}
			}

		case inCard.isBundle():
			switch set.Name {
			case "Core Set 2020 Promos":
			default:
				switch {
				case setDate.After(BuyABoxInExpansionSetsDate) &&
					set.Type == "expansion":
				default:
					continue
				}
			}

		case inCard.isFNM():
			switch {
			case strings.HasPrefix(set.Name, "Friday Night Magic "+maybeYear):
			case strings.HasSuffix(set.Name, "Promos"):
				skip := true
				foundCards := matchInSet(inCard, setCode)
				for _, card := range foundCards {
					if card.HasFrameEffect(mtgjson.FrameEffectInverted) {
						inCard.Variation = "FNM Promo"
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			default:
				continue
			}

		case strings.Contains(strings.ToLower(inCard.Variation), "judge"):
			if maybeYear == "" {
				if inCard.isGenericExtendedArt() {
					maybeYear = "2014"
				}
			}
			switch {
			case strings.HasPrefix(set.Name, "Judge Gift Cards "+maybeYear):
			default:
				continue
			}

		case inCard.isArena():
			maybeYear = inCard.arenaYear(maybeYear)
			switch {
			case set.Name == "DCI Legend Membership":
			case strings.HasPrefix(set.Name, "Arena League "+maybeYear):
			default:
				continue
			}

		// Some providers use "Textless" for MF cards
		case inCard.isRewards() && !inCard.isMagicFest():
			switch {
			case strings.HasPrefix(set.Name, "Magic Player Rewards "+maybeYear):
			default:
				continue
			}

		case inCard.isWPNGateway():
			switch set.Name {
			case "Summer of Magic",
				"Promotional Planes":
			default:
				switch {
				case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
				case strings.HasPrefix(set.Name, "Gateway "+maybeYear):
				default:
					continue
				}
			}

		// The JPN handling is due to Duress being present in IDW and Magazine Inserts
		case inCard.isIDWMagazineBook():
			switch {
			case !inCard.isJPN() && strings.HasPrefix(set.Name, "IDW Comics "+maybeYear):
			default:
				switch set.Name {
				case "HarperPrism Book Promos",
					"DCI Legend Membership",
					"Miscellaneous Book Promos":
				case "Magazine Inserts":
					skip := false
					foundCards := matchInSet(inCard, setCode)
					for _, card := range foundCards {
						if !inCard.isJPN() && card.HasUniqueLanguage(mtgjson.LanguageJapanese) {
							skip = true
							break
						}
					}
					if skip {
						continue
					}

				default:
					continue
				}
			}

		case strings.Contains(inCard.Variation, "Clash") ||
			(strings.Contains(inCard.Edition, "Clash") &&
				!strings.Contains(inCard.Edition, "Intro")): //cfb
			switch set.Name {
			case "Fate Reforged Clash Pack",
				"Magic 2015 Clash Pack",
				"Magic Origins Clash Pack":
			default:
				continue
			}

		case (strings.Contains(inCard.Variation, "Hero") && strings.Contains(inCard.Variation, "Path")) ||
			(strings.Contains(inCard.Edition, "Hero") && strings.Contains(inCard.Edition, "Path")):
			switch set.Name {
			case "Born of the Gods Hero's Path",
				"Journey into Nyx Hero's Path",
				"Theros Hero's Path":
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "Convention"):
			switch set.Name {
			case "URL/Convention Promos":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isWorldChamp():
			switch {
			case (maybeYear == "1996" || maybeYear == "") && set.Name == "Pro Tour Collector Set":
			case maybeYear != "" && strings.HasPrefix(set.Name, "World Championship Decks "+maybeYear):
			default:
				continue
			}

		case inCard.isMagicFest():
			// Some providers use GP2018 instead of MF2019
			if (maybeYear == "" || maybeYear == "2018") && inCard.isBasicLand() {
				maybeYear = "2019"
			}
			switch {
			case strings.HasPrefix(set.Name, "MagicFest "+maybeYear):
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "SDCC") ||
			Contains(inCard.Variation, "San Diego Comic-Con"):
			switch {
			case strings.HasPrefix(set.Name, "San Diego Comic-Con "+maybeYear):
			default:
				continue
			}

		case inCard.isDuelsOfThePW():
			switch {
			case strings.HasPrefix(set.Name, "Duels of the Planeswalkers"):
			default:
				continue
			}

		// DDA with deck variant specified in the variation
		case inCard.isDuelDecksAnthology():
			switch {
			case strings.HasPrefix(set.Name, "Duel Decks Anthology"):
				fields := strings.Fields(inCard.Variation)
				found := false
				for _, field := range fields {
					if len(field) < 4 {
						continue
					}
					if Contains(set.Name, field) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			default:
				continue
			}

		case inCard.isDuelDecks():
			variant := inCard.duelDecksVariant()
			switch {
			case strings.HasPrefix(set.Name, "Duel Decks") &&
				!strings.Contains(set.Name, "Anthology"):
				if !Contains(set.Name, variant) {
					continue
				}
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "Champs") ||
			strings.Contains(inCard.Variation, "States"):
			switch set.Name {
			case "Champs and States",
				"Gateway 2007":
			case "Grand Prix Promos":
				continue
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isPremiereShop():
			if maybeYear == "" {
				guilds := []string{
					"Azorius",
					"Boros",
					"Dimir",
					"Golgari",
					"Gruul",
					"Izzet",
					"Orzhov",
					"Rakdos",
					"Selesnya",
					"Simic",
				}
				for _, guild := range guilds {
					if strings.Contains(inCard.Variation, guild) {
						maybeYear = "2005"
						break
					}
				}
			}
			switch {
			case strings.HasPrefix(set.Name, "Magic Premiere Shop "+maybeYear):
			default:
				continue
			}

		case inCard.isBasicLand() && strings.Contains(inCard.Variation, "APAC"):
			if set.Name != "Asia Pacific Land Program" {
				continue
			}

		case inCard.isBasicLand() && Contains(inCard.Variation, "EURO"):
			if set.Name != "European Land Program" {
				continue
			}

		case strings.Contains(inCard.Edition, "Core Set") ||
			strings.Contains(inCard.Edition, "Core 20") ||
			strings.Contains(inCard.Edition, "Magic 20"):
			switch {
			case !inCard.isGenericPromo() && strings.HasSuffix(set.Name, "Promos"):
				continue
			case strings.HasPrefix(set.Name, "Core Set "+maybeYear):
			case strings.HasPrefix(set.Name, "Magic "+maybeYear):
			default:
				continue
			}
		}

		printings = append(printings, setCode)
	}

	return
}

// Deduplicate cards with the same name.
func filterCards(inCard *Card, cardSet map[string][]mtgjson.Card) (outCards []mtgjson.Card, foundCode []string) {
	for setCode, inCards := range cardSet {
		set := backend.Sets[setCode]
		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		for _, card := range inCards {
			// Super lucky case, we were expecting the card
			num, found := VariantsTable[set.Name][card.Name][strings.ToLower(inCard.Variation)]
			if found {
				if num == card.Number {
					outCards = append(outCards, card)
					foundCode = append(foundCode, setCode)
				}

				// If a variant is expected we assume that all cases are covered
				continue
			}

			// Handle full vs nonfull art basic lands, helps number parsing below
			if inCard.isBasicFullArt() && !card.IsFullArt {
				continue
			} else if inCard.isBasicNonFullArt() && card.IsFullArt {
				continue
			}

			// Lucky case, variation is just the collector number
			num = ExtractNumber(inCard.Variation)
			// But first, special handling for WCD (skip if player details are missing)
			if inCard.isWorldChamp() {
				prefix, sideboard := inCard.worldChampPrefix()
				if prefix != "" {
					// Copy this field so we can discard portions that have
					// already been used for deduplication
					cn := card.Number
					if sideboard && !strings.HasSuffix(cn, "sb") {
						continue
					} else if !sideboard && strings.HasSuffix(cn, "sb") {
						continue
					}
					cn = strings.Replace(cn, "sb", "", 1)

					// ML and MLP conflict with HasPrefix, so strip away
					// the numeric part and do a straight equal
					idx := strings.IndexFunc(cn, func(c rune) bool {
						return unicode.IsDigit(c)
					})
					if idx < 1 || prefix != cn[:idx] {
						continue
					}
					cn = strings.Replace(cn, prefix, "", 1)

					// Coming straight from ExtractNumber above
					if num != "" {
						cnn := cn
						if unicode.IsLetter(rune(cn[len(cn)-1])) {
							cnn = cn[:len(cn)-1]
						}
						if num != cnn {
							continue
						}
						cn = strings.Replace(cn, num, "", 1)
					}

					if len(cn) > 0 && unicode.IsLetter(rune(cn[len(cn)-1])) {
						suffix := inCard.possibleNumberSuffix()
						if suffix != "" && !strings.HasSuffix(cn, suffix) {
							continue
						}
					}
				} else {
					// Try looking at the collector number if it is in the correct form
					if inCard.Variation != "" &&
						!strings.Contains(inCard.Variation, "-") &&
						unicode.IsLetter(rune(inCard.Variation[0])) {
						ok := false
						for _, letter := range inCard.Variation {
							if unicode.IsDigit(rune(letter)) {
								ok = true
								break
							}
						}
						if ok && card.Number != inCard.Variation {
							continue
						}
					}
				}
			} else if num != "" {
				// The empty string will allow to test the number without any
				// additional prefix first
				possibleSuffixes := []string{""}
				variation := strings.Replace(inCard.Variation, "-", "", 1)
				fields := strings.Fields(strings.ToLower(variation))
				possibleSuffixes = append(possibleSuffixes, fields...)

				// Short circuit the possible suffixes if we know what we're dealing with
				if inCard.isJPN() {
					possibleSuffixes = []string{mtgjson.SuffixSpecial, "s" + mtgjson.SuffixSpecial}
				}
				// BFZ and ZEN intro lands non-fullart always have this
				if inCard.isBasicNonFullArt() {
					possibleSuffixes = append(possibleSuffixes, "a")
				}
				for _, numSuffix := range possibleSuffixes {
					// The self test is already expressed by the empty string
					// This avoids an odd case of testing 1.1 = 11
					if num == numSuffix {
						continue
					}
					number := num + numSuffix
					if number == card.Number {
						outCards = append(outCards, card)
						foundCode = append(foundCode, setCode)

						// Card was found, skip any other suffix
						break
					}
				}

				// If a variant is a number we expect that this information is
				// reliable, so skip anything else
				continue
			}

			// JPN
			if inCard.isJPN() && set.Name != "Magazine Inserts" {
				if !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
					continue
				}
			} else {
				if strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) && card.HasUniqueLanguage(mtgjson.LanguageJapanese) {
					continue
				}
			}

			// Prerelease
			if inCard.isPrerelease() {
				if !card.HasPromoType(mtgjson.PromoTypePrerelease) {
					continue
				}
				if card.Name == "Lu Bu, Master-at-Arms" {
					if strings.Contains(inCard.Variation, "April") && card.Number != "6" {
						continue
					} else if strings.Contains(inCard.Variation, "July") && card.Number != "8" {
						continue
					}
				}
			} else {
				if card.HasPromoType(mtgjson.PromoTypePrerelease) {
					continue
				}
				// Skip some Simplified Chinese cards that have different art
				// This should probably check for ZHS language, but the ForeignData
				// array is empty - as are cards from Prerelease, so check it here
				if len(card.ForeignData) == 0 && strings.HasSuffix(card.Number, "s") {
					continue
				}
			}

			// Promo pack
			if inCard.isPromoPack() && !card.HasPromoType(mtgjson.PromoTypePromoPack) {
				continue
			} else if !inCard.isPromoPack() && card.HasPromoType(mtgjson.PromoTypePromoPack) {
				continue
			}

			if setDate.After(PromosForEverybodyYay) {
				// ELD-Style borderless
				if inCard.isBorderless() {
					if card.BorderColor != mtgjson.BorderColorBorderless {
						continue
					}
				} else {
					// IKO may have showcase cards which happen to be borderless
					// or reskinned ones
					if card.BorderColor == mtgjson.BorderColorBorderless && !card.HasFrameEffect(mtgjson.FrameEffectShowcase) && card.FlavorName == "" {
						continue
					}
				}

				// ELD-Style extended art
				if inCard.isExtendedArt() {
					if !card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
						continue
					}
				} else {
					if card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
						continue
					}
				}

				// ELD-Style showcase
				if inCard.isShowcase() {
					if !card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
						continue
					}
				} else {
					if card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
						continue
					}
				}

				// IKO-Style cards with different names
				if inCard.isReskin() {
					if card.FlavorName == "" {
						continue
					}
				} else {
					if card.FlavorName != "" {
						continue
					}
				}

				// ELD-Style bundle
				if inCard.isBundle() && !card.HasPromoType(mtgjson.PromoTypeBundle) {
					continue
				} else if !inCard.isBundle() && card.HasPromoType(mtgjson.PromoTypeBundle) {
					continue
				}
			}

			// Special sets
			switch set.Name {
			// Light/Dark mana cost
			case "Arabian Nights":
				if inCard.isARNLightMana() {
					if !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana) {
						continue
					}
				} else if inCard.isARNDarkMana() || inCard.Variation == "" {
					if strings.HasSuffix(card.Number, mtgjson.SuffixLightMana) {
						continue
					}
				}
			// Try flavor text or artist fields for these sets (when variation is set)
			case "Alliances",
				"Arena League 2001",
				"Asia Pacific Land Program",
				"Commander Anthology Volume II",
				"European Land Program",
				"Fallen Empires",
				"Homelands":
				// Skip the check if this tag is empty, so that users can notice
				// there is an aliasing problem
				if inCard.Variation == "" {
					continue
				}

				// Since the check is field by field Foglio may alias Phil or Kaja
				variation := inCard.Variation
				if strings.Contains(inCard.Variation, "Foglio") {
					variation = strings.Replace(inCard.Variation, " Foglio", ":Foglio", 1)
				}

				fields := strings.Fields(variation)
				found := false

				// Keep flavor text author only
				flavor := card.FlavorText
				if strings.Contains(flavor, "—") {
					fields := strings.Split(flavor, "—")
					flavor = fields[len(fields)-1]
				}

				// Check field by field, it's usually enough for just two elements
				for _, field := range fields {
					// Skip short text like 'jr.' since they are often missed
					// Skip Land too for High and Low lands alias
					// Skip Sass due to the fact that 's' are ignored in NormContains
					if len(field) < 4 || strings.HasPrefix(field, "Land") || field == "Sass" {
						continue
					}
					if Contains(flavor, field) || Contains(card.Artist, field) {
						found = true
						break
					}
				}

				if !found && len(card.Variations) > 0 {
					numberSuffix := inCard.possibleNumberSuffix()
					if numberSuffix == "" ||
						(numberSuffix != "" && !strings.HasSuffix(card.Number, numberSuffix)) {
						continue
					}
				}
			// Check watermark when variation has no number information
			case "Magic Premiere Shop 2005",
				"GRN Guild Kit",
				"RNA Guild Kit":
				// Skip the check if this tag is empty, so that users can notice
				// there is an aliasing problem
				if inCard.Variation == "" {
					continue
				}

				if !Contains(inCard.Variation, card.Watermark) {
					continue
				}
			// Foil-only-booster cards, non-special version has both foil and non-foil
			case "Planeshift":
				if inCard.isGenericAltArt() && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
					continue
				} else if !inCard.isGenericAltArt() && strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
					continue
				}
			// Foil-only-booster cards, non-special version only have non-foil
			// (only works if card has no other duplicates within the same edition)
			case "Battlebond",
				"Conspiracy: Take the Crown",
				"Tenth Edition",
				"Unhinged":
				if inCard.Foil && card.HasNonFoil {
					continue
				} else if !inCard.Foil && card.HasFoil {
					continue
				}
			// Single letter variants
			case "Deckmasters",
				"Unstable":
				numberSuffix := inCard.possibleNumberSuffix()

				if len(card.Variations) > 0 && numberSuffix == "" {
					numberSuffix = "a"

					if set.Name == "Deckmasters" {
						if inCard.Foil || inCard.isGenericPromo() {
							numberSuffix = mtgjson.SuffixSpecial
						} else if card.HasNonFoil &&
							(card.Name == "Incinerate" || card.Name == "Icy Manipulator") {
							numberSuffix = ""
						}
					}
				}
				if numberSuffix != "" {
					if !strings.HasSuffix(card.Number, numberSuffix) {
						continue
					}
				}
			// Launch promos within the set itself
			case "Double Masters",
				"Jumpstart":
				if inCard.isRelease() && !card.IsAlternative {
					continue
				} else if !inCard.isRelease() && card.IsAlternative {
					continue
				}
			default:
				// Variants/misprints have different suffixes depending on foil or style
				expectedSuffix := mtgjson.SuffixVariant

				// Officially known misprints or just variants
				variation := inCard.Variation
				if card.Name == "Temple of Abandon" && set.Name == "Theros Beyond Death" && inCard.isExtendedArt() {
					expectedSuffix = mtgjson.SuffixSpecial
					if inCard.Foil {
						variation = "misprint"
					}
				} else if card.Name == "Reflecting Pool" && set.Name == "Shadowmoor" {
					expectedSuffix = mtgjson.SuffixSpecial
					if inCard.Foil {
						variation = "misprint"
					}
				} else if inCard.isPortalAlt() && set.Name == "Portal" {
					variation = "misprint"
				} else if inCard.Name == "Void Beckoner" {
					expectedSuffix = "A"
					if inCard.isReskin() {
						variation = "misprint"
					}
				} else if inCard.Name == "Island" && set.Name == "Arena League 1999" && Contains(inCard.Variation, "NO SYMBOL") {
					variation = "misprint"
				}

				if Contains(variation, "misprint") && !strings.HasSuffix(card.Number, expectedSuffix) {
					continue
				} else if !Contains(variation, "misprint") && strings.HasSuffix(card.Number, expectedSuffix) {
					continue
				}
			}

			outCards = append(outCards, card)
			foundCode = append(foundCode, setCode)
		}
	}

	// Check if there are multiple printings for Prerelease and Promo Pack cards
	// Sometimes these contain the ParentCode or the parent edition name in the field
	if len(outCards) > 1 && (inCard.isPrerelease() || inCard.isPromoPack()) {
		allSameEdition := true
		for i, code := range foundCode {
			if outCards[i].Name != outCards[0].Name || !strings.HasPrefix(code, "P") {
				allSameEdition = false
				break
			}
		}

		// All of my hate
		if outCards[0].Name == "Teferi, Master of Time" {
			allSameEdition = false
		}

		if allSameEdition {
			size := len(foundCode)
			for i := 0; i < size; i++ {
				set := backend.Sets[foundCode[i]]
				// Drop any printing that don't have the ParentCode
				// or the edition name itself in the Variation field
				if !(strings.Contains(inCard.Variation, set.ParentCode) ||
					strings.Contains(inCard.Variation, backend.Sets[set.ParentCode].Name) ||
					strings.Contains(inCard.Edition, backend.Sets[set.ParentCode].Name)) {
					foundCode = append(foundCode[:i], foundCode[i+1:]...)
					outCards = append(outCards[:i], outCards[i+1:]...)
					size--
					i = 0
				}
			}
		}
	}

	return
}
