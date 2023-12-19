package mtgmatcher

import (
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"
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
		case Equals(inCard.Edition, set.Name) && !inCard.isMysteryList():
			// pass-through

		case inCard.isPrerelease():
			switch set.Name {
			// Sets that could be marked as prerelease, but they aren't really
			case "M15 Prerelease Challenge",
				"Open the Helvault":
			// Sets that have prerelease cards mixed in
			case "Innistrad: Double Feature",
				"March of the Machine Commander",
				"The Lord of the Rings: Tales of Middle-earth":
				skip := true
				foundCards := MatchInSet(inCard.Name, setCode)
				for _, card := range foundCards {
					if card.HasPromoType(mtgjson.PromoTypePrerelease) {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			case "Duels of the Planeswalkers 2012 Promos",
				"Grand Prix Promos",
				"Pro Tour Promos",
				"Resale Promos",
				"World Championship Promos":
				continue
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
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
			skip := true
			foundCards := MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgjson.PromoTypeRelease) || card.HasPromoType(mtgjson.PromoTypeDraftWeekend) {
					skip = false
					break
				}
			}
			if skip {
				continue
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
			case set.Name == "Magic × Duel Masters Promos":
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
			// Short-circuit the upcoming number check to avoid the manual validation below
			_, found := VariantsTable[set.Name][inCard.Name][strings.ToLower(inCard.Variation)]
			if found {
				printings = append(printings, setCode)
				continue
			}

			switch set.Code {
			case "MB1":
				if inCard.Variation == "The List" || inCard.Edition == "The List" ||
					inCard.Contains("Heads I Win, Tails You Lose") ||
					inCard.Contains("From Cute to Brute") ||
					inCard.Contains("They're Just Like Us") ||
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
					inCard.Contains("Heads I Win, Tails You Lose") ||
					inCard.Contains("From Cute to Brute") ||
					inCard.Contains("They're Just Like Us") ||
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
					if inCard.Edition != set.Name {
						// Except the following cards, when they are not tagged as specified,
						// it means they are actually from this set
						switch inCard.Name {
						case "Sol Ring",
							"Reliquary Tower":
							if !inCard.Contains("21") {
								continue
							}
						case "Counterspell",
							"Temur Battle Rage":
							if !inCard.Contains("Legends") && !inCard.Contains("cmr") {
								continue
							}
						case "Fabricate",
							"Negate",
							"Spark Double",
							"Tribute Mage",
							"Blasphemous Act",
							"Daretti, Scrap Savant",
							"Fiery Gambit",
							"Goblin Archaeologist",
							"Goblin Kaboomist",
							"Karplusan Minotaur",
							"Krark, the Thumbless",
							"Tavern Scoundrel",
							"Frenetic Sliver",
							"Niv-Mizzet, Parun",
							"Ral Zarek",
							"Izzet Signet",
							"Lightning Greaves",
							"Mind Stone",
							"Swiftfoot Boots",
							"Sword of Vengeance",
							"Thought Vessel",
							"Whispersilk Cloak",
							"Buried Ruin",
							"Great Furnace",
							"Izzet Boilerworks",
							"Myriad Landscape",
							"Path of Ancestry",
							"Rogue's Passage",
							"Temple of Epiphany",
							"Wandering Fumarole",
							"Island",
							"Mountain":
							if !inCard.Foil {
								continue
							}
						default:
							if inCard.Foil {
								continue
							}
						}
					}
				}
			case "PCTB":
				// If the card is not foil, and has been printed somewhere else,
				// only pick this edition if explicilty requested
				if len(MatchInSet(inCard.Name, "MB1")) > 0 || len(MatchInSet(inCard.Name, "PLIST")) > 0 {
					if inCard.Edition != set.Name {
						// Except the following cards, when they are not tagged as specified,
						// it means they are actually from this set
						switch inCard.Name {
						case "Island", "Mountain", "Plains":
							if !inCard.Contains("amonkhet") && !inCard.Contains("akh") {
								continue
							}
						case "Swamp":
							if !inCard.Contains("hour") && !inCard.Contains("hou") {
								continue
							}
						case "Forest":
							if !inCard.Contains("amonkhet") && !inCard.Contains("akh") &&
								!inCard.Contains("hour") && !inCard.Contains("hou") {
								continue
							}
						default:
							continue
						}
					}
				}
			case "PAGL":
				// If the card is not foil, and has been printed somewhere else,
				// only pick this edition if explicilty requested
				if len(MatchInSet(inCard.Name, "MB1")) > 0 || len(MatchInSet(inCard.Name, "PLIST")) > 0 {
					if inCard.Edition != set.Name {
						// Except the following cards, when they are not tagged as specified,
						// it means they are actually from this set
						switch inCard.Name {
						case "Angelic Field Marshal",
							"Angel of Destiny",
							"Angel of Serenity",
							"Angel of Vitality",
							"Archangel of Tithes",
							"Emeria Shepherd",
							"Entreat the Angels",
							"Righteous Valkyrie",
							"Sephara, Sky's Blade",
							"Shattered Angel",
							"Sunblast Angel":
							if !inCard.Foil {
								continue
							}
						default:
							if inCard.Foil {
								continue
							}
						}
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
			case "Innistrad: Crimson Vow",
				"The Lost Cavern of Ixalan":
				skip := true
				foundCards := MatchInSet(inCard.Name, set.Code)
				for _, card := range foundCards {
					if card.HasPromoType(mtgjson.PromoTypeWPN) {
						skip = false
						break
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
				case "URL/Convention Promos":
				case "Hobby Japan Promos":
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
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
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

		// DDA with number or deck variant specified in the variation
		case inCard.isDuelDecksAnthology():
			switch {
			case strings.HasPrefix(set.Name, "Duel Decks Anthology"):
				found := false
				num := ExtractNumber(inCard.Variation)
				if num != "" {
					foundCards := MatchInSet(inCard.Name, setCode)
					for _, card := range foundCards {
						if card.Number == num {
							found = true
							break
						}
					}
				} else {
					fields := strings.Fields(inCard.Variation)
					for _, field := range fields {
						if len(field) < 4 {
							continue
						}
						if Contains(set.Name, field) {
							found = true
							break
						}
					}
				}
				if inCard.Variation != "" && !found {
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
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
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
			inCard.Contains("Open House") ||
			inCard.Contains("LGS Promo"):
			switch {
			case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
			case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
			case set.Name == "Grand Prix Promos":
				continue
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.Contains("30th"):
			switch set.Code {
			case "P30A", "P30H":
				if inCard.isJPN() && inCard.Name != "Tarmogoyf" {
					continue
				}
			case "P30HJPN":
				if !inCard.isJPN() {
					continue
				}
			case "P30T":
				if inCard.isRetro() {
					continue
				}
			case "P30M":
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
						if slices.Contains(card.Types, "Planeswalker") {
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

					// Check if edition-specific numbers need special suffixes
					numFilterFunc, found := numberFilterCallbacks[set.Code]
					if found {
						overrides := numFilterFunc(inCard)
						if overrides != nil {
							possibleSuffixes = overrides
						}
					}

					// Add any possible extra suffixes if we know what we're dealing with
					switch {
					case inCard.isPrerelease():
						possibleSuffixes = append(possibleSuffixes, "s")
					case inCard.isPromoPack():
						possibleSuffixes = append(possibleSuffixes, "p")
					case inCard.isChineseAltArt():
						possibleSuffixes = append(possibleSuffixes, "s", mtgjson.SuffixSpecial+"s", mtgjson.SuffixVariant+"s")
					case inCard.isSerialized():
						possibleSuffixes = append(possibleSuffixes, "z")
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
			switch set.Type {
			case "expansion", "core", "masters", "draft_innovation":
				if inCard.promoWildcard &&
					!card.HasPromoType(mtgjson.PromoTypeBoosterfun) &&
					!card.HasPromoType(mtgjson.PromoTypePromoPack) &&
					!card.HasPromoType(mtgjson.PromoTypeStarterDeck) &&
					!card.HasPromoType(mtgjson.PromoTypeIntroPack) &&
					!strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) &&
					!card.IsPromo {
					continue
				}
			}

			outCards = append(outCards, card)
		}
	}

	// Sort through the array of promo types
	if len(outCards) > 1 {
		var filteredOutCards []mtgjson.Card
		for _, card := range outCards {
			set, found := backend.Sets[card.SetCode]
			if !found {
				continue
			}
			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

			var shouldContinue bool
			for _, promoElement := range promoTypeElements {
				if setDate.Before(promoElement.ValidDate) {
					continue
				}
				if promoElement.CanBeWild && inCard.promoWildcard {
					continue
				}

				var tagPresent bool
				if promoElement.TagFunc != nil {
					tagPresent = promoElement.TagFunc(inCard)
				} else {
					for _, tag := range promoElement.Tags {
						if inCard.Contains(tag) {
							tagPresent = true
							break
						}
					}
				}

				if tagPresent && !card.HasPromoType(promoElement.PromoType) {
					shouldContinue = true
					break
				} else if !tagPresent && card.HasPromoType(promoElement.PromoType) {
					shouldContinue = true
					break
				}
			}
			if shouldContinue {
				continue
			}

			if inCard.beyondBaseSet {
				// Filter out any card that is located in the base set only
				num, err := strconv.Atoi(card.Number)
				if err == nil && num < set.BaseSetSize {
					continue
				}
			}
			filteredOutCards = append(filteredOutCards, card)
		}

		// Don't throw away what was found if filtering checks is too aggressive
		if len(filteredOutCards) > 0 {
			outCards = filteredOutCards
		}
	}

	// Sort through any custom per-edition filter(s)
	if len(outCards) > 1 {
		var filteredOutCards []mtgjson.Card
		for _, card := range outCards {
			cardFilterFunc, foundSimple := simpleFilterCallbacks[card.SetCode]
			cardFilterFuncs, foundComplex := complexFilterCallbacks[card.SetCode]
			if foundSimple {
				if cardFilterFunc(inCard, &card) {
					continue
				}
			} else if foundComplex {
				shouldContinue := false
				for _, fn := range cardFilterFuncs {
					if fn(inCard, &card) {
						shouldContinue = true
						break
					}
				}
				if shouldContinue {
					continue
				}
			} else {
				if inCard.Contains("Misprint") && !strings.HasSuffix(card.Number, mtgjson.SuffixVariant) && !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial) {
					continue
				} else if !inCard.Contains("Misprint") && (strings.HasSuffix(card.Number, mtgjson.SuffixVariant) || strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)) {
					continue
				}
			}

			filteredOutCards = append(filteredOutCards, card)
		}

		// Don't throw away what was found if filtering checks is too aggressive
		if len(filteredOutCards) > 0 {
			outCards = filteredOutCards
		}
	}

	if len(outCards) > 1 {
		logger.Println("Filtering status after main loop")
		for _, card := range outCards {
			logger.Println(card.SetCode, card.Number, card.Name)
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
			for _, code := range []string{
				"PLIST",
				"PHED",
				"PCTB",
				"PAGL",
			} {
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
	} else if len(outCards) > 1 && ExtractNumber(inCard.Variation) == "" {
		// Separate finishes have different collector numbers after this date
		if len(outCards) > 1 {
			var filteredOutCards []mtgjson.Card
			for _, card := range outCards {
				set, found := backend.Sets[card.SetCode]
				if !found {
					continue
				}
				setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
				if setDate.After(SeparateFinishCollectorNumberDate) && etchedCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// If above filters were not enough, check the boder, but dont skip card if it's showcase
		// ExtendedArt is fine as there cannot be a borderless one
		if len(outCards) > 1 {
			var filteredOutCards []mtgjson.Card
			for _, card := range outCards {
				if borderlessCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// If above filters were not enough, check the frameEffects
		// Extended Art
		if len(outCards) > 1 {
			var filteredOutCards []mtgjson.Card
			for _, card := range outCards {
				// This needs date check because some old full art promos are marked
				// as extended art, in a different way of what modern Extended Art is
				set, found := backend.Sets[card.SetCode]
				if !found {
					continue
				}
				setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
				if setDate.After(PromosForEverybodyYay) && extendedartCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// Showcase
		if len(outCards) > 1 {
			var filteredOutCards []mtgjson.Card
			for _, card := range outCards {
				if showcaseCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}
	}

	return
}
