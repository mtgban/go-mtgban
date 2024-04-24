package coolstuffinc

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Chant of the Vitu-Ghazi": "Chant of Vitu-Ghazi",
	"Deputized Protestor":     "Deputized Protester",
	"Erdwall Illuminator":     "Erdwal Illuminator",
	"Mistfoot Kirin":          "Misthoof Kirin",
	"Holy Justicar":           "Holy Justiciar",
	"Nearhearth Chaplain":     "Nearheath Chaplain",
	"Shatter Assumpions":      "Shatter Assumptions",
	"Stratozeppilid":          "Stratozeppelid",
	"Immerstrurm Predator":    "Immersturm Predator",
	"Inspiried Idea":          "Inspired Idea",
	"Shadow of Morality":      "Shadow of Mortality",

	"Startled Awake // Persistant Nightmare": "Startled Awake //  Persistent Nightmare",
}

var variantTable = map[string]string{
	"Jeff A Menges":                                    "Jeff A. Menges",
	"Jeff a Menges":                                    "Jeff A. Menges",
	"San Diego Comic-Con Promo M15":                    "SDCC 2014",
	"San Diego Comic-Con Promo M14":                    "SDCC 2013",
	"EURO Land White Cliffs of Dover Ben Thompson art": "EURO White Cliffs of Dover",
	"EURO Land Danish Island Ben Thompson art":         "EURO Land Danish Island",
	"Eighth Edition Prerelease Promo":                  "Release Promo",
	"Release 27 Promo":                                 "Release",
	"2/2 Power and Toughness":                          "misprint",
	"Big Furry Monster Left Side":                      "28",
	"Big Furry Monster Right Side":                     "29",

	"Deckmasters cards are white bordered and have a stylized D as their set symbol FOIL versions are black bordered": "",
}

