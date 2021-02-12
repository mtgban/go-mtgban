package magiccorner

import (
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Ashiok's Forerummer":     "Ashiok's Forerunner",
	"Fire (Fire/Ice) ":        "Fire",
	"Fire (Fire/Ice)":         "Fire",
	"Fire/Ice":                "Fire",
	"Kingsbaile Skirmisher":   "Kinsbaile Skirmisher",
	"Rough (Rough/Tumble)":    "Rough",
	"Time Shrike":             "Tine Shrike",
	"Treasure Find":           "Treasured Find",
	"Wax/Wane":                "Wax",
	"Who,What,When,Where,Why": "Who",
	"Who/What/When/Where/Why": "Who",
	"Skull of Arm":            "Skull of Orm",
	"Ramirez Di Pietro":       "Ramirez DePietro",
	"Sir Shandlar di Eberyn":  "Sir Shandlar of Eberyn",
	"Rohgahh di Kher":         "Rohgahh of Kher Keep",
	"El-Ajjaj":                "El-Hajjâj",
	"Frankenstein's Monst":    "Frankenstein's Monster",
	"The Tabernacle at Pe":    "The Tabernacle at Pendrell Vale",
	"Call of the Death-Dw":    "Call of the Death-Dweller",
	"Hallowed Spiritkeepe":    "Hallowed Spiritkeeper",
	"Iroas, God of Victor":    "Iroas, God of Victory",

	"Sedris, the King Traitor": "Sedris, the Traitor King",
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
	"L'Oscurità":                 "The Dark",
	"La Guerra della Scintilla":  "War of the Spark",
	"Labirinto del Drago":        "Dragon's Maze",
	"Landa Tenebrosa":            "Shadowmoor",
	"Leggende":                   "Legends",
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
	"Reinassance":                "Rinascimento",
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
	edition := card.Set

	// Grab the image url and keep only the image name
	extra := strings.TrimSuffix(path.Base(card.Extra), path.Ext(card.Extra))

	// Skip any token or similar cards
	if mtgmatcher.IsToken(cardName) {
		return nil, errors.New("not single")
	}

	// Circle of Protection: Red in Revised EU FWB???
	if card.Variants[index].Id == 223958 ||
		// Excruciator RAV duplicate card
		card.Variants[index].Id == 108840 ||
		// Wrong English name for Chain Lightning
		card.OrigName == "Crepaccio" {
		return nil, errors.New("duplicate")
	}

	isFoil := card.Variants[index].Foil == "Foil"

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
	// Use the tag information if available
	case "Promo":
		switch variation {
		case "War of the Spark: Extras", "Version1", "Version 2", "V.2":
			variation = "Prerelease"
		case "Core 2020: Extras", "Version 1", "V.1":
			variation = "Promo Pack"
		case "Judge Gift Program", "Judge Promo":
			switch cardName {
			case "Demonic Tutor",
				"Vampiric Tutor",
				"Vindicate",
				"Wasteland":
				if len(extra) > 5 {
					variation = extra[1:5]
					edition = "Judge"
				}
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
	default:
		// All the boosterfun prelease/promopack after THB (ELD is under "Promo")
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
			} else if cardName == "Flusterstorm" {
				edition = "Modern Horizons"
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
		case "Core 2021: Extras":
			variation = map[string]string{
				// ugin
				"1bacda35-bb91-4537-a14d-846650fa85f6": "279",
				"a11e75a0-17f0-429a-a77a-268fe6257010": "285",
				// basri
				"ed3906ea-df06-4299-a305-32e6ef476507": "280",
				"ca854101-5a9b-455d-bf47-ad8f3d57afa7": "286",
				// teferi
				"2d1ff397-2445-459a-ae4e-7bf1cd48f490": "275",
				"a8bf5708-4222-46d2-b108-0ed4cf3c83c3": "276",
				"d4949a0b-e320-470a-ba54-07ac75b0053b": "277",
				"ba95c4fc-f0fc-4bfe-bef7-694c3d82a6c7": "281",
				"f802369f-bd9f-4c2f-835b-b0b9c1ea61a7": "290",
				"c0ba42c9-480c-4236-b217-01910e51b290": "291",
				"20a271b3-c3a9-47c6-bc8b-b580ede1968b": "292",
				"a8fda4c0-d576-40e5-8e26-fc212c09691f": "293",
				// lili
				"a2edac86-02d4-4201-9b72-2ec64f163a72": "282",
				"e8dd3fda-d778-4be6-abd4-6cf704e352ea": "297",
				// garruk
				"bd16e1a6-0ece-44d5-8221-25892178e927": "284",
				"c5e92f14-776d-4fa6-b872-1acadca1e372": "305",
			}[extra]
		case "Ikoria: Lair of Behemoths: Extras":
			if cardName == "Void Beckoner" {
				if extra == "1ca7065e-88c1-44bb-ac68-e6e1df9e0726" {
					variation = "373"
				} else if extra == "29ae88c2-1f9b-4515-9f2e-78e1dc5468b1" {
					variation = "373A"
				}
			}
		case "Zendikar Rising: Extras":
			if cardName == "Charix, the Raging Isle" {
				if extra == "6e23886b-8200-427d-99af-068a0133795d" {
					variation = "Bundle"
				} else {
					variation = "Extended Art"
				}
			}
		case "Commander Legends: Extras":
			set, err := mtgmatcher.GetSet("CMR")
			if err != nil {
				return nil, err
			}
			for _, card := range set.Cards {
				if card.Name == cardName {
					if card.HasFrameEffect("showcase") {
						variation = "Showcase"
						break
					}
					if card.BorderColor == "borderless" {
						variation = "Borderless"
						break
					}
				}
			}
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
		if cardName == "Vampiric Tutor" {
			if variant == "V1" {
				variant = "Judge 2000"
			} else if variant == "V2" {
				variant = "Judge 2018"
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
		if cardName == "Surgical Extraction" {
			edition = "New Phyrexia Promos"
		}
	case "Promos":
		if cardName == "Hangarback Walker" {
			edition = "Love your LGS"
		}
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
