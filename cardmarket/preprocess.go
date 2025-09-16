package cardmarket

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type PreprocessError struct {
	Extra string
	Err   error
}

func (pe *PreprocessError) Error() string {
	return pe.Extra + " - " + pe.Err.Error()
}

var promo2editionTable = map[string]string{
	// Promos
	"Evolving Wilds":      "Tarkir Dragonfury",
	"Ruthless Cullblade":  "Worldwake Promos",
	"Pestilence Demon":    "Rise of the Eldrazi Promos",
	"Dreg Mangler":        "Return to Ravnica Promos",
	"Karametra's Acolyte": "Theros Promos",

	"Dragon Fodder":        "Tarkir Dragonfury",
	"Dragonlord's Servant": "Tarkir Dragonfury",
	"Foe-Razer Regent":     "Tarkir Dragonfury",

	"Reliquary Tower":   "Love Your LGS",
	"Hangarback Walker": "Love Your LGS",

	"Crucible of Worlds": "World Championship Promos",

	"Forest":   "2017 Gift Pack",
	"Island":   "2017 Gift Pack",
	"Mountain": "2017 Gift Pack",
	"Plains":   "2017 Gift Pack",
	"Swamp":    "2017 Gift Pack",
}

var convention2editionTable = map[string]string{
	"Serra Angel":         "PDOM",
	"Steward of Valeron":  "PURL",
	"Kor Skyfisher":       "PURL",
	"Bloodthrone Vampire": "PURL",
	"Merfolk Mesmerist":   "PURL",
	"Chandra's Fury":      "PURL",
	"Stealer of Secrets":  "PURL",
	"Aeronaut Tinkerer":   "PURL",
	"Nightpack Ambusher":  "PM20",
	"Deeproot Champion":   "PXLN",
	"Death Baron":         "PM19",
}

var dengeki2editionTable = map[string]string{
	"Shepherd of the Lost": "PURL",
	"Chandra's Spitfire":   "PMEI",
	"Cunning Sparkmage":    "PMEI",
	"Chandra's Outrage":    "PMEI",
}

var gameday2editionTable = map[string]string{
	"Consider":                 "PW22",
	"Recruitment Officer":      "GDY",
	"Touch the Spirit Realm":   "GDY",
	"Power Word Kill":          "GDY",
	"Fateful Absence":          "PW22",
	"Atsushi, the Blazing Sky": "PW22",
	"Skyclave Apparition":      "GDY",
	"All-Seeing Arbiter":       "GDY",
	"Shivan Devastator":        "GDY",
	"Workshop Warchief":        "GDY",
	"Braids, Arisen Nightmare": "GDY",
	"Surge Engine":             "GDY",
	"Supplant Form":            "PFRF",
}

func checkLoadedId(cardName string, productId int) []string {
	cardName = mtgmatcher.SplitVariants(cardName)[0]
	cardName = strings.TrimSuffix(cardName, " Token")
	testProductId := fmt.Sprint(productId)

	possibleIds, err := mtgmatcher.SearchContains(cardName)
	if err != nil {
		return nil
	}

	var ids []string
	for _, possibleId := range possibleIds {
		co, err := mtgmatcher.GetUUID(possibleId)
		if err != nil {
			continue
		}

		if co.Identifiers["mcmId"] == testProductId {
			ids = append(ids, co.UUID)
		}
	}

	return ids
}