func preprocess(cardName, edition, notes, maybeNum string) (*mtgmatcher.Card, error) {
	// Clean up notes, removing extra prefixes, and ueless characters
	variant := strings.TrimPrefix(notes, "Notes:")
	if strings.Contains(variant, "Deckmaster") {
		cuts := mtgmatcher.Cut(variant, "Deckmaster")
		variant = cuts[0]
	}
	variant = cleanVariant(variant)

	vars, found := variantTable[variant]
	if found {
		variant = vars
	}

	switch variant {
	case "Silver Planeswalker Symbol":
		switch cardName {
		// Impossible to decouple
		case "Sorcerous Spyglass",
			"Fabled Passage",
			"Temple of Epiphany",
			"Temple of Malady",
			"Temple of Mystery",
			"Temple of Silence",
			"Temple of Triumph":
			return nil, errors.New("unsupported")
		// Does not exist
		case "Elspeth's Devotee":
			return nil, errors.New("invalid")
		}
	case "Secret Lair":
		switch cardName {
		// Impossible to decouple (except for some fortunate cases from retail)
		case "Serum Visions":
			return nil, errors.New("unsupported")
		case "Thalia, Guardian of Thraben":
			if strings.HasPrefix(maybeNum, "ThaliaGuardianofThraben0") {
				variant = strings.TrimPrefix(maybeNum, "ThaliaGuardianofThraben0")
			} else {
				return nil, errors.New("unsupported")
			}
		}
	case "Secret Lair: Mountain Go":
		switch cardName {
		case "Lightning Bolt":
			if strings.HasPrefix(maybeNum, "SLD0") {
				variant = strings.TrimPrefix(maybeNum, "SLD0")
			} else {
				return nil, errors.New("unsupported")
			}
		}
	}

	isFoilFromName := false
	switch cardName {
	case "Teferi, Master of Time",
		"Lukka, Coppercoat Outcast",
		"Brokkos, Apex of Forever":
		return nil, errors.New("impossible to track")
	case "Demonic Tutor":
		if variant == "Daarken Judge Rewards Promo" {
			variant = "Judge 2008"
		} else if variant == "Anna Steinbauer Judge Promo" {
			variant = "Judge 2020"
		}
	case "Wasteland":
		if variant == "Judge Rewards Promo Carl Critchlow art" {
			variant = "Judge 2010"
		} else if variant == "Judge Rewards Promo Steve Belledin art" {
			variant = "Judge 2015"
		}
	case "Vindicate":
		if variant == "Judge Rewards Promo Mark Zug art" {
			variant = "Judge 2007"
		} else if variant == "Judge Rewards Promo Karla Ortiz art" {
			variant = "Judge 2013"
		}
	case "Fling":
		// Only one of the two is present
		if variant == "Gateway Promo Wizards Play Network Daren Bader art" {
			variant = "DCI"
		}
	case "Sylvan Ranger":
		if variant == "Judge Rewards Promo Mark Zug art" {
			variant = "WPN"
		}
	case "Goblin Warchief":
		if variant == "Friday Night Magic Promo Old Border" {
			variant = "FNM 2006"
		} else if variant == "Friday Night Magic Promo New Border" {
			variant = "FNM 2016"
		}
	case "Mind Warp":
		if variant == "Arena League Promo" {
			variant = "FNM 2000"
		}
	case "Cabal Therapy":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2003"
		}
	case "Rishadan Port":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2000"
		}
	case "Mana Crypt":
		if strings.HasPrefix(variant, "Harper Prism Promo") {
			if !strings.Contains(variant, "English") {
				return nil, errors.New("non-english")
			}
		}
	case "Hangarback Walker":
		if edition == "Promo" {
			edition = "Love your LGS"
		}
	case "Chord of Calling", "Wrath of God":
		if edition == "Promo" {
			edition = "Double Masters"
			variant = "Release"
		}

	default:
		if strings.Contains(cardName, "Booster Box") ||
			strings.Contains(cardName, "Booster Pack") ||
			strings.Contains(cardName, "Fat Pack") ||
			strings.Contains(cardName, "Bundle") ||
			strings.Contains(cardName, "Spindown") ||
			strings.Contains(cardName, "Box Set") ||
			strings.Contains(cardName, "Bulk") ||
			strings.Contains(cardName, "Signed by") ||
			strings.Contains(cardName, "Proxy Card") ||
			strings.Contains(cardName, "Chinese") {
			return nil, errors.New("not single")
		}
		if strings.Contains(variant, "Preorder") {
			return nil, errors.New("not out yet")
		}
		if strings.Contains(cardName, "FOIL") {
			cardName = strings.Replace(cardName, " FOIL", "", 1)
			isFoilFromName = true
		}
		if strings.Contains(cardName, "Signed") && strings.Contains(cardName, "by") {
			cuts := mtgmatcher.Cut(cardName, "Signed")
			cardName = cuts[0]
		}
	}

	if strings.Contains(edition, "Challenger Decks") {
		return nil, errors.New("set not mtg")
	}
	switch edition {
	case "", "Overwhelming Swarm", "Special Offers", "Unique Boutique", "Magic Mics Merch",
		"Authenticated Collectibles", "New Player Series", "Heavy Metal Magic Supplies",
		"Online Arena":
		return nil, errors.New("set not mtg")
	case "Prerelease Promos":
		if variant != "" {
			variant = " "
		}
		variant += "Prerelease"
	case "Portal 3 Kingdoms", "Jace vs. Chandra":
		if strings.Contains(cardName, "Japanese") {
			return nil, errors.New("not english")
		}
	case "Duel Decks: Anthology":
		if len(maybeNum) == 3 {
			edition = maybeNum
		}
	case "Commander Anthology Volume II":
		if maybeNum == "" {
			switch cardName {
			case "Solemn Simulacrum",
				"Golgari Signet",
				"Simic Signet",
				"Temple of the False God":
				return nil, errors.New("unsupported")
			}
		}
		variant = maybeNum
	case "Unstable", "Unsanctioned":
		switch cardName {
		case "Amateur Auteur",
			"Everythingamajig",
			"Very Cryptic Command":
		default:
			variant = maybeNum
		}
		if maybeNum == "" || maybeNum == "." {
			if cardName == "Extremely Slow Zombie" || cardName == "Sly Spy" {
				variant = "a"
			}
			if cardName == "Secret Base" {
				variant = "b"
			}
		}
	case "Guilds of Ravnica", "Ravnica Allegiance":
		_, err := strconv.Atoi(maybeNum)
		if err == nil && !strings.Contains(maybeNum, "486") {
			if variant != "" {
				variant += " "
			}
			variant += maybeNum
		}
	case "Global Series - Planeswalker Decks - Jiang Yanggu & Mu Yanling":
		edition = "Global Series Jiang Yanggu & Mu Yanling"
	case "Ikoria: Lair of Behemoths: Variants":
		vars := mtgmatcher.SplitVariants(cardName)
		if len(vars) > 1 {
			cardName = vars[0]
			variant = vars[1]
		}
		if strings.Contains(variant, "Japanese") {
			switch cardName {
			case "Dirge Bat", "Mysterious Egg", "Crystalline Giant":
				variant = "godzilla"
			default:
				return nil, errors.New("not english")
			}
		}
	case "Zendikar Rising":
		if strings.HasPrefix(cardName, "Blank Card") {
			return nil, errors.New("untracked")
		}
	case "Mystical Archive", "Double Masters: Variants":
		variant = strings.Replace(variant, "Showcase Frame", "", 1)
	case "D&D: Adventures in the Forgotten Realms: Variants",
		"Unfinity: Variants":
		if cardName == "Zariel, Archduke of Avernus" && variant == "Showcase Frame" {
			variant = "Borderless"
		}
	case "Promo":
		switch cardName {
		case "Eye of Ugin":
			if variant == "" {
				edition = "J20"
			}
		case "Bloodchief's Thirst",
			"Into the Roil":
			variant = "Promo Pack"
		case "Dauntless Dourbark":
			variant = "PDCI"
		case "Jaya Ballard, Task Mage":
			variant = "Resale"
		case "Greater Auramancy":
			if variant == "" {
				edition = "P22"
			}
		case "Conjurer's Closet":
			if variant == "" {
				edition = "PW21"
			}
		}
	default:
		if strings.HasSuffix(edition, ": Variants") && variant == "" {
			switch {
			case mtgmatcher.HasEtchedPrinting(cardName):
				variant = "Etched"
			case mtgmatcher.HasExtendedArtPrinting(cardName):
				variant = "Extended Art"
			case mtgmatcher.HasBorderlessPrinting(cardName):
				variant = "Borderless"
			case mtgmatcher.HasShowcasePrinting(cardName):
				variant = "Showcase"
			}
		}
	}

	// Cut a few tags a the end of the card
	if strings.HasSuffix(cardName, "Promo") {
		cuts := mtgmatcher.Cut(cardName, "Promo")
		cardName = cuts[0]
	} else if strings.HasSuffix(cardName, "PROMO") {
		cuts := mtgmatcher.Cut(cardName, "PROMO")
		cardName = cuts[0]
		if strings.HasSuffix(cardName, "Textless") {
			cuts = mtgmatcher.Cut(cardName, "Textless")
			cardName = cuts[0]
		}
	}

	variants := mtgmatcher.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		switch edition {
		// Skip the check for the following sets, known to contain JPN-only cards
		case "War of the Spark":
		case "Promo":
			// Black Lotus - Ultra Pro Puzzle - Eight of 9
			if cardName == "Black Lotus" {
				return nil, errors.New("untracked")
			}
		default:
			// If notes contain the same language there are good chances
			// it's a foreign card
			if !strings.Contains(variant, variants[1]) {
				return nil, errors.New("non-english")
			}
		}
	}
	if strings.Contains(cardName, " - ") {
		variants := strings.Split(cardName, " - ")
		cardName = variants[0]
		if len(variants) > 1 {
			if variant != "" {
				variant += " "
			}
			variant += variants[1]
		}
	}

	cardName = strings.TrimSuffix(cardName, " -")
	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	if mtgmatcher.IsBasicLand(cardName) {
		cardName = strings.TrimSuffix(cardName, "Gift Pack")

		if variant == "" || len(variant) == 1 || variant == "This is NOT the full art version" {
			if variant != "" {
				variant += " "
			}

			maybeNum = strings.Replace(maybeNum, "v2", "", 1)
			maybeNum = strings.TrimPrefix(strings.ToLower(maybeNum), "plains")
			maybeNum = strings.TrimPrefix(strings.ToLower(maybeNum), "island")
			maybeNum = strings.TrimPrefix(strings.ToLower(maybeNum), "swamp")
			maybeNum = strings.TrimPrefix(strings.ToLower(maybeNum), "mountain")
			maybeNum = strings.TrimPrefix(strings.ToLower(maybeNum), "forest")
			variant += maybeNum
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoilFromName,
	}, nil
}

