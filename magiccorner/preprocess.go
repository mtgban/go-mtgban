package magiccorner

import (
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Fire (Fire/Ice) ":        "Fire // Ice",
	"Fire (Fire/Ice)":         "Fire // Ice",
	"Fire/Ice":                "Fire // Ice",
	"Kingsbaile Skirmisher":   "Kinsbaile Skirmisher",
	"Rough (Rough/Tumble)":    "Rough",
	"Time Shrike":             "Tine Shrike",
	"Treasure Find":           "Treasured Find",
	"Wax/Wane":                "Wax // Wane",
	"Who,What,When,Where,Why": "Who",
	"Who/What/When/Where/Why": "Who",
	"Ramirez Di Pietro":       "Ramirez DePietro",
	"Sir Shandlar di Eberyn":  "Sir Shandlar of Eberyn",
	"Rohgahh di Kher":         "Rohgahh of Kher Keep",
	"El-Ajjaj":                "El-Hajjâj",
	"Immagina Fantasma":       "Phantasmal Image",

	"Valentin, Dean of the Vein // Lisette, Dean of the": "Valentin, Dean of the Vein",
	"Mourning Patrol // Mourning Apparition":             "Mourning Patrol",
	"Growing Rites of Itlimoc / Itlimoc, Cradle of the":  "Growing Rites of Itlimoc",

	"Insegnamenti Mistici": "Mystical Teachings",
	"Massa Chimerica ":     "Chimeric Mass",
	"Condannare":           "Condemn",
	"Novelle":              "Tidings",
	"Torpore":              "Stupor",
	"Tattica del Cenn":     "Cenn's Tactician",
	"Aeronaut":             "Helionaut",

	"Lullamage's Familiar":    "Lullmage's Familiar",
	"Sword of Heart and Home": "Sword of Hearth and Home",
	"Allosaurus Sheperd":      "Allosaurus Shepherd",
	"Brazen Bucaneers":        "Brazen Buccaneers",
}

var editionTable = map[string]string{
	"Apocalisse":                 "Apocalypse",
	"Ascesa Oscura":              "Dark Ascension",
	"Ascesa degli Eldrazi":       "Rise of the Eldrazi",
	"Assalto":                    "Onslaught",
	"Aurora":                     "Morningtide",
	"Battaglia per Zendikar":     "Battle for Zendikar",
	"Campioni di Kamigawa":       "Champions of Kamigawa",
	"Caos Dimensionale":          "Planar Chaos",
	"Cavalcavento":               "Weatherlight",
	"Cicatrici di Mirrodin":      "Scars of Mirrodin",
	"Commander Arsenal":          "Commander's Arsenal",
	"Congiunzione":               "Planeshift",
	"Decima Edizione":            "Tenth Edition",
	"Destino di Urza":            "Urza's Destiny",
	"Discordia":                  "Dissension",
	"Draghi di Tarkir":           "Dragons of Tarkir",
	"Era Glaciale":               "Ice Age",
	"Eredità di Urza":            "Urza's Legacy",
	"Esodo":                      "Exodus",
	"Figli degli Dei":            "Born of the Gods",
	"Flagello":                   "Scourge",
	"Fortezza":                   "Stronghold",
	"Frammenti di Alara":         "Shards of Alara",
	"Giuramento dei Guardiani":   "Oath of the Gatewatch",
	"I Khan di Tarkir":           "Khans of Tarkir",
	"Il Patto delle Gilde":       "Guildpact",
	"Invasione":                  "Invasion",
	"Irruzione":                  "Gatecrash",
	"L'Era della Rovina":         "Hour of Devastation",
	"L'Oscurità":                 "The Dark Italian",
	"La Guerra della Scintilla":  "War of the Spark",
	"Labirinto del Drago":        "Dragon's Maze",
	"Landa Tenebrosa":            "Shadowmoor",
	"Leggende":                   "Legends Italian",
	"Legioni":                    "Legions",
	"Liberatori di Kamigawa":     "Saviors of Kamigawa",
	"Luna Spettrale":             "Eldritch Moon",
	"Maschere di Mercadia":       "Mercadian Masques",
	"Mirrodin Assediato":         "Mirrodin Besieged",
	"Nona Edizione":              "Ninth Edition",
	"Nuova Phyrexia":             "New Phyrexia",
	"Odissea":                    "Odyssey",
	"Ombre su Innistrad":         "Shadows over Innistrad",
	"Ondata Glaciale":            "Coldsnap",
	"Orizzonti di Modern":        "Modern Horizons",
	"Ottava Edizione":            "Eighth Edition",
	"Profezia":                   "Prophecy",
	"Quarta Edizione":            "Fourth Edition",
	"Quinta Alba":                "Fifth Dawn",
	"Quinta Edizione":            "Fifth Edition",
	"Ravnica: Città delle Gilde": "Ravnica: City of Guilds",
	"Revised EU FBB":             "Foreign Black Border",
	"Revised EU FWB":             "Foreign White Border",
	"Riforgiare il Destino":      "Fate Reforged",
	"Rinascita di Alara":         "Alara Reborn",
	"Ritorno a Ravnica":          "Return to Ravnica",
	"Ritorno di Avacyn":          "Avacyn Restored",
	"Rivali di Ixalan":           "Rivals of Ixalan",
	"Rivolta dell'Etere":         "Aether Revolt",
	"Saga di Urza":               "Urza's Saga",
	"Sentenza":                   "Judgment",
	"Sesta Edizione":             "Classic Sixth Edition",
	"Settima Edizione":           "Seventh Edition",
	"Spirale Temporale":          "Time Spiral",
	"Tempesta":                   "Tempest",
	"Theros: Oltre la Morte":     "Theros Beyond Death",
	"Tormento":                   "Torment",
	"Traditori di Kamigawa":      "Betrayers of Kamigawa",
	"Trono di Eldraine":          "Throne of Eldraine",
	"Vespro":                     "Eventide",
	"Viaggio Verso Nyx":          "Journey into Nyx",
	"Visione Futura":             "Future Sight",
	"Visioni":                    "Visions",

	"Duel Deck: Ajani Vs Bolas":        "Duel Decks: Ajani vs. Nicol Bolas",
	"Duel Deck: Cavalieri vs Draghi":   "Duel Decks: Knights vs. Dragons",
	"Duel Deck: Elfi Vs Goblin":        "Duel Decks: Elves vs. Goblins",
	"Duel Deck: Elspeth Vs Tezzereth":  "Duel Decks: Elspeth vs. Tezzeret",
	"Duel Decks: Cavalieri vs. Draghi": "Duel Decks: Knights vs. Dragons",

	"Eight Edition":  "Eighth Edition",
	"Fifth Ediiton":  "Fifth Edition",
	"Fifth Editon":   "Fifth Edition",
	"Journey to Nyx": "Journey into Nyx",
}

