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
	// Dengeki Maoh Promos
	"Shepherd of the Lost": "URL/Convention Promos",

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

	"Sol Ring":           "MagicFest 2019",
	"Crucible of Worlds": "World Championship Promos",

	"Forest":   "2017 Gift Pack",
	"Island":   "2017 Gift Pack",
	"Mountain": "2017 Gift Pack",
	"Plains":   "2017 Gift Pack",
	"Swamp":    "2017 Gift Pack",
}

func Preprocess(cardName, variant, edition string) (*mtgmatcher.Card, error) {
	for _, name := range filteredExpansions {
		if edition == name {
			return nil, mtgmatcher.ErrUnsupported
		}
	}

	ogEdition := edition
	number := variant

	switch number {
	case "1 of 1":
		return nil, mtgmatcher.ErrUnsupported
	}

	vars := mtgmatcher.SplitVariants(cardName)
	if len(vars) > 1 {
		cardName = vars[0]
		variant = vars[1]
	}

	switch edition {

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

	case "Arabian Nights":
		if variant == "V.1" {
			variant = "dark"
		} else if variant == "V.2" {
			variant = "light"
		}

	case "Champions of Kamigawa":
		if cardName == "Brothers Yamazaki" {
			if variant == "V.1" {
				variant = "160a"
			} else if variant == "V.2" {
				variant = "160b"
			}
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

	case "Unglued":
		if variant == "V.1" {
			variant = "28"
		} else if variant == "V.2" {
			variant = "29"
		}

	case "Duel Decks: Anthology":
		switch cardName {
		case "Giant Growth":
			if variant == "V.1" {
				edition = "GVL"
			} else if variant == "V.2" {
				edition = "EVG"
			}
		case "Flamewave Invoker":
			if variant == "V.1" {
				edition = "JVC"
			} else if variant == "V.2" {
				edition = "EVG"
			}
		case "Corrupt":
			if variant == "V.1" {
				edition = "DVD"
			} else if variant == "V.2" {
				edition = "GVL"
			}
		case "Harmonize":
			if variant == "V.1" {
				edition = "EVG"
			} else if variant == "V.2" {
				edition = "GVL"
			}
		default:
			variant = ""
			if mtgmatcher.IsBasicLand(cardName) {
				return nil, mtgmatcher.ErrUnsupported
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

	case "Planeshift":
		if variant == "V.2" {
			variant = "Alt Art"
		}

	case "Visions":
		if variant == "V.2" {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Fourth Edition: Black Bordered":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Commander Anthology II",
		"Ravnica Allegiance",
		"Guilds of Ravnica",
		"Conspiracy: Take the Crown",
		"Battlebond",
		"Kaldheim",
		"Secret Lair Drop Series":
		// Could have been lost in SplitVariant, and it's more reliable
		variant = number

	case "Secret Lair Drop Series: Secretversary 2021":
		edition = "PHED"

	case "Theros",
		"Born of the Gods",
		"Journey into Nyx":
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

	case "Zendikar":
		switch variant {
		case "V.1", "V.3", "V.5", "V.7":
			variant = number + "a"
		default:
			variant = number
		}

	case "Battle for Zendikar",
		"Oath of the Gatewatch":
		switch variant {
		case "V.2", "V.4", "V.6", "V.8", "V.10":
			variant = number + "a"
		default:
			variant = number
		}

	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		if variant == "V.1" {
			variant = "Japanese"
			edition = "War of the Spark"
		} else if variant == "V.2" {
			variant = "Prerelease Japanese"
			edition = "War of the Spark Promos"
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

	case "Duel Decks: Jace vs. Chandra":
		if variant == "V.2" {
			variant = "Japanese"
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
		}

	case "Release Promos":
		if len(mtgmatcher.MatchInSet(cardName, "PCMD")) == 1 {
			edition = "PCMD"
		} else if cardName == "Shivan Dragon" {
			return nil, mtgmatcher.ErrUnsupported
		}

	// Catch-all sets for anything promo
	case "Dengeki Maoh Promos",
		"Promos",
		"DCI Promos",
		"Magic the Gathering Products",
		"MagicCon Products":
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
			case "10", "11":
				edition = "SLP"
				variant = number
			}
		case "Mystical Dispute":
			if variant == "V.1" {
				edition = "PWCS"
			} else if variant == "V.2" {
				edition = "PR23"
			}
		case "Snapcaster Mage ":
			if variant == "V.1" {
				edition = "PPRO"
			} else if variant == "V.2" {
				edition = "PR23"
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
			if number == "1" {
				edition = "P30M"
				variant = "1F★"
				if edition == "MagicCon Products" {
					variant = "1F"
				}
			} else if variant == "3" {
				edition = "PSVC"
				variant = ""
			}

		default:
			for _, code := range []string{
				"PMEI", "PPRO", "PEWK", "PNAT", "WMC", "PWCS", "PRES",
				"P30T", "PF23", "PLG21",
				"SLD", "SLP",
				"PW21", "PW22", "PW23",
				"PL21", "PL22", "PL23",
			} {
				// number is often wrong, so
				num, _ := strconv.Atoi(number)
				results := mtgmatcher.MatchInSetNumber(cardName, code, strings.TrimLeft(number, "0"))
				switch code {
				case "PMEI", "PEWK", "PW21", "PW22", "PW23":
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
		// Decouple the foils from this set, they need to marked as foil
		if len(mtgmatcher.MatchInSet(cardName, "FMB1")) > 0 {
			edition = "Mystery Booster Retail Edition Foils"
		}
		variant = ""

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

	case "Modern Horizons 2: Extras":
		// Note: order of these printing checks matters
		if mtgmatcher.IsBasicLand(cardName) {
			switch variant {
			case "V.1", "V.3":
				variant = number
			case "V.2", "V.4":
				variant = number + " Etched"
			}
		} else if mtgmatcher.HasExtendedArtPrinting(cardName) {
			switch variant {
			case "V.1":
				variant = "Retro Frame"
			case "V.2":
				variant = "Retro Frame Foil Etched"
			case "V.3":
				variant = "Extended Art"
			}
		} else if mtgmatcher.HasBorderlessPrinting(cardName) {
			switch variant {
			case "V.1":
				variant = "Borderless"
			case "V.2":
				variant = "Retro Frame"
				if mtgmatcher.HasShowcasePrinting(cardName) {
					variant = "Showcase"
				}
			case "V.3":
				variant = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasShowcasePrinting(cardName) {
			switch variant {
			case "V.1":
				variant = "Showcase"
			case "V.2":
				variant = "Retro Frame"
			case "V.3":
				variant = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasRetroFramePrinting(cardName) {
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

	case "Universes Beyond: Transformers":
		switch variant {
		case "V.1":
			variant = ""
		case "V.2":
			variant = "Shattered"
		}

	case "Retro Frame Artifacts":
		switch variant {
		case "V.1":
			variant = ""
		case "V.2":
			variant = "Schematic"
		case "V.3":
			variant = "Serialized"
		}
		edition = "BRR"

	case "Multiverse Legends":
		switch variant {
		case "V.1":
			variant = ""
		case "V.2":
			variant = "Etched"
		case "V.3":
			variant = "Halo"
		case "V.4":
			variant = "Serialized"
		}

	case "Commander's Arsenal":
		if len(mtgmatcher.MatchInSet(cardName, "OCM1")) == 1 {
			edition = "OCM1"
		}

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

	case "Commander Legends: Battle for Baldur's Gate: Promos":
		variant = "Prerelease"

	case "Commander: Streets of New Capenna: Promos":
		variant = "Promo Pack"

	case "Jumpstart 2022":
		if !mtgmatcher.IsBasicLand(cardName) {
			switch variant {
			case "V.1":
				variant = "Anime"
			case "V.2":
				variant = ""
			}
		}

	case "March of the Machine: The Aftermath: Extras":
		variant = number

	default:
		switch {
		// Pre-search the card, if not found it's likely a sideboard variant
		case strings.HasPrefix(edition, "Pro Tour 1996:"),
			strings.HasPrefix(edition, "WCD "):
			_, err := mtgmatcher.Match(&mtgmatcher.Card{
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

			// Anything between KTK and ELD
			// These sets are always Prerelease, except for a couple of intro packs
			// that are marked in an unpredictable way
			if setDate.After(mtgmatcher.NewPrereleaseDate) &&
				setDate.Before(mtgmatcher.PromosForEverybodyYay) {
				variant = "Prerelease"

				prereleaseTag := "V.1"
				switch editionNoSuffix {
				case "Eldrich Moon",
					"Shadows over Innistrad",
					"Oath of the Gatewatch",
					"Magic Origins":
					prereleaseTag = "V.2"
				}
				if ogVariant == prereleaseTag {
					variant = "Prerelease"
				} else if ogVariant != number {
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
				case "Innistrad: Crimson Vow: Promos":
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
		}
	}

	// Lands are named as "Island (V.1)" and similar, keep the collector number
	// which is surprisingly accurate (errors are ignored for lands anyway)
	if mtgmatcher.IsBasicLand(cardName) {
		switch ogEdition {
		default:
			// Old editions do not have any number assigned, if so, then keep
			// the V.1 V.2 etc style and process in variants.go
			if number != "" {
				variant = number
			}
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
