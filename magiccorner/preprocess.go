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
}

var editionTable = map[string]string{
	"Alleanze":                   "Alliances",
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
	"Fedeltà di Ravnica":         "Ravnica Allegiance",
	"Figli degli Dei":            "Born of the Gods",
	"Flagello":                   "Scourge",
	"Fortezza":                   "Stronghold",
	"Frammenti di Alara":         "Shards of Alara",
	"Gilde di Ravnica":           "Guilds of Ravnica",
	"Giuramento dei Guardiani":   "Oath of the Gatewatch",
	"I Khan di Tarkir":           "Khans of Tarkir",
	"Il Patto delle Gilde":       "Guildpact",
	"Invasione":                  "Invasion",
	"Irruzione":                  "Gatecrash",
	"L'Era della Rovina":         "Hour of Devastation",
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
	"Origini":                    "Homelands",
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
	"Duel Deck: Elfi vs Goblin":        "Duel Decks: Elves vs. Goblins",
	"Duel Deck: Elspeth vs Tezzereth":  "Duel Decks: Elspeth vs. Tezzeret",
	"Duel Decks: Cavalieri vs. Draghi": "Duel Decks: Knights vs. Dragons",
}

func preprocess(card *MCCard, index int) (*mtgmatcher.Card, error) {
	cardName := card.Name
	edition := card.Set

	// Grab the image url and keep only the image name
	extra := strings.TrimSuffix(path.Base(card.Extra), path.Ext(card.Extra))

	// Skip any token or similar cards
	if strings.Contains(cardName, "Token") ||
		strings.Contains(cardName, "token") ||
		strings.Contains(cardName, "Art Series") ||
		strings.Contains(cardName, "Checklist") ||
		strings.Contains(cardName, "Check List") ||
		strings.Contains(cardName, "Check-List") ||
		strings.Contains(cardName, "Emblem") ||
		cardName == "Punch Card" ||
		cardName == "The Monarch" ||
		cardName == "Spirit" ||
		cardName == "City's Blessing" {
		return nil, errors.New("non-mtg")
	}

	// Circle of Protection: Red in Revised EU FWB???
	if card.Variants[index].Id == 223958 ||
		// Excruciator RAV duplicate card
		card.Variants[index].Id == 108840 {
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
	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		if variation == "Version 2" {
			variation = "Japanese Prerelease"
			edition = "War of the Spark Promos"
		}
	// Use the tag information if available
	case "Promo":
		switch variation {
		case "War of the Spark: Extras", "Version1", "Version 2":
			variation = "Prerelease"
		case "Core 2020: Extras", "Version 1":
			variation = "Promo Pack"
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
	case "Giuramento dei Guardiani":
		if cardName == "Wastes" && len(extra) > 3 {
			variation = extra[3:]
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
	}

	id := ""
	if variation == "" {
		switch edition {
		// Work around missing tags until (if) they add them
		case "Promo":
			if strings.HasPrefix(extra, "p2019ELD") || strings.HasPrefix(extra, "p2020THB") {
				internalNumber := strings.TrimLeft(extra[8:], "0")
				num, err := strconv.Atoi(internalNumber)
				if err == nil {
					if num < 69 {
						variation = "Prerelease"
					} else {
						variation = "Promo Pack"
					}
				}
			}
		// These editions contain numbers that can be used safely
		case "Throne of Eldraine: Extras", "Theros Beyond Death: Extras",
			"Gilde di Ravnica", "Fedeltà di Ravnica", "Unglued":
			variation = extra
			if len(extra) > 3 {
				internalNumber := strings.TrimLeft(extra[3:], "0")
				_, err := strconv.Atoi(internalNumber)
				if err == nil {
					variation = internalNumber
				}
			}
		// Image is scryfall ID
		case "Ikoria: Lair of Behemoths":
			id = extra
		// Only one of them
		case "Campioni di Kamigawa":
			if cardName == "Brothers Yamazaki" {
				variation = "Facing Left"
			}
		// These are the editions that need table lookup
		case "Antiquities", "Fallen Empires", "Chronicles",
			"Alleanze", "Rinascimento", "Origini":
			variation = extra
		// Same for this one, except the specifier is elsewhere
		case "Commander Anthology 2018":
			variation = card.URL
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
