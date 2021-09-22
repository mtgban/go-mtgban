package cardmarket

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
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

	"Goblin Chieftain":        "Resale Promos",
	"Oran-Rief, the Vastwood": "Resale Promos",
	"Loam Lion":               "Resale Promos",

	"Dragon Fodder":        "Tarkir Dragonfury",
	"Dragonlord's Servant": "Tarkir Dragonfury",
	"Foe-Razer Regent":     "Tarkir Dragonfury",

	"Reliquary Tower":   "Love Your LGS",
	"Hangarback Walker": "Love Your LGS",

	"Geist of Saint Traft": "World Magic Cup Qualifiers",
	"Sol Ring":             "MagicFest 2019",
	"Crucible of Worlds":   "World Championship Promos",

	"Forest":   "2017 Gift Pack",
	"Island":   "2017 Gift Pack",
	"Mountain": "2017 Gift Pack",
	"Plains":   "2017 Gift Pack",
	"Swamp":    "2017 Gift Pack",

	// Media Inserts
	"Archangel":                           "PMEI",
	"Ascendant Evincar":                   "PMEI",
	"Cast Down":                           "PMEI",
	"Crop Rotation":                       "PMEI",
	"Dark Ritual":                         "PMEI",
	"Darksteel Juggernaut":                "PMEI",
	"Daxos, Blessed by the Sun":           "PMEI",
	"Diabolic Edict":                      "PMEI",
	"Duress":                              "PMEI",
	"Heliod's Pilgrim":                    "PMEI",
	"Hypnotic Sprite // Mesmeric Glare":   "PMEI",
	"Jamuraan Lion":                       "PMEI",
	"Kuldotha Phoenix":                    "PMEI",
	"Lava Coil":                           "PMEI",
	"Phantasmal Dragon":                   "PMEI",
	"Shivan Dragon":                       "PMEI",
	"Shock":                               "PMEI",
	"Sprite Dragon":                       "PMEI",
	"Staggering Insight":                  "PMEI",
	"Tangled Florahedron // Tangled Vale": "PMEI",
	"Thorn Elemental":                     "PMEI",
	"Voltaic Key":                         "PMEI",

	// DCI Promos
	"Abrupt Decay":                "World Magic Cup Qualifiers",
	"Inkmoth Nexus":               "World Magic Cup Qualifiers",
	"Thalia, Guardian of Thraben": "World Magic Cup Qualifiers",
	"Vengevine":                   "World Magic Cup Qualifiers",
}

