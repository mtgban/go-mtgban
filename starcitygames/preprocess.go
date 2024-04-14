package starcitygames

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Who / What / When / Where / Why": "Who // What // When // Where // Why",

	"Jushi Apprentice // Tomoya The Revealer // Tomoya The Revealer": "Jushi Apprentice // Tomoya The Revealer",
}

func languageTags(language, edition, variant, number string) (string, string, error) {
	switch language {
	case "Japanese", "ja":
		switch edition {
		case "Chronicles":
			edition = "BCHR"
		case "4th Edition - Black Border":
			edition = "4BB"
			variant = strings.TrimSuffix(variant, " BB")
		case "Strixhaven Mystical Archive",
			"Strixhaven Mystical Archive - Foil Etched":
			num, err := strconv.Atoi(strings.TrimLeft(number, "0"))
			if err != nil {
				return "", "", err
			}
			if num < 64 {
				return "", "", errors.New("non-english")
			}
		case "War of the Spark":
			if !strings.Contains(variant, "Alternate Art") {
				return "", "", errors.New("non-english")
			}
		default:
			if variant != "" {
				variant += " "
			}
			variant += "Japanese"
		}
	case "Italian", "it":
		if edition == "Renaissance" {
			edition = "Rinascimento"
		} else {
			if variant != "" {
				variant += " "
			}
			variant += "Italian"
		}
	}
	return edition, variant, nil
}

