package coolstuffinc

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Banewhip Punishe":           "Banewhip Punisher",
	"Chant of the Vitu-Ghazi":    "Chant of Vitu-Ghazi",
	"Curse of the Fool's Wisdom": "Curse of Fool's Wisdom",
	"Deputized Protestor":        "Deputized Protester",
	"Erdwall Illuminator":        "Erdwal Illuminator",
	"Mistfoot Kirin":             "Misthoof Kirin",
	"Holy Justicar":              "Holy Justiciar",
	"Nearhearth Chaplain":        "Nearheath Chaplain",
	"Shatter Assumpions":         "Shatter Assumptions",
	"Stratozeppilid":             "Stratozeppelid",
	"Elder Garganoth":            "Elder Gargaroth",
	"Immerstrurm Predator":       "Immersturm Predator",
	"Inspiried Idea":             "Inspired Idea",
	"Shadow of Morality":         "Shadow of Mortality",

	"Circle of Protection Red FNM Foil": "Circle of Protection: Red",

	"Who/What/When/Where/Why": "Who",

	"Startled Awake // Persistant Nightmare": "Startled Awake //  Persistent Nightmare",

	"Needleverge Patheway // Pillarverge Pathway": "Needleverge Pathway // Pillarverge Pathway",
}

var variantTable = map[string]string{
	"Tarkir Dragonfury Prerelease Promo":               "Tarkir Dragonfury Promo",
	"Jeff A Menges":                                    "Jeff A. Menges",
	"Jeff a Menges":                                    "Jeff A. Menges",
	"Not available in Draft Boosters":                  "",
	"San Diego Comic-Con Promo M15":                    "SDCC 2014",
	"San Diego Comic-Con Promo M14":                    "SDCC 2013",
	"Arena Foil no Urza's Saga symbol Donato Giancola": "Arena 1999 misprint",
	"EURO Land White Cliffs of Dover Ben Thompson art": "EURO  White Cliffs of Dover",
	"EURO Land Danish Island Ben Thompson art":         "EURO Land Danish Island",
	"Aether Revolt Prerelease Promo":                   "AER Prerelease",
	"Core 21 Prerelease Promo":                         "M21 Prerelease",
	"Silver Planeswalker Symbol Core 21":               "M21 Promo Pack",
	"Throne of Eldraine Prerelease promo":              "ELD Prerelease",
	"Eighth Edition Prerelease Promo":                  "Release Promo",
	"Release 27 Promo":                                 "Release",
	"2/2 Power and Toughness":                          "misprint",
	"Big Furry Monster Left Side":                      "28",
	"Big Furry Monster Right Side":                     "29",
	"Arena League Promo 2001 Mark Poole art":           "Arena 2002",
}

func preprocess(cardName, edition, notes, maybeNum string) (*mtgmatcher.Card, error) {
	// Clean up notes, removing extra prefixes, and ueless characters
	variant := strings.TrimPrefix(notes, "Notes:")
	if strings.Contains(variant, "Deckmaster") {
		cuts := mtgmatcher.Cut(variant, "Deckmaster")
		variant = cuts[0]
	}
	if strings.Contains(variant, "Picture") {
		variant = strings.Replace(variant, "Picture 1", "", 1)
		variant = strings.Replace(variant, "Picture 2", "", 1)
		variant = strings.Replace(variant, "Picture 3", "", 1)
		variant = strings.Replace(variant, "Picture 4", "", 1)
	}
	if strings.Contains(variant, "Artist") {
		variant = strings.Replace(variant, "Artist ", "", 1)
	}
	variant = strings.Replace(variant, "\n", " ", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)
	variant = strings.Replace(variant, ")", "", -1)
	variant = strings.Replace(variant, ",", "", -1)
	variant = strings.Replace(variant, "- ", "", 1)
	variant = strings.Replace(variant, "  ", " ", -1)
	variant = strings.TrimSuffix(variant, " ")
	variant = strings.TrimSuffix(variant, ".")
	variant = strings.TrimSpace(variant)

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
		if variant == "Judge Rewards Promo" {
			variant = "Judge 2008"
		} else if variant == "Judge Promo" {
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
			variant = "WPN 2010"
		}
	case "Sylvan Ranger":
		if variant == "Judge Rewards Promo Mark Zug art" {
			variant = "WPN 2011"
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
		"Authenticated Collectibles", "New Player Series", "Heavy Metal Magic Supplies":
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
		switch cardName {
		case "Boros Guildgate":
			return nil, errors.New("unsupported")
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
	case "Mystical Archive":
		if strings.Contains(variant, "Showcase Frame") {
			variant = strings.Replace(variant, "Showcase Frame", "", 1)
		}
	case "D&D: Adventures in the Forgotten Realms: Variants":
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
			variant = "Gateway 2007"
		case "Jaya Ballard, Task Mage":
			variant = "Resale"
		}
	}

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

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	if strings.HasSuffix(cardName, " -") {
		cardName = cardName[:len(cardName)-2]
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

var promoTable = map[string]string{
	"Abrupt Decay":               "WMC",
	"Abzan Beastmaster":          "F15",
	"Accumulated Knowledge":      "F04",
	"Acidic Slime":               "F12",
	"Acquire":                    "PI14",
	"Aether Hub":                 "F17",
	"Ainok Tracker":              "UGIN",
	"Ajani Steadfast":            "PS14",
	"Ajani, Caller of the Pride": "PSDC",
	"Albino Troll":               "F02",
	"Altar of the Brood":         "UGIN",
	"Anathemancer":               "F10",
	"Ancient Grudge":             "F12",
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
		variant += fields[1]
	}

	switch edition {
	case "Black Bordered (foreign)":
		switch variant {
		case "German", "French":
			return nil, errors.New("not supported")
		case "Spanish", "Chinese", "Japanese":
			edition = "4BB"
			return nil, errors.New("not supported")
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
	case "Mystical Archive":
		if strings.Contains(variant, "Showcase Frame") {
			variant = strings.Replace(variant, "Showcase Frame", "", 1)
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      card.IsFoil,
	}, nil
}
