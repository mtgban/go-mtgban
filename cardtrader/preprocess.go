package cardtrader

import (
	"errors"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Turn to Pumpkin":                "Turn into a Pumpkin",
	"Wall of Stolen Identities":      "Wall of Stolen Identity",
	"Vivien Reid (vers. 1)":          "Vivien Reid",
	"Thalia, Protettrice di Thraben": "Thalia, Guardian of Thraben",
	"iko Crystalline Resonance":      "Crystalline Resonance",
	"iko Cartographer's Hawk":        "Cartographer's Hawk",
	"iko Martial Impetus":            "Martial Impetus",
	"Barrin, Tolarian Archmag":       "Barrin, Tolarian Archmage",
	"Soul's Mighty":                  "Soul's Might",
	"Captain Vargus Ira":             "Captain Vargus Wrath",
	"Yurlok, the Mana Burner":        "Yurlok of Scorch Thrash",
	"Explor":                         "Explore",

	"Karametra, God of Harvests  Karametra, God of Harvests ": "Karametra, God of Harvests",
}

var card2edition = map[string]string{
	"Vampiric Tutor (vers. 1)":  "G00",
	"Vampiric Tutor (vers. 2)":  "J18",
	"Goblin Warchief (vers. 1)": "F06",
	"Goblin Warchief (vers. 2)": "F16",
	"Sylvan Ranger (vers. 1)":   "PWP10",
	"Sylvan Ranger (vers. 2)":   "PWP11",
	"Fling (vers. 1)":           "PWP10",
	"Fling (vers. 2)":           "PWP11",
	"Demonic Tutor (Vers. 1)":   "G08",
	"Demonic Tutor (Vers. 2)":   "J20",
}

var id2edition = map[int]string{
	// Wasteland
	19144: "G10",
	10702: "J15",
	// Vindicate
	14713: "J13",
	22805: "G07",
	// Lightning Bolt
	32746: "JGP",
	// Sol Ring
	58550: "PF19",

	// Arena Plains
	// the only land from the PARL series to miss the year in their full name
	34701: "PARL",
	31623: "PAL99",
	30002: "PAL00",
	29057: "PAL01",
	27027: "PAL03",
	25897: "PAL04",
	24949: "PAL05",
	23847: "PAL06",
}