func preprocess(card *SCGCardVariant, cardEdition, language string, foil bool, cn string) (*mtgmatcher.Card, error) {
	// Processing variant first because it gets added on later
	variant := strings.Replace(card.Subtitle, "&amp;", "&", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	cardName := strings.Replace(card.Name, "&amp;", "&", -1)
	cardName = strings.Replace(cardName, "{", "", -1)
	cardName = strings.Replace(cardName, "}", "", -1)

	// Check tokens with the same name as certain cards (and skip them)
	isToken := strings.HasPrefix(card.Name, "{") && strings.HasSuffix(card.Name, "}")
	if isToken {
		switch cardName {
		case "Phyrexian Hydra":
			return nil, mtgmatcher.ErrUnsupported
		}
	}

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	edition := strings.Replace(cardEdition, "&amp;", "&", -1)
	if strings.HasSuffix(edition, "(Foil)") {
		edition = strings.TrimSuffix(edition, " (Foil)")
		foil = true
	}
	vars = mtgmatcher.SplitVariants(edition)
	edition = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	var err error
	edition, variant, err = languageTags(language, edition, variant, cn)
	if err != nil {
		return nil, err
	}

	// SKU documented as
	// * for singles:
	// SGL-[Brand]-[Set]-[Collector Number]-[Language][Foiling][Condition]
	// * for world champtionship:
	// SGL-[Brand]-WCPH-[Year][Player Initials][Set][Collector Number][Sideboard]-[Language][Foiling][Condition]
	// * for promotional cards:
	// SGL-[Brand]-PRM-[Promo][Set][Collector Number]-[Language][Foiling][Condition]
	//
	// examples
	// * SGL-MTG-PRM-SECRET_SLD_1095-ENN1
	// * SGL-MTG-PRM-PP_MKM_187-ENN
	// * SGL-MTG-PWSB-PCA_115-ENN1

	canProcessSKU := language == "en" || language == "English"
	// We can't use the numbers reported because they match the plain version
	// and the Match search doesn't upgrade these custom tags
	if strings.Contains(variant, "Serial") || strings.Contains(variant, "Compleat") {
		canProcessSKU = false
	}

	var number string
	fields := strings.Split(card.Sku, "-")
	if len(fields) > 3 && canProcessSKU {
		setCode := fields[2]
		number = strings.TrimLeft(fields[3], "0")

		switch setCode {
		case "MPS2":
			setCode = "MPS"
		case "MPS3":
			setCode = "MP2"
		case "PWSB":
			setCode = "PLST"
			fields := strings.Split(number, "_")
			if len(fields) == 2 {
				number = fields[0] + "-" + strings.TrimLeft(fields[1], "0")
			} else if len(fields) == 4 {
				if fields[0] == "PRM" {
					number = fields[2] + "-" + strings.TrimLeft(fields[3], "0")
				}
			}
		case "PRM", "PRM3":
			fields := strings.Split(number, "_")
			if len(fields) > 2 && fields[0] == "SECRET" {
				setCode = fields[1]
				number = strings.TrimLeft(fields[2], "0")
			} else if strings.HasPrefix(number, "PRE_LTR_") {
				number = strings.TrimPrefix(number, "PRE_LTR_")
				if strings.HasSuffix(number, "a") {
					setCode = "PLTR"
					number = strings.Replace(number, "a", "s", 1)
				} else if strings.HasSuffix(number, "b") {
					setCode = "LTR"
					number = strings.TrimSuffix(number, "b")
				}
			}
		default:
			// Disable quick search for Oversized cards, as they are embdded
			// in the same set and id lookup doesn't support extending over
			if strings.Contains(variant, "Oversized") {
				setCode = ""
				break
			}

			if strings.Contains(cardName, "//") && strings.HasSuffix(number, "a") {
				number = strings.TrimSuffix(number, "a")
			}

			// Handle set renames like OTC2 and LTR2
			_, err := mtgmatcher.GetSet(setCode)
			if err != nil && len(setCode) > 3 && unicode.IsDigit(rune(setCode[len(setCode)-1])) {
				setCode = setCode[:len(setCode)-1]
			}
		}

		// Check if we found it and return the id
		out := mtgmatcher.MatchWithNumber(cardName, setCode, number)
		if len(out) == 1 {
			return &mtgmatcher.Card{
				Id:        out[0].UUID,
				Foil:      foil,
				Variation: variant,
				Language:  language,
			}, nil
		}
	}

	switch edition {
	case "Promo: General",
		"Promo: General - Alternate Foil":
		switch cardName {
		case "Swiftfoot Boots":
			if variant == "Launch" {
				edition = "PW22"
				variant = ""
			}
		case "Dismember":
			if variant == "Commander Party Phyrexian" {
				edition = "PW22"
			}
		case "Rukh Egg":
			if variant == "10th Anniversary" {
				edition = "P8ED"
			}
		case "Mind Stone":
			if variant == "Bring-a-Friend" {
				edition = "PW21"
				variant = ""
			}
		case "Scavenging Ooze":
			if variant == "Love Your LGS Retro Frame" {
				edition = "PLG21"
			}
		case "Arcane Signet":
			switch variant {
			case "Play Draft Retro Frame":
				edition = "P30M"
				variant = "1P"
			case "Festival":
				edition = "P30M"
				variant = "1F"
			case "Festival Foil Etched":
				edition = "P30M"
				variant = "1F★"
			}
		case "Counterspell":
			switch variant {
			case "Festival Full Art":
				edition = "PF24"
			}
		case "Llanowar Elves":
			switch variant {
			case "Resale Retro Frame":
				edition = "PRES"
			}
		case "Pyromancer's Gauntlet":
			switch variant {
			case "Hasbro Retro Frame":
				edition = "PMEI"
			}
		case "Lathril, Blade of the Elves":
			switch variant {
			case "Resale Foil Etched":
				edition = "PRES"
			}
		case "Rampant Growth":
			if variant == "Release Foil Etched" {
				edition = "PW23"
			}
		case "Commander's Sphere":
			if variant == "Play Draft" {
				edition = "PW24"
			}
		}
	case "Unfinity":
		if strings.Contains(cardName, "Sticker Sheet") {
			edition = "SUNF"
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
		Language:  language,
	}, nil
}

var sealedRenames = map[string]string{
	"Commander Legends: Battle for Baldur's Gate Commander Deck - Set of 4": "Commander Legends Battle for Baldurs Gate Commander Deck Display",

	"Doctor Who Commander Deck - Set of 4":        "Doctor Who Commander Deck Display",
	"Modern Event Deck - March of the Multitudes": "Magic Modern Event Deck",
	"Global Series: Jiang Yanggu vs. Mu Yanling":  "Global Series Jiang Yanggu and Mu Yanling",
	"Deckmasters: Garfield vs. Finkel":            "Garfield vs Finkel 2001 Deckmasters Tin",
	"Battle Royale Multi-Player Box Set":          "Battle Royale Multi Player Box set",

	"HASCON 2017 - Silver-Bordered Card Boxed Set of 4": "2017 Magic The Gathering Hascon Collection",

	"Modern Masters (2013 Edition) Booster Box":  "Modern Masters Booster Box",
	"Modern Masters (2013 Edition) Booster Pack": "Modern Masters Booster Pack",
	"Modern Masters (2015 Edition) Booster Box":  "Modern Masters 2015 Booster Box",
	"Modern Masters (2015 Edition) Booster Pack": "Modern Masters 2015 Booster Pack",
	"Modern Masters (2017 Edition) Booster Box":  "Modern Masters 2017 Booster Box",
	"Modern Masters (2017 Edition) Booster Pack": "Modern Masters 2017 Booster Pack",

	"Guild Kit - Boros":    "Guilds of Ravnica Guild Kit Boros",
	"Guild Kit - Dimir":    "Guilds of Ravnica Guild Kit Dimir",
	"Guild Kit - Golgari":  "Guilds of Ravnica Guild Kit Golgari",
	"Guild Kit - Izzet":    "Guilds of Ravnica Guild Kit Izzet",
	"Guild Kit - Selesnya": "Guilds of Ravnica Guild Kit Selesnya",
	"Guild Kit - Azorius":  "Ravnica Allegiance Guild Kit Azorius",
	"Guild Kit - Gruul":    "Ravnica Allegiance Guild Kit Gruul",
	"Guild Kit - Orzhov":   "Ravnica Allegiance Guild Kit Orzhov",
	"Guild Kit - Rakdos":   "Ravnica Allegiance Guild Kit Rakdos",
	"Guild Kit - Simic":    "Ravnica Allegiance Guild Kit Simic",

	"San Diego Comic-Con 2013 - Black Planeswalker Card Boxed Set-of-5": "SDCC 2013 Box-Set (Set of 5 Planeswalkers)",
	"San Diego Comic-Con 2014 - Black Planeswalker Card Boxed Set-of-6": "SDCC 2014 Box-Set (Set of 6 Planeswalkers) - No Axe",
	"San Diego Comic-Con 2015 - Black Planeswalker Card Boxed Set-of-5": "SDCC 2015 Box-Set (Set of 5 Planeswalkers) - Book Included",
	"San Diego Comic-Con 2016 - Planeswalker Card Boxed Set-of-5":       "SDCC 2016 Box-Set (Set of 5 Planeswalkers)",
	"San Diego Comic-Con 2017 - Planeswalker Card Boxed Set-of-6":       "SDCC 2017 Box-Set (Set of 6 Planeswalkers) - No Poster",
	"San Diego Comic-Con 2018 - Planeswalker Card Boxed Set-of-5":       "SDCC 2018 Box-Set (Set of 5 Planeswalkers)",
	"San Diego Comic-Con 2019 - Dragon's Endgame Card Boxed Set-of-5":   "SDCC 2019 Box-Set",

	"Mystery Booster: Convention Edition Booster Box (2019)":  "Mystery Booster Booster Box Convention Edition",
	"Mystery Booster: Convention Edition Booster Pack (2019)": "Mystery Booster Booster Pack Convention Edition",
	"Mystery Booster: Convention Edition Booster Box (2021)":  "Mystery Booster Booster Box Convention Edition 2021",
	"Mystery Booster: Convention Edition Booster Pack (2021)": "Mystery Booster Booster Pack Convention Edition 2021",
	"Mystery Booster: WPN Edition Booster Box":                "Mystery Booster Booster Box Retail Exclusive",
	"Mystery Booster: WPN Edition Booster Pack":               "Mystery Booster Booster Pack Retail Exclusive",
}

func preprocessSealed(productName string) (string, error) {
	switch {
	case strings.Contains(productName, "History Promo Cards"),
		strings.Contains(productName, "APAC Land"),
		strings.Contains(productName, "Euro Land"):
		return "", errors.New("unsupported")
	}

	rename, found := sealedRenames[productName]
	if found {
		productName = rename
	}

	productName = strings.Replace(productName, "The Lord of the Rings", "The Lord of the Rings Tales of Middle earth", 1)
	productName = strings.Replace(productName, "3rd Edition/Revised", "Revised Edition", 1)
	productName = strings.Replace(productName, "Edition Core Set", "Edition", 1)
	productName = strings.Replace(productName, "Magic 20", "20", 1)

	if strings.Contains(productName, "World Championship") {
		year := mtgmatcher.ExtractYear(productName)
		if year != "" {
			productName = strings.Replace(productName, year+" ", "", 1)
			productName = year + " " + productName
		}
	}

	sets := mtgmatcher.GetSets()

	var uuid string
	for _, set := range sets {
		for _, sealedProduct := range set.SealedProduct {
			if mtgmatcher.SealedEquals(sealedProduct.Name, productName) {
				uuid = sealedProduct.UUID
				break
			}
		}
		if uuid != "" {
			break
		}
	}

	if uuid == "" {
		// The year shouldn't be strictly needed, but since the Contains search
		// on everything is very wide, it's better to avoid false positives
		year := mtgmatcher.ExtractYear(productName)
		if year != "" && (strings.Contains(productName, "Deck") || strings.Contains(productName, "Archenemy")) {
			for _, set := range sets {
				if !strings.Contains(set.Name, year) && !strings.Contains(set.Name, "Archenemy") {
					continue
				}
				for _, sealedProduct := range set.SealedProduct {
					decks, found := sealedProduct.Contents["deck"]
					if found {
						for _, deck := range decks {
							if mtgmatcher.SealedContains(productName, deck.Name) {
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
	}

	return uuid, nil
}