func Preprocess(cardName, number, edition string) (*mtgmatcher.InputCard, error) {
	var foil bool

	for _, tag := range filteredExpansionsTags {
		if strings.Contains(edition, tag) {
			return nil, mtgmatcher.ErrUnsupported
		}
	}

	switch number {
	case "1 of 1":
		return nil, mtgmatcher.ErrUnsupported
	default:
		// Strip extra letters that may interfere later on
		// (ie IDW Promos P4 becomeing Starter 2000)
		number = strings.TrimLeft(number, "P")
	}
	number = strings.TrimLeft(number, "0")
	number = strings.TrimSpace(number)

	switch cardName {
	case "Magic Guru":
		return nil, mtgmatcher.ErrUnsupported
	}

	var variant string
	vars := mtgmatcher.SplitVariants(cardName)
	if len(vars) > 1 {
		cardName = vars[0]
		variant = vars[1]
	}
	ogVariant := variant
	ogEdition := edition

	switch edition {
	case "Anthologies",
		"Deckmasters",
		"Unstable":
		// Variants good as is

	case "The Dark Italian",
		"Foreign Black Bordered",
		"Coldsnap Theme Decks",
		"Summer Magic",
		"Introductory Two-Player Set",
		"Duels of the Planeswalkers Decks",
		"Archenemy":
		// Drop variant and number
		variant = ""

		if mtgmatcher.IsBasicLand(cardName) {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Misprints":
		switch cardName {
		case "Laquatus's Champion":
			if variant == "V.1" {
				variant = "prerelease misprint dark"
			} else if variant == "V.2" {
				variant = "prerelease misprint"
			}
		case "Corpse Knight",
			"Stocking Tiger":
			variant = "misprint"
		case "Beast of Burden":
			variant = "prerelease misprint"
		case "Demigod of Revenge":
			return nil, mtgmatcher.ErrUnsupported
		case "Island":
			variant = "Arena 1999 misprint"
		default:
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Simplified Chinese Alternate Art Cards":
		switch cardName {
		case "Drudge Skeletons":
			switch variant {
			case "V.1":
				edition = "6ED"
			case "V.2", "V.3":
				edition = "7ED"
			case "V.4", "V.5":
				edition = "8ED"
			case "V.6", "V.7":
				edition = "9ED"
			}
			switch variant {
			case "V.3", "V.5", "V.7":
				foil = true
			}
			variant = "Simplified Chinese Alternate Art Cards"
		case "Raise Dead":
			switch variant {
			case "V.1", "V.2":
				edition = "7ED"
				foil = (variant == "V.2")
			case "":
				edition = "POR"
			}
			variant = "Simplified Chinese Alternate Art Cards"
		case "Blaze":
			edition = "POR"
			switch variant {
			case "V.1":
				variant = "118†s"
			case "V.2":
				variant = "118s"
			}
		case "Charcoal Diamond":
			edition = "7ED"
			foil = (variant == "V.2")
			variant = "Simplified Chinese Alternate Art Cards"
		}

	case "Arabian Nights":
		if variant == "V.1" {
			variant = "dark"
		} else if variant == "V.2" {
			variant = "light"
		}

	case "Fallen Empires",
		"Homelands":
		for _, num := range mtgmatcher.VariantsTable[edition][cardName] {
			if (variant == "V.1" && strings.HasSuffix(num, "a")) ||
				(variant == "V.2" && strings.HasSuffix(num, "b")) ||
				(variant == "V.3" && strings.HasSuffix(num, "c")) ||
				(variant == "V.4" && strings.HasSuffix(num, "d")) {
				variant = num
				break
			}
		}

	case "Portal":
		switch cardName {
		case "Armored Pegasus",
			"Bull Hippo",
			"Cloud Pirates",
			"Feral Shadow",
			"Snapping Drake",
			"Storm Crow":
			if variant == "V.2" {
				variant = "Reminder Text"
			}
		default:
			if variant == "V.1" && !mtgmatcher.IsBasicLand(cardName) {
				variant = "Reminder Text"
			}
		}

	case "Visions":
		if variant == "V.2" {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Fourth Edition: Black Bordered":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Planeshift":
		if variant == "V.2" {
			variant = "Alt Art"
		}

	case "Judge Rewards Promos":
		switch cardName {
		case "Demonic Tutor",
			"Vampiric Tutor",
			"Vindicate",
			"Wasteland":
			vars = strings.Split(number, "-")
			variant = vars[0]
		}
		if mtgmatcher.IsBasicLand(cardName) {
			if variant == "V.1" {
				edition = "J14"
				variant = ""
			} else if variant == "V.2" {
				edition = "P23"
				variant = ""
			}
		}

	case "MagicFest Promos":
		vars = strings.Split(number, "-")
		edition += " " + vars[0]
		if len(vars) > 1 {
			variant = vars[1]
		}

	case "Prerelease Promos":
		switch cardName {
		case "Dirtcowl Wurm":
			if variant == "V.2" {
				return nil, mtgmatcher.ErrUnsupported
			}
		case "Lu Bu, Master-at-Arms":
			if variant == "V.1" {
				variant = "April"
			} else if variant == "V.2" {
				variant = "July"
			}
		case "Chord of Calling",
			"Wrath of God":
			edition = "2XM"
			variant = "Launch"
		case "Weathered Wayfarer",
			"Bring to Light":
			edition = "2X2"
			variant = "Launch"
		default:
			if len(mtgmatcher.MatchInSet(cardName, "PHEL")) == 1 {
				edition = "PHEL"
			}
		}
		// This set has incorrect numbering
		if variant == number {
			variant = ""
		}

	case "Harper Prism Promos":
		if variant == "V.2" {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Gateway Promos":
		switch cardName {
		case "Fling",
			"Sylvan Ranger":
			if variant == "V.1" {
				variant = "DCI"
			} else if variant == "V.2" {
				variant = "WPN"
			}
		}

	case "Friday Night Magic Promos":
		switch cardName {
		case "Goblin Warchief":
			if variant == "V.1" {
				edition = "F06"
			} else if variant == "V.2" {
				edition = "F16"
			}
		}

	case "Weekend Promos":
		switch cardName {
		case "Time Wipe",
			"Karn's Bastion":
			edition = "War of the Spark Promos"
			variant = "" // Drop WAR
		default:
			edition = "Ravnica Weekend"
			variant = number
		}

	case "Player Rewards Promos":
		switch cardName {
		case "Lightning Bolt":
			if variant == "V.1" {
				variant = "oversized"
			}
		case "Comet Storm",
			"Emrakul, the Aeons Torn",
			"Feral Hydra",
			"Glissa, the Traitor",
			"Hero of Bladehold",
			"Rampaging Baloths",
			"Spellbreaker Behemoth",
			"Sun Titan",
			"Wurmcoil Engine":
			variant = "oversized"
		}

	case "Arena League Promos":
		switch variant {
		case "V.1":
			edition = "PARL"
		case "V.2":
			edition = "PAL99"
		case "V.3":
			edition = "PAL00"
		case "V.4":
			edition = "PAL01"
			if cardName == "Forest" {
				variant = "1"
			}
		case "V.5":
			if cardName == "Island" {
				edition = "PAL02"
			} else if cardName == "Forest" {
				edition = "PAL01"
				variant = "11"
			} else {
				edition = "PAL03"
			}
		case "V.6":
			edition = "PAL04"
			if cardName == "Forest" || cardName == "Island" {
				edition = "PAL03"
			}
		case "V.7":
			edition = "PAL05"
			if cardName == "Forest" || cardName == "Island" {
				edition = "PAL04"
			}
		case "V.8":
			edition = "PAL06"
			if cardName == "Forest" || cardName == "Island" {
				edition = "PAL05"
			}
		case "V.9":
			// Only for "Forest" or "Island"
			edition = "PAL06"
		default:
			if cardName == "Mad Auntie" {
				edition = "Japan Junior Tournament"
			}
		}

	case "APAC Lands":
		var landMap = map[string]string{
			"R1": "4",
			"R2": "2",
			"R3": "5",
			"R4": "3",
			"R5": "1",
			"B1": "9",
			"B2": "7",
			"B3": "10",
			"B4": "8",
			"B5": "6",
			"C1": "14",
			"C2": "12",
			"C3": "15",
			"C4": "13",
			"C5": "11",
		}
		variant = landMap[number]

	case "Euro Lands":
		var landMap = map[string]string{
			"1":  "1",
			"2":  "6",
			"3":  "11",
			"4":  "2",
			"5":  "7",
			"6":  "12",
			"7":  "3",
			"8":  "8",
			"9":  "13",
			"10": "4",
			"11": "9",
			"12": "14",
			"13": "5",
			"14": "10",
			"15": "15",
		}
		variant = landMap[number]

	case "Magic Premiere Shop Promos":
		switch variant {
		default:
			edition = "PMPS"
			// Further untangling in variants.go
		case "V.5":
			edition = "PMPS06"
		case "V.6":
			edition = "PMPS07"
		case "V.7":
			edition = "PMPS08"
		case "V.8":
			edition = "PMPS09"
		case "V.9":
			edition = "PMPS10"
		case "V.10":
			edition = "PMPS11"
		}

	case "Standard Showdown Promos":
		switch variant {
		case "V.1":
			edition = "XLN Standard Showdown"
		case "V.2":
			edition = "M19 Standard Showdown"
		case "V.3":
			edition = "MKM Standard Showdown"
		default:
			if len(mtgmatcher.MatchInSet(cardName, "PCBB")) == 1 {
				edition = "PCBB"
			}
		}

	case "Release Promos":
		switch cardName {
		case "Plots That Span Centuries", "Tazeem":
			edition = "DCI"
		case "Stairs to Infinity":
			edition = "PHOP"
		case "Reya Dawnbringer":
			edition = "P10E"
		case "Shivan Dragon":
			return nil, mtgmatcher.ErrUnsupported
		default:
			if len(mtgmatcher.MatchInSet(cardName, "PCMD")) == 1 {
				edition = "PCMD"
			}
		}

	case "Resale Promos":
		variant = ""

	case "MagicCon Products":
		switch cardName {
		case "Relentless Rats":
			switch number {
			case "11":
				edition = "SLP"
				variant = number
			}
		case "Arcane Signet":
			if number == "1" {
				edition = "P30M"
				variant = "1F"
			}
		default:
			for _, code := range []string{
				"SLD", "SLP",
			} {
				if len(mtgmatcher.MatchInSetNumber(cardName, code, number)) == 1 {
					edition = code
					variant = number
				}
			}
		}

	case "Magic the Gathering Products":
		switch cardName {
		case "Culling the Weak",
			"Disenchant",
			"Snuff Out":
			edition = "PMEI"
		case "Darksteel Colossus":
			edition = "PW24"
		case "Approach of the Second Sun":
			edition = "Q06"
		case "Lay Down Arms", "Sleight of Hand", "Cut Down":
			edition = "PLG24"
		case "Counterspell":
			if number == "4 JAN 2023" {
				edition = "SLD"
				variant = number
			}
		default:
			variant = number
		}

	// Catch-all sets for anything promo
	case "Promos",
		"DCI Promos":
		switch cardName {
		case "Mayor of Avabruck / Howlpack Alpha",
			"Ancestral Recall",
			"Scrubland",
			"Time Walk",
			"Gaea's Cradle":
			variant = "oversized"
		case "Nalathni Dragon":
			return nil, mtgmatcher.ErrUnsupported
		case "Relentless Rats":
			switch number {
			case "":
				edition = "PM11"
			case "10":
				edition = "SLP"
				variant = number
			}
		case "Mystical Dispute":
			if variant == "V.1" {
				edition = "PWCS"
			} else if variant == "V.2" {
				edition = "PR23"
			}
		case "Snapcaster Mage":
			if variant == "V.1" {
				edition = "PPRO"
				variant = "2016"
			} else if variant == "V.2" {
				edition = "PR23"
				variant = "2"
			}
		case "Liliana of the Veil":
			if variant == "V.1" {
				edition = "PPRO"
			} else if variant == "V.2" {
				edition = "PWCS"
			}
		case "Orb of Dragonkind":
			edition = "PLG21"
			if variant == "V.1" {
				variant = "J1"
			} else if variant == "V.2" {
				variant = "J2"
			} else if variant == "V.3" {
				variant = "J3"
			}
		case "Arcane Signet":
			if strings.Contains(number, "BD/1") {
				edition = "P30M"
				variant = "1P"
			} else if number == "1" {
				edition = "P30M"
				variant = "1F★"
			} else if number == "3" {
				edition = "PSVC"
				variant = ""
			}
		case "Chaos Warp":
			if number == "2024/1" {
				edition = "PW24"
				variant = "7"
			}
		case "Commander's Sphere":
			if number == "2024/2" {
				edition = "PW24"
				variant = "8"
			}
		case "Sol Ring":
			if edition == "Promos" {
				if number == "001" {
					edition = "PF19"
				} else if number == "LGS" {
					edition = "PLG22"
				}
			}
		case "Shivan Dragon":
			if edition == "Promos" {
				if variant == "V.1" {
					edition = "PMEI"
				} else if variant == "V.2" {
					edition = "P30T"
				}
			}
		case "Destroy Evil":
			if edition == "Promos" {
				edition = "P30T"
			}
		case "Mental Misstep":
			variant = ""
			if edition == "Promos" {
				edition = "PMEI"
			} else if edition == "DCI Promos" {
				edition = "PEWK"
			}
		case "Sauron, the Dark Lord":
			edition = "LTR"
			variant = "301"

		default:
			for _, code := range []string{
				"PMEI", "PPRO", "PEWK", "PNAT", "WMC", "PWCS",
				"PF23", "PLG21", "PLG22", "PLG23", "PLG24",
				"SLC", "SLP",
				"PRCQ", "PRC23", "PRC24",
				"PW21", "PW22", "PW23", "PW24",
				"PL21", "PL22", "PL23", "PL24",
			} {
				// number is often wrong, so
				num, _ := strconv.Atoi(number)
				results := mtgmatcher.MatchInSetNumber(cardName, code, number)
				switch code {
				case "PMEI", "PEWK", "PW21", "PW22", "PW23", "PW24":
					results = mtgmatcher.MatchInSet(cardName, code)
				default:
					if num < 10 {
						results = mtgmatcher.MatchInSet(cardName, code)
					}
				}
				if len(results) == 1 {
					edition = code
					variant = ""
					if num >= 10 {
						variant = number
					}
					break
				}
			}

		}

		ed, found := promo2editionTable[cardName]
		if found {
			edition = ed
		}

	case "Dengeki Maoh Promos":
		ed, found := dengeki2editionTable[cardName]
		if found {
			edition = ed
			variant = ""
		}

	case "Convention Promos":
		ed, found := convention2editionTable[cardName]
		if found {
			edition = ed
			variant = ""
		}

	case "Game Day Promos":
		ed, found := gameday2editionTable[cardName]
		if found {
			edition = ed
			variant = ""
		}

	case "Duel Decks: Jace vs. Chandra":
		variant = number
		switch cardName {
		case "Chandra Nalaar", "Jace Beleren":
			if ogVariant == "V.2" {
				variant = "Japanese"
			}
		}

	case "Duel Decks: Anthology":
		if len(number) > 2 {
			variant = number[2:]
			if cardName == "Tranquil Thicket" {
				variant = "26"
			} else if mtgmatcher.IsBasicLand(cardName) {
				switch number[:2] {
				case "10", "20":
					edition = "EVG"
				case "30", "40":
					edition = "JVC"
				case "50", "60":
					edition = "DVD"
				case "70", "80":
					edition = "GVL"
				}
			}
		}

	case "Champions of Kamigawa":
		variant = number
		if cardName == "Brothers Yamazaki" {
			if ogVariant == "V.1" {
				variant = "160a"
			} else if ogVariant == "V.2" {
				variant = "160b"
			}
		}

	case "Zendikar":
		switch variant {
		case "V.1", "V.3", "V.5", "V.7":
			variant = number + "a"
		default:
			variant = number
		}

	case "Theros",
		"Born of the Gods",
		"Journey into Nyx":
		variant = number
		if strings.HasPrefix(variant, "C") {
			switch edition {
			case "Theros":
				edition = "Face the Hydra"
			case "Born of the Gods":
				edition = "Battle the Horde"
			case "Journey into Nyx":
				edition = "Defeat a God"
			}
		}

	case "Battle for Zendikar",
		"Oath of the Gatewatch":
		switch variant {
		case "V.2", "V.4", "V.6", "V.8", "V.10":
			variant = number + "a"
		default:
			variant = number
		}

	case "Shadows over Innistrad":
		if cardName == "Tamiyo's Journal" {
			switch variant {
			case "V.1", "V.2", "V.3", "V.4", "V.5", "V.6":
				// Sorted out later in variants.go
			default:
				// Other languages have other numbers, but Scryfall only tracks
				// the English ones
				return nil, mtgmatcher.ErrUnsupported
			}
		}

	case "Guilds of Ravanica: Extras",
		"Ravnica Allegiance: Extras",
		"War of the Spark: Extras":
		// All cards from these set are Prerelease
		variant = "Prerelease"
		// Except for Planeswalker Decks cards
		set, err := mtgmatcher.GetSetByName(strings.TrimSuffix(edition, ": Extras"))
		if err == nil {
			num, _ := strconv.Atoi(number)
			if num > set.BaseSetSize && len(mtgmatcher.MatchInSet(cardName, set.Code)) > 0 {
				edition = set.Code
				variant = number
			}
		}

	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		if variant == "V.1" {
			variant = "Japanese"
			edition = "War of the Spark"
		} else if variant == "V.2" {
			variant = "Prerelease Japanese"
			edition = "War of the Spark Promos"
		}

	case "Core 2019: Promos":
		variant = "Prerelease"
		edition = "PM19"
		if strings.HasPrefix(number, "GP") {
			variant = ""
			edition = "M19 Gift Pack"
		}

	case "Core 2020: Extras":
		switch cardName {
		case "Chandra's Regulator":
			if variant == "V.1" {
				variant = "131"
				edition = "PM20"
			} else if variant == "V.2" {
				variant = "Promo Pack"
			} else if variant == "V.3" {
				variant = "Prerelease"
			}
		case "Nicol Bolas, Dragon-God":
			variant = "Promo Pack"
		default:
			if variant == "V.1" || mtgmatcher.IsBasicLand(cardName) {
				variant = "Promo Pack"
			} else if variant == "V.2" {
				variant = "Prerelease"
			} else if mtgmatcher.HasPromoPackPrinting(cardName) { // Needs to be after V.2 check
				variant = "Promo Pack"
			} else {
				variant = ""
			}
		}

	case "Mystery Booster":
		edition = "PLST"
		variant = number
		switch cardName {
		case "Laboratory Maniac":
			variant = "UMA-61"
		case "Plains":
			variant = "UGL-84"
		}

	case "Mystery Booster 2: Playtest Cards":
		edition = "MB2"

	case "Mystery Booster 2: Reprints from Across Magic's History":
		edition = "PLST"

	case "The List":
		variant = number
		switch cardName {
		case "Laboratory Maniac":
			variant = "ISD-61"
		case "Bottle Gnomes":
			variant = "TMP-278"
		case "Man-o'-War":
			variant = "VIS-37"
		case "Ineffable Blessing",
			"Everythingamajig":
			variant = ogVariant
		case "Imperious Perfect":
			variant = "PCMP-9"
		case "Burst Lightning":
			variant = "P10-8"
		case "Plains":
			variant = "AKH-256"
		}

	case "Secret Lair Drop Series":
		variant = number
		switch cardName {
		case "Plague Sliver",
			"Toxin Sliver",
			"Virulent Sliver":
			if ogVariant == "V.2" {
				variant += " step-and-compleat"
			}
		case "Shadowborn Apostle":
			if ogVariant == "V.8" {
				variant += " step-and-compleat"
			}
		case "Fellwar Stone":
			if ogVariant == "V.2" {
				variant += " etched"
			}
		}

	case "Secret Lair Drop Series: October Superdrop 2021",
		"Secret Lair Drop Series: October Superdrop 2023",
		"Secret Lair Drop Series: June Superdrop 2022",
		"Secret Lair Drop Series: August Superdrop 2022":
		variant = number
		edition = "SLD"
		if ogVariant == "V.2" {
			variant += " etched"
		}

	// Some cards from PLST overflow here
	case "Secret Lair Drop Series: Secretversary 2021":
		variant = number
		edition = "SLD"
		if ogVariant == "V.2" {
			variant += " etched"
		}

		switch cardName {
		case "Okaun, Eye of Chaos",
			"Zndrsplt, Eye of Wisdom":
			if ogVariant == "V.2" {
				variant = number + " display"
			}
		default:
			for _, card := range mtgmatcher.MatchInSet(cardName, "PLST") {
				if strings.HasSuffix(card.Number, "-"+number) {
					edition = "PLST"
					variant = card.Number
				}
			}
		}

	// This set is missing PLST collector numbers for no reason
	case "Secret Lair Commander Deck: Angels: They're Just Like Us but Cooler":
		edition = "SLD"
		variant = number
		if variant == "" {
			edition = "PLST"
		}

	case "Modern Horizons 2":
		switch variant {
		case "V.1":
			variant = number
		case "V.2":
			variant = number + " Etched"
		default:
			variant = number
		}

	case "Modern Horizons 2: Extras":
		// Note: order of these printing checks matters
		if mtgmatcher.IsBasicLand(cardName) {
			switch variant {
			case "V.1", "V.3":
				variant = number
			case "V.2", "V.4":
				variant = number + " Etched"
			}
		} else if mtgmatcher.HasExtendedArtPrinting(cardName, "MH2") {
			switch variant {
			case "V.1":
				variant = "Retro Frame"
			case "V.2":
				variant = "Retro Frame Foil Etched"
			case "V.3":
				variant = "Extended Art"
			}
		} else if mtgmatcher.HasBorderlessPrinting(cardName, "MH2") {
			switch variant {
			case "V.1":
				variant = "Borderless"
			case "V.2":
				variant = "Retro Frame"
				if mtgmatcher.HasShowcasePrinting(cardName, "MH2") {
					variant = "Showcase"
				}
			case "V.3":
				variant = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasShowcasePrinting(cardName, "MH2") {
			switch variant {
			case "V.1":
				variant = "Showcase"
			case "V.2":
				variant = "Retro Frame"
			case "V.3":
				variant = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasRetroFramePrinting(cardName, "MH2") {
			switch variant {
			case "V.1":
				variant = "Retro Frame"
			case "V.2":
				variant = "Retro Frame Foil Etched"
			}
		} else {
			switch variant {
			case "V.1":
				variant = ""
			case "V.2":
				variant = "Foil Etched"
			}
		}

	case "Modern Horizons: Retro Frame Cards":
		switch variant {
		case "V.1":
			variant = "Retro Frame"
		case "V.2":
			variant = "Foil Etched"
		}

	case "Commander: Modern Horizons 3: Extras":
		edition = "M3C"
		switch variant {
		case "V.2":
			for _, card := range mtgmatcher.MatchInSetNumber(cardName, "M3C", number) {
				if card.HasPromoType(mtgmatcher.PromoTypeRippleFoil) {
					variant += " ripplefoil"
				}
			}
		}

	case "Mystical Archive":
		switch variant {
		case "V.1":
			variant = ""
		case "V.2":
			variant = "JPN"
		case "V.3":
			variant = "Foil-Etched"
		case "V.4":
			variant = "JPN Foil-Etched"
		}

	// Skip Attraction lights, too many
	case "Unfinity":
		if len(mtgmatcher.MatchInSet(cardName, "UNF")) > 1 {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Commander's Arsenal":
		if len(mtgmatcher.MatchInSet(cardName, "OCM1")) == 1 {
			edition = "OCM1"
		}

	// Oversized commander cards
	case "Commander",
		"Commander 2013",
		"Commander 2014",
		"Commander 2015",
		"Commander 2016",
		"Commander 2017",
		"Commander 2018",
		"Commander 2019",
		"Commander: Ikoria":
		variant = number
		if ogVariant == "V.2" && !mtgmatcher.IsBasicLand(cardName) {
			variant = "oversized"
		}

	// Display commander (V.2 separate edition)
	case "Commander: Strixhaven: Extras",
		"Commander: Adventures in the Forgotten Realms: Extras":
		variant = number
		if ogVariant == "V.2" {
			variant = "Display"
			set, err := mtgmatcher.GetSetByName(strings.TrimSuffix(edition, ": Extras"))
			if err == nil {
				edition = "O" + set.Code
			}
		}

	// Display commander (V.3 separate edition)
	case "Commander: Innistrad: Midnight Hunt",
		"Commander: Innistrad: Crimson Vow":
		variant = number
		if ogVariant == "V.3" {
			variant = "Display"
			set, err := mtgmatcher.GetSetByName(strings.TrimSuffix(edition, ": Extras"))
			if err == nil {
				edition = "O" + set.Code
			}
		}

	case "Commander Legends: Battle for Baldur's Gate: Promos":
		variant = "Prerelease"

	case "Commander: Streets of New Capenna: Promos":
		variant = "Promo Pack"

	case "Commander: The Lord of the Rings: Tales of Middle-earth: Extras":
		variant = number
		switch cardName {
		case "Sol Ring":
			switch ogVariant {
			case "V.2", "V.4", "V.6":
				variant += " serial"
			}
		}

	case "The Lord of the Rings: Tales of Middle-earth Holiday Release":
		variant = number
		if len(mtgmatcher.MatchInSet(cardName, "LTC")) > 0 {
			edition = "LTC"
			if mtgmatcher.HasSerializedPrinting(cardName, "LTC") {
				variant = "serial"
			}
		} else if len(mtgmatcher.MatchInSet(cardName, "LTR")) > 0 {
			edition = "LTR"
			switch ogVariant {
			case "V.2":
				variant += " silverfoil"
				// The other V.2 being extendedart are only for LTC
			case "V.5":
				variant += " serial"
			}
		}

	case "Murders at Karlov Manor":
		variant = number
		if !mtgmatcher.IsBasicLand(cardName) {
			if ogVariant == "V.1" {
				variant = "a"
			} else if ogVariant == "V.2" {
				variant = "b"
			}
		}

	case "30th Anniversary History Promos":
		variant = number
		if ogVariant == "V.2" {
			variant = "retro"
		}

	case "Universes Beyond: Warhammer 40,000":
		variant = number
		edition = "40K"
		switch cardName {
		case "Abaddon the Despoiler",
			"Be'lakor, the Dark Master",
			"Imotekh the Stormlord",
			"Inquisitor Greyfax",
			"Magus Lucea Kane",
			"Marneus Calgar",
			"Szarekh, the Silent King",
			"The Swarmlord":
			switch ogVariant {
			case "V.1":
				variant += " surge foil"
			case "V.2":
				variant += ""
			case "V.3":
				variant += " display surge foil"
			case "V.4":
				variant += " display etched foil"
			}
		default:
			if ogVariant == "V.2" && mtgmatcher.HasFoilPrinting(cardName, "40K") {
				variant += " surge foil"
			}
		}

	case "Retro Frame Artifacts":
		variant = number
		if ogVariant == "V.3" {
			variant += " serialized"
		}

	case "Multiverse Legends":
		variant = number
		if ogVariant == "V.4" {
			variant += " serialized"
		}

	default:
		switch {
		// Try to derive the serialized status from the various Extras sets
		case strings.HasSuffix(edition, ": Extras") && variant == "V.3" && mtgmatcher.HasSerializedPrinting(cardName, strings.TrimSuffix(edition, ": Extras")):
			variant = "serial"

		// Pre-search the card, if not found it's likely a sideboard variant
		case strings.HasPrefix(edition, "Pro Tour 1996:"),
			strings.HasPrefix(edition, "WCD "):
			_, err := mtgmatcher.Match(&mtgmatcher.InputCard{
				Name:    cardName,
				Edition: edition,
			})
			if err != nil {
				edition += " Sideboard"
			}

		// All the various promos
		case strings.Contains(edition, ": Promos"):
			editionNoSuffix := edition
			editionNoSuffix = strings.TrimSuffix(editionNoSuffix, ": Promos")
			editionNoSuffix = strings.Replace(editionNoSuffix, "Core", "Core Set", 1)
			editionNoSuffix = strings.TrimSpace(editionNoSuffix)

			// Retrieve the set date because different tags mean different things
			// depending on the epoch
			set, err := mtgmatcher.GetSetByName(editionNoSuffix)
			if err != nil {
				return nil, &PreprocessError{
					Extra: fmt.Sprintf("%s | %s | %s", cardName, edition, variant),
					Err:   err,
				}
			}
			setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
			if err != nil {
				return nil, &PreprocessError{
					Extra: fmt.Sprintf("%s | %s | %s", cardName, edition, variant),
					Err:   err,
				}
			}

			// These sets are always Prerelease, except for a couple of intro packs
			// that are marked in an unpredictable way
			if setDate.After(mtgmatcher.NewPrereleaseDate) &&
				setDate.Before(mtgmatcher.PromosForEverybodyYay) {

				variant = "Prerelease"
				if ogVariant == "V.2" {
					variant = number
				}
			} else if setDate.After(mtgmatcher.PromosForEverybodyYay) {
				// Default tags
				customVariant := ""
				specialTag := "V.0" // custom, ignored for most cases
				prerelTag := "V.1"
				promoTag := "V.2"
				bundleTag := "V.3"

				// Special cases
				switch editionNoSuffix {
				case "Theros Beyond Death",
					"Core 2021",
					"Ikoria: Lair of Behemoths",
					"Zendikar Rising":
					promoTag = "V.1"
					prerelTag = "V.2"
					switch cardName {
					case "Arasta of the Endless Web":
						promoTag = "V.2"
						prerelTag = "V.3"
					case "Colossification":
						if number == "364" {
							bundleTag = ""
						}
					}
				case "Adventures in the Forgotten Realms":
					specialTag = "V.3"
					customVariant = "Ampersand"
					bundleTag = "V.4"
				case "Kamigawa: Neon Dynasty",
					"The Brothers' War":
					specialTag = "V.3"
					customVariant = "oversized"
					bundleTag = "V.4"
				case "Innistrad: Crimson Vow":
					specialTag = "V.3"
					customVariant = "Play Promo"
				}

				switch variant {
				case specialTag:
					variant = customVariant
				case promoTag:
					variant = "Promo Pack"
				case prerelTag:
					variant = "Prerelease"
				case bundleTag:
					variant = "Bundle"
				default:
					if strings.Contains(cardName, "//") {
						variant = number
					} else if mtgmatcher.HasPromoPackPrinting(cardName) {
						variant = "Promo Pack"
					}
				}
			}
		default:
			// Old editions do not have any number assigned, if so, then keep
			// the V.1 V.2 etc style and process in variants.go
			if number != "" {
				variant = number
			}
		}
	}

	// Try separating SLD and PLST cards if possible
	if strings.Contains(ogEdition, "Secret Lair Commander Deck") {
		for _, card := range mtgmatcher.MatchInSet(cardName, "PLST") {
			if strings.HasSuffix(card.Number, "-"+number) {
				edition = "PLST"
				variant = card.Number
				break
			}
		}

		// Detect thick display commander from these sets
		if !mtgmatcher.IsBasicLand(cardName) && ogVariant == "V.2" {
			variant += " Thick"
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
	}, nil
}
