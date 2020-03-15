package magiccorner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

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

// These don't have any variant, but this table only applies to Promo edition
var promo2setTable = map[string]string{
	"Flusterstorm": "Modern Horizons",
	"Astral Drift": "Modern Horizons Promos",
	"Negate":       "Core Set 2020 Promos",
}

var ddaVariants = map[string]map[string]string{
	"Flamewave Invoker": map[string]string{
		"DDA036": "36",
		"DDA140": "40",
	},
	"Harmonize": map[string]string{
		"DDA022": "22",
		"DDA321": "21",
	},
	"Corrupt": map[string]string{
		"DDA255": "55",
		"DDA357": "57",
	},
}

func (mc *Magiccorner) parseSet(c *mcCard) (setName string, setCheck mtgban.SetCheckFunc) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Set

	ed, found := mtgban.EditionTable[setName]
	if found {
		setName = ed
		return
	}

	set, found := mc.db[c.setCode]
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
		ed, found = promo2setTable[c.Name]
		if found {
			setName = ed
			return
		}
	}

	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	if strings.HasPrefix(c.extra, "p2019") {
		possibleCode := c.extra[5:8]
		if possibleCode != "M20" && possibleCode != "FNM" {
			set, found := mc.db["P"+strings.ToUpper(possibleCode)]
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
	case "Friday Night Magic":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Friday Night Magic")
		}
	case "Pre-Release Promo":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Prerelease Events" || (strings.HasSuffix(set.Name, "Promos") && set.Name != "Resale Promos")
		}
	case "Release Event":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Release Events" || strings.HasSuffix(set.Name, "Promos")
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
			// AKA: "Buy a Box", "Game Day Promo", "Intro Pack", "Open House Promo",
			//      "League Promo", "Convention Promo", "Store Championship"
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Promos")
			}
		}
	}

	return
}

func (mc *Magiccorner) parseNumber(c *mcCard, setName string) (cardName string, numberCheck mtgban.NumberCheckFunc) {
	cardName = c.Name
	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}
	cardName = variants[0]

	number := ""
	if c.Number != mcNumberNotAvailable {
		number = c.Number
	}

	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				// Reduce aliasing by making sure that only "<xxx> Promos"
				// (except Resale) have numbers with 'p' or 's' at the end.
				if (set.Name == "Resale Promos" || !strings.HasSuffix(set.Name, "Promos")) && (strings.HasSuffix(card.Number, "p") || strings.Contains(card.Number, "s")) {
					return false
				}
				return card.Number == number
			}
		}

		variants = mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		switch {
		// Only keep one of the split/ transform cards
		case strings.Contains(cardName, " // "):
			cn := strings.Split(cardName, " // ")
			cardName = cn[0]
		// Only keep one of the transform cards
		case strings.Contains(cardName, " / "):
			cn := strings.Split(cardName, " / ")
			cardName = cn[0]
		}
	}()

	no, found := mtgban.VariantsTable[setName][cardName][specifier]
	if found {
		number = no
		return
	}

	no, found = mtgban.VariantsTable[setName][cardName][c.extra]
	if found {
		number = no
		return
	}

	switch setName {
	case "Duel Decks Anthology":
		no, found = ddaVariants[cardName][c.extra]
		if found {
			number = no
		}
	case "Ravnica Weekend":
		cn := strings.Fields(c.Name)
		cardName = cn[0]
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
			s := strings.Fields(cardName)
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
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasSuffix(card.Number, "s")
				}
			} else {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasSuffix(card.Number, "p")
				}
			}
		}
	case "Core Set 2020 Promos":
		if specifier == "Version 1" {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return strings.HasSuffix(card.Number, "p")
			}
		} else if specifier == "Version 2" {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return strings.HasSuffix(card.Number, "s")
			}
		}
	default:
		if specifier == "Pre-Release Promo" {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				if strings.HasSuffix(set.Name, "Promos") {
					setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
					if setDate.After(mtgban.NewPrereleaseDate) {
						return strings.Contains(card.Number, "s")
					} else {
						return !strings.Contains(card.Number, "s")
					}
				}

				if number != "" {
					return card.Number == number
				} else {
					return !strings.Contains(card.Number, "s")
				}
				return true
			}
		} else if strings.Contains("Core 2020: Extras", c.orig) {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return strings.HasSuffix(card.Number, "p")
			}
		} else if specifier == "Resale Promo" {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return !strings.HasSuffix(card.Number, "p") && !strings.Contains(card.Number, "s")
			}
		}
	}

	return
}
