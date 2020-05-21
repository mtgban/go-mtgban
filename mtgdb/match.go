package mtgdb

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgjson"
)

func (db *Database) Match(inCard *Card, logger *log.Logger) (outCard *Card, err error) {
	// Work around some cards that cannot be mapped in mtgjson v4
	if inCard.Name == "Bind" {
		inCard.Id = "08eb3d67-5821-5e8f-a1f4-1a44a4fc3428"
	} else if inCard.Name == "Liberate" {
		inCard.Id = "5e878a45-8b2c-50e0-82be-6ca11452bf9f"
	} else if strings.HasPrefix(inCard.Name, "Bind") && strings.HasSuffix(inCard.Name, "Liberate") {
		inCard.Id = "4814c418-2dd9-5d4d-b613-4ec775e3989a"
	} else if inCard.Name == "Start" {
		inCard.Id = "e822363d-438d-50c1-9102-9f62302a27d8"
	} else if strings.HasPrefix(inCard.Name, "Start") && strings.HasSuffix(inCard.Name, "Finish") {
		inCard.Id = "f5d836dc-ea44-5edb-ac09-ddd5469dfa07"
	} else if strings.HasPrefix(inCard.Name, "Trial") && strings.HasSuffix(inCard.Name, "Error") && strings.HasPrefix(inCard.Edition, "Commander 2016") {
		inCard.Id = "46cd1ee7-7119-5d5f-b6f0-be5569481eb0"
	} else if strings.HasPrefix(inCard.Name, "Trial") && strings.HasSuffix(inCard.Name, "Error") && inCard.Edition == "Dissension" {
		inCard.Id = "72c9549f-61c1-5014-92bb-40503775bccb"
	} else if strings.Contains(inCard.Name, "Smelt") && strings.Contains(inCard.Name, "Herd") && strings.Contains(inCard.Name, "Saw") {
		inCard.Id = "f8f84c2c-b875-5960-803d-c07b2066fb99"
	}

	// Look up by uuid
	if inCard.Id != "" {
		id := inCard.Id
		if strings.HasSuffix(id, "_f") {
			id = id[:len(id)-2]
		}
		for _, set := range internal.Sets {
			for _, card := range set.Cards {
				if id == card.UUID || id == card.ScryfallId {
					return inCard.output(card, set), nil
				}
			}
		}
	}

	entry, found := db.Cards[mtgjson.Normalize(inCard.Name)]
	if !found {
		db.tryAdjustName(inCard)
		// Load the card again
		entry, found = db.Cards[mtgjson.Normalize(inCard.Name)]
		if !found {
			err = fmt.Errorf("card '%s' does not exist", inCard.Name)
			return
		}
	}

	db.tryAdjustEdition(inCard)

	logger.Println("Processing", inCard, entry.Printings)
	printings := entry.Printings
	if len(printings) > 1 {
		printings = db.filterPrintings(inCard, entry)
		logger.Println("Filtered printings:", printings)

		if len(printings) == 0 {
			err = fmt.Errorf("edition '%s' does not apply to '%s'", inCard.Edition, inCard.Name)
			return
		}
	}

	cardSet := map[string][]mtgjson.Card{}

	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		cardSet[printings[0]] = matchSimple(inCard, db.Sets[printings[0]])
	} else {
		logger.Println("Several printings found, iterating over edition name")
		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if mtgjson.NormEquals(db.Sets[setCode].Name, inCard.Edition) {
				logger.Println("Found a perfect match with", inCard.Edition, setCode)
				cardSet[setCode] = matchSimple(inCard, db.Sets[setCode])
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		if len(cardSet) == 0 {
			logger.Println("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := db.Sets[setCode]
				if mtgjson.NormContains(set.Name, inCard.Edition) ||
					(inCard.isGenericPromo() && strings.HasSuffix(set.Name, "Promos")) {
					logger.Println("Found a possible match with", inCard.Edition, setCode)
					cardSet[setCode] = matchSimple(inCard, set)
				}
			}
		}

		// Third loop, YOLO
		if len(cardSet) == 0 {
			logger.Println("No loose match found, trying all")
			for _, setCode := range printings {
				cardSet[setCode] = matchSimple(inCard, db.Sets[setCode])
			}
		}
	}

	// Dertermine if any deduplication needs to be performed
	logger.Println("Found these possible matches")
	var foundCode []string
	single := len(cardSet) == 1
	for setCode, cards := range cardSet {
		foundCode = []string{setCode}
		single = single && len(cards) == 1
		for _, card := range cards {
			logger.Println(setCode, card.Name, card.Number)
		}
	}

	var outCards []mtgjson.Card
	if single {
		logger.Println("Single printing, using it right away")
		outCards = []mtgjson.Card{cardSet[foundCode[0]][0]}
	} else {
		logger.Println("Now filtering...")
		outCards, foundCode = db.filterCards(inCard, cardSet)

		for i, card := range outCards {
			logger.Println(foundCode[i], card.Name, card.Number)
		}
	}

	// Just keep the first card found for these sets
	if inCard.isWorldChamp() && len(outCards) > 1 {
		logger.Println("Dropping a few WCD entries...")
		logger.Println(outCards[1:])
		outCards = []mtgjson.Card{outCards[0]}
	}

	switch len(outCards) {
	case 0:
		logger.Println("No matches...")

		err = fmt.Errorf("edition '%s' does not apply to '%s'", inCard.Edition, inCard.Name)
	case 1:
		outCard = inCard.output(outCards[0], db.Sets[foundCode[0]])
	default:
		out := ""
		for i, card := range outCards {
			out += fmt.Sprintf("\n%s - %s (%s)", foundCode[i], card.Name, card.Number)
		}
		err = fmt.Errorf("aliasing detected%s", out)
	}

	return
}