func cleanVariant(variant string) string {
	if strings.Contains(variant, "Picture") {
		variant = strings.Replace(variant, "Picture 1", "", 1)
		variant = strings.Replace(variant, "Picture 2", "", 1)
		variant = strings.Replace(variant, "Picture 3", "", 1)
		variant = strings.Replace(variant, "Picture 4", "", 1)
	}
	if strings.Contains(variant, "Artist") {
		variant = strings.Replace(variant, "Artist ", "", 1)
	}
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)
	variant = strings.Replace(variant, ",", "", -1)
	variant = strings.Replace(variant, ".", "", -1)
	variant = strings.Replace(variant, "- ", "", 1)
	variant = strings.Replace(variant, "  ", " ", -1)
	variant = strings.Replace(variant, "\r\n", " ", -1)
	return strings.TrimSpace(variant)
}

var preserveTags = []string{
	"Etched",
	"Step-and-Compleat",
	"Serialized",
}

func PreprocessBuylist(card CSIPriceEntry) (*mtgmatcher.Card, error) {
	num := strings.TrimLeft(card.Number, "0")
	cleanVar := cleanVariant(card.Notes)
	edition := card.ItemSet
	isFoil := card.IsFoil == 1
	cardName := card.Name
	variant := num
	if variant == "" {
		variant = cleanVar
	}

	fixup, found := cardTable[cardName]
	if found {
		cardName = fixup
	}
	vars, found := variantTable[variant]
	if found {
		variant = vars
		cleanVar = vars
	}

	switch edition {
	case "Coldsnap Theme Deck":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, mtgmatcher.ErrUnsupported
		}
	case "Zendikar", "Battle for Zendikar", "Oath of the Gatewatch":
		// Strip the extra letter from the name
		if mtgmatcher.IsBasicLand(cardName) {
			cardName = strings.Fields(cardName)[0]
		}
	case "Mystery Booster - The List",
		"Secret Lair":
		variant = cleanVar

		if num != "" && cardName != "Everythingamajig" && cardName != "Ineffable Blessing" {
			variant = num + " " + cleanVar
		}
	case "Promo":
		variant = cleanVar
		switch variant {
		case "Ravnica Weekend Promo":
			edition = variant
			variant = num
		case "Stained Glass Art":
			edition = "SLD"
			variant = num
		default:
			switch cardName {
			case "Demonic Tutor":
				if variant == "Daarken Judge Rewards Promo" {
					variant = "Judge 2008"
				} else if variant == "Anna Steinbauer Judge Promo" {
					variant = "Judge 2020"
				}
			case "Vampiric Tutor":
				if variant == "Judge Rewards Promo Old Border" {
					variant = "Judge 2000"
				} else if variant == "Judge Rewards Promo New Border" {
					variant = "Judge 2018"
				}
			case "Wasteland":
				if variant == "Judge Rewards Promo Carl Critchlow art" {
					variant = "Judge 2010"
				} else if variant == "Judge Rewards Promo Steve Belledin art" {
					variant = "Judge 2015"
				}
			case "Vindicate":
				if variant == "Judge Rewards Promo Mark Zug art" {
					variant = "Judge 2007"
				} else if variant == "Judge Rewards Promo Karla Ortiz art" {
					variant = "Judge 2013"
				}
			case "Fling":
				// Only one of the two is present
				if variant == "Gateway Promo Wizards Play Network Daren Bader art" {
					variant = "DCI"
				}
			case "Sylvan Ranger":
				if variant == "Judge Rewards Promo Mark Zug art" {
					variant = "WPN"
				}
			case "Goblin Warchief":
				if variant == "Friday Night Magic Promo Old Border" {
					variant = "FNM 2006"
				} else if variant == "Friday Night Magic Promo New Border" {
					variant = "FNM 2016"
				}
			case "Cabal Therapy":
				if strings.HasPrefix(variant, "Gold-bordered") {
					variant = "2003"
				}
			case "Rishadan Port":
				if strings.HasPrefix(variant, "Gold-bordered") {
					variant = "2000"
				}
			case "Hangarback Walker":
				edition = "Love your LGS"
			case "Chord of Calling", "Wrath of God":
				edition = "Double Masters"
				variant = "Release"
			case "Conjurer's Closet":
				edition = "PW21"
			case "Bolas's Citadel":
				edition = "PWAR"
				variant = "79"
			case "Cryptic Command":
				if variant == "Qualifier Promo" {
					edition = "PPRO"
					variant = "2020-1"
				}
			case "Dauntless Dourbark":
				edition = "PDCI"
				variant = "12"
			case "Eye of Ugin":
				edition = "J20"
			case "Serra Avatar":
				if variant == "Junior Super Series Promo Dermot Power art" {
					edition = "PSUS"
					variant = "2"
				}
			case "Steward of Valeron":
				edition = "PURL"
			case "Goblin Guide":
				if variant == "Love Your Local Game Store Promo" {
					edition = "PLG21"
				}
			case "Llanowar Elves":
				if variant == "Friday Night Magic Promo" {
					edition = "FNM"
					variant = "11"
				} else if variant == "Open House Promo" {
					edition = "PDOM"
					variant = "168"
				}
			case "Masked Vandal":
				edition = "KHM"
				variant = "405"
			}
		}
	}

	// Add previously removed/ignored tags
	for _, tag := range preserveTags {
		if strings.Contains(cleanVar, tag) && !strings.Contains(variant, tag) {
			variant += " " + tag
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}

func Preprocess(card CSICard) (*mtgmatcher.Card, error) {
	cardName := card.Name
	variant := card.Variation
	edition := card.Edition

	if mtgmatcher.Contains(cardName, "Signed by") {
		return nil, errors.New("not singles")
	}

	fields := mtgmatcher.SplitVariants(cardName)
	cardName = fields[0]
	if len(fields) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(fields[1:], " ")
	}

	switch edition {
	case "Online Arena":
		return nil, errors.New("not supported")
	case "Black Bordered (foreign)":
		switch variant {
		case "German", "French", "Spanish", "Chinese", "Korean":
			return nil, errors.New("not supported")
		case "Italian":
			edition = "FBB"
		case "Japanese":
			edition = "4BB"
		}
	case "Ikoria: Lair of Behemoths: Variants":
		if variant == "Japanese" {
			switch cardName {
			case "Dirge Bat", "Mysterious Egg", "Crystalline Giant":
				variant += " Godzilla"
			default:
				return nil, errors.New("not supported")
			}
		}
	case "Prerelease Promo":
		switch cardName {
		case "On Serra's Wings":
			return nil, errors.New("does not exist")
		}
	case "Portal 3 Kingdoms":
		if variant == "Japanese" || variant == "Chinese" {
			return nil, errors.New("not english")
		}
	case "Mystical Archive", "Double Masters: Variants":
		variant = strings.Replace(variant, "Showcase Frame", "", 1)
	case "Dominaria United: Variants":
		if variant == "Stained Glass Frame" {
			variant = "Showcase"
		}
	}

	return &mtgmatcher.Card{
		Id:        card.ScryfallId,
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      card.IsFoil,
	}, nil
}