func Preprocess(cardName, variant, edition string) (*mtgmatcher.Card, error) {
	skipCard := false
	for _, name := range filteredCards {
		if cardName == name {
			skipCard = true
			break
		}
	}
	for _, name := range filteredExpansions {
		if edition == name {
			skipCard = true
			break
		}
	}
	if strings.Contains(edition, "Token") ||
		strings.Contains(edition, "Oversized") ||
		strings.Contains(edition, "Player Cards") {
		skipCard = true
	}

	if skipCard ||
		mtgmatcher.IsToken(cardName) ||
		strings.Contains(cardName, "On Your Turn") {
		return nil, errors.New("not single")
	}

	ogEdition := edition
	number := variant
	vars := mtgmatcher.SplitVariants(cardName)
	if len(vars) > 1 {
		cardName = vars[0]
		variant = vars[1]
	}

	switch edition {
	case "The Dark Italian":
		number = ""
		variant = ""
	case "Player Rewards Promos":
		switch cardName {
		case "Lightning Bolt":
			if variant == "V.1" {
				return nil, errors.New("oversized")
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
			return nil, errors.New("oversized")
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
			return nil, errors.New("unsupported misprint")
		case "Island":
			variant = "Arena 1999 misprint"
		default:
			return nil, errors.New("untracked")
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
			return nil, errors.New("unsupported")
		}
	case "Foreign Black Bordered",
		"Tenth Edition":
		variant = ""
	case "Fourth Edition: Black Bordered":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, errors.New("unsupported")
		}
	case "Commander's Arsenal":
		switch cardName {
		case "Azusa, Lost but Seeking",
			"Brion Stoutarm",
			"Glissa, the Traitor",
			"Godo, Bandit Warlord",
			"Grimgrin, Corpse-Born",
			"Karn, Silver Golem",
			"Karrthus, Tyrant of Jund",
			"Mayael the Anima",
			"Sliver Queen",
			"Zur the Enchanter":
			return nil, errors.New("oversize")
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
		if cardName == "Squire" {
			return nil, errors.New("untracked")
		}
	case "Theros",
		"Born of the Gods",
		"Journey into Nyx":
		// Skip the token-based cards, TFTH, TBTH, and TDAG
		if strings.HasPrefix(variant, "C") {
			return nil, errors.New("untracked")
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
				return nil, errors.New("unsupported misprint")
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
		}
		// This set has incorrect numbering
		if variant == number {
			variant = ""
		}
	case "Harper Prism Promos":
		if variant == "V.2" {
			return nil, errors.New("non-english")
		}
	case "Simplified Chinese Alternate Art Cards":
		return nil, errors.New("non-english")
	case "Duel Decks: Jace vs. Chandra":
		if variant == "V.2" {
			variant = "Japanese"
		}
	case "Gateway Promos":
		switch cardName {
		case "Naya Sojourners":
			edition = "PM10"
		case "Fling",
			"Sylvan Ranger":
			if variant == "V.1" {
				edition = "PWP10"
			} else if variant == "V.2" {
				edition = "PWP11"
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
	case "Standard Showdown Promos":
		if variant == "V.1" {
			edition = "XLN Standard Showdown"
		} else if variant == "V.2" {
			edition = "M19 Standard Showdown"
		}
	case "Release Promos":
		switch cardName {
		case "Basandra, Battle Seraph",
			"Edric, Spymaster of Trest",
			"Nin, the Pain Artist",
			"Skullbriar, the Walking Grave",
			"Vish Kal, Blood Arbiter":
			return nil, errors.New("oversize")
		case "Shivan Dragon":
			return nil, errors.New("non english")
		}
	// Catch-all sets for anything promo
	case "Dengeki Maoh Promos",
		"Promos",
		"DCI Promos":
		switch cardName {
		case "Mayor of Avabruck / Howlpack Alpha":
			return nil, errors.New("oversize")
		case "Nalathni Dragon":
			return nil, errors.New("non english")
		}
		// Variant is always unreliable
		variant = ""
		ed, found := promo2editionTable[cardName]
		if found {
			edition = ed
		}
	case "Magic Origins: Promos":
		if variant == "V.2" {
			variant = number
		} else {
			variant = "Prerelease"
		}
	case "Core 2019: Promos":
		variant = "Prerelease"
		edition = "PM19"
		if strings.HasPrefix(number, "GP") {
			variant = ""
			edition = "M19 Gift Pack"
		}
	case "Core 2020: Extras":
		if cardName == "Chandra's Regulator" {
			if variant == "V.1" {
				variant = "131"
				edition = "PM20"
			} else if variant == "V.2" {
				variant = "Promo Pack"
			} else if variant == "V.3" {
				variant = "Prerelease"
			}
		} else if cardName == "Nicol Bolas, Dragon-God" {
			return nil, errors.New("dupe")
		} else {
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
	case "Commander: Zendikar Rising":
		if variant == "V.2" {
			variant = "Extended Art"
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
	case "Modern Horizons 2: Extras":
		// Note: order of these printing checks matters
		if mtgmatcher.HasExtendedArtPrinting(cardName) {
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
		}
	case "Modern Horizons: Retro Frame Cards":
		switch variant {
		case "V.1":
			variant = "Retro Frame"
		case "V.2":
			variant = "Foil Etched"
		}

	default:
		// Double parse commander so that it is possible to drop the oversize
		// cards as well as catch the Extras in the block below
		if strings.Contains(edition, "Commander") {
			hasExtra := strings.HasSuffix(edition, ": Extras")
			edition = mtgmatcher.ParseCommanderEdition(edition)

			if strings.HasPrefix(edition, "Commander 20") && variant == "V.2" {
				return nil, errors.New("oversized")
			}
			// Restore the tag for correct parsing of CMR Collectors cards
			if hasExtra {
				edition += ": Extras"
			}
		}

		if strings.HasPrefix(edition, "Pro Tour 1996:") || strings.HasPrefix(edition, "WCD ") {
			if variant != "" {
				variant += " "
			}
			variant += edition

			// Pre-search the card, if not found it's likely a sideboard variant
			_, err := mtgmatcher.Match(&mtgmatcher.Card{
				Name:    cardName,
				Edition: edition,
			})
			if err != nil {
				variant = "sideboard"
			}
		} else if strings.Contains(edition, ": Extras") || strings.Contains(edition, ": Promos") {
			// Some cards escape the previous checks
			if strings.Contains(cardName, "Art Series: ") {
				return nil, errors.New("untracked")
			}

			// These sets usually have incorrect numbering
			if variant == number {
				variant = ""
			}

			editionNoSuffix := strings.TrimSuffix(edition, ": Extras")
			editionNoSuffix = strings.TrimSuffix(editionNoSuffix, ": Promos")
			editionNoSuffix = strings.Replace(editionNoSuffix, "Core", "Core Set", 1)

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

			if strings.Contains(edition, ": Extras") {
				if setDate.Before(mtgmatcher.PromosForEverybodyYay) {
					if mtgmatcher.HasPrereleasePrinting(cardName) {
						variant = "Prerelease"
						edition = strings.Replace(edition, ": Extras", " Promos", 1)
					}
					if mtgmatcher.IsBasicLand(cardName) {
						edition = strings.Replace(edition, " Promos", "", 1)
					}
				} else {
					// KHM: Extras has spurious promo pack cards in it
					if len(mtgmatcher.MatchInSet(cardName, set.Code)) != 0 {
						variant = number
					} else {
						if mtgmatcher.HasPromoPackPrinting(cardName) {
							variant = "Promo Pack"
						} else if mtgmatcher.HasPrereleasePrinting(cardName) {
							variant = "Prerelease"
						}
					}
				}
			} else if strings.Contains(edition, ": Promos") {
				if setDate.After(mtgmatcher.NewPrereleaseDate) &&
					setDate.Before(mtgmatcher.PromosForEverybodyYay) {
					if strings.HasPrefix(variant, "V.") {
						if variant == "V.1" {
							variant = number
						} else if variant == "V.2" {
							variant = "Prerelease"
						}
					} else {
						variant = "Prerelease"
					}
				} else if setDate.After(mtgmatcher.PromosForEverybodyYay) {
					promopTag := "V.1"
					prerelTag := "V.2"
					bundleTag := "V.3"

					switch edition {
					case "Theros Beyond Death: Promos":
						if cardName == "Arasta of the Endless Web" {
							promopTag = "V.2"
							prerelTag = "V.3"
							bundleTag = "V.1"
						}
					case "Throne of Eldraine: Promos",
						"Kaldheim: Promos":
						promopTag = "V.2"
						prerelTag = "V.1"
					case "Zendikar Rising: Promos":
						if variant == "V.1" && strings.Contains(cardName, " // ") {
							return nil, errors.New("not exist")
						}
					}

					if variant == bundleTag {
						variant = "bundle"
						edition = strings.TrimSuffix(edition, ": Promos")
					} else if cardName == "Colossification" && variant == "" {
						variant = "bundle"
						edition = strings.TrimSuffix(edition, ": Promos")
					} else if variant == promopTag {
						variant = "Promo Pack"
					} else if variant == prerelTag {
						variant = "Prerelease"
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
		// Too much
		case "Foreign Black Bordered",
			"Deckmasters",
			"Coldsnap Theme Decks",
			"Summer Magic",
			"Introductory Two-Player Set",
			"Duels of the Planeswalkers Decks",
			"Duel Decks: Anthology",
			"Archenemy":
			return nil, errors.New("unsupported")
		case "Core 2020: Extras":
			variant = "Promo Pack"
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
		// Use as is, drop variant
		case "Guru Lands":
			variant = ""
		// Sets preprocessed earlier, or with invalid numbers, don't do anything
		case "Arena League Promos",
			"APAC Lands",
			"Euro Lands",
			"Anthologies",
			"Misprints":
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
