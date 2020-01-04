package magiccorner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgjson"
)

var setTable = map[string]string{
	"Alleanze":                   "Alliances",
	"Alpha":                      "Limited Edition Alpha",
	"Apocalisse":                 "Apocalypse",
	"Ascesa Oscura":              "Dark Ascension",
	"Ascesa degli Eldrazi":       "Rise of the Eldrazi",
	"Assalto":                    "Onslaught",
	"Aurora":                     "Morningtide",
	"Beta":                       "Limited Edition Beta",
	"Campioni di Kamigawa":       "Champions of Kamigawa",
	"Caos Dimensionale":          "Planar Chaos",
	"Cavalcavento":               "Weatherlight",
	"Cicatrici di Mirrodin":      "Scars of Mirrodin",
	"Commander Arsenal":          "Commander's Arsenal",
	"Commander":                  "Commander 2011",
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
	"I Khan di Tarkir":           "Khans of Tarkir",
	"Il Patto delle Gilde":       "Guildpact",
	"Invasione":                  "Invasion",
	"Irruzione":                  "Gatecrash",
	"Labirinto del Drago":        "Dragon's Maze",
	"Landa Tenebrosa":            "Shadowmoor",
	"Leggende":                   "Legends",
	"Legioni":                    "Legions",
	"Liberatori di Kamigawa":     "Saviors of Kamigawa",
	"Maschere di Mercadia":       "Mercadian Masques",
	"Mirrodin Assediato":         "Mirrodin Besieged",
	"Modern Event Deck":          "Modern Event Deck 2014",
	"Nona Edizione":              "Ninth Edition",
	"Nuova Phyrexia":             "New Phyrexia",
	"Odissea":                    "Odyssey",
	"Ondata Glaciale":            "Coldsnap",
	"Origini":                    "Homelands",
	"Ottava Edizione":            "Eighth Edition",
	"Profezia":                   "Prophecy",
	"Quarta Edizione":            "Fourth Edition",
	"Quinta Alba":                "Fifth Dawn",
	"Quinta Edizione":            "Fifth Edition",
	"Ravnica: Città delle Gilde": "Ravnica: City of Guilds",
	"Revised":                    "Revised Edition",
	"Revised EU FBB":             "Foreign Black Border",
	"Revised EU FWB":             "Foreign White Border",
	"Riforgiare il Destino":      "Fate Reforged",
	"Rinascita di Alara":         "Alara Reborn",
	"Ritorno a Ravnica":          "Return to Ravnica",
	"Ritorno di Avacyn":          "Avacyn Restored",
	"Saga di Urza":               "Urza's Saga",
	"Sentenza":                   "Judgment",
	"Sesta Edizione":             "Classic Sixth Edition",
	"Settima Edizione":           "Seventh Edition",
	"Spirale Temporale":          "Time Spiral",
	"Tempesta":                   "Tempest",
	"Timeshifted":                "Time Spiral Timeshifted",
	"Tormento":                   "Torment",
	"Traditori di Kamigawa":      "Betrayers of Kamigawa",
	"Unlimited":                  "Unlimited Edition",
	"Vespro":                     "Eventide",
	"Viaggio Verso Nyx":          "Journey into Nyx",
	"Visione Futura":             "Future Sight",
	"Visioni":                    "Visions",

	"Collector's Edition":               "Collectors’ Edition",
	"International Collector's Edition": "Intl. Collectors’ Edition",
	"Mythic Edition Gilde di Ravnica":   "Mythic Edition",
	"Mythic Edition Fedeltà di Ravnica": "Mythic Edition",
	"Ravnica Allegiance: Guild Kits":    "RNA Guild Kit",
	"Guilds of Ravnica: Guild Kits":     "GRN Guild Kit",

	"Duel Deck: Elfi vs Goblin":       "Duel Decks: Elves vs. Goblins",
	"Duel Deck: Cavalieri vs Draghi":  "Duel Decks: Knights vs. Dragons",
	"Duel Deck: Ajani Vs Bolas":       "Duel Decks: Ajani vs. Nicol Bolas",
	"Duel Deck: Elspeth vs Tezzereth": "Duel Decks: Elspeth vs. Tezzeret",

	"Masterpiece Series: Amonkhet Invocations": "Amonkhet Invocations",
	"Masterpiece Series: Kaladesh Inventions":  "Kaladesh Inventions",
	"Masterpiece Series: Zendikar Inventions":  "Zendikar Expeditions",
}

