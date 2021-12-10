package tcgplayer

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func Preprocess(product *TCGProduct, editions map[int]string) (*mtgmatcher.Card, error) {
	cardName := product.Name
	variant := ""

	if strings.Contains(cardName, " - ") {
		fields := strings.Split(cardName, " - ")
		cardName = fields[0]
		if len(fields) > 1 {
			variant = strings.Join(fields[1:], " ")
		}
	}
	if strings.Contains(cardName, " [") {
		cardName = strings.Replace(cardName, "[", "(", -1)
		cardName = strings.Replace(cardName, "]", ")", -1)
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

	edition := editions[product.GroupId]

	if strings.Contains(edition, "Art Series") {
		return nil, errors.New("non-single card")
	}

	// Early to skip the Oversize early return
	if variant == "Commander Launch Promo" {
		edition = "PCMD"
	}

	ogVariant := variant
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
	case "Hero's Path Promos",
		"Oversize Cards",
		"Special Occasion",
		"Revised Edition (Foreign White Border)",
		"Fourth Edition (Foreign White Border)":
		return nil, errors.New("unsupported")
	case "Portal":
		switch cardName {
		case "Warrior's Charge",
			"Blaze",
			"Raging Goblin",
			"Anaconda",
			"Elite Cat Warrior",
			"Monstrous Growth":
			if variant == "Flavor Text" {
				variant = ""
			} else if variant == "" {
				variant = "Reminder Text"
			}
		case "Hand of Death":
			// Variants are fine as is
		case "Armored Pegasus",
			"Bull Hippo",
			"Cloud Pirates",
			"Feral Shadow",
			"Snapping Drake",
			"Storm Crow":
			if variant == "Reminder Text" {
				edition = "Portal Demo Game"
			}
		}
	case "Battle for Zendikar",
		"Oath of the Gatewatch":
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
	case "Kaladesh", "Aether Revolt":
		if variant == "17/264" {
			variant = "Intro Pack"
		}
	case "Media Promos",
		"Pro Tour Promos",
		"Unique and Miscellaneous Promos",
		"Launch Party & Release Event Promos",
		"League Promos",
		"Game Day & Store Championship Promos",
		"WMCQ Promo Cards",
		"WPN & Gateway Promos":
		if strings.HasSuffix(variant, "Ultra Pro Puzzle Quest") ||
			variant == "Redemption Program" { // JPN Nalathni Dragon
			return nil, errors.New("unsupported")
		}
		for _, ext := range product.ExtendedData {
			if ext.Name == "OracleText" && strings.Contains(ext.Value, "Heroes of the Realm") {
				return nil, errors.New("unsupported")
			}
		}

		ed, found := map[string]string{
			"Balduvian Horde":             "PWOR",
			"Char":                        "P15A",
			"Deathless Angel":             "PROE",
			"Jaya Ballard, Task Mage":     "PMPS08",
			"Kamahl, Pit Fighter":         "P15A",
			"Sword of Dungeons & Dragons": "H17",
			"Arcbound Ravager":            "PPRO",
			"Goblin Chieftain":            "PRES",
			"Oran-Rief, the Vastwood":     "PRES",
			"Loam Lion":                   "PRES",
			"Shepherd of the Lost":        "PURL",
			"Sethron, Hurloon General":    "PL21",
			"Moraug, Fury of Akoum":       "PL21",
			"Ox of Agonas":                "PL21",
			"Angrath, the Flame-Chained":  "PL21",
			"Tahngarth, First Mate":       "PL21",
			"Arbor Elf":                   "PWP21",
			"Collected Company":           "PWP21",
			"Fabled Passage":              "PWP21",
			"Wurmcoil Engine":             "PWP21",
		}[cardName]
		if found {
			edition = ed
		} else if edition == "Media Promos" {
			// Keep SDCC Chandra separate from the Q06 version
			if !strings.Contains(variant, "SDCC") {
				edition = "Magazine Inserts"
			} else {
				edition = "SDCC"
			}
		} else if len(mtgmatcher.MatchInSet(cardName, "CP1")) == 1 {
			edition = "CP1"
		} else if len(mtgmatcher.MatchInSet(cardName, "CP2")) == 1 {
			edition = "CP2"
		} else if len(mtgmatcher.MatchInSet(cardName, "CP3")) == 1 {
			edition = "CP3"
		} else if len(mtgmatcher.MatchInSet(cardName, "Q06")) == 1 {
			edition = "Q06"
		} else if edition == "Launch Party & Release Event Promos" && mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		} else if edition == "WPN & Gateway Promos" && variant == "Retro Frame" {
			edition = "PLG21"
		}

		if edition == "PWP21" {
			if variant == "Winner" || variant == "Top 8" {
				return nil, errors.New("untracked")
			}
		}

		switch cardName {
		case "Arasta of the Endless Web":
			edition = "THB"
			variant = "352"
		case "Lotus Bloom":
			edition = "TSR"
			variant = "411"
		case "Archmage Emeritus":
			edition = "STX"
			variant = "377"
		case "Yusri, Fortune's Flame":
			edition = "MH2"
			variant = "492"
		case "Sigarda's Summons":
			edition = "VOW"
			variant = "404"
		case "Duress":
			if variant == "IDW Comics 2014" {
				edition = variant
			}
		case "Fling":
			edition = "PWP11"
			if variant == "DCI" {
				edition = "PWP10"
			}
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
				edition = "PLG20"
			} else {
				edition = "PM19"
			}
		case "Mind Stone":
			if variant == "2021" {
				edition = "PWP21"
			} else {
				variant = "Gateway"
			}
		case "Orb of Dragonkind":
			num := mtgmatcher.ExtractNumber(variant)
			variant = "J" + num
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
		} else if variant == "Japan Junior Series" {
			edition = "PJJT"
		} else if len(mtgmatcher.MatchInSet(cardName, "PSUS")) == 1 {
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
		switch cardName {
		case "Lu Bu, Master-at-Arms":
			if variant == "Japan 4/29/99" {
				variant = "April"
			} else if variant == "Singapore 7/4/99" {
				variant = "July"
			}
		case "Beast of Burden":
			if variant == "No Date" {
				variant = "misprint"
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
		} else {
			variant = product.getNum()
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
	case "Alliances",
		"Portal Second Age":
		if variant == "" {
			variant = product.getNum()
			// Missing everything
			if cardName == "Awesome Presence" {
				variant = "arms spread"
			}
		}
	case "Homelands":
		num := product.getNum()
		fixup, found := map[string]string{
			"Abbey Matron":    "2",
			"Aliban's Tower":  "61",
			"Ambush Party":    "63",
			"Anaba Bodyguard": "66",
			"Anaba Shaman":    "67",
			"Carapace":        "84",
			"Cemetery Gate":   "44",
			"Reef Pirates":    "",
		}[cardName]
		if found {
			num = fixup
		}
		if variant == "" {
			variant = num + "a"
		} else if variant == "Version 2" {
			variant = num + "b"
		}
	case "Fallen Empires":
		if variant == "" {
			fixup, found := map[string]string{
				"Armor Thrull":           "33a",
				"Basal Thrull":           "34a",
				"Elven Fortress":         "65a",
				"Elvish Hunter":          "67a",
				"Elvish Scout":           "68a",
				"Farrel's Zealot":        "3a",
				"Goblin Chirurgeon":      "54a",
				"High Tide":              "18a",
				"Homarid":                "19a",
				"Icatian Infantry":       "7a",
				"Icatian Moneychanger":   "10a",
				"Merseine":               "23a",
				"Orcish Veteran":         "62a",
				"Order of Leitbur":       "16c",
				"Order of the Ebon Hand": "42a",
				"Spore Cloud":            "72a",
				"Thorn Thallid":          "80a",
				"Tidal Flats":            "27a",
				"Vodalian Mage":          "30a",
			}[cardName]
			if found {
				variant = fixup
			}
		}
	case "Starter 2000":
		switch cardName {
		case "Spined Wurm",
			"Wind Drake":
			return nil, errors.New("does not exist")
		}
	case "Battle Royale Box Set":
		if mtgmatcher.IsBasicLand(cardName) {
			variant = product.getNum()
		}
	case "Adventures in the Forgotten Realms":
		if variant == "Dungeon Module" {
			variant = "Showcase"
		}
	case "Mystery Booster Cards":
		edition = "MB1"
	case "Innistrad: Double Feature":
		variant = product.getNum()
	}

	// Outside the main loop to catch everything
	if mtgmatcher.IsBasicLand(cardName) && variant == "" {
		variant = product.getNum()
	}

	// Handle any particular finish
	isFoil := strings.Contains(variant, "Foil") || edition == "Mystery Booster Retail Exclusives"
	if strings.Contains(ogVariant, "Etched") {
		if variant != "" {
			variant += " "
		}
		variant += "Etched"
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      isFoil,
	}, nil
}

func (tcgp *TCGProduct) getNum() string {
	for _, extData := range tcgp.ExtendedData {
		if extData.Name == "Number" {
			return extData.Value
		}
	}
	return ""
}
