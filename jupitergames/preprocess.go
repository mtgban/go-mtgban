package jupitergames

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

var cardTable = map[string]string{
	"Siege Modificataion":                   "Siege Modification",
	"Renegade's Gateway":                    "Renegade's Getaway",
	"Decomission":                           "Decommission",
	"Forsake the Worldy":                    "Forsake the Worldly",
	"Temmet, Vizer of Naktamun":             "Temmet, Vizier of Naktamun",
	"Holy Justicar":                         "Holy Justiciar",
	"United Front":                          "Unified Front",
	"Archetype of Agression":                "Archetype of Aggression",
	"Bladeing the Risen":                    "Bladewing the Risen",
	"Dreadbringer Lampands":                 "Dreadbringer Lampads",
	"Glint-Eye Nephilium":                   "Glint-Eye Nephilim",
	"Higland Lake":                          "Highland Lake",
	"Necromatic Selection":                  "Necromantic Selection",
	"Ancient Ampitheater":                   "Ancient Amphitheater",
	"Avacyn's Judgement":                    "Avacyn's Judgment",
	"Biomass Mutaion":                       "Biomass Mutation",
	"Kordoza Guildmage":                     "Korozda Guildmage",
	"Fearless Halbredier":                   "Fearless Halberdier",
	"Siege Mastadon":                        "Siege Mastodon",
	"Skynight Vanguard":                     "Skyknight Vanguard",
	"Karagan Dragonrider":                   "Kargan Dragonrider",
	"Maurauder's Axe":                       "Marauder's Axe",
	"Stony Quarry":                          "Stone Quarry",
	"Centaur Chieftan":                      "Centaur Chieftain",
	"Archon of the Triumvrate":              "Archon of the Triumvirate",
	"Azorius Justicar":                      "Azorius Justiciar",
	"Primal Vistiation":                     "Primal Visitation",
	"Pyromantics":                           "Pyromatics",
	"Djeru's Renuncation":                   "Djeru's Renunciation",
	"Incendiary Sabatoge":                   "Incendiary Sabotage",
	"Revoke Priviledges":                    "Revoke Privileges",
	"Maurauding Boneslasher":                "Marauding Boneslasher",
	"Ixali's Keeper":                        "Ixalli's Keeper",
	"Sage of Shalia's Claim":                "Sage of Shaila's Claim",
	"Vile Maifestation":                     "Vile Manifestation",
	"Stampede Diver":                        "Stampede Driver",
	"Wolly Loxodon":                         "Woolly Loxodon",
	"Profane Momento":                       "Profane Memento",
	"Imposter of the Sixth Pride":           "Impostor of the Sixth Pride",
	"Wooly Thoctar":                         "Woolly Thoctar",
	"Mortapod":                              "Mortarpod",
	"Viashino Slaughermaster":               "Viashino Slaughtermaster",
	"Eyeblight's Endling":                   "Eyeblight's Ending",
	"Gleeful Sabatoge":                      "Gleeful Sabotage",
	"Greebelt Rampager":                     "Greenbelt Rampager",
	"Milikin":                               "Millikin",
	"Vessel of Volitility":                  "Vessel of Volatility",
	"Briutalizer Exarch":                    "Brutalizer Exarch",
	"Miotic Slime":                          "Mitotic Slime",
	"Spawnbringer Mage":                     "Spawnbinder Mage",
	"Steppe Gilder":                         "Steppe Glider",
	"Survey the Wreakage":                   "Survey the Wreckage",
	"Trained Carcal":                        "Trained Caracal",
	"Justicar's Portal":                     "Justiciar's Portal",
	"Twilght Panther":                       "Twilight Panther",
	"Erdwall Illuminator":                   "Erdwal Illuminator",
	"Geier Reach Bandit | Vidin-Pack Alpha": "Geier Reach Bandit | Vildin-Pack Alpha",
	"Murder's Axe":                          "Murderer's Axe",
	"Skin Invasion | Skin Shredder":         "Skin Invasion | Skin Shedder",
	"True-Faith Censor":                     "True-Faith Censer",
	"Twins of Marer Estate":                 "Twins of Maurer Estate",
	"Scholar of Atheros":                    "Scholar of Athreos",
	"Who|What|When|Where|Why":               "Who",
	"Hazmat Suit _Used_":                    "Hazmat Suit (Used)",
	"Erase [Not the Urza's Legacy One]":     "Erase (Not the Urza's Legacy One)",
	"Kill Destroy":                          "Kill! Destroy!",
	"Dr. Juilus Jumblemorph":                "Dr. Julius Jumblemorph",
	"Villanous Wealth":                      "Villainous Wealth",

	"The Ultimate Nightmare of Wizards of the Coast? Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",

	"Breya, the Etherium Shaper": "Breya, Etherium Shaper",
	"Kinght of the White Orchid": "Knight of the White Orchid",
	"Mythos of Vardok":           "Mythos of Vadrok",
	"Sharbraz, the Skyshark":     "Shabraz, the Skyshark",
}

