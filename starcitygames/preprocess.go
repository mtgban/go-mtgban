package starcitygames

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Who / What / When / Where / Why": "Who // What // When // Where // Why",

	"Jushi Apprentice // Tomoya The Revealer // Tomoya The Revealer": "Jushi Apprentice // Tomoya The Revealer",
}

func preprocess(card *SCGCardVariant, edition, language string, foil bool, number string) (*mtgmatcher.Card, error) {
	cardName := strings.Replace(card.Name, "&amp;", "&", -1)

	if edition != "Unfinity" && strings.Contains(cardName, "{") && strings.Contains(cardName, "}") {
		return nil, errors.New("non-single")
	}

	cardName = strings.Replace(cardName, "{", "", -1)
	cardName = strings.Replace(cardName, "}", "", -1)

	edition = strings.Replace(edition, "&amp;", "&", -1)
	if strings.HasSuffix(edition, "(Foil)") {
		edition = strings.TrimSuffix(edition, " (Foil)")
		foil = true
	}

	variant := strings.Replace(card.Subtitle, "&amp;", "&", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	vars = mtgmatcher.SplitVariants(edition)
	edition = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	switch {
	// These are the sealed packs
	case strings.HasPrefix(cardName, "APAC Land"),
		strings.HasPrefix(cardName, "Euro Land"):
		return nil, errors.New("non-single")
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	if mtgmatcher.IsBasicLand(cardName) {
		if strings.Contains(variant, "APAC") {
			edition = "Asia Pacific Land Program"
		} else if strings.Contains(variant, "Euro") {
			edition = "European Land Program"
		}
	}

	// Make sure not to pollute variants with the language otherwise multiple
	// variants may be aliased (ie urzalands)
	switch language {
	case "Japanese", "ja":
		switch edition {
		case "Chronicles":
			edition = "BCHR"
		case "4th Edition - Black Border":
			edition = "4BB"
			variant = strings.TrimSuffix(variant, " BB")
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

	switch edition {
	case "Promo: General":
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
		}
		if strings.Contains(variant, "The Lord of the Rings") && number != "" {
			variant = number
		}
	case "Unfinity":
		if strings.Contains(variant, "/") && number != "" {
			variant = number
		}
		if strings.Contains(cardName, "Sticker Sheet") {
			edition = "SUNF"
		}
	case "The Lord of the Rings Commander - Serialized":
		if cardName == "Sol Ring" && number != "" {
			variant = number
		}
	case "The Lost Caverns of Ixalan - Alternate Foil":
		if cardName == "Cavern of Souls" && number != "" {
			variant = number
		}
	case "Special Guests - Alternate Foil":
		if cardName == "Mana Crypt" && number != "" {
			variant = number
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