var promosetTable = map[string]string{
	"Book Promo":           "HarperPrism Book Promos",
	"Resale Promo":         "Resale Promos",
	"Gran Prix Promo":      "Grand Prix Promos",
	"Regional PTQ Promo":   "Pro Tour Promos",
	"Pro Tour Promo":       "Pro Tour Promos",
	"Ugin’s Fate Promo":    "Ugin's Fate",
	"URL Convention Promo": "URL/Convention Promos",
	"Junior Super Series":  "Junior Super Series",
	"Junior Series":        "Junior Series Europe",
	"Junior APAC Series":   "Junior APAC Series",

	"San Diego Comic-Con 2014": "San Diego Comic-Con 2014",
	"San Diego Comic-Con 2015": "San Diego Comic-Con 2015",
	"San Diego Comic-Con 2016": "San Diego Comic-Con 2016",
	"San Diego Comic-Con 2017": "San Diego Comic-Con 2017",
	"San Diego Comic-Con 2018": "San Diego Comic-Con 2018",
	"San Diego Comic-Con":      "San Diego Comic-Con 2019",
}

var card2setTable = map[string]string{
	"Glory (Pre-Release Promo) *Only in Hebrew*": "Prerelease Events",

	"Jace Beleren (Book Promo)": "Miscellaneous Book Promos",

	"Phyrexian Metamorph (Release Event)":    "New Phyrexia Promos",
	"Endbringer (Release Event)":             "Oath of the Gatewatch Promos",
	"Saheeli's Artistry (Release Event)":     "Kaladesh Promos",
	"Opt (Friday Night Magic)":               "Dominaria Promos",
	"Cast Down (Friday Night Magic)":         "Dominaria Promos",
	"Nexus of Fate (Buy a Box)":              "Core Set 2019",
	"Militia Bugler (Friday Night Magic)":    "Core Set 2019 Promos",
	"Murder (Friday Night Magic)":            "Core Set 2019 Promos",
	"Conclave Tribunal (Friday Night Magic)": "Guilds of Ravnica Promos",
	"Sinister Sabotage (Friday Night Magic)": "Guilds of Ravnica Promos",
	"The Haunt of Hightower (Buy a Box)":     "Ravnica Allegiance",
}

var promo2setTable = map[string]string{
	// These don't have any variant, but this table only applies to Promo edition
	"Flusterstorm": "Modern Horizons",
	"Astral Drift": "Modern Horizons Promos",
	"Negate":       "Core Set 2020 Promos",
}

var newPrereleaseDate = time.Date(2014, time.September, 1, 0, 0, 0, 0, time.UTC)

func (c *MCCard) parseNumber(origName, setName, specifier string) (cardName, number string, numberCheck func(mtgjson.Set, string) bool) {
	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, n string) bool {
				// Reduce aliasing by making sure that only "<xxx> Promos" (except
				// Resale) have numbers with 'p' or 's' at the end.
				if (set.Name == "Resale Promos" || !strings.HasSuffix(set.Name, "Promos")) && (strings.HasSuffix(n, "p") || strings.Contains(n, "s")) {
					return false
				}
				return n == number
			}
		}
	}()

	cardName = origName
	if c.Number != mcNumberNotAvailable {
		number = c.Number
	}

	// Look up card number
	no, found := setVariants[setName][cardName][specifier]
	if found {
		number = no
		return
	}

	no, found = setVariants[setName][cardName][c.extra]
	if found {
		number = no
		return
	}

	switch setName {
	case "Ultimate Box Topper":
		number = "U" + number
	case "Unstable":
		_, err := strconv.Atoi(number[:len(number)-1])
		if err != nil {
			number = ""
		}
	case "War of the Spark":
		number = strings.Replace(number, "b", mtgjson.SuffixSpecial, 1)
	case "Zendikar":
		if strings.HasPrefix(cardName, "Forest") ||
			strings.HasPrefix(cardName, "Island") ||
			strings.HasPrefix(cardName, "Mountain") ||
			strings.HasPrefix(cardName, "Plains") ||
			strings.HasPrefix(cardName, "Swamp") {
			s := strings.Split(cardName, " ")
			if len(s) > 1 {
				cardName = s[0]
				number = s[1] + "a"
			}
		}
	case "Throne of Eldraine Promos":
		internalNumber := strings.Replace(c.extra, "p2019ELD", "", 1)
		internalNumber = strings.TrimLeft(internalNumber, "0")
		num, err := strconv.Atoi(internalNumber)
		if err == nil {
			if num < 69 {
				numberCheck = func(set mtgjson.Set, n string) bool {
					return strings.HasSuffix(n, "s")
				}
			} else {
				numberCheck = func(set mtgjson.Set, n string) bool {
					return strings.HasSuffix(n, "p")
				}
			}
		}
	case "Core Set 2020 Promos":
		if specifier == "Version 1" {
			numberCheck = func(set mtgjson.Set, n string) bool {
				return strings.HasSuffix(n, "p")
			}
		} else if specifier == "Version 2" {
			numberCheck = func(set mtgjson.Set, n string) bool {
				return strings.HasSuffix(n, "s")
			}
		}
	default:
		if specifier == "Pre-Release Promo" {
			numberCheck = func(set mtgjson.Set, n string) bool {
				if strings.HasSuffix(set.Name, "Promos") {
					setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
					if setDate.After(newPrereleaseDate) {
						return strings.Contains(n, "s")
					}
				}
				if number != "" {
					return n == number
				}
				return true
			}
		} else if strings.Contains("Core 2020: Extras", c.orig) {
			numberCheck = func(set mtgjson.Set, n string) bool {
				return strings.HasSuffix(n, "p")
			}
		} else if specifier == "Resale Promo" {
			numberCheck = func(set mtgjson.Set, n string) bool {
				return !strings.HasSuffix(n, "p") && !strings.HasSuffix(n, "s")
			}
		}
	}

	return
}

