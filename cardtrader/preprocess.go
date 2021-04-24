package cardtrader

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Turn to Pumpkin":                "Turn into a Pumpkin",
	"Wall of Stolen Identities":      "Wall of Stolen Identity",
	"Vivien Reid (vers. 1)":          "Vivien Reid",
	"Thalia, Protettrice di Thraben": "Thalia, Guardian of Thraben",
	"iko Yidaro, Wandering Monster":  "Yidaro, Wandering Monster",
	"iko Cartographer's Hawk":        "Cartographer's Hawk",
	"iko Martial Impetus":            "Martial Impetus",
	"Barrin, Tolarian Archmag":       "Barrin, Tolarian Archmage",
	"Soul's Mighty":                  "Soul's Might",
	"Explor":                         "Explore",
	"Swamp (V.2)":                    "Swamp",

	"Karametra, God of Harvests  Karametra, God of Harvests ": "Karametra, God of Harvests",
}

var id2edition = map[int]string{
	// Wasteland
	19144: "G10",
	10702: "J15",
	// Demonic Tutor
	21492: "G08",
	62651: "J20",
	// Vampiric Tutor
	30009: "G00",
	2515:  "J18",
	// Vindicate
	14713: "J13",
	22805: "G07",
	// Lightning Bolt
	32746: "JGP",
	// Sol Ring
	58550: "PF19",

	// Goblin Warchief
	23929: "F06",
	8503:  "F16",

	// Sylvan Ranger
	19128: "PWP10",
	17634: "PWP11",
	// Fling
	19129: "PWP10",
	17635: "PWP11",

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

func preprocess(bp *Blueprint) (*mtgmatcher.Card, error) {
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

	// Some, but not all, have a proper id we can reuse right away
	u, err := url.Parse(bp.ScryfallId)
	if err == nil {
		base := path.Base(u.Path)
		if base != "" && base != "." {
			return &mtgmatcher.Card{
				Id:      base,
				Name:    cardName,
				Edition: edition,
			}, nil
		}
	}

	switch edition {
	case "":
		if bp.Properties.Language == "jp" {
			edition = "PMEI"
			if cardName == "Load Lion" {
				edition = "PRES"
			}
		} else {
			return nil, errors.New("missing edition")
		}
	case "Ultra-Pro Puzzle Cards",
		"Celebration Cards",
		"Foreign White Bordered",
		"Fourth Edition: Alternate",
		"Fallen Empires: Wyvern Misprints",
		"Filler Cards":
		return nil, errors.New("not mtg")
	case "Battle the Horde",
		"Defeat a God",
		"Face the Hydra":
		return nil, errors.New("unsupported")
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
		variant = number
	case "Chronicles Japanese",
		"Rinascimento":
		variant = number
	case "Alliances", "Fallen Empires", "Homelands",
		"Guilds of Ravnica",
		"Ravnica Allegiance",
		"Kaldheim",
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
		if strings.HasSuffix(number, "b") {
			variant = "light"
		} else if strings.Contains(number, "a") {
			variant = "dark"
		}
	case "Champions of Kamigawa":
		if cardName == "Brothers Yamazaki" {
			if bp.Version == "1" {
				variant = "160a"
			} else if bp.Version == "2" {
				variant = "160b"
			}
		}
	case "Buy a Box ",
		"Armada Comics",
		"Prerelease Promos":
		variant = edition
	case "Factory Misprints":
		variant = edition
		switch cardName {
		case "Sapphire Medallion",
			"Thunderheads",
			"Winged Sliver":
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
		ed, found := id2edition[bp.Id]
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
	case "Grand Prix Promos":
		if cardName == "Wilt-Leaf Cavaliers" {
			edition = "PG08"
		}
	default:
		if strings.HasSuffix(edition, "Collectors") {
			variant = number

			switch edition {
			case "Throne of Eldraine Collectors":
				if cardName == "Castle Vantress" && number == "360" {
					variant = "390"
				}
			case "Theros: Beyond Death Collectors":
				if cardName == "Purphoros's Intervention" && number == "313" {
					return nil, errors.New("duplicate")
				}
			case "Commander Legends Collectors":
				if cardName == "Three Visits" && number == "685" {
					variant = "686"
				}
			case "Strixhaven: School of Mages Collectors":
				if cardName == "Magma Opus" && number == "336" {
					variant = "346"
				}
			case "Commander: Strixhaven Collectors":
				if cardName == "Inkshield" && number == "395" {
					variant = "398"
				} else if cardName == "Willowdusk, Essence Seer" && number == "331" {
					variant = "333"
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
				variant += " Prerelease"
			}
		} else if strings.HasSuffix(edition, "Promos") {
			variant = number

			switch edition {
			case "Zendikar Promos":
				if cardName == "Jace, Mirror Mage" {
					variant = "Promo Pack"
					edition = "Zendikar Rising Promos"
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
				version := mtgmatcher.ExtractNumber(strings.Replace(bp.Slug, "-", " ", -1))
				if cardName == "Chandra's Regulator" {
					if version == "1" {
						variant = number
					} else if version == "2" {
						variant = "Promo Pack"
					} else if version == "3" {
						variant = "Prerelease"
					}
				} else if version == "1" {
					variant = "Promo Pack"
				} else if version == "2" {
					variant = "Prerelease"
				} else {
					if mtgmatcher.HasPromoPackPrinting(cardName) {
						variant = "Promo Pack"
					} else {
						edition = "Core Set 2020"
						if mtgmatcher.IsBasicLand(cardName) {
							edition = "M20 Promo Packs"
						}
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
		variant = number
		switch edition {
		case "International Edition",
			"Introductory Two-Player Set",
			"Collectors’ Edition":
			return nil, errors.New("pass")
		// Some basic land foil are mapped to the Promos
		case "Guilds of Ravnica Promos":
			edition = "Guilds of Ravnica"
			if strings.HasPrefix(variant, "A") {
				edition = "GRN Ravnica Weekend"
			}
		case "Ravnica Allegiance Promos":
			edition = "Ravnica Allegiance"
			if strings.HasPrefix(variant, "B") {
				edition = "RNA Ravnica Weekend"
			}
		// Some lands have years set
		case "Arena League Promos":
			variant = mtgmatcher.ExtractYear(strings.Replace(bp.Slug, "-", " ", -1))

			switch variant {
			case "2001":
				switch cardName {
				case "Forest":
					variant = "2001 1"
				case "Mountain", "Swamp":
					variant = "2000"
				}
			case "2002":
				switch cardName {
				case "Forest":
					variant = "2001 11"
				case "Mountain", "Swamp":
					variant = "2001"
				}
			}
		case "Game Night 2019":
			if cardName == "Swamp" && number == "61" {
				variant = "60"
			}
		case "Theros: Beyond Death Theme Deck":
			if cardName == "Swamp" && number == "281" {
				variant = "282"
			}
		case "Core Set 2021 Collectors":
			if cardName == "Mountain" && number == "310" {
				variant = "312"
			}
		case "Magic Premiere Shop":
			if number == "" {
				number = fmt.Sprint(bp.Id)
			}
			variant = pmpsTable[number]
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
			if number == "6" {
				variant = "April"
			} else if number == "8" {
				variant = "July"
			}
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
