package trollandtoad

import (
	"errors"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgdb"
)

var cardTable = map[string]string{
	"Brineborn Cutthorat":   "Brineborn Cutthroat",
	"Herald of Anafenze":    "Herald of Anafenza",
	"Shimmer of Possiblity": "Shimmer of Possibility",

	"Chandra, Fire of Kaladesh // Chandra The Roaring Flame": "Chandra, Fire of Kaladesh // Chandra, Roaring Flame",

	"Sarpadian Empires, Vol.": "Sarpadian Empires, Vol. VII",
	"Nalathni Dragon 1994":    "Nalathni Dragon",
	"Merfolk Mesmerist Promo": "Merfolk Mesmerist",
	"Japanese Shivan Dragon":  "Shivan Dragon",

	"Who/ What/ When/ Where/ Why":  "Who",
	"Our Market Research Shows...": "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

var tagsTable = []string{
	"Box Topper",
	"Buy-A-Box Promo",
	"DotP",
	"Game Day Promo",
	"IDW Promo",
	"Judge Promo",
	"MagicFest Textless Promo",
	"Media Promo",
	"Mystery Booster",
	"Planeswalker Deck Exclusive",
	"Planeswalker Deck",
	"Prerelease Promo",
	"Walmart Promo",
	"Welcome Deck 2019 Exclusive",
}

func preprocess(fullName, edition string) (*mtgdb.Card, error) {
	if edition == "Bulk" || fullName == "" {
		return nil, errors.New("bulk")
	}

	switch {
	case strings.Contains(fullName, "Token"),
		strings.Contains(fullName, "Oversize"),
		strings.Contains(fullName, "Checklist"),
		strings.Contains(fullName, "Splendid Genesis"),
		strings.Contains(fullName, "Blank Proxy Card"),
		strings.Contains(fullName, "Basic Land Set"),
		strings.Contains(fullName, "Hasbro Card Set"),
		strings.Contains(fullName, "Battle the Horde Challenge Deck"),
		strings.Contains(fullName, "Face the Hydra Challenge Deck"),
		strings.Contains(fullName, "Emblem"):
		return nil, errors.New("not single")
	case strings.Contains(edition, "Duel Decks") && strings.Contains(edition, "Japanese"),
		strings.Contains(fullName, "Spanish"),
		strings.Contains(fullName, "Portuguese"),
		strings.Contains(fullName, "Japanese Emrakul"),
		strings.Contains(fullName, "Italian"):
		return nil, errors.New("not english")
	case strings.Contains(fullName, "Bounty Agent") && strings.Contains(fullName, "Prerelease"):
		return nil, errors.New("doesn't exist")
	case strings.Contains(fullName, "Beast of Burden") && strings.Contains(fullName, "Misprint"),
		strings.Contains(fullName, "Etali, Primal Storm") && strings.Contains(fullName, "Media Promo"):
		return nil, errors.New("unsupported")
	case fullName == "Marit Lage - Foil 16/16":
		return nil, errors.New("token")
	}

	isFoil := (strings.Contains(strings.ToLower(fullName), " foil") && !strings.Contains(fullName, "Non ")) ||
		(strings.Contains(edition, " Foil") && !strings.Contains(edition, "Non "))

	if isFoil {
		fullName = strings.Replace(fullName, " - Foil", "", -1)
		// Some cards have the foil tag leaking to the card name
		fullName = strings.Replace(fullName, "- Foil", "", -1)
		fullName = strings.Replace(fullName, " Foil", "", -1)
	}

	// Sometimes the buylist specifies tags at the end of the card name,
	// but without parenthesis, so make sure they are present.
	for _, tag := range tagsTable {
		if strings.HasSuffix(fullName, tag) {
			fullName = strings.Replace(fullName, tag, "("+tag+")", 1)
			break
		}
	}

	fullName = strings.TrimPrefix(fullName, "Basic Land - ")

	// Every edition has "Singles", the foil ones have "Foil Singles"
	edition = strings.TrimSuffix(edition, " Singles")
	edition = strings.TrimSuffix(edition, " Foil")
	edition = strings.TrimSuffix(edition, " English")
	edition = strings.TrimPrefix(edition, "MTG ")
	edition = strings.TrimPrefix(edition, "Magic: The Gathering ")

	switch {
	case strings.Contains(fullName, "God - Pharaoh"):
		fullName = strings.Replace(fullName, "God - Pharaoh", "God-Pharaoh", 1)
	case strings.Contains(fullName, "B.F.M."):
		if strings.Contains(fullName, "Left") {
			fullName = "B.F.M. (28)"
		} else {
			fullName = "B.F.M. (29)"
		}
	case strings.HasPrefix(fullName, "Plains (Ozhov) 050/133"):
		fullName = "Plains 050/133 (Ozhov)"
	case strings.Contains(fullName, "Euro Land"), strings.Contains(fullName, "Apac"):
		fullName = strings.Replace(fullName, "1", "one", 1)
		fullName = strings.Replace(fullName, "2", "two", 1)
		fullName = strings.Replace(fullName, "3", "three", 1)
		if strings.Contains(fullName, "Apac") && mtgdb.IsBasicLand(fullName) {
			edition = "Asia Pacific Land Program"
		}
	}

	// Split in two, use the second part as variant
	s := strings.Split(fullName, " - ")
	cardName := s[0]
	variant := ""
	if len(s) > 1 {
		variant = strings.Join(s[1:], " ")
	}

	switch {
	case strings.Contains(edition, "Silver Stamped"):
		variant = "Promo Pack"
		// Due to the everloved Sorcerous Spyglass
		if !strings.Contains(edition, "Ixalan") &&
			!strings.Contains(edition, "Eldraine") {
			edition = "Promo Pack"
		}
	case edition == "Unstable":
		// This variants resides just outside this poorly formatted tag
		// Look for it, and only keep the interesting parts
		if strings.Contains(cardName, ")-") {
			f := ""
			for _, field := range strings.Fields(fullName) {
				if strings.Contains(field, ")-") {
					f = field
					break
				}
			}
			s := strings.Split(cardName, f)
			cardName = strings.TrimSpace(s[0])
			if len(s) > 1 {
				variant = strings.TrimSpace(s[1])
			}

			// utf8 is love
			if cardName == "Novellamental" {
				variant = strings.Replace(variant, "â€œ", "''", 1)
				variant = strings.Replace(variant, "...â€", "…''", 1)
			}
		}
	}

	// This need to be at the end, for FTV and Core Sets
	se := mtgdb.SplitVariants(edition)
	edition = se[0]

	if !strings.HasPrefix(cardName, "Erase (Not the Urza's Legacy One)") {
		vars := mtgdb.SplitVariants(cardName)
		cardName = vars[0]
		if len(vars) > 1 {
			variant = vars[1]
		}
	}

	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	cardName = strings.TrimSuffix(cardName, " -")
	cardName = strings.TrimSuffix(cardName, "-")

	fields := strings.Fields(cardName)
	if len(fields) < 1 {
		return nil, errors.New("invalid card name")
	}
	last := ""
	if len(fields) > 1 {
		last = fields[len(fields)-1]
	}
	if strings.Contains(last, "/") {
		if !mtgdb.IsBasicLand(cardName) || (mtgdb.IsBasicLand(cardName) && edition == "Promo Cards") {
			// Some cards have their number appended at the very end, strip it out
			cardName = strings.Join(fields[:len(fields)-1], " ")
		}
	} else if len(last) == 3 && last == strings.ToUpper(last) && !unicode.IsDigit(rune(last[0])) && !strings.HasPrefix(edition, "Un") {
		// Some cards are tagged as "CODE Prerelease Promo", strip the last part
		// unless it's a funny set, since there are the Look at Me cards
		cardName = strings.Join(fields[:len(fields)-1], " ")
	}

	cardName = strings.TrimSuffix(cardName, " -")
	cardName = strings.TrimSuffix(cardName, "-")
	cardName = strings.Replace(cardName, "|", "//", 1)

	if mtgdb.IsBasicLand(cardName) {
		fields := strings.Fields(cardName)
		if len(fields) > 1 {
			cardName = fields[0]
			if variant != "" {
				variant += " "
			}
			variant += strings.Join(fields[1:], " ")
		} else if edition != "Promo Cards" && last != "" {
			if variant != "" {
				variant += " "
			}
			variant += last
		}
	}

	switch edition {
	case "Starter Series", "Starter 2000":
		return nil, errors.New("alias 1999 and 2000")

	case "Aether Revolt":
		if variant == "Kaladesh Inventions" {
			edition = variant
		}
	case "Alliances",
		"Champions of Kamigawa",
		"Fallen Empires":
		for _, num := range mtgdb.VariantsTable[edition][cardName] {
			if (variant == "Ver. 1" && strings.HasSuffix(num, "a")) ||
				(variant == "Ver. 2" && strings.HasSuffix(num, "b")) ||
				(variant == "Ver. 3" && strings.HasSuffix(num, "c")) {
				variant = num
				break
			}
		}
	case "Anthologies",
		"Portal Second Age",
		"Portal",
		"Tempest",
		"Mirage",
		"Ice Age",
		"4th Edition",
		"5th Edition",
		"Revised":
		if mtgdb.IsBasicLand(cardName) {
			if edition == "Revised" {
				edition = "Revised Edition"
			} else if edition == "4th Edition" {
				edition = "Fourth Edition"
			} else if edition == "5th Edition" {
				edition = "Fifth Edition"
			}
			for key, num := range mtgdb.VariantsTable[edition][cardName] {
				if (variant == "1" && key == "a") ||
					(variant == "2" && key == "b") ||
					(variant == "3" && key == "c") ||
					(variant == "4" && key == "d") {
					variant = num
					break
				}
			}
		}
	case "Battle Royale":
		if mtgdb.IsBasicLand(cardName) {
			fields := strings.Fields(variant)
			if len(fields) > 1 {
				variant = fields[1]
			}
		}
	case "Secret Lair Drop Series":
		num := mtgdb.ExtractNumber(fullName)
		if num != "" {
			variant = num
			cardName = strings.Replace(cardName, " "+num, "", 1)
		}
	case "Commander Anthology Volume II",
		"Ravnica Allegiance",
		"Guilds of Ravnica":
		variant = last
	case "Duel Decks Anthology":
		for _, code := range strings.Fields(variant) {
			if len(code) == 3 {
				edition = code
				break
			}
		}
	case "Promo Cards":
		switch cardName {
		case "Feral Throwback":
			edition = "PPRE"
		case "Island":
			if variant == "Arena Ice Age Art 2001" {
				variant = "Arena 2001"
			} else if variant == "Arena Beta Art 2001" {
				variant = "Arena 2002"
			} else if variant == "Arena No Symbol 1999" {
				variant = "Arena 1999 misprint"
			}

		case "Goblin Warchief":
			if strings.Contains(fullName, "005/012") {
				variant = "FNM 2016"
			} else {
				variant = "FNM 2006"
			}
		case "Fling":
			if strings.Contains(variant, "Gateway") {
				variant = "WPN 2010"
			}
		case "Vampiric Tutor":
			if strings.Contains(variant, "Judge") && !strings.Contains(variant, "2018") {
				variant = "Judge 2000"
			}
		case "Demonic Tutor":
			if strings.Contains(variant, "Judge") && !strings.Contains(variant, "2020") {
				variant = "Judge 2008"
			}
		case "Elesh Norn, Grand Cenobite":
			if variant == "Phyrexian Language" {
				variant = "Judge"
			}
		case "Soltari Priest":
			if variant == "JSS" {
				variant = "Euro JSS Promo"
			}
		case "Fiery Temper":
			if variant == "Arena" {
				variant = "Gateway"
			} else if variant == "FNM Promo" {
				variant = "FNM"
			}

		case "Curse of Wizardry",
			"Kor Duelist",
			"Mind Control",
			"Pathrazer of Ulamog",
			"Reckless Wurm",
			"Rise from the Grave",
			"Syphon Mind",
			"Vampire Nighthawk":
			variant = "WPN"
		case "Boomerang",
			"Wood Elves",
			"Yixlid Jailer",
			"Zoetic Cavern",
			"Icatian Javelineers":
			variant = "Gateway"
		case "Lu Bu, Master-at-Arms":
			variant += " Prerelease"
		case "Goblin Mime", "Circle of Protection: Art", "Booster Tutor":
			variant = "Arena"
		case "Rukh Egg", "Budoka Pupil":
			variant = "Release"
		case "Underworld Dreams":
			variant = "2HG"
		case "Powder Keg", "Psychatog", "Hypnotic Specter":
			variant = "Rewards"
		case "Crystalline Sliver":
			variant = "FNM"
		case "Kamahl, Pit Fighter":
			variant = "15th Anniversary"
		default:
			if strings.Contains(fullName, "005") && strings.Contains(fullName, "GP") {
				edition = "G18"
			}

			if strings.Contains(variant, "Prerelease") {
				cs := strings.Fields(cardName)
				for i := range cs {
					if mtgdb.ExtractNumber(cs[i]) != "" {
						cs[i] = ""
					}
				}
				cardName = strings.Join(cs, " ")
				cardName = strings.Replace(cardName, "  ", " ", -1)
			} else if strings.Contains(variant, "Top 8") {
				variant = strings.Replace(variant, "Top 8", "", 1)
			}
		}
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	if strings.Contains(variant, "Sealed") {
		return nil, errors.New("sealed")
	}

	return &mtgdb.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
