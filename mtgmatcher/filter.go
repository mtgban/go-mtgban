package mtgmatcher

import (
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
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
			case "30th Anniversary History Japanese Promos",
				"30th Anniversary History Promos",
				"30th Anniversary Play Promos":
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
				case strings.HasPrefix(set.Name, "30th Anniversary"):
					continue
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
				"Jumpstart",
				"Double Masters 2022",
				"Dominaria Remastered",
				"Warhammer 40,000 Commander":
				// If the list of cards is present in any other edition they need special casing
				switch inCard.Name {
				case "Chord of Calling",
					"Scholar of the Lost Trove",
					"Weathered Wayfarer",
					"Bring to Light",
					"Counterspell",
					"Fabricate":
				case "Wrath of God":
					if set.Name != "Double Masters" {
						continue
					}
				default:
					continue
				}
			case "30th Anniversary History Japanese Promos",
				"30th Anniversary History Promos",
				"30th Anniversary Play Promos":
				continue
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
					inCard.Edition == "Heads I Win, Tails You Lose" ||
					inCard.Edition == "From Cute to Brute" ||
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
					inCard.Edition == "Heads I Win, Tails You Lose" ||
					inCard.Edition == "From Cute to Brute" ||
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
			case "PHED":
				// If the card is not foil, and has been printed somewhere else,
				// only pick this edition if explicilty requested
				if len(MatchInSet(inCard.Name, "MB1")) > 0 || len(MatchInSet(inCard.Name, "PLIST")) > 0 {
					if inCard.Edition != "Heads I Win, Tails You Lose" {
						// Except the following cards, when they are not tagged as specified,
						// it means they are actually from this set
						switch inCard.Name {
						case "Sol Ring",
							"Reliquary Tower":
							if !inCard.Contains("2021") {
								continue
							}
						case "Counterspell",
							"Temur Battle Rage":
							if !inCard.Contains("Legends") {
								continue
							}
						case "Island", "Mountain":
							if !inCard.Contains("Battlebond") {
								continue
							}
						default:
							if !inCard.Foil {
								continue
							}
						}
					}
				}
			case "PCTB":
				// If the card is not foil, and has been printed somewhere else,
				// only pick this edition if explicilty requested
				if len(MatchInSet(inCard.Name, "MB1")) > 0 || len(MatchInSet(inCard.Name, "PLIST")) > 0 {
					if inCard.Edition != "From Cute to Brute" {
						continue
					}
				}
			case "UPLIST":
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
			case "DCI Promos":
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
				case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
				default:
					continue
				}
			}

		case inCard.isIDWMagazineBook():
			switch {
			case strings.HasPrefix(set.Name, "30th Anniversary"):
				continue
			case !inCard.isJPN() && set.Name == "IDW Comics Inserts":
			case !inCard.isJPN() && strings.HasPrefix(set.Name, "Duels of the Planeswalkers "+maybeYear):
			case !inCard.isJPN() && strings.HasSuffix(set.Name, "Promos"):
				switch set.Name {
				case "Grand Prix Promos",
					"Planeswalker Championship Promos",
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
			case "30th Anniversary History Japanese Promos",
				"30th Anniversary History Promos",
				"30th Anniversary Play Promos":
				continue
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.Contains("Premier Play"):
			switch {
			case set.Name == "Pro Tour Promos":
			case strings.HasPrefix(set.Name, "Regional Championship Qualifiers "+maybeYear):
			default:
				continue
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
			case set.Code == "P30A":
				if inCard.Name != "Arcane Signet" && inCard.Name != "Richard Garfield, Ph.D." {
					continue
				}
			case set.Code == "PLG21":
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
			case "Champs and States":
			case "Grand Prix Promos":
				continue
			case "30th Anniversary History Japanese Promos",
				"30th Anniversary History Promos",
				"30th Anniversary Play Promos":
				continue
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isPremiereShop():
			if maybeYear == "" {
				guilds := append(GRNGuilds, ARNGuilds...)
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
			case "PLNY":
			default:
				if !strings.HasPrefix(set.Name, "Year of the ") {
					continue
				}
			}

		case inCard.isThickDisplay():
			switch set.Code {
			// The sets with thick display cards separate from the main commander set
			case "OC21", "OAFC", "OMIC", "OVOC":
			// SLD may contain DFC with thick display
			case "SLD":
			default:
				// Skip any set before this date if not from the sets above
				if setDate.Before(SeparateFinishCollectorNumberDate) {
					continue
				}
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

		case inCard.Contains("30th Anniversary"):
			switch set.Code {
			case "P30A", "P30H":
				if inCard.isJPN() {
					continue
				}
			case "P30HJPN":
				if !inCard.isJPN() {
					continue
				}
			default:
				continue
			}

		case inCard.Contains("Planeswalker") && inCard.Contains("Promos"):
			switch set.Code {
			case "PWCS":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		// For all the promos with "Extended Art" which refer to the full art promo
		case inCard.isExtendedArt() && !inCard.Contains("Game Day"):
			if setDate.Before(PromosForEverybodyYay) {
				switch set.Code {
				case "PDCI":
				default:
					continue
				}
			}

		// Last resort, if this is set on the input card, and there were
		// no better descriptors earlier, try looking at the set type
		case inCard.promoWildcard:
			switch set.Type {
			case "promo":
				// Skip common promos, they are usually correctly listed
				if strings.HasPrefix(set.Name, "Judge Gift") ||
					strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
				skip := false
				foundCards := MatchInSet(inCard.Name, setCode)
				// It is required to set a proper tag to parse non-English
				// cards or well-known promos
				for _, card := range foundCards {
					if card.Language == mtgjson.LanguageJapanese {
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

			// Support the Simplified Chinese Alternative Art Cards
			// Needs to be before number parsing to handle number variants
			schineseTag := inCard.Contains("Chinese") || (strings.Contains(inCard.Variation, "CS") && inCard.isGenericAltArt())
			if schineseTag && !card.HasPromoType(mtgjson.PromoTypeSChineseAltArt) {
				continue
			} else if !schineseTag && card.HasPromoType(mtgjson.PromoTypeSChineseAltArt) {
				continue
			}

			checkNum := true
			if inCard.Contains("Misprint") ||
				inCard.isWorldChamp() ||
				(card.AttractionLights != nil && strings.Contains(inCard.Variation, "/")) {
				checkNum = false
			}
			if checkNum {
				// Lucky case, variation is just the collector number
				num = ExtractNumber(inCard.Variation)
				if num != "" {
					// The empty string will allow to test the number without any
					// additional prefix first
					possibleSuffixes := []string{""}
					variation := strings.Replace(inCard.Variation, "-", "", 1)
					fields := strings.Fields(strings.ToLower(variation))
					possibleSuffixes = append(possibleSuffixes, fields...)

					// Short circuit the possible suffixes if we know what we're dealing with
					if inCard.isJPN() {
						switch set.Code {
						case "WAR":
							possibleSuffixes = []string{mtgjson.SuffixSpecial}
						case "PWAR":
							possibleSuffixes = []string{"s" + mtgjson.SuffixSpecial}
						}
					} else if inCard.isPrerelease() {
						possibleSuffixes = append(possibleSuffixes, "s")
					} else if inCard.isPromoPack() {
						possibleSuffixes = append(possibleSuffixes, "p")
					} else if schineseTag {
						possibleSuffixes = []string{"s"}
					} else if inCard.Contains("Serial") {
						switch set.Code {
						case "SLD", "MOM":
						default:
							possibleSuffixes = []string{"z"}
						}
					} else if inCard.isStepAndCompleat() && set.Code == "SLD" {
						possibleSuffixes = []string{"φ"}
					}

					// BFZ and ZEN intro lands non-fullart always have this
					if inCard.isBasicNonFullArt() {
						possibleSuffixes = append(possibleSuffixes, "a")
					}

					// 40K could have numbers reported alongside the surge tag
					if inCard.isSurgeFoil() && !inCard.isThickDisplay() {
						// Exclude the first 8 cards that do not have the special suffix
						cn, err := strconv.Atoi(card.Number)
						if err != nil || cn > 8 {
							possibleSuffixes = []string{mtgjson.SuffixSpecial}
						}
					}

					// Some editions duplicate foil and nonfoil in the same set
					if inCard.Foil {
						switch set.Code {
						case "7ED", "8ED", "9ED":
							possibleSuffixes = []string{mtgjson.SuffixSpecial}
						case "10E", "UNH":
							possibleSuffixes = []string{"", mtgjson.SuffixSpecial}
						}
					}

					for _, numSuffix := range possibleSuffixes {
						// The self test is already expressed by the empty string
						// This avoids an odd case of testing 1.1 = 11
						if num == numSuffix {
							continue
						}
						number := num
						if numSuffix != "" && !strings.HasSuffix(number, numSuffix) {
							number += numSuffix
						}
						if number == strings.ToLower(card.Number) {
							// Repeat promo pack check for sets where "p" and "" may be mixed
							if strings.HasSuffix(set.Name, "Promos") {
								if inCard.isPromoPack() && !card.HasPromoType(mtgjson.PromoTypePromoPack) {
									continue
								} else if !inCard.isPromoPack() && card.HasPromoType(mtgjson.PromoTypePromoPack) {
									continue
								}
							}

							outCards = append(outCards, card)

							// Card was found, skip any other suffix
							break
						}
					}

					// If a variant is a number we expect that this information is
					// reliable, so skip anything else
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
					if !card.HasPromoType(mtgjson.PromoTypeBoosterfun) && !card.HasPromoType(mtgjson.PromoTypePromoPack) && !card.HasPromoType(mtgjson.PromoTypeStarterDeck) {
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
				// MAT has prerelease cards with showcase tag
			} else if !inCard.isShowcase() {
				if card.HasPromoType(mtgjson.PromoTypePrerelease) {
					continue
				}
			}

			// Promo pack and Play promo
			if inCard.isPromoPack() && !card.HasPromoType(mtgjson.PromoTypePromoPack) {
				continue
			} else if !inCard.isPromoPack() && card.HasPromoType(mtgjson.PromoTypePromoPack) {
				continue
			}
			if inCard.isPlayPromo() && !card.HasPromoType(mtgjson.PromoTypePlayPromo) {
				continue
			} else if !inCard.isPlayPromo() && card.HasPromoType(mtgjson.PromoTypePlayPromo) {
				continue
			}

			if inCard.beyondBaseSet {
				// Filter out any card that is located in the base set only
				num, err := strconv.Atoi(card.Number)
				if err == nil && num < set.BaseSetSize {
					continue
				}
			} else if (setDate.After(PromosForEverybodyYay) || set.Code == "ALA") && !inCard.isMysteryList() {
				// ELD-Style borderless
				if inCard.isBorderless() {
					if card.BorderColor != mtgjson.BorderColorBorderless {
						continue
					}
					// BaB are allowed to have borderless, same as a few foiling types
				} else if !card.HasPromoType(mtgjson.PromoTypeBuyABox) &&
					!card.HasPromoType(mtgjson.PromoTypeTextured) &&
					!card.HasFrameEffect(mtgjson.FrameEffectShattered) &&
					!card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) &&
					!card.HasPromoType(mtgjson.PromoTypeOilSlick) &&
					!card.HasPromoType(mtgjson.PromoTypeStepAndCompleat) &&
					!card.HasPromoType(mtgjson.PromoTypeConcept) &&
					!card.HasPromoType(mtgjson.PromoTypeThickDisplay) &&
					!card.IsDFCSameName() {
					// IKO may have showcase cards which happen to be borderless
					// or reskinned ones.
					if card.BorderColor == mtgjson.BorderColorBorderless &&
						!card.HasFrameEffect(mtgjson.FrameEffectShowcase) && card.FlavorName == "" {
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
				if inCard.isShowcase() || inCard.isGilded() {
					if !card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
						continue
					}
					//
				} else if !card.HasPromoType(mtgjson.PromoTypeOilSlick) &&
					// Every card in this set is tagged as showcase
					card.SetCode != "MUL" &&
					// Halo foil may be showcase
					!card.HasPromoType(mtgjson.PromoTypeHaloFoil) &&
					// Phyrexian cards _may_ be showcase sometimes
					card.Language != mtgjson.LanguagePhyrexian {
					// NEO has showcase cards that aren't marked as such when they are Etched
					// same for DMU and Textured
					if card.HasFrameEffect(mtgjson.FrameEffectShowcase) && !inCard.isEtched() && !inCard.isTextured() {
						continue
					}
				}

				// ELD-Style bundle
				if inCard.isBundle() && !card.HasPromoType(mtgjson.PromoTypeBundle) {
					continue
				} else if !inCard.isBundle() && card.HasPromoType(mtgjson.PromoTypeBundle) {
					// oilslick cards may not have the bundle tag attached to them
					if !card.HasPromoType(mtgjson.PromoTypeOilSlick) {
						continue
					}
				}

				// ZNR-Style buy-a-box but card is also present in main set
				if setDate.After(BuyABoxNotUniqueDate) {
					if inCard.isBaB() && !card.HasPromoType(mtgjson.PromoTypeBuyABox) {
						continue
					} else if !inCard.isBaB() && card.HasPromoType(mtgjson.PromoTypeBuyABox) {
						continue
					}
				}

				// SNC-Style gilded
				if inCard.isGilded() {
					if !card.HasPromoType(mtgjson.PromoTypeGilded) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeGilded) {
						continue
					}
				}

				if inCard.isPhyrexian() {
					if card.Language != mtgjson.LanguagePhyrexian {
						continue
					}
				} else {
					if card.Language == mtgjson.LanguagePhyrexian {
						continue
					}
				}

				if inCard.isTextured() {
					if !card.HasPromoType(mtgjson.PromoTypeTextured) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeTextured) {
						continue
					}
				}

				if inCard.isGalaxyFoil() {
					if !card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeGalaxyFoil) {
						continue
					}
				}

				if inCard.isSurgeFoil() {
					if !card.HasPromoType(mtgjson.PromoTypeSurgeFoil) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeSurgeFoil) {
						continue
					}
				}

				// ONE-style, present across many editions
				if inCard.isStepAndCompleat() {
					if !card.HasPromoType(mtgjson.PromoTypeStepAndCompleat) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeStepAndCompleat) {
						continue
					}
				}

				if inCard.isConcept() {
					if !card.HasPromoType(mtgjson.PromoTypeConcept) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeConcept) {
						continue
					}
				}

				if inCard.isOilSlick() {
					if !card.HasPromoType(mtgjson.PromoTypeOilSlick) {
						continue
					}
				} else {
					if card.HasPromoType(mtgjson.PromoTypeOilSlick) {
						continue
					}
				}

				isHalo := inCard.Contains("Halo")
				if isHalo && !card.HasPromoType(mtgjson.PromoTypeHaloFoil) {
					continue
				} else if !isHalo && card.HasPromoType(mtgjson.PromoTypeHaloFoil) {
					continue
				}

				// Separate finishes have different collector numbers
				if set.Code == "SLD" || set.Code == "CMR" || setDate.After(SeparateFinishCollectorNumberDate) {
					if inCard.isEtched() && !card.HasFinish(mtgjson.FinishEtched) {
						continue
						// Some thick display cards are not marked as etched
					} else if !inCard.isEtched() && !inCard.isThickDisplay() && card.HasFinish(mtgjson.FinishEtched) {
						continue
					}

					if inCard.isThickDisplay() && !card.HasPromoType(mtgjson.PromoTypeThickDisplay) {
						continue
					} else if !inCard.isThickDisplay() && card.HasPromoType(mtgjson.PromoTypeThickDisplay) {
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

			cardFilterFunc, found := cardFilterCallbacks[setCode]
			if found {
				if cardFilterFunc(inCard, &card) {
					continue
				}
			} else {
				if inCard.Contains("Misprint") && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
					continue
				} else if !inCard.Contains("Misprint") && strings.HasSuffix(card.Number, mtgjson.SuffixVariant) {
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
				// or the edition name itself in the Variation or Edition field
				// (by looking at the longest word present in the parent Edition
				// to avoid aliasing with short words that could ger Normalized away)
				// or the year matches across
				keyword := longestWordInEditionName(backend.Sets[set.ParentCode].Name)
				if strings.Contains(inCard.Variation, set.ParentCode) ||
					(year == "" && inCard.Contains(keyword)) ||
					(year != "" && inCard.Contains(year)) {
					filteredOutCards = append(filteredOutCards, card)
				}
			}

			// Don't throw away what was found if filtering checks is too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}
	}
	// In case card is indistinguishable between MB1, PLIST or PHED:
	// - first check whether we have an exact variant:number association
	// - if not, and there is a PLIST printing, select PLIST
	// - repeat for PHED, PCTB, and other mixed sets
	if len(outCards) > 1 && inCard.isMysteryList() {
		var filteredOutCards []mtgjson.Card

		cn, found := mb1plistVariants[inCard.Name][strings.ToLower(inCard.Variation)]
		if found {
			for _, card := range outCards {
				if card.Number != cn {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}
		} else {
			for _, code := range []string{"PLIST", "PHED", "PCTB"} {
				if len(MatchInSet(inCard.Name, code)) > 0 {
					for _, card := range outCards {
						if card.SetCode != code {
							continue
						}
						filteredOutCards = append(filteredOutCards, card)
					}
					break
				}
			}
		}

		outCards = filteredOutCards
	}

	return
}
