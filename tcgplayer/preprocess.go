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

	edition := editions[product.GroupId]

	if cardName == "Bruna, Light of Alabaster" && variant == "Commander 2018" {
		return nil, errors.New("does not exist")
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
	case "Game Day & Store Championship Promos":
		if variant == "Winner" || variant == "Top 8" {
			return nil, errors.New("untracked")
		}
		switch cardName {
		case "Consider":
			edition = "PW22"
		}
	case "Launch Party & Release Event Promos":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		}
		switch cardName {
		case "Lotus Bloom":
			edition = "TSR"
			variant = "411"
		}
	case "Media Promos":
		switch cardName {
		case "Nalathni Dragon":
			if variant == "Redemption Program" {
				edition = variant
			}
		case "Duress":
			if variant == "IDW Comics 2014" {
				edition = "IDW Comics 2014"
			}
		case "Stocking Tiger":
			variant = "misprint"
		case "Sword of Dungeons & Dragons":
			edition = "H17"
		case "Shepherd of the Lost":
			edition = "PURL"
		default:
			if !strings.Contains(variant, "SDCC") {
				edition = "Magazine Inserts"
			} else {
				edition = "SDCC"
			}
		}
	case "Pro Tour Promos":
		switch cardName {
		case "Char":
			edition = "P15A"
		}
	case "WMCQ Promo Cards":
		switch cardName {
		case "Arcbound Ravager":
			edition = "PPRO"
		}
	case "WPN & Gateway Promos":
		switch cardName {
		case "Mind Stone":
			if variant == "2021" {
				variant = "WPN 2021"
			} else {
				variant = "Gateway"
			}
		case "Orb of Dragonkind":
			num := mtgmatcher.ExtractNumber(variant)
			variant = "J" + num
		}
	case "Unique and Miscellaneous Promos":
		for _, ext := range product.ExtendedData {
			if ext.Name == "OracleText" && strings.Contains(ext.Value, "Heroes of the Realm") {
				edition = "Heroes of the Realm"
				break
			}
		}

		switch cardName {
		case "Fiendish Duo":
			edition = "PKHM"
		case "Arasta of the Endless Web":
			edition = "THB"
			variant = "352"
		case "Archmage Emeritus":
			edition = "STX"
			variant = "377"
		case "Yusri, Fortune's Flame":
			edition = "MH2"
			variant = "492"
		case "Sigarda's Summons":
			edition = "VOW"
			variant = "404"
		case "Serra Angel":
			if variant == "" {
				edition = "PWOS"
			} else if variant == "25th Anniversary Exposition" {
				edition = "PDOM"
			}
		}
	case "Special Occasion":
		if len(mtgmatcher.MatchInSet(cardName, "PCEL")) == 1 {
			edition = "PCEL"
		} else {
			variant = "Happy Holidays"
		}
	case "Junior Series Promos":
		// TCG has a single version but there are multiple ones available
		// So just preserve whichever is filed in Scryfall
		ed, found := map[string]string{
			"Sakura-Tribe Elder": "PJSE",
			"Shard Phoenix":      "PJSE",
			"Whirling Dervish":   "PJSE",
			"Mad Auntie":         "PJJT",
		}[cardName]
		if found {
			edition = ed
		} else if variant == "Japan Junior Series" {
			edition = "PJJT"
		} else if cardName != "Royal Assassin" && len(mtgmatcher.MatchInSet(cardName, "PSUS")) == 1 {
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
		ogVariant := variant
		variant = product.getNum()
		switch cardName {
		case "Plains // Battlefield Forge":
			cardName = "Battlefield Forge"
			variant = "669"
		case "Hadoken":
			cardName = "Lightning Bolt"
			variant = "675"
		// These cards have a different number than reported
		case "Demonlord Belzenlok",
			"Griselbrand",
			"Liliana's Contract",
			"Kothophed, Soul Hoarder",
			"Razaketh, the Foulblooded":
			if strings.Contains(ogVariant, "Etched") {
				variant = "etched"
			}
		}
	case "Planeswalker Event Promos":
		variant = ""
		if len(mtgmatcher.MatchInSet(cardName, "PPRO")) != 0 || cardName == "Fae of Wishes" {
			edition = "PPRO"
		}
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
	case "Innistrad: Double Feature",
		"Kamigawa: Neon Dynasty",
		"Unfinity":
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