var commonTags = []string{
	"2019 League",
	"Arena",
	"Book",
	"Buy-A-Box",
	"Buy-a-Box",
	"Comics",
	"Draft Weekend",
	"Duelist",
	"Friday Night Magic",
	"Game Day",
	"Gateway",
	"Grand Prix",
	"Holiday Box",
	"Intro Pack",
	"Ixalan League",
	"Judge",
	"Launch",
	"Magazine",
	"Open House",
	"Player Rewards",
	"Prerelease",
	"Preview",
	"Promo Pack",
	"Qualifier",
	"SDCC",
	"Store Championship",
	"Textless",
	"WPN",

	"April", "July",
}

func preprocess(cardName, variant, edition, format string) (*mtgmatcher.Card, error) {
	switch {
	case strings.Contains(cardName, "Checklist"),
		strings.Contains(cardName, "Punch Out Card"),
		strings.Contains(cardName, "Experience Counter Card"),
		strings.Contains(cardName, "Token"):
		return nil, errors.New("not single")
	}

	if edition == "Promo - Ikoria: Lair of Behemoths Prerelease" {
		return nil, errors.New("not yet supported")
	}

	isFoil := strings.Contains(format, "FOIL") && !strings.Contains(format, "NON")
	if strings.HasPrefix(edition, "Promo - ") {
		edition = strings.TrimPrefix(edition, "Promo - ")
	}
	if strings.Contains(edition, "Prelease") || strings.Contains(edition, "Prerlease") {
		edition += " Prerelease"
	}

	extra := ""
	// Sometimes variant contains a number and the actual collector number
	// but the collector number is correct only for lands and a few sets,
	// so pick and choose, here, and below
	if strings.Contains(variant, "[") {
		variants := strings.Split(variant, " [")
		variant = variants[0]
		if len(variants) > 1 {
			if mtgmatcher.IsBasicLand(cardName) ||
				edition == "Secret Lair Drop Sets" ||
				edition == "Commander Anthology 2018" ||
				edition == "Unglued" {
				switch edition {
				case "Arena League", "Standard Showdown", "Unsanctioned":
				default:
					extra = strings.Replace(variants[1], "]", "", 1)
				}
			}
		}
	}

	s := strings.Split(variant, " - ")
	variant = s[0]
	if len(s) > 1 {
		extra = s[1]
	}

	// Move common tags from edition to variant
	for _, tag := range commonTags {
		if strings.Contains(edition, tag) {
			if variant != "" {
				variant += " "
			}
			variant += tag + " Promo"
		}
	}

	// Try decoupling lands as much as possible, in particular retrieve the
	// year of Arena League by looking at the single digit number
	if mtgmatcher.IsBasicLand(cardName) || strings.Contains(cardName, "Guildgate") {
		switch edition {
		case "Portal Second Age",
			"Portal",
			"Ice Age",
			"Revised Edition",
			"Tempest",
			"Mirage":
			vars := map[string]string{
				"1": "a",
				"2": "b",
				"3": "c",
				"4": "d",
			}
			variant = vars[variant]
		case "Arena League":
			var year string
			variation := strings.TrimSuffix(variant, " Arena Promo")
			switch cardName {
			case "Forest":
				year = map[string]string{
					"1": "2006",
					"2": "2005",
					"3": "2004",
					"4": "2003",
					"5": "2001 11",
					"6": "2001 1",
					"7": "2000",
					"8": "1999",
					"9": "1996",
				}[variation]
			case "Island":
				year = map[string]string{
					"1": "2006",
					"2": "2005",
					"3": "2004",
					"4": "2003",
					"5": "2002",
					"6": "2001",
					"7": "2000",
					"8": "1999",
					"9": "1996",
				}[variation]
			default:
				year = map[string]string{
					"1": "2006",
					"2": "2005",
					"3": "2004",
					"4": "2003",
					"5": "2001",
					"6": "2000",
					"7": "1999",
					"8": "1996",
				}[variation]
			}
			variant = "Arena " + year
		case "Judge Gift Program":
			variant = "Judge Promo"
		case "Standard Showdown", "Unsanctioned":
			// skip, handled below
		case "Limited Edition Alpha":
			if cardName == "Mountain" {
				if variant == "2" {
					extra = "#284"
				}
			}
			fallthrough
		default:
			variant = extra
		}
	}

	variants := mtgmatcher.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += variants[1]
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	// Only for buylist mode, needs to be after cardTable
	if strings.Contains(cardName, "[") {
		variants := strings.Split(cardName, " [")
		cardName = variants[0]
	}

	switch cardName {
	case "Dreg Mangler":
		if edition == "Holiday Box 2012" {
			variant = "Game Day"
		}
	case "Wasteland":
		if strings.Contains(variant, "Judge") {
			if strings.Contains(variant, "2015") {
				edition = "J15"
			} else {
				edition = "G10"
			}
		}
	case "Fling":
		if variant == "WPN Promo" {
			edition = "PWP10"
		}
	case "Sylvan Ranger":
		if variant == "WPN Promo" {
			edition = "PWP10"
		}
	case "Bolas's Citadel",
		"Ghalta, Primal Hunger",
		"Glorybringer",
		"Lu Bu, Master-at-Arms":
		variant = strings.Replace(variant, "1", "Promo", 1)
		variant = strings.Replace(variant, "2", "Promo", 1)
	case "Cryptic Command":
		if variant == "Qualifier" {
			edition = "Pro Tour Promos"
		}
	case "Death Baron":
		if edition == "Convention 2018" {
			edition = "PM19"
		}
		variant = strings.Replace(variant, "1", "Promo", 1)
		variant = strings.Replace(variant, "2", "Promo", 1)
	default:
		if strings.HasSuffix(cardName, "of 2") {
			cardName = strings.TrimSuffix(cardName, " 1 of 2")
			cardName = strings.TrimSuffix(cardName, " 2 of 2")
		}
	}

	switch edition {
	case "ALT - Ikoria: Lair of Behemoths":
		// Decouple showcase and boderless from this tag
		if strings.Contains(variant, "BORDERLESS") {
			set, err := mtgmatcher.GetSet("IKO")
			if err != nil {
				return nil, err
			}
			for _, card := range set.Cards {
				if card.Name == cardName {
					if card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
						variant = "Showcase"
						break
					}
					if card.BorderColor == mtgjson.BorderColorBorderless {
						variant = "Borderless"
						break
					}
				}
			}
		}
	case "Commander":
		edition = "MagicFest 2019"
	case "Magic: The Gathering-Commander":
		edition = "Commander 2011"
	case "Antiquities",
		"Alliances",
		"Champions of Kamigawa",
		"Chronicles",
		"Deckmasters",
		"Fallen Empires",
		"Homelands",
		"Unstable":
		for _, num := range mtgmatcher.VariantsTable[edition][cardName] {
			if (variant == "1" && strings.HasSuffix(num, "a")) ||
				(variant == "2" && strings.HasSuffix(num, "b")) ||
				(variant == "3" && strings.HasSuffix(num, "c")) ||
				(variant == "4" && strings.HasSuffix(num, "d")) ||
				(variant == "5" && strings.HasSuffix(num, "e")) ||
				(variant == "6" && strings.HasSuffix(num, "f")) {
				variant = num
				break
			}
		}
	case "Fourth Edition",
		"Fifth Edition",
		"Limited Edition Beta":
		number := mtgmatcher.ExtractNumber(variant)
		if number != "" {
			for key, num := range mtgmatcher.VariantsTable[edition][cardName] {
				if strings.Contains(key, number) {
					variant = num
					break
				}
			}
		}
	case "Commander Anthology 2018",
		"Unglued":
		variant = extra
	case "Planeshift":
		if variant == "2" {
			variant = "Alt Art"
		} else {
			variant = ""
		}
	case "Portal":
		if variant == "2" {
			variant = "No Reminder Text"
		} else if variant == "1" {
			variant = ""
		}
	case "Planeswalker Weekend":
		edition = "Promos"
		variant = ""
	case "Ravnica Weekend":
		variant = strings.Replace(variant, "#P", "Ravnica Weekend ", 1)
	case "Secret Lair Drop Sets":
		variant = strings.Replace(extra, "#P", "", 1)
	case "Standard Showdown":
		if strings.Contains(variant, "Rebecca Guay") || strings.Contains(variant, "2017") {
			variant = "Rebecca Guay Standard Showdown"
		} else if strings.Contains(variant, "Alayna Danner") || strings.Contains(variant, "2018") {
			variant = "Alayna Danner Standard Showdown"
		} else {
			variant = "Standard Series"
		}
	default:
		if strings.Contains(edition, "Guild Kit") {
			variant = edition
		}
	}

	if mtgmatcher.Contains(variant, "extended") {
		variant += " extended art"
	} else if mtgmatcher.Contains(variant, "full") {
		variant += " full art"
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