var sealedRenames = map[string]string{
	"Global Series - Planeswalker Decks - Jiang Yanggu & Mu Yanling": "Global Series Jiang Yanggu and Mu Yanling",

	"War of the Spark -  Booster Box (Japanese)": "War of the Spark Booster Box (Non-English JAPANESE)",

	"Revised (Third Edition) - Booster Box": "Revised Edition Booster Box",
	"Pro Tour 1996 Collector Set":           "Pro Tour Collector Set",
	"Time Spiral - Tournament Deck":         "Time Spiral Tournament Pack",

	"Starter - Booster Box":  "Starter 1999 Booster Box",
	"Starter - Booster Pack": "Starter 1999 Booster Pack",

	"Mystery Booster (Convention Edition) - Booster Box":       "Mystery Booster Booster Box (Convention Edition)",
	"Mystery Booster (Convention Edition) - Booster Pack":      "Mystery Booster Booster Pack (Convention Edition)",
	"Mystery Booster (Convention Edition 2021) - Booster Box":  "Mystery Booster Booster Box (Convention Edition - 2021)",
	"Mystery Booster (Convention Edition 2021) - Booster Pack": "Mystery Booster Booster Pack (Convention Edition - 2021)",

	"Secret Lair Drop Series - Ultimate Edition":           "Secret Lair Ultimate Edition Box",
	"Phyrexia: All Will Be One - Bundle: Compleat Edition": "Phyrexia All Will Be One Compleat Bundle",
}