func preprocess(bp Blueprint) (*mtgmatcher.Card, error) {
	cardName := bp.Name
	edition := bp.Expansion.Name
	number := strings.TrimLeft(bp.Properties.Number, "0")
	variant := ""

	if mtgmatcher.IsToken(cardName) {
		return nil, errors.New("not singles")
	}

	altName, found := cardTable[cardName]
	if found {
		cardName = altName
	}

	switch edition {
	case "Ultra-Pro Puzzle Cards",
		"Celebration Cards",
		"Foreign White Bordered",
		"Fourth Edition: Alternate",
		"Fallen Empires: Wyvern Misprints",
		"Filler Cards":
		return nil, errors.New("not mtg")
	case "Salvat 2005":
		return nil, errors.New("foreign")
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
	case "Fourth Edition Black Bordered":
		edition = "Fourth Edition Foreign Black Border"
	case "Chronicles Japanese":
		edition = "Chronicles"
		variant = number
	case "Alliances", "Fallen Empires", "Homelands",
		"Guilds of Ravnica",
		"Ravnica Allegiance",
		"Asia Pacific Land Program",
		"European Land Program",
		"Commander Anthology Volume II",
		"Unglued",
		"Chronicles",
		"Antiquities":
		variant = number
	case "Commander Legends: Commander Decks":
		edition = "Commander Legends"
		variant = number
	case "Arabian Nights":
		if strings.Contains(bp.DisplayName, "(light)") {
			variant = "light"
		} else if strings.Contains(bp.DisplayName, "(dark)") {
			variant = "dark"
		}
	case "Champions of Kamigawa":
		if bp.DisplayName == "Brothers Yamazaki (vers. 1)" {
			variant = "160a"
		} else if bp.DisplayName == "Brothers Yamazaki (vers. 2)" {
			variant = "160b"
		}
	case "Buy a Box ",
		"Armada Comics",
		"Prerelease Promos":
		variant = edition
	case "Factory Misprints":
		variant = edition
		switch cardName {
		case "Sapphire Medallion", "Thunderheads":
			return nil, errors.New("unknown")
		case "Laquatus's Champion":
			variant = "prerelease misprint"
		case "Island":
			edition = "PAL99"
		}
	case "Judge Gift Cards",
		"Wizards Play Network",
		"Arena League Promos",
		"Friday Night Magic",
		"Player Rewards Promos":
		ed, found := card2edition[bp.DisplayName]
		if found {
			edition = ed
		}
		ed, found = id2edition[bp.Id]
		if found {
			edition = ed
		}
	case "Secret Lair Drop Series":
		variant = number
		switch cardName {
		case "Treasure",
			"Walker":
			return nil, errors.New("not single")
		}
	case "Champs and States":
		if cardName == "Crucible of Worlds" {
			edition = "World Championship Promos"
		}
	case "Core Set 2021":
		if cardName == "Teferi, Master of Time" {
			variant = number
		}
	case "Signature Spellbook: Chandra":
		if bp.Id == 133840 {
			cardName = "Cathartic Reunion"
		} else if bp.Id == 133841 {
			cardName = "Pyroblast"
		}
	case "DCI Promos":
		switch cardName {
		case "Cryptic Command":
			edition = "PPRO"
		case "Flooded Strand":
			edition = "PNAT"
		}
	default:
		if strings.HasSuffix(edition, "Collectors") {
			variant = number

			switch edition {
			case "Throne of Eldraine Collectors":
				if cardName == "Castle Vantress" {
					variant = "390"
				}
			case "Theros: Beyond Death Collectors":
				if cardName == "Purphoros's Intervention" && number == "313" {
					return nil, errors.New("duplicate")
				}
			case "Ikoria: Lair of Behemoths Collectors":
				if strings.Contains(bp.DisplayName, "Vers. 1") {
					cardName = strings.TrimPrefix(cardName, "iko ")
					variant = number
				} else if strings.Contains(bp.DisplayName, "Vers, 2") ||
					strings.Contains(bp.DisplayName, "Vers. 2") {
					variant = "Godzilla"
				}
			case "Core Set 2021 Collectors":
				if cardName == "Mountain" && number == "310" {
					variant = "312"
				}
			case "Commander Legends Collectors":
				if cardName == "Three Visits" && number == "685" {
					variant = "686"
				}
			}
		} else if strings.HasPrefix(edition, "WCD") ||
			strings.HasPrefix(edition, "Pro Tour 1996") {
			variant = number
			if strings.HasPrefix(variant, "sr") {
				variant = strings.Replace(variant, "sr", "shr", 1)
			}
			// Scrabbling Claws
			if bp.Id == 25481 {
				variant = "jn237sb"
			} else if bp.Id == 35075 { // Shatter
				variant = "gb219sb"
			}
		} else if strings.Contains(edition, "Japanese") {
			variant = "Japanese"
			if strings.Contains(edition, "Promo") {
				variant = "Prerelease"
			}
			edition = "War of the Spark"
		} else if strings.HasSuffix(edition, "Promos") {
			variant = number

			switch edition {
			case "Zendikar Promos":
				if cardName == "Jace, Mirror Mage" {
					variant = "Promo Pack"
					edition = "Zendikar Rising Promos"
				}
			case "Core Set 2019 Promos":
				// g18 cards are folded in pm19 edition
				if strings.HasPrefix(number, "GP") {
					edition = "G18"
				}
			case "Guilds of Ravnica Promos":
				switch cardName {
				case "Attendant of Vraska",
					"Kraul Raider",
					"Precision Bolt",
					"Ral's Dispersal",
					"Ral's Staticaster",
					"Ral, Caller of Storms",
					"Vraska's Stoneglare",
					"Vraska, Regal Gorgon":
					edition = "Guilds of Ravnica"
				}
			// Starting from RNA, the collector numbers do not have the
			// right suffix any more, we need to special case a lot of cards
			case "Ravnica Allegiance Promos":
				switch cardName {
				case "Light Up the Stage",
					"Growth Spiral",
					"Mortify",
					"Rakdos Firewheeler",
					"Simic Ascendancy":
					variant = number
				case "Dovin, Architect of Law",
					"Elite Arrester",
					"Dovin's Dismissal",
					"Dovin's Automaton",
					"Domri, City Smasher",
					"Ragefire",
					"Charging War Boar",
					"Domri's Nodorog":
					edition = "Ravnica Allegiance"
				default:
					variant = number + "s"
				}
			case "War of the Spark Promos":
				switch cardName {
				case "Augur of Bolas",
					"Liliana's Triumph",
					"Paradise Druid",
					"Dovin's Veto":
					variant = number
				case "Bolas's Citadel",
					"Karn's Bastion":
					variant = number
					if bp.Id == 56810 || bp.Id == 56746 {
						variant = "Prerelease"
					}
				case "Desperate Lunge",
					"Gideon's Battle Cry",
					"Gideon's Company",
					"Gideon, the Oathsworn",
					"Guildpact Informant",
					"Jace's Projection",
					"Jace's Ruse",
					"Jace, Arcane Strategist",
					"Orzhov Guildgate",
					"Simic Guildgate",
					"Tezzeret, Master of the Bridge":
					edition = "War of the Spark"
				default:
					variant = "Prerelease"
				}
			// This set acts mostly as a catch-all for anything prior :(
			case "Core Set 2020 Promos":
				if cardName == "Chandra's Regulator" {
					if strings.Contains(bp.DisplayName, "Vers. 1") {
						variant = number
					} else if strings.Contains(bp.DisplayName, "Vers. 2") {
						variant = "Promo Pack"
					} else if strings.Contains(bp.DisplayName, "Vers. 2") {
						variant = "Prerelease"
					}
				} else if strings.Contains(bp.DisplayName, "Vers. 1") {
					variant = "Promo Pack"
				} else if strings.Contains(bp.DisplayName, "Vers. 2") {
					variant = "Prerelease"
				} else {
					if mtgmatcher.HasPromoPackPrinting(cardName) {
						variant = "Promo Pack"
					} else if !mtgmatcher.IsBasicLand(cardName) {
						// Lands are adjusted below
						edition = "Core Set 2020"
					} else {
						variant = number
					}
				}
			default:
				set, err := mtgmatcher.GetSet(bp.Expansion.Code)
				if err != nil {
					return nil, err
				}
				setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
				if err != nil {
					return nil, err
				}

				if setDate.After(mtgmatcher.PromosForEverybodyYay) {
					if mtgmatcher.HasPromoPackPrinting(cardName) {
						variant = "Promo Pack"
					} else {
						edition = strings.TrimSuffix(edition, " Promos")
					}
				}
			}
		}
	}

	if mtgmatcher.IsBasicLand(cardName) {
		switch edition {
		case "International Edition",
			"Collectors’ Edition":
			return nil, errors.New("pass")
		// Skip assignment for these editions
		case "Judge Gift Cards",
			"Grand Prix Promos",
			"European Land Program",
			"Asia Pacific Land Program":
		// Some basic land foil are mapped to the Promos
		case "Guilds of Ravnica Promos":
			edition = "Guilds of Ravnica"
			variant = number
			if strings.HasPrefix(variant, "A") {
				edition = "GRN Ravnica Weekend"
			}
		case "Ravnica Allegiance Promos":
			edition = "Ravnica Allegiance"
			variant = number
			if strings.HasPrefix(variant, "B") {
				edition = "RNA Ravnica Weekend"
			}
		case "Core Set 2020 Promos":
			edition = "M20 Promo Packs"
		case "Oath of the Gatewatch":
			// Handle the "a" suffix
			if cardName == "Wastes" {
				variant = number
			}
		// Some lands have years set
		case "Arena League Promos":
			if mtgmatcher.ExtractYear(bp.DisplayName) != "" {
				variant = bp.DisplayName
			}
			switch variant {
			case "Forest (2001)":
				variant = "2001 1"
			case "Forest (2002)":
				variant = "2001 11"
			case "Mountain (2001)", "Swamp (2001)":
				variant = "2000"
			case "Mountain (2002)", "Swamp (2002)":
				variant = "2001"
			}
		case "Game Night 2019":
			if cardName == "Swamp" && number == "61" {
				variant = "60"
			} else {
				cardName = bp.DisplayName
			}
		case "Theros: Beyond Death Theme Deck":
			if cardName == "Swamp" && number == "281" {
				variant = "282"
			} else {
				cardName = bp.DisplayName
			}
		default:
			switch {
			// Use number as is
			case strings.HasPrefix(edition, "Duel Decks:"),
				strings.HasPrefix(edition, "WCD"),
				strings.HasPrefix(edition, "Pro Tour 1996"),
				strings.HasPrefix(edition, "Commander"):
				variant = number
			// Maybe there is a number in the full name
			case strings.Contains(bp.DisplayName, " "):
				cardName = bp.DisplayName
			}
		}
	}

	if strings.Contains(edition, "Prerelease") {
		edition = strings.Replace(edition, "Prerelease", "Promos", 1)
		variant = "Prerelease"

		switch cardName {
		case "Curious Pair // Treats to Share":
			edition = "Throne of Eldraine"
			variant = "Showcase"
		case "Lu Bu, Master-at-Arms":
			edition = "Prerelease Events"
			variant = number
		case "Chord of Calling", "Wrath of God":
			edition = "Double Masters"
			variant = number
		}
	} else if strings.HasSuffix(edition, "Theme Deck") {
		edition = strings.TrimSuffix(edition, " Theme Deck")
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