func (c *MCCard) parseSet(db mtgjson.MTGDB, origName, specifier string) (cardName, setName string, setCheck func(set mtgjson.Set) bool) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	cardName = origName
	setName = c.Set

	ed, found := setTable[setName]
	if found {
		setName = ed
		return
	}

	set, found := db[c.setCode]
	if found {
		setName = set.Name
		return
	}

	switch {
	case strings.HasPrefix(setName, "Duel Deck:"):
		setName = strings.Replace(setName, "Deck: ", "Decks: ", 1)
		setName = strings.Replace(setName, " Vs. ", " vs. ", 1)
		setName = strings.Replace(setName, " vs ", " vs. ", 1)
		setName = strings.Replace(setName, " Vs ", " vs. ", 1)
		setName = strings.Replace(setName, " VS ", " vs. ", 1)
		setName = strings.Replace(setName, "The", "the", 1)
		return
	case strings.HasPrefix(setName, "Premium Deck"):
		setName = strings.Replace(setName, "Premium Deck: ", "Premium Deck Series: ", 1)
		return
	case strings.HasPrefix(setName, "From The Vault: "):
		setName = strings.Replace(setName, "The", "the", 1)
		return
	case setName == "Duel Decks Anthology":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duel Decks Anthology")
		}
		return
	}

	ed, found = card2setTable[c.Name]
	if found {
		setName = ed
		return
	}

	if setName == "Promo" {
		ed, found = promo2setTable[origName]
		if found {
			setName = ed
			return
		}
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	if strings.HasPrefix(c.extra, "p2019") {
		possibleCode := c.extra[5:8]
		if possibleCode != "M20" && possibleCode != "FNM" {
			set, found := db["P"+strings.ToUpper(possibleCode)]
			if found {
				setName = set.Name
				return
			}
			fmt.Println("Code not found", possibleCode, "from", c.extra)
		}
	}

	switch specifier {
	case "Version1":
		if strings.Contains(c.extra, "WAR") {
			setName = "War of the Spark Promos"
		}
	case "Version 1", "Version 2":
		if strings.Contains(c.extra, "M20") {
			setName = "Core Set 2020 Promos"
		}
	case "Ravnica Weekend":
		cn := strings.Split(origName, " ")
		cardName = cn[0]
	case "Friday Night Magic":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Friday Night Magic")
		}
	case "Pre-Release Promo":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Prerelease Events" || strings.HasSuffix(set.Name, "Promos")
		}
	case "Gateway Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Gateway") || strings.HasPrefix(set.Name, "Wizards Play Network")
		}
	case "Wizard Play Network":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Wizards Play Network")
		}
	case "Players Reward":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case "Clash Pack Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Clash Pack")
		}
	case "Release Event":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Release Events" || strings.HasSuffix(set.Name, "Promos")
		}
	case "Hero's Path":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Hero's Path")
		}
	case "Judge Gift Program":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Judge Gift Cards")
		}
	case "IDW Comics Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics")
		}
	case "Arena League":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Arena League")
		}
	case "Duels of the Planeswalkers":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
		}
	default:
		if setName == "Promo" {
			//case "Buy a Box", "Game Day Promo", "Intro Pack", "Open House Promo", "League Promo", "Convention Promo", "Store Championship":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Promos")
			}
		}
	}

	return
}