func (db *Database) filterPrintings(inCard *Card, entry *mtgjson.SimpleCard) (printings []string) {
	maybeYear := ExtractYear(inCard.Variation)
	if maybeYear == "" {
		maybeYear = ExtractYear(inCard.Edition)
	}

	for _, setCode := range entry.Printings {
		set, found := db.Sets[setCode]
		if !found || setCode == "PRED" {
			continue
		}

		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		switch {
		// If the edition matches, use it as is
		case mtgjson.NormEquals(inCard.Edition, set.Name):
			// pass-through

		case inCard.isPrerelease():
			switch set.Name {
			case "Duels of the Planeswalkers 2012 Promos", //possibly a bug in scryfall
				"Duels of the Planeswalkers 2013 Promos",
				"Grand Prix Promos",
				"Resale Promos",
				"Pro Tour Promos",
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
			case "Grand Prix Promos",
				"Dragon's Maze Promos":
				continue
			case "M20 Promo Packs":
			default:
				switch {
				case strings.HasSuffix(set.Name, "Promos"):
				case setDate.After(PromosForEverybodyYay) && set.Type == "expansion":
					skip := true
					cards := matchSimple(inCard, set)
					for _, card := range cards {
						if card.HasFrameEffect(mtgjson.FrameEffectInverted) {
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
			case "Release Events",
				"Launch Parties":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.isBaB():
			switch set.Name {
			case "Launch Parties":
			case "Modern Horizons":
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
				cards := matchSimple(inCard, set)
				for _, card := range cards {
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

		case strings.Contains(inCard.Variation, "Judge"):
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

		case strings.Contains(inCard.Variation, "Arena"):
			if maybeYear == "" {
				switch {
				case strings.Contains(inCard.Variation, "Tony Roberts"):
					maybeYear = "1996"
				case strings.Contains(inCard.Variation, "Urza"),
					strings.Contains(inCard.Variation, "Saga"),
					strings.Contains(inCard.Variation, "Anthony S. Waters"),
					strings.Contains(inCard.Variation, "Donato Giancola"):
					maybeYear = "1999"
				case strings.Contains(inCard.Variation, "Mercadian"),
					strings.Contains(inCard.Variation, "Masques"):
					maybeYear = "2000"
				case strings.Contains(inCard.Variation, "Ice Age"),
					strings.Contains(inCard.Variation, "IA"),
					strings.Contains(inCard.Variation, "Pat Morrissey"),
					strings.Contains(inCard.Variation, "Anson Maddocks"),
					strings.Contains(inCard.Variation, "Tom Wanerstrand"),
					strings.Contains(inCard.Variation, "Christopher Rush"),
					strings.Contains(inCard.Variation, "Douglas Shuler"):
					maybeYear = "2001"
				case strings.Contains(inCard.Variation, "Mark Poole"):
					maybeYear = "2002"
				case strings.Contains(inCard.Variation, "Rob Alexander"):
					maybeYear = "2003"
				case strings.Contains(inCard.Variation, "Don Thompson"):
					maybeYear = "2005"
				case strings.Contains(inCard.Variation, "Beta"):
					switch inCard.Name {
					case "Forest":
						maybeYear = "2001"
					case "Island":
						maybeYear = "2002"
					}
				}
			} else if maybeYear == "2002" && inCard.Name == "Forest" {
				maybeYear = "2001"
			}
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
			switch {
			case set.Name == "Summer of Magic":
			case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
			case strings.HasPrefix(set.Name, "Gateway "+maybeYear):
			default:
				continue
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
					cards := matchSimple(inCard, set)
					for _, card := range cards {
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

		case strings.Contains(inCard.Variation, "Clash"):
			switch set.Name {
			case "Magic 2015 Clash Pack",
				"Magic Origins Clash Pack",
				"Fate Reforged Clash Pack":
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "Hero's Path") ||
			strings.Contains(inCard.Edition, "Hero's Path"):
			switch set.Name {
			case "Journey into Nyx Hero's Path",
				"Born of the Gods Hero's Path",
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
			mtgjson.NormContains(inCard.Variation, "San Diego Comic-Con"):
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
					if mtgjson.NormContains(set.Name, field) {
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
				if !mtgjson.NormContains(set.Name, variant) {
					continue
				}
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "Champs") ||
			strings.Contains(inCard.Variation, "States"):
			switch set.Name {
			case "Gateway 2007":
			case "Champs and States":
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

		case inCard.isBasicLand() && mtgjson.NormContains(inCard.Variation, "EURO"):
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

func (db *Database) filterCards(inCard *Card, cardSet map[string][]mtgjson.Card) (cards []mtgjson.Card, foundCode []string) {

	for setCode, inCards := range cardSet {
		set := db.Sets[setCode]
		setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)

		for _, card := range inCards {
			// Super lucky case, we were expecting the card
			// Note, we need to use the input card name because there might be variants
			// in the names provided by mtgjson
			num, found := VariantsTable[set.Name][inCard.Name][strings.ToLower(inCard.Variation)]
			if found {
				if num == card.Number {
					cards = append(cards, card)
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
						cards = append(cards, card)
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
				prereleaseSuffix := "s"
				if inCard.isJPN() {
					prereleaseSuffix += mtgjson.SuffixSpecial
				}
				if setDate.After(NewPrereleaseDate) && !strings.HasSuffix(card.Number, prereleaseSuffix) {
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
				if strings.HasSuffix(card.Number, "s") ||
					strings.HasSuffix(card.Number, "s"+mtgjson.SuffixSpecial) {
					// Scryfall bug
					if set.Name != "Deckmasters" {
						continue
					}
				}
			}

			// Promo pack
			if inCard.isPromoPack() && !inCard.isBasicLand() {
				if !strings.HasSuffix(card.Number, "p") && !card.HasFrameEffect(mtgjson.FrameEffectInverted) {
					continue
				}
			} else if !inCard.isBasicLand() {
				if strings.HasSuffix(card.Number, "p") || (card.HasFrameEffect(mtgjson.FrameEffectInverted) && !inCard.isFNM()) {
					continue
				}
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
				// Since the check is field by field Foglio may alias Phil or Kaja
				variation := inCard.Variation
				if strings.Contains(inCard.Variation, "Foglio") {
					variation = strings.Replace(inCard.Variation, " Foglio", ":Foglio", 1)
				}

				fields := strings.Fields(variation)
				found := false

				// Keep flavor text author only
				flavor := card.FlavorText
				if strings.Contains(flavor, "\" —") {
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
					if mtgjson.NormContains(flavor, field) || mtgjson.NormContains(card.Artist, field) {
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
				if !mtgjson.NormContains(inCard.Variation, card.Watermark) {
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
			default:
				// Special singles
				switch card.Name {
				// Duplicated in several promo sets
				case "Sorcerous Spyglass":
					if set.ParentCode != "" &&
						!(strings.Contains(inCard.Variation, set.ParentCode) ||
							strings.Contains(inCard.Variation, db.Sets[set.ParentCode].Name) ||
							strings.Contains(inCard.Edition, db.Sets[set.ParentCode].Name)) {
						continue
					}
				case "Piper of the Swarm":
					if inCard.isBundle() && card.Number != "392" {
						continue
					} else if !inCard.isBundle() && !inCard.isPrerelease() && !inCard.isPromoPack() && !inCard.isExtendedArt() && card.Number != "100" {
						continue
					}
				case "Arasta of the Endless Web":
					if inCard.isBundle() && card.Number != "352" {
						continue
					} else if !inCard.isBundle() && !inCard.isPrerelease() && !inCard.isPromoPack() && !inCard.isExtendedArt() && card.Number != "165" {
						continue
					}
				case "Colossification":
					if inCard.isBundle() && card.Number != "364" {
						continue
					} else if !inCard.isBundle() && !inCard.isPrerelease() && !inCard.isPromoPack() && !inCard.isExtendedArt() && card.Number != "148" {
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
					}

					if mtgjson.NormContains(variation, "misprint") && !strings.HasSuffix(card.Number, expectedSuffix) {
						continue
					} else if !mtgjson.NormContains(variation, "misprint") && strings.HasSuffix(card.Number, expectedSuffix) {
						continue
					}
				}
			}

			cards = append(cards, card)
			foundCode = append(foundCode, setCode)
		}
	}

	return
}

func matchSimple(inCard *Card, set *mtgjson.Set) (outCards []mtgjson.Card) {
	for _, card := range set.Cards {
		cName := card.Name
		// Strip the parentheses for Unstable variations,
		// but keep them if they're part of the card name
		if set.Name == "Unstable" {
			names := SplitVariants(cName)
			if len(names) > 1 && len(names[1]) == 1 {
				cName = names[0]
			}
		}
		if mtgjson.NormEquals(inCard.Name, cName) {
			outCards = append(outCards, card)
		}
	}

	return
}

func (db *Database) tryAdjustName(inCard *Card) {
	// Move the card number from name to variation
	num := ExtractNumber(inCard.Name)
	if num != "" {
		fields := strings.Fields(inCard.Name)
		for i, field := range fields {
			if strings.Contains(field, num) {
				fields = append(fields[:i], fields[i+1:]...)
				break
			}
		}
		inCard.Name = strings.Join(fields, " ")
		if inCard.Variation != "" {
			inCard.Variation += " "
		}
		inCard.Variation += num
		return
	}

	// Move any single letter variation from name to beginning variation
	if inCard.IsBasicLand() {
		fields := strings.Fields(inCard.Name)
		if len(fields) > 1 && len(fields[1]) == 1 {
			oldVariation := inCard.Variation
			cuts := Cut(inCard.Name, " "+fields[1])

			inCard.Name = cuts[0]
			inCard.Variation = cuts[1]
			if oldVariation != "" {
				inCard.Variation += " " + oldVariation
			}
			return
		}
	}

	// Check if the input name is the reskinned one
	if strings.Contains(inCard.Edition, "Ikoria") {
		for _, card := range db.Sets["IKO"].Cards {
			if mtgjson.NormEquals(inCard.Name, card.FlavorName) {
				inCard.Name = card.Name
				if inCard.Variation != "" {
					inCard.Variation += " "
				}
				inCard.Variation += "Godzilla"
				return
			}
		}
	}

	// Many provide the full name with a variety of characters
	if strings.Contains(inCard.Name, " | ") ||
		strings.Contains(inCard.Name, " // ") ||
		strings.Contains(inCard.Name, " / ") ||
		strings.Contains(inCard.Name, " and ") ||
		strings.Contains(inCard.Name, " to ") {
		// Loop over the db, find the one with different Layout and matching name
		for _, card := range db.Cards {
			// Ignore side b for flip and split cards, they add too many duplicates
			if mtgjson.NormPrefix(inCard.Name, card.Name) &&
				((card.Side != "b" && card.Layout != "normal" && card.Layout != "meld") || card.Layout == "meld") {
				threshold := 0
				// At least two names need to be present for a successful match
				for _, subname := range card.Names {
					if mtgjson.NormContains(inCard.Name, subname) {
						threshold++
					}
				}
				if threshold >= 2 {
					inCard.Name = card.Name
					break
				}
			}
		}
	} else {
		for _, card := range db.Cards {
			if strings.HasPrefix(card.Name, inCard.Name) {
				switch card.Printings[0] {
				case "UGL", "UNH", "UST", "UND":
					inCard.Name = card.Name
					break
				}
			}
		}
	}
}

func (db *Database) tryAdjustEdition(inCard *Card) {
	edition := inCard.Edition
	set, found := db.Sets[edition]
	if found {
		edition = set.Name
	}
	ed, found := EditionTable[edition]
	if found {
		edition = ed
	}
	ed, found = EditionTable[inCard.Variation]
	if found {
		edition = ed
	}

	switch {
	case strings.HasSuffix(edition, "(Collector Edition)"):
		edition = strings.Replace(edition, " (Collector Edition)", "", 1)
	case strings.HasSuffix(edition, "Extras"):
		edition = strings.Replace(edition, " Extras", "", 1)
		edition = strings.Replace(edition, ":", "", 1)
	case strings.HasSuffix(edition, "Variants"):
		edition = strings.Replace(edition, " Variants", "", 1)
		edition = strings.Replace(edition, ":", "", 1)
	case strings.Contains(edition, "Mythic Edition"):
		edition = "Mythic Edition"
	case strings.Contains(edition, "Invocations"):
		edition = "Amonkhet Invocations"
	case strings.Contains(edition, "Inventions"):
		edition = "Kaladesh Inventions"
	case strings.Contains(edition, "Expeditions"):
		edition = "Zendikar Expeditions"
	}

	variation := inCard.Variation
	switch {
	case strings.Contains(variation, "Ravnica Weekend"):
		num := ExtractNumber(variation)
		if strings.HasPrefix(num, "A") {
			edition = "GRN Ravnica Weekend"
		} else if strings.HasPrefix(num, "B") {
			edition = "RNA Ravnica Weekend"
		}
	case strings.Contains(variation, "APAC Set") || strings.Contains(variation, "Euro Set"):
		num := ExtractNumber(variation)
		if num != "" {
			variation = strings.Replace(variation, num+" ", "", 1)
		}
	case strings.HasPrefix(variation, "Junior") && strings.Contains(variation, "APAC"),
		strings.HasPrefix(variation, "Junior APAC Series U"):
		edition = "Junior APAC Series"
	case strings.HasPrefix(variation, "Junior Super Series"),
		strings.HasPrefix(variation, "MSS Foil"),
		strings.HasPrefix(variation, "MSS #J"),
		strings.HasPrefix(variation, "MSS Promo J"),
		strings.HasPrefix(variation, "JSS #J"),
		strings.Contains(variation, "JSS Foil") && !mtgjson.NormContains(variation, "euro"):
		edition = "Junior Super Series"
	case strings.HasPrefix(variation, "Junior Series Europe"),
		strings.HasPrefix(variation, "Junior Series E"),
		strings.HasPrefix(variation, "Junior Series #E"),
		strings.HasPrefix(variation, "Junior Series Promo E"),
		strings.HasPrefix(variation, "Junior Series Promo Foil E"),
		strings.HasPrefix(variation, "ESS Foil E"),
		strings.HasPrefix(variation, "European JrS E"),
		strings.HasPrefix(variation, "European JSS Foil E"):
		edition = "Junior Series Europe"
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// XLN Treasure Chest
	case inCard.isBaB() && len(matchSimple(inCard, db.Sets["PXTC"])) != 0:
		inCard.Edition = db.Sets["PXTC"].Name
	// BFZ Standard Series
	case inCard.isGenericAltArt() && len(matchSimple(inCard, db.Sets["PSS1"])) != 0:
		inCard.Edition = db.Sets["PSS1"].Name
	// Champs and States
	case inCard.isGenericExtendedArt() && len(matchSimple(inCard, db.Sets["PCMP"])) != 0:
		inCard.Edition = db.Sets["PCMP"].Name
	// Portal Demo Game
	case ((mtgjson.NormContains(inCard.Variation, "Reminder Text") &&
		!strings.Contains(inCard.Variation, "No")) ||
		mtgjson.NormContains(inCard.Variation, "No Flavor Text")) &&
		len(matchSimple(inCard, db.Sets["PPOD"])) != 0:
		inCard.Edition = db.Sets["PPOD"].Name
	// Secret Lair Ultimate
	case strings.Contains(inCard.Edition, "Secret Lair") &&
		len(matchSimple(inCard, db.Sets["SLU"])) != 0:
		inCard.Edition = db.Sets["SLU"].Name
	// Summer of Magic
	case (inCard.isWPNGateway() || strings.Contains(inCard.Variation, "Summer")) &&
		len(matchSimple(inCard, db.Sets["PSUM"])) != 0:
		inCard.Edition = db.Sets["PSUM"].Name

	// Single cards mismatch
	case mtgjson.NormEquals(inCard.Name, "Rhox") && inCard.isGenericAltArt():
		inCard.Edition = "Starter 2000"
	case mtgjson.NormEquals(inCard.Name, "Balduvian Horde") && (strings.Contains(inCard.Variation, "Judge") || strings.Contains(inCard.Edition, "Promo")):
		inCard.Edition = "World Championship Promos"
	case mtgjson.NormEquals(inCard.Name, "Nalathni Dragon") && inCard.isIDWMagazineBook():
		inCard.Edition = "Dragon Con"
	case mtgjson.NormEquals(inCard.Name, "Ass Whuppin'") && inCard.isPrerelease():
		inCard.Variation = "Release Events"
	case mtgjson.NormEquals(inCard.Name, "Ajani Vengeant") && inCard.isRelease():
		inCard.Variation = "Prerelease"
	case mtgjson.NormEquals(inCard.Name, "Tamiyo's Journal") && inCard.Variation == "" && inCard.Foil:
		inCard.Variation = "Foil"
	}
}