func preprocessSealed(productName, edition string) (string, error) {
	switch edition {
	case "Mystery Booster - The List":
		edition = "MB1"
	case "Bulk Magic":
		return "", mtgmatcher.ErrUnsupported
	case "World Championship Decks":
		year := mtgmatcher.ExtractYear(productName)
		if year == "1996" {
			edition = "PTC"
		} else {
			productName = strings.Replace(productName, "("+year+") ", "", 1)
			productName = strings.TrimSuffix(productName, " Deck")
			productName = strings.Replace(productName, " - ", " ", 1)
			productName = year + " " + productName
			edition += " " + year
		}
	case "Secret Lair":
		if strings.Contains(productName, "Ultimate") {
			edition = "SLU"
		} else {
			edition = "SLD"
		}
	default:
		if strings.Contains(productName, "Challenger Deck") {
			edition = ""
		}
	}

	// If edition is empty, do not return and instead loop through
	var setCode string
	set, err := mtgmatcher.GetSetByName(edition)
	if err != nil {
		if edition != "" {
			return "", err
		}
	} else {
		setCode = set.Code
	}

	productName = strings.TrimSuffix(productName, " (1)")
	productName = strings.Replace(productName, "(6)", "Case", 1)

	rename, found := sealedRenames[productName]
	if found {
		productName = rename
	}

	switch {
	case strings.Contains(productName, "Life Counter"),
		strings.Contains(productName, "Booster Box (3)"),
		strings.Contains(productName, "Scene Box"),
		strings.Contains(productName, "Player's Guide"),
		strings.Contains(productName, "Bundle Card Box"),
		strings.Contains(productName, "D20 Set"),
		strings.Contains(productName, "Born of the Gods - Japanese"),
		strings.Contains(productName, "Variety Pack"):
		return "", mtgmatcher.ErrUnsupported
	case strings.HasPrefix(productName, "From the Vault"),
		strings.HasPrefix(productName, "Signature Spellbook"):
		productName = strings.TrimSuffix(productName, " - Box Set")
	}

	var uuid string
	for _, set := range mtgmatcher.GetSets() {
		if setCode != "" && setCode != set.Code {
			continue
		}

		for _, sealedProduct := range set.SealedProduct {
			if mtgmatcher.SealedEquals(sealedProduct.Name, productName) {
				uuid = sealedProduct.UUID
				break
			}
		}

		if uuid == "" {
			for _, sealedProduct := range set.SealedProduct {
				// If not found, look if the a chunk of the name is present in the deck name
				switch {
				case strings.Contains(productName, "Archenemy"),
					strings.Contains(productName, "Duels of the Planeswalkers"),
					strings.Contains(productName, "Commander"),
					strings.Contains(productName, "Secret Lair"),
					strings.Contains(productName, "Planechase"):
					decks, found := sealedProduct.Contents["deck"]
					if found {
						for _, deck := range decks {
							// Work around internal names that are too long, like
							// "Teeth of the Predator - the Garruk Wildspeaker Deck"
							deckName := strings.Split(deck.Name, " - ")[0]
							if mtgmatcher.SealedContains(productName, deckName) {
								uuid = sealedProduct.UUID
								break
							}
							// Scret Lair may have
							deckName = strings.TrimSuffix(strings.ToLower(deckName), " foil")
							if mtgmatcher.SealedContains(productName, deckName) {
								uuid = sealedProduct.UUID
								break
							}
						}
					}
				}
				if uuid != "" {
					break
				}
			}
		}

		// Last chance (in case edition is known)
		if uuid == "" && setCode != "" && len(set.SealedProduct) == 1 {
			uuid = set.SealedProduct[0].UUID
		}

		if uuid != "" {
			break
		}

	}

	return uuid, nil
}
