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

	"Circle of Protection Red FNM Foil": "Circle of Protection: Red",

	"Who/What/When/Where/Why": "Who",

	"Startled Awake // Persistant Nightmare": "Startled Awake //  Persistent Nightmare",

	"The Ultimate Nightmare of Wizards of the Coast(R": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",

	"Our Market Research Shows That Players Like Really Long Card Names So We Made This Card to Have the Absolute Longest Card Nam": "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
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
	"Core 21 Prerelease Promo":                         "Prerelease",
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

	isFoilFromName := false
	switch cardName {
	case "City's Blessing",
		"Experience Counter",
		"Manifest",
		"Morph",
		"Poison Counter",
		"The Monarch":
		return nil, errors.New("not single")
	case "Teferi, Master of Time",
		"Lukka, Coppercoat Outcast",
		"Brokkos, Apex of Forever":
		return nil, errors.New("impossible to track")
	case "Corpse Knight":
		if variant == "2/2 Power and Toughness" {
			variant = "misprint"
		}
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
		if variant == "Gateway Promo Wizards Play Network DCI Logo" {
			variant = "WPN 2010"
		} else if variant == "Judge Rewards Promo Mark Zug art" {
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
	case "Deathbringer Regent":
		if variant == "Release 27 Promo" {
			variant = "Release"
		}
	case "Rukh Egg":
		if variant == "Eighth Edition Prerelease Promo" {
			variant = "Release Promo"
		}
	case "B.F.M. (Big Furry Monster)":
		if variant == "Big Furry Monster Left Side" {
			variant = "28"
		} else {
			variant = "29"
		}
	case "Cabal Therapy":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2003"
		}
	case "Rishadan Port":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2000"
		}
	case "Island":
		if variant == "Arena League Promo 2001 Mark Poole art" {
			variant = "Arena 2002"
		}
	case "Solemn Simulacrum":
		if edition == "Commander Anthology Volume II" && maybeNum == "" {
			return nil, errors.New("unsupported")
		}
	case "Temple of the False God":
		if edition == "Commander Anthology Volume II" && maybeNum == "" {
			return nil, errors.New("unsupported")
		}
	case "Sorcerous Spyglass":
		if variant == "Silver Planeswalker Symbol" {
			return nil, errors.New("unsupported")
		}
	case "Disenchant":
		if strings.Contains(variant, "Arena") {
			variant = "Arena 1996"
		}
	case "Mana Crypt":
		if strings.HasPrefix(variant, "Harper Prism Promo") {
			if !strings.Contains(variant, "English") {
				return nil, errors.New("non-english")
			}
		}

	default:
		if strings.Contains(strings.ToLower(cardName), "token") ||
			strings.Contains(strings.ToLower(cardName), "emblem") ||
			strings.Contains(cardName, "Checklist") ||
			strings.Contains(cardName, "Oversized") ||
			strings.Contains(cardName, "Booster Box") ||
			strings.Contains(cardName, "Booster Pack") ||
			strings.Contains(cardName, "Fat Pack") ||
			strings.Contains(cardName, "Bundle") ||
			strings.Contains(cardName, "Series") ||
			strings.Contains(cardName, "Spindown") ||
			strings.Contains(cardName, "Box Set") ||
			strings.Contains(cardName, "Bulk") ||
			strings.Contains(cardName, "Signed by") ||
			strings.Contains(cardName, "Proxy Card") ||
			strings.Contains(cardName, "MTG Arena Code Card") ||
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

	switch edition {
	case "", "Overwhelming Swarm", "Special Offers", "Unique Boutique", "Magic Mics Merch",
		"Misprints & Oddities", "Authenticated Collectibles", "New Player Series",
		"Heavy Metal Magic Supplies", "Oversized Cards", "Modern Horizons: Art Series",
		"Art Series: Modern Horizons", "Art Series: Modern Horizon", "Vanguard",
		"Fourth (Alternate Edition)",
		"Challenger Decks 2020", "Challenger Decks 2019", "Challenger Decks 2018":
		return nil, errors.New("set not mtg")
	case "Prerelease Promos":
		variant = "Prerelease Foil"
	case "Portal 3 Kingdoms", "Jace vs. Chandra":
		if strings.Contains(cardName, "Japanese") {
			return nil, errors.New("not english")
		}
	case "Duel Decks: Anthology":
		if len(maybeNum) == 3 {
			edition = maybeNum
		}
	case "Commander Anthology Volume II", "Unstable", "Unsanctioned":
		if cardName != "Amateur Auteur" && cardName != "Everythingamajig" {
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
	case "Legends":
		if strings.Contains(cardName, "Italian") || strings.Contains(variant, "Italian") {
			return nil, errors.New("non-english")
		}
	default:
		if strings.Contains(variant, "Oversized") {
			return nil, errors.New("not single")
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
		if !strings.Contains(variant, variants[1]) {
			return nil, errors.New("non-english")
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
