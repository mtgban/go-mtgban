package cardtrader

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func Preprocess(bp *Blueprint) (*mtgmatcher.InputCard, error) {
	cardName := bp.Name
	edition := bp.Expansion.Name
	number := strings.TrimLeft(bp.Properties.Number, "0")
	variant := ""

	// Some, but not all, have a proper id we can reuse right away
	id := mtgmatcher.Scryfall2UUID(bp.ScryfallId)
	if id != "" {
		return &mtgmatcher.InputCard{
			Id: id,
			// Not needed, but helps debugging
			Name:    cardName,
			Edition: edition,
			// Needed to detect etched finish
			Variation: bp.Version,
		}, nil
	}

	switch edition {
	case "Alliances", "Fallen Empires", "Homelands",
		"Guilds of Ravnica",
		"Ravnica Allegiance",
		"Kaldheim",
		"Asia Pacific Land Program",
		"European Land Program",
		"Commander's Arsenal",
		"Commander Anthology Volume II",
		"Unglued",
		"Mystical Archive: Japanese alternate-art",
		"Chronicles",
		"Chronicles Japanese",
		"Rinascimento",
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
	case "Buy a Box ", "Buy a Box",
		"Armada Comics",
		"Prerelease Promos":
		variant = edition
		if cardName == "Voldaren Estate" {
			variant = "Dracula"
		}
	case "Factory Misprints":
		variant = edition
		switch cardName {
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
		if bp.Id == 29063 {
			variant = "1"
		} else if bp.Id == 29053 {
			variant = "11"
		}
	case "Secret Lair Drop Series":
		variant = number
	case "Champs and States":
		if cardName == "Crucible of Worlds" {
			edition = "World Championship Promos"
		}
	case "Core Set 2021":
		if cardName == "Teferi, Master of Time" {
			variant = number
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
			edition = "DCI"
		}
	case "The List":
		switch cardName {
		case "Everythingamajig", "Ineffable Blessing":
			variant = strings.Fields(bp.Version)[0]
		default:
			if len(mtgmatcher.MatchInSetNumber(cardName, "PLST", number)) > 0 {
				variant = number
			}
		}
	case "Mystery Booster: Convention Edition Playtest Cards":
		variant = bp.Version
	case "Modern Horizons 2",
		"Modern Horizons 1: Timeshifted":
		variant = number
		if strings.HasSuffix(variant, "e") {
			variant = strings.TrimSuffix(variant, "e")
			variant += " Etched"
		}
	case "Commander: The Lord of the Rings - Tales of Middle-earth Collectors",
		"The Lord of the Rings: Tales of Middle-earth Holiday Release":
		variant = strings.Replace(number, "s", "z", 1)
	case "Simplified Chinese Alternate Art Cards":
		switch cardName {
		case "Drudge Skeletons":
			if bp.Version != "" {
				edition = bp.Version
			}
			variant = number
			if !strings.HasSuffix(variant, "s") {
				variant += "s"
			}
		}
	case "Secret Lair Showdown":
		switch cardName {
		case "Relentless Rats":
			variant = number
		}
	default:
		if strings.HasSuffix(edition, "Collectors") {
			variant = number
		} else if strings.HasPrefix(edition, "WCD") ||
			strings.HasPrefix(edition, "Pro Tour 1996") {
			variant = number
			if strings.HasPrefix(variant, "sr") {
				variant = strings.Replace(variant, "sr", "shr", 1)
			}

			switch bp.Id {
			case 25481: // Scrabbling Claws
				variant = "jn237sb"
			case 25491: // Chrome Mox
				variant = "mb152"
			case 25501: // Seething Song
				variant = "ap104sb"
			case 32184: // Aura of Silence
				variant = "bh7bsb"
			case 35075: // Shatter
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
					} else if bp.Id == 60193 || bp.Id == 105989 {
						variant = "Promo Pack"
					}
				case "Feather, the Redeemed":
					variant = "Promo Pack"
					if bp.Id == 56782 {
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
				if mtgmatcher.IsBasicLand(cardName) {
					edition = "M20 Promo Packs"
				} else if cardName == "Chandra's Regulator" {
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
						if cardName == "Sorcerous Spyglass" {
							edition = "PXLN"
						}
					} else {
						edition = "Core Set 2020"
					}
				}
			case "D&D: Adventures in the Forgotten Realms Promos":
				variant = "Promo Pack"
				edition = "PAFR"
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
					notPromoPack := false
					num, convErr := strconv.Atoi(number)
					parentSet, setErr := mtgmatcher.GetSet(set.ParentCode)
					if convErr == nil && setErr == nil {
						notPromoPack = num > parentSet.BaseSetSize
					}

					if mtgmatcher.HasPromoPackPrinting(cardName) && !notPromoPack {
						variant = "Promo Pack"
					} else {
						edition = strings.TrimSuffix(edition, " Promos")
					}
				} else {
					switch edition {
					case "Hour of Devastation Promos":
						if cardName == "Nicol Bolas, God-Pharaoh" && number == "140" {
							variant = "Prerelase"
						}
					case "Dominaria Promos":
						if cardName == "Steel Leaf Champion" && bp.Id == 1833 {
							variant = "182"
						}
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
		case "Guilds of Ravnica Promos",
			"Ravnica Allegiance Promos":
			edition = strings.TrimSuffix(edition, " Promos")
			if strings.HasPrefix(variant, "A") {
				edition = "GRN Ravnica Weekend"
			} else if strings.HasPrefix(variant, "B") {
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
		case "Magic Missile":
			edition = "ARF"
			variant = "401"
		}
	} else if strings.HasSuffix(edition, "Theme Deck") {
		edition = strings.TrimSuffix(edition, " Theme Deck")
	}

	if strings.Contains(bp.Version, "Etched") && !strings.Contains(variant, "Etched") {
		if variant != "" {
			variant += " "
		}
		variant += "Etched"
	}

	// Serialized uses a different suffix than scryfall
	if strings.Contains(bp.Version, "Serialized") {
		variant = "Serialized"
	}

	// Make sure the token tag is always present
	if bp.CategoryId == CategoryMagicTokens && !strings.Contains(cardName, "Token") {
		cardName += " Token"
		if variant == "" {
			variant = strings.TrimPrefix(number, "T")
		}
	}
	if bp.CategoryId == CategoryMagicOversized {
		switch {
		// Preserve the Display Commander tag
		case strings.Contains(bp.Version, "Display"):
			variant += " Display"
		// Make sure the oversize tag is always present
		case !strings.Contains(edition, "Oversize"):
			edition += " Oversize"
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
