package tcgplayer

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var tokenIds = map[int]string{
	78630:  "P03",
	78526:  "L13",
	78417:  "L12",
	78623:  "P03",
	78631:  "PR2",
	78444:  "L13",
	82612:  "L13",
	108437: "L14",
	108436: "L13",
	78618:  "MPR",
	108434: "L14",
	78636:  "PR2",
	78613:  "P04",
}

var cardIds = map[int]string{
	// Serra Angel
	284951: "P30A",
	284923: "P30H",
	284921: "P30H",
	// Shivan Dragon
	515925: "P30T",
	284937: "P30H",
	284939: "P30H",
}

func Preprocess(product *TCGProduct, editions map[int]string) (*mtgmatcher.Card, error) {
	cardName, variant := product.GetNameAndVariant()

	edition := editions[product.GroupId]

	// Unsupported cards depending on their variant
	switch cardName {
	case "Bruna, Light of Alabaster":
		if variant == "Commander 2018" {
			return nil, errors.New("does not exist")
		}
	case "Glissa, the Traitor":
		if variant == "Mirrodin Besieged" {
			return nil, errors.New("untracked")
		}
	case "Elvish Vanguard":
		if strings.Contains(variant, "Spanish") ||
			strings.Contains(variant, "French") ||
			strings.Contains(variant, "Italian") {
			return nil, errors.New("non english")
		}
	case "Tamiyo's Journal":
		if edition == "Shadows over Innistrad" {
			_, found := mtgmatcher.VariantsTable[edition][variant]
			if !found {
				return nil, errors.New("non english")
			}
		}
	case "Tarmogoyf":
		if variant == "JP Exclusive" {
			variant = ""
		}
	default:
		if strings.Contains(variant, "JP Amazon Exclusive") ||
			strings.Contains(variant, "SEA Exclusive") ||
			strings.Contains(variant, "JP WonderGOO Exclusive") ||
			strings.Contains(variant, "JP Hareruya Exclusive") {
			return nil, errors.New("unofficial")
		} else if product.IsToken() && strings.Contains(product.CleanName, "Double") {
			return nil, errors.New("duplicate")
		} else if strings.Contains(edition, "Tales of Middle-earth") && strings.HasSuffix(cardName, "Scene") {
			return nil, errors.New("unsupported")
		}
	}

	ogVariant := variant
	switch edition {
	case "Portal":
		switch cardName {
		case "Blaze":
			switch variant {
			case "CS Alternate Art":
				variant = "118†s"
			case "Starter Deck CS Alternate Art":
				variant = "118s"
			case "Flavor Text":
				variant = "118"
			default:
				variant = "118†"
			}
		case "Warrior's Charge",
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
	case "Alliances",
		"Portal Second Age":
		if variant == "" {
			variant = product.GetNumber()
			// Missing everything
			if cardName == "Awesome Presence" {
				variant = "arms spread"
			}
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
	case "Homelands":
		num := product.GetNumber()
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
	case "Media Promos":
		switch cardName {
		case "Nalathni Dragon":
			if variant == "Redemption Program" {
				edition = variant
			}
		case "Duress":
			if variant == "IDW Comics 2014" {
				edition = "PIDW"
			}
		case "Stocking Tiger":
			variant = "misprint"
		case "Sword of Dungeons & Dragons":
			edition = "H17"
		case "Shepherd of the Lost":
			edition = "PURL"
		case "Counterspell":
			edition = "PMEI"
		case "Llanowar Elves":
			edition = "PRES"
		default:
			if !strings.Contains(variant, "SDCC") {
				edition = "PMEI"
			} else {
				edition = "SDCC"
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
	case "Junior Series Promos":
		if variant == "Japan Junior Tournament" {
			edition = "PJJT"
		} else {
			switch cardName {
			case "Royal Assassin",
				"Sakura-Tribe Elder":
			default:
				if len(mtgmatcher.MatchInSet(cardName, "PSUS")) == 1 {
					edition = "PSUS"
				}
			}
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
		case "Moraug, Fury of Akoum", "Ox of Agonas", "Angrath, the Flame-Chained", "Tahngarth, First Mate":
			if strings.Contains(variant, "JP Exclusive") {
				return nil, errors.New("non-english")
			}
		case "Yuriko, the Tiger's Shadow":
			if variant == "" {
				edition = "PL22"
			}
		case "Relentless Rats":
			if strings.Contains(variant, "Italian") {
				return nil, errors.New("non-english")
			}
		case "Lotus Petal":
			edition = "P30M"
		case "Gala Greeters":
			if variant == "English" {
				edition = "SNC"
				variant = "450"
			}
		default:
			if variant == "JP Exclusive Summer Vacation" && len(mtgmatcher.MatchInSet(cardName, "PL21")) == 0 {
				edition = "PSVC"
			} else if product.IsToken() && strings.Contains(variant, "JP") && strings.Contains(variant, "Exclusive") {
				edition = "WDMU"
			}
		}
	case "Pro Tour Promos":
		switch cardName {
		case "Char":
			edition = "P15A"
		case "Gideon, Ally of Zendikar", "Selfless Spirit", "Thraben Inspector":
			edition = "PRCQ"
		case "Thing in the Ice":
			edition = "PR23"
		case "Snapcaster Mage":
			if variant == "Regional Championship Qualifiers 2023" {
				edition = variant
			}
		case "Sauron, the Dark Lord":
			edition = "LTR"
			variant = "301"
		default:
			for _, code := range []string{"PR23", "SLP"} {
				if len(mtgmatcher.MatchInSet(cardName, code)) > 0 {
					edition = code
					break
				}
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
	case "Standard Showdown Promos":
		edition = "PSS1"
		if variant == "Rebecca Guay" {
			edition = "PSS2"
		} else if variant == "Alayna Danner" {
			edition = "PSS3"
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
	case "Launch Party & Release Event Promos":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		}
		switch cardName {
		case "Lotus Bloom":
			edition = "TSR"
			variant = "411"
		case "Shivan Dragon":
			if variant == "Moscow 2005" {
				edition = "P9ED"
			}
		case "Water Gun Balloon Game":
			edition = "UNF"
			variant = "538"
		case "Fabricate":
			edition = "40K"
			variant = "181"
		case "Disrupt Decorum":
			edition = "CMM"
			variant = "1067"
		}
	case "Renaissance":
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
	case "WMCQ Promo Cards":
		switch cardName {
		case "Arcbound Ravager":
			edition = "PPRO"
		}
	case "Kaladesh":
		if variant == "17/264" {
			variant = "Intro Pack"
		}
	case "Secret Lair Drop Series":
		variant = product.GetNumber()
		switch cardName {
		case "Plains // Battlefield Forge":
			cardName = "Battlefield Forge"
			variant = "669"
		case "Hadoken":
			cardName = "Lightning Bolt"
			variant = "675"
		case "Zndrsplt, Eye of Wisdom", "Okaun, Eye of Chaos":
			variant = ogVariant
		case "Counterspell":
			if ogVariant == "SL PLAYTEST" {
				variant = "SCTLR"
			}
		default:
			// Preserve all etched/galaxy/rainbow foil properties
			if strings.Contains(ogVariant, "Foil") {
				variant += " " + ogVariant
			}
		}
	case "AFR Ampersand Promos":
		variant = product.GetNumber() + "a"
	case "WPN & Gateway Promos":
		switch cardName {
		case "Mind Stone":
			edition = "PDCI"
			if variant == "2021" {
				edition = "PW21"
			}
		case "Orb of Dragonkind":
			variant = "J" + product.GetNumber()
		}
	case "Game Day & Store Championship Promos":
		if variant == "Winner" || variant == "Top 8" {
			return nil, errors.New("untracked")
		}
		switch cardName {
		case "Saruman of Many Colors":
			edition = "LTR"
			variant = "300"
		}
	case "Special Occasion":
		if len(mtgmatcher.MatchInSet(cardName, "PCEL")) == 1 {
			edition = "PCEL"
		} else if len(mtgmatcher.MatchInSet(cardName, "HHO")) == 1 {
			edition = "HHO"
		} else {
			return nil, errors.New("untracked")
		}
	case "Unfinity":
		// Skip attractions, number is incorrect
		if !strings.Contains(variant, "-") {
			variant = product.GetNumber()
		}
	case "Murders at Karlov Manor":
		num := product.GetNumber()

		if num != "" && variant != "a" && variant != "b" {
			variant = num
		}
	case "Fourth Edition",
		"Revised Edition",
		"Mirage",
		"Tempest",
		"Planechase 2012",
		"Anthologies",
		"Deckmasters Garfield vs Finkel",
		"Duel Decks: Anthology",
		"International Edition",
		"Collector's Edition",
		"Duel Decks: Jace vs. Chandra",
		"Revised Edition (Foreign Black Border)",
		"Fourth Edition (Foreign Black Border)",
		"Unstable",
		"Shadows over Innistrad",
		"War of the Spark",
		"Modern Horizons",
		"30th Anniversary Promos",
		"Universes Beyond: Warhammer 40,000",
		"The Brothers' War: Retro Frame Artifacts",
		"Mystery Booster Cards",
		"MagicFest Cards",
		"Planeshift",
		"": // cosmetic
		// Variants are fine as is
	default:
		num := product.GetNumber()

		if num != "" && mtgmatcher.ExtractYear(variant) == "" {
			variant = num
		}
	}

	// Override any complex cases
	ed, found := cardIds[product.ProductId]
	if found {
		edition = ed
	}

	if product.IsToken() && edition != "Unfinity" {
		// Strip pw/tou numbers that could be misinterpreted as numbers
		if strings.Contains(variant, "/") {
			variant = ""
		}
		// If number is available, use it as it's usually accurate
		num := product.GetNumber()
		if num != "" {
			variant = num
		}
		// Decouple
		ed, found := tokenIds[product.ProductId]
		if found {
			edition = ed
		}

		if edition == "L13" {
			if product.ProductId == 82612 {
				variant = "4"
			} else if product.ProductId == 78444 {
				variant = "1"
			}
		}
	}

	// Special tags
	if strings.Contains(strings.ToLower(ogVariant), "misprint") {
		variant = "misprint"
	}
	if strings.Contains(strings.ToLower(ogVariant), "display commander") {
		variant = ogVariant
	}
	if strings.Contains(strings.ToLower(ogVariant), "serial") {
		if variant != "" {
			variant += " "
		}
		variant += "serial"
	}

	// Handle any particular finish
	isFoil := strings.Contains(ogVariant, "Foil")
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
