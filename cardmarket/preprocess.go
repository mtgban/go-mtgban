package cardmarket

import (
	"errors"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var promo2editionTable = map[string]string{
	// Dengeki Maoh Promos
	"Shepherd of the Lost": "URL/Convention Promos",

	// Release Promos
	"Basandra, Battle Seraph":       "Commander 2011 Launch Party",
	"Edric, Spymaster of Trest":     "Commander 2011 Launch Party",
	"Nin, the Pain Artist":          "Commander 2011 Launch Party",
	"Skullbriar, the Walking Grave": "Commander 2011 Launch Party",
	"Vish Kal, Blood Arbiter":       "Commander 2011 Launch Party",

	// Promos
	"Evolving Wilds":                     "Tarkir Dragonfury",
	"Ruthless Cullblade":                 "Worldwake Promos",
	"Pestilence Demon":                   "Rise of the Eldrazi Promos",
	"Dreg Mangler":                       "Return to Ravnica Promos",
	"Karametra's Acolyte":                "Theros Promos",
	"Mayor of Avabruck / Howlpack Alpha": "Innistrad Promos",

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

	"Archangel":                 "Magazine Inserts",
	"Ascendant Evincar":         "Magazine Inserts",
	"Cast Down":                 "Magazine Inserts",
	"Darksteel Juggernaut":      "Magazine Inserts",
	"Daxos, Blessed by the Sun": "Magazine Inserts",
	"Diabolic Edict":            "Magazine Inserts",
	"Duress":                    "Magazine Inserts",
	"Jamuraan Lion":             "Magazine Inserts",
	"Kuldotha Phoenix":          "Magazine Inserts",
	"Lava Coil":                 "Magazine Inserts",
	"Phantasmal Dragon":         "Magazine Inserts",
	"Shivan Dragon":             "Magazine Inserts",
	"Shock":                     "Magazine Inserts",
	"Sprite Dragon":             "Magazine Inserts",
	"Thorn Elemental":           "Magazine Inserts",
	"Voltaic Key":               "Magazine Inserts",

	// Game Day Promos
	"Hall of Triumph": "Journey into Nyx Hero's Path",

	// DCI Promos
	"Abrupt Decay":                "World Magic Cup Qualifiers",
	"Inkmoth Nexus":               "World Magic Cup Qualifiers",
	"Thalia, Guardian of Thraben": "World Magic Cup Qualifiers",
	"Vengevine":                   "World Magic Cup Qualifiers",
}

func Preprocess(cardName, variant, edition string) (*mtgmatcher.Card, error) {
	number := variant
	vars := mtgmatcher.SplitVariants(cardName)
	if len(vars) > 1 {
		cardName = vars[0]
		variant = vars[1]
	}

	switch edition {
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
			variant = "prerelease misprint"
			if variant == "V.1" {
				variant += " dark"
			}
		case "Corpse Knight",
			"Stocking Tiger":
			variant = "misprint"
		case "Beast of Burden",
			"Demigod of Revenge":
			variant = "prerelease misprint"
		case "Island":
			variant = "arena 1999 misprint"
		default:
			return nil, errors.New("untracked")
		}

	case "Arabian Nights":
		if variant == "V.1" {
			variant = "dark"
		} else if variant == "V.2" {
			variant = "light"
		}
	case "Alliances",
		"Champions of Kamigawa",
		"Fallen Empires",
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
	case "Unsanctioned":
		if cardName == "Surgeon Commander" {
			cardName = "Surgeon ~General~ Commander"
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
			if variant == "V.1" {
				variant = "Reminder Text"
			}
		}
	case "Planeshift":
		if variant == "V.2" {
			variant = "Alt Art"
		}
	case "Foreign Black Bordered",
		"Tenth Edition":
		variant = ""
	case "Commander Anthology II",
		"Ravnica Allegiance",
		"Guilds of Ravnica",
		"Conspiracy: Take the Crown",
		"Battlebond",
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
			variant = ""
			edition = "War of the Spark"
		} else if variant == "V.2" {
			variant = "Prerelease"
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
	case "Prerelease Promos":
		switch cardName {
		case "Dirtcoil Wurm":
			if variant == "V.2" {
				variant = "misprint"
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
				variant = "PAL02"
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
	case "Standard Showdown Promos":
		if variant == "V.1" {
			edition = "XLN Standard Showdown"
		} else if variant == "V.2" {
			edition = "M19 Standard Showdown"
		}
	// Catch-all sets for anything promo
	case "Dengeki Maoh Promos",
		"Release Promos",
		"Promos",
		"DCI Promos",
		"Game Day Promos":
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
	case "Core 2020: Extras":
		if variant == "V.1" || mtgmatcher.IsBasicLand(cardName) {
			variant = "Promo Pack"
		} else if variant == "V.2" {
			variant = "Prerelease"
		} else if mtgmatcher.HasPromoPackPrinting(cardName) { // Needs to be after V.2 check
			variant = "Promo Pack"
		} else {
			variant = ""
		}

	default:
		if strings.Contains(edition, "Commander") {
			if variant == "V.2" {
				return nil, errors.New("oversized")
			}
		} else if strings.HasPrefix(edition, "Pro Tour 1996:") || strings.HasPrefix(edition, "WCD ") {
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

			set, err := mtgmatcher.GetSetByName(edition)
			if err != nil {
				return nil, err
			}
			setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
			if err != nil {
				return nil, err
			}

			if strings.Contains(edition, ": Extras") {
				if setDate.Before(mtgmatcher.PromosForEverybodyYay) {
					if mtgmatcher.HasPrereleasePrinting(cardName) {
						variant = "Prerelease"
					}
				} else if edition == "Ikoria: Lair of Behemoths: Extras" {
					if variant == "V.1" {
						variant = "showcase"
					} else if variant == "V.2" {
						variant = "godzilla"
					}
				} else if edition == "Commander Legends: Extras" {
					variant = number
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
					if edition == "Throne of Eldraine: Promos" {
						promopTag = "V.2"
						prerelTag = "V.1"
					}
					if variant == bundleTag {
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
		if edition == "Zendikar" {
			switch variant {
			case "V.1", "V.3", "V.5", "V.7":
				variant = number + "a"
			default:
				variant = number
			}
		} else if edition == "Battle for Zendikar" {
			switch variant {
			case "V.2", "V.4", "V.6", "V.8", "V.10":
				variant = number + "a"
			default:
				variant = number
			}
		} else if edition == "Magic Premiere Shop Promos" {
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
		} else {
			variant = number
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
