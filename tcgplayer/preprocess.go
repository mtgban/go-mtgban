package tcgplayer

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func Preprocess(product *TCGProduct) (*mtgmatcher.Card, error) {
	cardName := product.Name
	variant := ""

	if strings.Contains(cardName, " - ") {
		fields := strings.Split(cardName, " - ")
		cardName = fields[0]
		if len(fields) > 1 {
			variant = strings.Join(fields[1:], " ")
		}
	}
	if strings.Contains(cardName, " (") {
		fields := mtgmatcher.SplitVariants(cardName)
		cardName = fields[0]
		if len(fields) > 1 {
			if variant != "" {
				variant += " "
			}
			variant += strings.Join(fields[1:], " ")

			variant = strings.TrimSuffix(variant, " CE")
			variant = strings.TrimSuffix(variant, " IE")
		}
	}

	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	// Skip non-singles cards
	switch {
	case mtgmatcher.IsToken(cardName) || mtgmatcher.IsToken(product.CleanName):
		return nil, errors.New("non-single card")
	}

	edition := ""
	fields := strings.Split(product.URL, "/")
	if len(fields) > 4 {
		edition = fields[4]
		edition = strings.Replace(edition, "-", " ", -1)
		edition = strings.Title(edition)
	}

	switch edition {
	case "Renaissance":
		// Only keep the German for this edition
		if strings.Contains(variant, "French") || strings.Contains(variant, "German") {
			return nil, errors.New("non english")
		}
		if strings.HasSuffix(variant, "Italian") {
			edition = "Rinascimento"
			variant = strings.TrimSuffix(variant, " Italian")
			variants := strings.Split(variant, "\" ")
			if len(variants) > 1 {
				variant = variants[1]
			}
		}
	case "Heros Path Promos",
		"Oversize Cards",
		"Special Occasion",
		"Revised Edition Foreign White Border",
		"Fourth Edition Foreign White Border":
		return nil, errors.New("unsupported")
	case "Portal":
		if variant == "Flavor Text" {
			variant = ""
		} else if variant == "" {
			if len(mtgmatcher.MatchInSet(cardName, "PPOD")) == 0 {
				variant = "Reminder Text"
			}
		}
	case "Battle For Zendikar",
		"Oath Of The Gatewatch":
		if mtgmatcher.IsBasicLand(cardName) {
			if cardName == "Wastes" && variant == "" {
				variant = "183a"
			} else if !strings.HasSuffix(product.CleanName, "Full Art") {
				variant += "a"
			}
		}
	case "Core Set 2020":
		if strings.Contains(variant, "Misprint") {
			variant = "misprint"
		}
	case "Throne Of Eldraine":
		if cardName == "Kenrith, the Returned King" {
			return nil, errors.New("unsupported")
		}
	case "Kaladesh", "Aether Revolt":
		if variant == "17/264" {
			variant = "Intro Pack"
		}
	case "Media Promos",
		"Pro Tour Promos",
		"Unique And Miscellaneous Promos",
		"Launch Party And Release Event Promos",
		"League Promos",
		"Game Day And Store Championship Promos",
		"Wmcq Promo Cards",
		"Wpn And Gateway Promos":
		if strings.HasSuffix(variant, "Ultra Pro Puzzle Quest") ||
			variant == "Redemption Program" { // JPN Nalathni Dragon
			return nil, errors.New("unsupported")
		}
		ed, found := map[string]string{
			"Balduvian Horde":             "PWOR",
			"Char":                        "P15A",
			"Deathless Angel":             "PROE",
			"Jaya Ballard, Task Mage":     "PMPS08",
			"Kamahl, Pit Fighter":         "P15A",
			"Hall of Triumph":             "THP3",
			"Sword of Dungeons & Dragons": "H17",
			"Arcbound Ravager":            "PPRO",
			"Goblin Chieftain":            "PRES",
			"Oran-Rief, the Vastwood":     "PRES",
			"Loam Lion":                   "PRES",
			"Shepherd of the Lost":        "PURL",
			"Sethron, Hurloon General":    "PL21",
		}[cardName]
		if found {
			edition = ed
		} else if edition == "Media Promos" && !strings.Contains(variant, "SDCC") {
			edition = "Magazine Inserts"
		} else if len(mtgmatcher.MatchInSet(cardName, "CP1")) == 1 {
			edition = "CP1"
		} else if len(mtgmatcher.MatchInSet(cardName, "CP2")) == 1 {
			edition = "CP2"
		} else if len(mtgmatcher.MatchInSet(cardName, "CP3")) == 1 {
			edition = "CP3"
		} else if cardName == "Fling" && variant != "DCI" {
			edition = "PWP11"
		} else if edition == "Launch Party And Release Event Promos" && mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		} else {
			switch cardName {
			case "Arasta of the Endless Web":
				edition = "THB"
				variant = "352"
			}
		}

		switch cardName {
		case "Serra Angel":
			if variant == "" {
				edition = "PWOS"
			} else if variant == "25th Anniversary Exposition" {
				edition = "PDOM"
			}
		case "Stocking Tiger":
			if variant == "No Date" {
				variant = "misprint"
			}
		case "Reliquary Tower":
			if variant == "Bring a Friend Promo" {
				edition = "PLGS"
			} else {
				edition = "PM19"
			}
		}
	case "Junior Series Promos":
		// TCG has a single version but there are multiple ones available
		// So just preserve whichever is filed in Scryfall
		ed, found := map[string]string{
			"Royal Assassin":     "PJSE",
			"Sakura-Tribe Elder": "PJSE",
			"Shard Phoenix":      "PJSE",
			"Whirling Dervish":   "PJSE",
			"Mad Auntie":         "PJJT",
		}[cardName]
		if found {
			edition = ed
		} else if variant != "" && len(mtgmatcher.MatchInSet(cardName, "PSUS")) == 1 {
			edition = "PSUS"
		}
	case "Judge Promos":
		switch cardName {
		case "Demonic Tutor":
			if variant == "" {
				variant = "2008"
			} else if variant == "J20" {
				variant = "2020"
			}
		case "Vampiric Tutor":
			if variant == "" {
				variant = "2000"
			} else if variant == "J18" {
				variant = "2018"
			}
		case "Wasteland":
			if variant == "" {
				variant = "2010"
			}
		}
	case "Prerelease Cards":
		if cardName == "Lu Bu, Master-at-Arms" {
			if variant == "Japan 4/29/99" {
				variant = "April"
			} else if variant == "Singapore 7/4/99" {
				variant = "July"
			}
		}
	case "Standard Showdown Promos":
		if variant == "Rebecca Guay" {
			edition = "PSS2"
		} else if variant == "Alayna Danner" {
			edition = "PSS3"
		} else {
			edition = "PSS1"
		}
	case "Secret Lair Drop Series":
		if cardName == "Squire" {
			return nil, errors.New("unsupported")
		} else if cardName == "Thalia, Guardian of Thraben" {
			if variant == "" {
				variant = "37"
			}
		} else if cardName == "Swamp" && variant == "Full Art" {
			variant = "119"
		}
	case "Planeswalker Event Promos":
		variant = ""
	case "Core Set 2021":
		if variant == "Alternate Art" {
			variant = "Borderless"
		}
	case "World Championship Decks":
		// Typo
		variant = strings.Replace(variant, "SD", "SB", 1)

		// These do not exist
		if (cardName == "Red Elemental Blast" && variant == "1996 George Baxter 4ED") ||
			(cardName == "Island" && variant == "2002 Raphael Levy 7ED 337") {
			return nil, errors.New("does not exist")
		}
		// Try parsing the rightmost portion of the data, by looking up
		// any recurring tag available in the VariantsTable
		sets := mtgmatcher.GetSets()
		for _, code := range []string{"FEM", "4ED", "TMP"} {
			if strings.Contains(variant, code) {
				fields := mtgmatcher.Cut(variant, code)
				// Clean up by removing the set code and the option sideboard tag
				vars := strings.TrimPrefix(fields[1], code)
				vars = strings.TrimSpace(vars)
				vars = strings.TrimPrefix(vars, "SB")
				vars = strings.TrimSpace(vars)
				vars = strings.ToLower(vars)

				if vars != "" {
					tag := mtgmatcher.VariantsTable[sets[code].Name][cardName][vars]
					if tag != "" {
						variant += " " + tag
					}
				}
				break
			}
		}
	}

	isFoil := strings.Contains(variant, "Foil") || edition == "Mystery Booster Retail Exclusives"

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      isFoil,
	}, nil
}