func preprocess(card *MCCard, index int) (*mtgmatcher.Card, error) {
	cardName := card.Name
	edition := card.Edition

	if strings.Contains(card.Extra, "pokemon") {
		return nil, errors.New("pokemon")
	}

	// Grab the image url and keep only the image name
	extra := strings.TrimSuffix(path.Base(card.Extra), path.Ext(card.Extra))

	// Circle of Protection: Red in Revised EU FWB???
	if card.Variants[index].Id == 223958 ||
		// Excruciator RAV duplicate card
		card.Variants[index].Id == 108840 ||
		// Wrong English name for Chain Lightning
		card.OrigName == "Crepaccio" {
		return nil, errors.New("duplicate")
	}

	isFoil := card.Variants[index].Foil == "Foil"

	// Some cards do not have an English name field
	if cardName == "" {
		cardName = card.OrigName
	}

	cn, found := cardTable[cardName]
	if found {
		cardName = cn
	}

	variation := ""
	variants := mtgmatcher.SplitVariants(cardName)
	if len(variants) > 1 {
		variation = variants[1]
	}
	cardName = variants[0]
	if variation == "" {
		variants = mtgmatcher.SplitVariants(card.OrigName)
		if len(variants) > 1 {
			variation = variants[1]
		}
	}

	// Run the lookup to catch any variant
	cn, found = cardTable[cardName]
	if found {
		cardName = cn
	}

	switch edition {
	case "Unlimited":
		cardName = mtgmatcher.Cut(cardName, "Unlimited")[0]
	case "Arabian Nights":
		if variation == "V.1" {
			variation = "dark"
		} else if variation == "V.2" {
			variation = "light"
		}
	case "Antiquities":
		if variation != "" {
			variation = extra
		}
	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		variation = "Japanese"
		edition = "War of the Spark"
		if variation == "Version 2" {
			variation = "Japanese Prerelease"
			edition = "War of the Spark Promos"
		}
	case "Core 2020: Extras":
		edition = "PM20"
		if cardName == "Chandra's Regulator" {
			if variation == "1" {
				variation = "131"
			} else if variation == "V.2" {
				variation = "Promo Pack"
			} else if variation == "V.3" {
				variation = "Prerelease"
			}
		} else {
			switch variation {
			case "V.1":
				variation = "Promo Pack"
			case "V.2":
				variation = "Prerelease"
			default:
				if mtgmatcher.HasPromoPackPrinting(cardName) {
					variation = "Promo Pack 2020"
					edition = "Promos"
					if cardName == "Sorcerous Spyglass" {
						edition = "PXLN"
					}
				}
			}
		}
	case "Throne of Eldraine: Promos":
		switch variation {
		case "V.1":
			variation = "Prerelease"
		case "V.2":
			variation = "Promo Pack"
		default:
			if mtgmatcher.HasPromoPackPrinting(cardName) {
				variation = "Promo Pack ELD"
				edition = "Promos"
			}
		}
	case "Kaldheim: Extras":
		if cardName == "Vorinclex, Monstrous Raider" {
			if variation == "V.1" {
				variation = "Showcase"
			} else if variation == "V.2" {
				variation = "Phyrexian"
			}
		} else {
			if variation == "V.1" {
				variation = "Borderless"
			} else if variation == "V.2" {
				variation = "Showcase"
			}
		}
	case "Modern Horizons 2: Extras":
		// Note: order of these printing checks matters
		if mtgmatcher.HasExtendedArtPrinting(cardName) {
			switch variation {
			case "V.1":
				variation = "Retro Frame"
			case "V.2":
				variation = "Retro Frame Foil Etched"
			case "V.3":
				variation = "Extended Art"
			}
		} else if mtgmatcher.HasBorderlessPrinting(cardName) {
			switch variation {
			case "V.1":
				variation = "Borderless"
			case "V.2":
				variation = "Retro Frame"
				if mtgmatcher.HasShowcasePrinting(cardName) {
					variation = "Showcase"
				}
			case "V.3":
				variation = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasShowcasePrinting(cardName) {
			switch variation {
			case "V.1":
				variation = "Showcase"
			case "V.2":
				variation = "Retro Frame"
			case "V.3":
				variation = "Retro Frame Foil Etched"
			}
		} else if mtgmatcher.HasRetroFramePrinting(cardName) {
			switch variation {
			case "V.1":
				variation = "Retro Frame"
			case "V.2":
				variation = "Retro Frame Foil Etched"
			}
		}
	// Use the tag information if available
	case "Promo":
		switch variation {
		case "War of the Spark: Extras", "Version1", "Version 2", "V.2":
			variation = "Prerelease"
		case "Core 2020: Extras", "Version 1", "V.1":
			variation = "Promo Pack"
		case "Judge Gift Program", "Judge Promo":
			switch cardName {
			case "Demonic Tutor":
				variation = "Judge 2020"
			case "Vampiric Tutor":
				variation = "Judge 2018"
			case "Vindicate",
				"Wasteland":
				if len(extra) > 5 {
					variation = extra[1:5]
					edition = "Judge"
				}
			}
		default:
			switch cardName {
			case "Sword of Dungeons & Dragons":
				edition = "H17"
			}
		}
	// Use the number from extra if present, or keep the current version
	case "Unstable":
		if strings.HasPrefix(variation, "Version") && strings.HasPrefix(extra, "UST") {
			variation = strings.TrimLeft(extra[3:], "0")
		}
	case "Unsanctioned":
		if len(extra) > 3 {
			variation = strings.TrimLeft(extra[3:], "0")
		}
	// Handle wastes
	case "Oath of the Gatewatch":
		if cardName == "Wastes" {
			variation = extra
		}
	// DDA correct edition is hidden in the extra
	case "Duel Decks Anthology":
		switch {
		case strings.HasPrefix(extra, "DDA0"):
			edition = "Duel Decks Anthology: Elves vs. Goblins"
		case strings.HasPrefix(extra, "DDA1"):
			edition = "Duel Decks Anthology: Jace vs. Chandra"
		case strings.HasPrefix(extra, "DDA2"):
			edition = "Duel Decks Anthology: Divine vs. Demonic"
		case strings.HasPrefix(extra, "DDA3"):
			edition = "Duel Decks Anthology: Garruk vs. Liliana"
		}
		variation = strings.TrimLeft(extra[4:], "0")
	case "Commander Legends: Extras":
		if mtgmatcher.HasEtchedPrinting(cardName, "CMR") {
			variation = "etched"
		} else if mtgmatcher.HasExtendedArtPrinting(cardName, "CMR") {
			variation = "extended art"
		}
	case "Secret Lair Drop Series":
		switch cardName {
		case "Serum Visions", "Faerie Rogue Token":
			if strings.HasPrefix(extra, "SLD") {
				variation = strings.TrimLeft(extra[3:], "0")
			}
		}
	default:
		// All the prelease/promopack versions >= THB
		if strings.HasSuffix(edition, ": Promos") {
			switch variation {
			case "V.1":
				variation = "Promo Pack"
			case "V.2":
				variation = "Prerelease"
			case "V.3":
				variation = "Bundle"
				edition = strings.TrimSuffix(edition, ": Promos")
			default:
				if mtgmatcher.HasPromoPackPrinting(cardName) {
					variation = "Promo Pack"
				}
			}
		}
	}

	id := ""
	if variation == "" {
		switch edition {
		// Work around missing tags until (if) they add them
		case "Promo":
			if strings.HasPrefix(extra, "p2019ELD") {
				internalNumber := strings.TrimLeft(extra[8:], "0")
				num, err := strconv.Atoi(internalNumber)
				if err == nil {
					if num < 69 {
						variation = "Prerelease"
					} else {
						variation = "Promo Pack"
					}
				}
			} else {
				switch cardName {
				case "Flusterstorm":
					edition = "Modern Horizons"
				case "Gideon Blackblade":
					edition = "SLD"
				case "Angelic Guardian":
					edition = "G18"
				case "Wrath of God":
					edition = "2XM"
					variation = "383"
				}
			}
		// These editions contain numbers that can be used safely
		case "Throne of Eldraine: Extras", "Theros Beyond Death: Extras",
			"Guilds of Ravnica", "Ravnica Allegiance", "Unglued":
			variation = extra
			if len(extra) > 3 {
				internalNumber := strings.TrimLeft(extra[3:], "0")
				_, err := strconv.Atoi(internalNumber)
				if err == nil {
					variation = internalNumber
				}
			}
		// These are the editions that need table lookup
		case "Antiquities", "Fallen Empires", "Chronicles",
			"Alliances", "Reinassance", "Rinascimento", "Homelands":
			variation = extra
		// Same for this one, except the specifier is elsewhere
		case "Commander Anthology 2018":
			variation = card.URL
		case "Core 2020: Extras":
			variation = "Promo Pack"
		case "Core 2021: Extras",
			"Ikoria: Lair of Behemoths: Extras",
			"Zendikar Rising: Extras":
			id = extra
		// Full-art Zendikar lands
		case "Zendikar":
			if mtgmatcher.IsBasicLand(cardName) {
				s := strings.Fields(cardName)
				if len(s) > 1 {
					cardName = s[0]
					variation = s[1] + "a"
				}
			}
		default:
			// Try using the number, except the set code can randomly be
			// 2 or 3 characters.
			if mtgmatcher.IsBasicLand(cardName) {
				for _, lengthToDrop := range []int{2, 3} {
					if len(extra) > lengthToDrop {
						internalNumber := strings.TrimLeft(extra[lengthToDrop:], "0")
						val, err := strconv.Atoi(internalNumber)
						if err == nil && val < 400 {
							variation = internalNumber
							break
						}
					}
				}
			}
		}
	}

	lutName, found := editionTable[edition]
	if found {
		edition = lutName
	}

	return &mtgmatcher.Card{
		Id:        id,
		Name:      cardName,
		Variation: variation,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}

func preprocessBL(cardName, edition string) (*mtgmatcher.Card, error) {
	if strings.HasSuffix(cardName, " HP") {
		return nil, errors.New("duplicate")
	}
	if strings.HasSuffix(cardName, " NM") {
		cardName = strings.TrimSuffix(cardName, " NM")
	}

	variant := ""
	if strings.Contains(edition, "(") {
		vars := mtgmatcher.SplitVariants(edition)
		edition = vars[0]
		if len(vars) > 1 {
			variant = vars[1]
		}
	}

	switch edition {
	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		edition = "War of the Spark"
		variant = "Japanese"
	case "DCI Promos":
		switch cardName {
		case "Abrupt Decay", "Inkmoth Nexus", "Vengevine", "Thalia, Guardian of Thraben":
			edition = "World Magic Cup Qualifiers"
		}
	case "Judge Rewards":
		switch cardName {
		case "Vampiric Tutor":
			if variant == "V1" {
				variant = "Judge 2000"
			} else if variant == "V2" {
				variant = "Judge 2018"
			}
		case "Vindicate":
			if variant == "V1" {
				variant = "Judge 2007"
			} else if variant == "V2" {
				variant = "Judge 2013"
			}
		}
	case "Judge Rewards Promos":
		if cardName == "Demonic Tutor" {
			if variant == "V1" {
				variant = "Judge 2008"
			} else if variant == "V2" {
				variant = "Judge 2020"
			}
		}
	case "Buy a Box Promos":
		switch cardName {
		case "Surgical Extraction":
			edition = "New Phyrexia Promos"
		case "Chord of Calling":
			edition = "2XM"
		}
	case "Promos":
		if cardName == "Hangarback Walker" {
			edition = "Love your LGS"
		}
	case "Modern Horizons II: Old Frame":
		edition = "Modern Horizons 2"
		variant += " Retro Frame"
	}

	cn, found := cardTable[cardName]
	if found {
		cardName = cn
	}

	lutName, found := editionTable[edition]
	if found {
		edition = lutName
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
