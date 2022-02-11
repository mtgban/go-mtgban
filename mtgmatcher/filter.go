package mtgmatcher

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

// Remove any unrelated edition from the input array.
func filterPrintings(inCard *Card, editions []string) (printings []string) {
	maybeYear := ExtractYear(inCard.Variation)
	if maybeYear == "" {
		maybeYear = ExtractYear(inCard.Edition)
	}

	for _, setCode := range editions {
		set, found := backend.Sets[setCode]
		if !found {
			continue
		}

		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		switch {
		// If the edition matches, use it as is
		case Equals(inCard.Edition, set.Name):
			// pass-through

		case inCard.isPrerelease():
			switch set.Name {
			// Sets that have prerelease cards mixed in
			case "Open the Helvault",
				"Promotional Planes",
				"Innistrad: Double Feature":
			case "Duels of the Planeswalkers 2012 Promos",
				"Grand Prix Promos",
				"Pro Tour Promos",
				"Resale Promos",
				"World Championship Promos":
				continue
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
				case setDate.After(PromosForEverybodyYay) && (set.Type == "expansion" || set.Type == "core"):
					skip := true
					foundCards := MatchInSet(inCard.Name, setCode)
					for _, card := range foundCards {
						if card.HasPromoType(mtgjson.PromoTypePromoPack) || card.HasPromoType(mtgjson.PromoTypePlayPromo) {
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
			case "Promotional Planes":
			case "Double Masters",
				"Jumpstart":
				switch inCard.Name {
				case "Wrath of God",
					"Chord of Calling",
					"Scholar of the Lost Trove":
				default:
					continue
				}
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isBaB():
			skip := true
			foundCards := MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgjson.PromoTypeBuyABox) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}

		case inCard.isBundle():
			skip := true
			foundCards := MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgjson.PromoTypeBundle) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}

		case inCard.isFNM():
			switch {
			case strings.HasPrefix(set.Name, "Friday Night Magic "+maybeYear):
			case strings.HasSuffix(set.Name, "Promos"):
				skip := true
				foundCards := MatchInSet(inCard.Name, setCode)
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

		case inCard.isJudge():
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

		// This needs to be above any possible printing type below
		case inCard.isMysteryList():
			switch set.Code {
			case "MB1":
				if inCard.Variation == "The List" || inCard.Edition == "The List" ||
					inCard.Foil || (inCard.Contains("Foil") && !inCard.Contains("Non")) {
					continue
				}
			case "FMB1":
				if inCard.Variation == "The List" || inCard.Edition == "The List" ||
					inCard.Contains("Non-Foil") {
					continue
				}
			case "PLIST":
				if inCard.Variation == "Mystery Booster" || inCard.Edition == "Mystery Booster" ||
					inCard.Foil || (inCard.Contains("Foil") && !inCard.Contains("Non") ||
					// Explicitly skip playtest cards unless using the correct edition is used
					// They are visually the same as CMB1 and nobody tracks them separately
					(len(MatchInSet(inCard.Name, "CMB1")) > 0 && inCard.Edition != "The List")) {
					continue
				}
			case "CMB1":
				if inCard.Contains("No PW Symbol") || inCard.Contains("No Symbol") || strings.Contains(inCard.Variation, "V.2") {
					continue
				}
			case "CMB2":
				if !(inCard.Contains("No PW Symbol") || inCard.Contains("No Symbol") || strings.Contains(inCard.Variation, "V.2")) {
					continue
				}
			default:
				continue
			}

		// Some providers use "Textless" for MF cards
		case inCard.isRewards() && !inCard.isMagicFest():
			maybeYear = inCard.playerRewardsYear(maybeYear)
			switch {
			case strings.HasPrefix(set.Name, "Magic Player Rewards "+maybeYear):
			default:
				continue
			}

		case inCard.isWPNGateway():
			switch set.Name {
			case "Summer of Magic",
				"Promotional Planes":
			case "Innistrad: Crimson Vow":
				skip := true
				foundCards := MatchInSet(inCard.Name, "VOW")
				for _, card := range foundCards {
					switch card.Number {
					case "408", "409", "410", "411", "412":
						skip = false
					}
				}
				if skip {
					continue
				}
			default:
				switch {
				case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
				case strings.HasPrefix(set.Name, "Gateway "+maybeYear):
				case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
				default:
					continue
				}
			}

		case inCard.isIDWMagazineBook():
			switch {
			case !inCard.isJPN() && strings.HasPrefix(set.Name, "IDW Comics "+maybeYear):
			case !inCard.isJPN() && strings.HasPrefix(set.Name, "Duels of the Planeswalkers "+maybeYear):
			case !inCard.isJPN() && strings.HasSuffix(set.Name, "Promos"):
				switch set.Name {
				case "Grand Prix Promos",
					"Pro Tour Promos",
					"World Championship Promos":
					continue
				case "HarperPrism Book Promos",
					"Miscellaneous Book Promos",
					"Resale Promos":
					// Relevant sets falling into the HasSuffix above
				}
			default:
				switch set.Name {
				case "DCI Legend Membership":
				case "Media Inserts":
					// This is the only card present in IDW and Media Inserts
					// so make sure it is properly tagged
					if inCard.Name == "Duress" && !inCard.isJPN() {
						continue
					}
				default:
					continue
				}
			}

		case inCard.Contains("Hero") && inCard.Contains("Path"):
			switch set.Name {
			case "Born of the Gods Hero's Path",
				"Journey into Nyx Hero's Path",
				"Journey into Nyx Promos",
				"Theros Hero's Path",
				"Defat a God",
				"Face the Hydra",
				"Battle the Horde":
			default:
				continue
			}

		case inCard.Contains("Convention"):
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
			if maybeYear == "2018" && inCard.isBasicLand() {
				maybeYear = "2019"
			}
			switch {
			case strings.HasPrefix(set.Name, "MagicFest "+maybeYear):
			default:
				continue
			}

		case inCard.isSDCC():
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
				if maybeYear == "" && strings.HasSuffix(inCard.Variation, "Cycle") {
					maybeYear = map[string]string{
						"Time Spiral Cycle":       "2006",
						"Lorwyn Cycle":            "2007",
						"Shards of Alara Cycle":   "2008",
						"Zendikar Cycle":          "2009",
						"Scars of Mirrodin Cycle": "2010",
						"Innistrad Cycle":         "2011",
					}[inCard.Variation]
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

		case inCard.Contains("Grand Prix"):
			switch set.Name {
			case "Grand Prix Promos":
			default:
				if !strings.HasPrefix(set.Name, "MagicFest") {
					continue
				}
			}

		case inCard.Contains("Lunar New Year"):
			switch set.Code {
			case "PLNY", "PL21":
			default:
				continue
			}

		case inCard.Contains("Bring-A-Friend") ||
			inCard.Contains("Love Your LGS") ||
			inCard.Contains("Welcome Back") ||
			inCard.Contains("LGS Promo"):
			switch {
			case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
			// There is a lot overlap in this set
			case set.Name == "Wizards Play Network 2021":
			default:
				continue
			}

		// Last resort, if this is set on the input card, and there were
		// no better descriptors earlier, try looking at the set type
		case inCard.promoWildcard:
			switch set.Type {
			case "promo":
				// Skip Judge promos, they are usually correctly listed
				if strings.HasPrefix(set.Name, "Judge Gift") {
					continue
				}
				skip := false
				foundCards := MatchInSet(inCard.Name, setCode)
				// It is required to set a proper tag to parse non-English
				// cards or well-known promos
				for _, card := range foundCards {
					if card.HasUniqueLanguage(mtgjson.LanguageJapanese) {
						skip = true
						break
					}
				}

				if skip {
					continue
				}
			case "starter":
				if !strings.HasSuffix(set.Name, "Clash Pack") {
					continue
				}
			case "expansion", "core", "masters", "draft_innovation":
				skip := true
				foundCards := MatchInSet(inCard.Name, setCode)
				for _, card := range foundCards {
					// Skip boosterfun because they are inherently non-promo
					if card.IsPromo && !card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			case "box":
				skip := true
				switch setCode {
				// Only keep the planeswalkers from SLD for this category
				case "SLD":
					foundCards := MatchInSet(inCard.Name, setCode)
					for _, card := range foundCards {
						if card.IsPlaneswalker() {
							skip = false
							break
						}
					}
				}
				if skip {
					continue
				}
			case "funny":
				if set.Name != "HasCon 2017" {
					continue
				}
			default:
				continue
			}

		// Tokens need correct set names or special handling earlier
		case (strings.HasSuffix(inCard.Name, "Token") &&
			backend.Cards[Normalize(inCard.Name)].Layout == "token") ||
			(!strings.HasSuffix(inCard.Name, "Token") &&
				backend.Cards[Normalize(inCard.Name+" Token")].Layout == "token"):
			if !Equals(inCard.Edition, set.Name) {
				continue
			}
		}

		printings = append(printings, setCode)
	}

	return
}

// Deduplicate cards with the same name.
func filterCards(inCard *Card, cardSet map[string][]mtgjson.Card) (outCards []mtgjson.Card) {
	for setCode, inCards := range cardSet {
		set := backend.Sets[setCode]
		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		for _, card := range inCards {
			// Super lucky case, we were expecting the card
			num, found := VariantsTable[set.Name][card.Name][strings.ToLower(inCard.Variation)]
			if found {
				if num == card.Number {
					outCards = append(outCards, card)
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
				wcdNum := extractWCDNumber(inCard.Variation, prefix, sideboard)

				// If a wcdNum is found, check that it's matching the card number
				// Else rebuild the number manually using prefix, sideboard, and num as hints
				if wcdNum != "" {
					if wcdNum == card.Number {
						outCards = append(outCards, card)
					}
					// Skip anything else, the number needs to be correct
					continue
				} else if prefix != "" {
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
						// Strip last character if it's a letter
						if unicode.IsLetter(rune(cn[len(cn)-1])) {
							cnn = cn[:len(cn)-1]
						}
						// Try both simple number and original collector number
						if num != cnn && num != cn {
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
				}
			} else if num != "" && !inCard.Contains("Misprint") {
				// The empty string will allow to test the number without any
				// additional prefix first
				possibleSuffixes := []string{""}
				variation := strings.Replace(inCard.Variation, "-", "", 1)
				fields := strings.Fields(strings.ToLower(variation))
				possibleSuffixes = append(possibleSuffixes, fields...)

				// Short circuit the possible suffixes if we know what we're dealing with
				if inCard.isJPN() {
					switch set.Code {
					case "WAR", "PWAR":
						possibleSuffixes = []string{mtgjson.SuffixSpecial, "s" + mtgjson.SuffixSpecial}
					}
				} else if inCard.isPrerelease() {
					possibleSuffixes = append(possibleSuffixes, "s")
				} else if inCard.isPromoPack() {
					possibleSuffixes = append(possibleSuffixes, "p")
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

						// Card was found, skip any other suffix
						break
					}
				}

				// If a variant is a number we expect that this information is
				// reliable, so skip anything else
				continue
			}

			// JPN
			if inCard.isJPN() && set.Code != "PMEI" && !strings.HasPrefix(set.Name, "Magic Premiere Shop") && set.Code != "STA" {
				if !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) && !card.HasUniqueLanguage(mtgjson.LanguageJapanese) {
					continue
				}
			} else {
				if strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) && card.HasUniqueLanguage(mtgjson.LanguageJapanese) {
					continue
				}
			}

			// The last-ditch effort from above - when this is set, only check
			// the non-promo sets as some promos can be mixed next to the
			// normal cards - in this way, promo sets can process as normal and
			// deduplicate all the various prerelease and promo packs
			if inCard.promoWildcard {
				switch set.Type {
				case "expansion", "core", "masters", "draft_innovation":
					if !card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
						// Consider Promos, non-promo Intro Pack cards, and non-promo special cards
						if card.IsPromo || card.HasPromoType(mtgjson.PromoTypeIntroPack) || strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
							outCards = append(outCards, card)
						}
						continue
					}
				}
			}

			// Prerelease
			if inCard.isPrerelease() {
				if !card.HasPromoType(mtgjson.PromoTypePrerelease) {
					continue
				}
				if card.Name == "Lu Bu, Master-at-Arms" {
					if (strings.Contains(inCard.Variation, "April") || strings.Contains(inCard.Variation, "4/29/1999")) && card.OriginalReleaseDate != "1999-04-29" {
						continue
					} else if (strings.Contains(inCard.Variation, "July") || strings.Contains(inCard.Variation, "7/4/1999")) && card.OriginalReleaseDate != "1999-07-04" {
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

			// Promo pack and Play promo
			if inCard.isPromoPack() && !card.HasPromoType(mtgjson.PromoTypePromoPack) && !card.HasPromoType(mtgjson.PromoTypePlayPromo) {
				continue
			} else if !inCard.isPromoPack() && (card.HasPromoType(mtgjson.PromoTypePromoPack) || card.HasPromoType(mtgjson.PromoTypePlayPromo)) {
				continue
			}

			if inCard.beyondBaseSet {
				// Filter out any card that is located in the base set only
				num, err := strconv.Atoi(card.Number)
				if err == nil && num < set.BaseSetSize {
					continue
				}
			} else if setDate.After(PromosForEverybodyYay) && !inCard.isMysteryList() {
				// ELD-Style borderless
				if inCard.isBorderless() {
					if card.BorderColor != mtgjson.BorderColorBorderless {
						continue
					}
				} else {
					// IKO may have showcase cards which happen to be borderless
					// or reskinned ones. Also all cards from STA are borderless,
					// so they are never tagged as such.
					if card.BorderColor == mtgjson.BorderColorBorderless &&
						!card.HasFrameEffect(mtgjson.FrameEffectShowcase) && card.FlavorName == "" &&
						set.Name != "Strixhaven Mystical Archive" {
						continue
					}
				}

				// ELD-Style extended art
				if inCard.isExtendedArt() {
					if !card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) {
						continue
					}
					// BaB are allowed to have extendedart
				} else if !card.HasPromoType(mtgjson.PromoTypeBuyABox) {
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

				// ELD-Style bundle
				if inCard.isBundle() && !card.HasPromoType(mtgjson.PromoTypeBundle) {
					continue
				} else if !inCard.isBundle() && card.HasPromoType(mtgjson.PromoTypeBundle) {
					continue
				}

				// ZNR-Style buy-a-box but card is also present in main set

				if setDate.After(BuyABoxNotUniqueDate) {
					if inCard.isBaB() && !card.HasPromoType(mtgjson.PromoTypeBuyABox) {
						continue
					} else if !inCard.isBaB() && card.HasPromoType(mtgjson.PromoTypeBuyABox) {
						continue
					}
				}
			}

			// Only do this check if we are in a safe parsing status
			if !inCard.beyondBaseSet {
				// IKO-Style cards with different names
				// Needs to be outside of the above block due to promos
				// originally printed in an older edition
				// Also some providers do not tag Japanese-only Godzilla
				// cards as such
				if inCard.isReskin() && !card.HasPromoType(mtgjson.PromoTypeGodzilla) && !card.HasPromoType(mtgjson.PromoTypeDracula) {
					continue
				} else if !inCard.isReskin() && (card.HasPromoType(mtgjson.PromoTypeGodzilla) || card.HasPromoType(mtgjson.PromoTypeDracula)) {
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
				if inCard.Foil && card.HasFinish(mtgjson.FinishNonfoil) {
					continue
				} else if !inCard.Foil && card.HasFinish(mtgjson.FinishFoil) {
					continue
				}
			// Single letter variants
			case "Deckmasters",
				"Unstable":
				numberSuffix := inCard.possibleNumberSuffix()

				if len(card.Variations) > 0 && numberSuffix == "" {
					numberSuffix = "a"

					if set.Name == "Deckmasters" {
						if inCard.Foil || inCard.Contains("Promo") {
							numberSuffix = mtgjson.SuffixSpecial
						} else if card.HasFinish(mtgjson.FinishNonfoil) &&
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
			// Variants related to flavor text presence
			case "Portal":
				if inCard.isPortalAlt() && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
					continue
				} else if !inCard.isPortalAlt() && strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
					continue
				}
			// Launch promos within the set itself
			case "Double Masters",
				"Jumpstart":
				if (inCard.isRelease() || inCard.isBaB()) && !card.IsAlternative {
					continue
				} else if !(inCard.isRelease() || inCard.isBaB()) && card.IsAlternative && !card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
					continue
				}
			// Identical cards
			case "Commander Legends":
				// This set uses a different card frame for etched cards
				if inCard.isEtched() && !card.HasFrameEffect(mtgjson.FrameEffectEtched) {
					continue
				} else if !inCard.isEtched() && card.HasFrameEffect(mtgjson.FrameEffectEtched) {
					continue
				}

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
				if !inCard.isExtendedArt() && hasAlternate {
					if inCard.Variation == "" && card.IsAlternative {
						continue
					} else if inCard.Variation != "" && !card.IsAlternative {
						continue
					}
				}
			// Intro pack
			case "Aether Revolt",
				"Kaladesh":
				if !Contains(inCard.Variation, "Intro") && card.HasPromoType(mtgjson.PromoTypeIntroPack) {
					continue
				} else if Contains(inCard.Variation, "Intro") && !card.HasPromoType(mtgjson.PromoTypeIntroPack) {
					continue
				}
			// Separate JPN and non-JPN cards (foil-etched are parsed above)
			case "Strixhaven Mystical Archive":
				cn, _ := strconv.Atoi(card.Number)
				if (inCard.isJPN() || inCard.isGenericAltArt()) && cn <= 63 {
					continue
				} else if !inCard.isJPN() && cn > 63 {
					continue
				}
			case "Modern Horizons 2":
				cn, _ := strconv.Atoi(card.Number)
				if Contains(inCard.Variation, "Retro") && (cn <= 380 || cn > 441) {
					continue
				} else if !(Contains(inCard.Variation, "Retro") || inCard.beyondBaseSet) && !(cn <= 380 || cn > 441) {
					continue
				}
			// Due to the WPN lands
			case "Innistrad: Crimson Vow":
				if inCard.isWPNGateway() && !card.HasPromoType(mtgjson.PromoTypeWPN) {
					continue
				} else if !inCard.isWPNGateway() && card.HasFinish(mtgjson.PromoTypeWPN) {
					continue
				}
			default:
				// Variants/misprints have different suffixes depending on foil or style
				expectedSuffix := mtgjson.SuffixVariant

				// Always check suffix for misprints
				checkNumberSuffix := inCard.Contains("Misprint")

				// Officially known misprints or just variants
				switch card.Name {
				case "Temple of Abandon":
					if set.Name == "Theros Beyond Death" && inCard.isExtendedArt() {
						expectedSuffix = mtgjson.SuffixSpecial
						checkNumberSuffix = inCard.Foil
					}
				case "Reflecting Pool":
					if set.Name == "Shadowmoor" {
						expectedSuffix = mtgjson.SuffixSpecial
						checkNumberSuffix = inCard.Foil
					}
				case "Void Beckoner":
					if set.Name == "Ikoria: Lair of Behemoths" {
						expectedSuffix = "A"
						checkNumberSuffix = inCard.isReskin()
					}
				case "Island":
					if set.Name == "Arena League 1999" && Contains(inCard.Variation, "NO SYMBOL") {
						checkNumberSuffix = true
					}
				case "Laquatus's Champion":
					if set.Name == "Torment Promos" {
						if Contains(inCard.Variation, "dark") {
							if card.Number != "67†a" {
								continue
							}
						} else if Contains(inCard.Variation, "misprint") {
							if card.Number != "67†" {
								continue
							}
						} else {
							if card.Number != "67" {
								continue
							}
						}
						// Make below check pass, we already filtered above
						checkNumberSuffix = true
						expectedSuffix = card.Number
					}
				case "Demonlord Belzenlok",
					"Griselbrand",
					"Liliana's Contract",
					"Kothophed, Soul Hoarder",
					"Razaketh, the Foulblooded":
					if set.Name == "Secret Lair Drop" {
						expectedSuffix = mtgjson.SuffixSpecial
						checkNumberSuffix = inCard.isEtched()
					}
				}

				if checkNumberSuffix && !strings.HasSuffix(card.Number, expectedSuffix) {
					continue
				} else if !checkNumberSuffix && strings.HasSuffix(card.Number, expectedSuffix) {
					continue
				}
			}

			outCards = append(outCards, card)
		}
	}

	// Check if there are multiple printings for Prerelease and Promo Pack cards
	// Sometimes these contain the ParentCode or the parent edition name in the field
	if len(outCards) > 1 && (inCard.isPrerelease() || inCard.isPromoPack()) {
		allSameEdition := true
		for _, card := range outCards {
			if card.Name != outCards[0].Name || !strings.HasPrefix(card.SetCode, "P") {
				allSameEdition = false
				break
			}
		}

		if allSameEdition {
			var filteredOutCards []mtgjson.Card
			for _, card := range outCards {
				set := backend.Sets[card.SetCode]
				// The year is necessary to decouple PM20 and PM21 cards
				// when the edition name is different than the canonical one
				// ie "Core 2021" instead of "Core Set 2021"
				year := ExtractYear(set.Name)
				// Drop any printing that don't have the ParentCode
				// or the edition name itself in the Variation field
				// or the year matches across
				if strings.Contains(inCard.Variation, set.ParentCode) ||
					inCard.Contains(backend.Sets[set.ParentCode].Name) ||
					(year != "" && inCard.Contains(year)) {
					filteredOutCards = append(filteredOutCards, card)
				}
			}
			outCards = filteredOutCards
		}
	}
	// If card is indistinguishable across MB1 or PLIST, select PLIST
	// Duplicate MB1 cards should have an entry in variants.go
	if len(outCards) > 1 && inCard.isMysteryList() {
		var filteredOutCards []mtgjson.Card
		for _, card := range outCards {
			if card.SetCode != "PLIST" {
				continue
			}
			filteredOutCards = append(filteredOutCards, card)
		}
		outCards = filteredOutCards
	}

	return
}
