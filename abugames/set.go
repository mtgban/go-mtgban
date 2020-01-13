package abugames

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgjson"
)

var promosetTable = map[string]string{
	"15th Anniversary":        "15th Anniversary Cards",
	"2HG":                     "Two-Headed Giant Tournament",
	"Armada Comics":           "Magazine Inserts",
	"Book":                    "HarperPrism Book Promos",
	"Dragon'Con 1994":         "Dragon Con",
	"Dragonfury":              "Tarkir Dragonfury",
	"HASCON 2017":             "HasCon 2017",
	"Redemption Original Art": "Wizards of the Coast Online Store",
	"Standard Series":         "BFZ Standard Series",

	"MagicFest":          "MagicFest 2019",
	"Judge Extended Art": "Judge Gift Cards 2014",
	"States 2008":        "Gateway 2007",
	"Stained Glass":      "Secret Lair Promos",

	"MCQ":        "Pro Tour Promos",
	"Pro Tour":   "Pro Tour Promos",
	"RPTQ":       "Pro Tour Promos",
	"WMCQ":       "World Magic Cup Qualifiers",
	"Grand Prix": "Grand Prix Promos",
	"Nationals":  "Nationals Promos",

	"European Junior Series": "Junior Series Europe",
	"JSS": "Junior Super Series",
	"MSS": "Junior Super Series",

	"Gift Pack 2017": "2017 Gift Pack",
	"Gift Pack 2018": "M19 Gift Pack",

	"Standard Showdown 2017": "XLN Standard Showdown",
	"Standard Showdown 2018": "M19 Standard Showdown",

	"Resale":          "Resale Promos",
	"Walmart Resale":  "Resale Promos",
	"Resale Walmart":  "Resale Promos",
	"Resale Walmart ": "Resale Promos",

	// Repeated (mostly) as-is to get picked as-is
	"Ugin's Fate":              "Ugin's Fate",
	"Ugins Fate":               "Ugin's Fate",
	"Guru":                     "Guru",
	"Magic 2015 Clash Deck":    "Magic 2015 Clash Pack",
	"Magic 2015 Clash Pack":    "Magic 2015 Clash Pack",
	"Fate Reforged Clash Pack": "Fate Reforged Clash Pack",
	"Magic Origins Clash Pack": "Magic Origins Clash Pack",
}

var card2setTable = map[string]string{
	"Forest (Arena 2002 Beta)": "Arena League 2001",
	"Skirk Marauder (FOIL)":    "Arena League 2003",

	"Elvish Aberration (FNM)": "Arena League 2003",
	"Elvish Lyrist (FNM)":     "Junior Super Series",
	"Psychatog (FNM)":         "Magic Player Rewards 2005",

	"Ajani Vengeant (Shards of Alara Release)":      "Prerelease Events",
	"Lu Bu, Master-at-Arms (Prerelease April 29th)": "Prerelease Events",
	"Lu Bu, Master-at-Arms (Prerelease July 4th)":   "Prerelease Events",

	"Lightning Hounds (TopDeck Magazine)": "Magazine Inserts",
	"Spined Wurm (Book TopDeck Magazine)": "Magazine Inserts",
	"Warmonger (Book)":                    "Magazine Inserts",

	"Fling (WPN #50)": "Wizards Play Network 2010",
	"Fling (WPN #69)": "Wizards Play Network 2011",

	"Counterspell (Arena)":    "DCI Legend Membership",
	"Incinerate (Arena)":      "DCI Legend Membership",
	"Balduvian Horde (Judge)": "Worlds",

	"Treetop Village (Gateway)": "Summer of Magic",
	"Faerie Conclave (Gateway)": "Summer of Magic",

	"Lightning Bolt (MagicFest Textless)": "MagicFest 2019",
	"Sol Ring (Commander)":                "MagicFest 2019",

	"Ass Whuppin' (Prerelease)":                       "Release Events",
	"Rukh Egg (Prerelease)":                           "Release Events",
	"Moonsilver Spear (Prerelease)":                   "Avacyn Restored Promos",
	"Mayor of Avabruck / Howlpack Alpha (Prerelease)": "Innistrad Promos",

	"Pristine Talisman (Game Day Mirrodin Pure Preview)": "New Phyrexia Promos",
	"Jace Beleren (Book Agents of Artiface)":             "Miscellaneous Book Promos",
	"Rhox (Alternate Art)":                               "Starter 2000",
	"Astral Drift (Preview)":                             "Modern Horizons Promos",
	"Hall of Triumph (Game Day)":                         "Journey into Nyx Hero's Path",

	"Sylvan Ranger (WPN #51)": "Wizards Play Network 2010",
	"Sylvan Ranger (WPN #70)": "Wizards Play Network 2011",

	"Piper of the Swarm (Bundle)":  "Throne of Eldraine",
	"Chandra's Regulator (Bundle)": "Core Set 2020 Promos",

	"Burning Sun's Avatar (Buy-a-Box)":    "Ixalan Promos",
	"Firesong and Sunspeaker (Buy-a-Box)": "Dominaria",
	"Honor of the Pure (Buy-a-Box)":       "Magic 2010 Promos",
	"Impervious Greatwurm (Buy-a-Box)":    "Guilds of Ravnica",
	"Nexus of Fate (Buy-a-Box)":           "Core Set 2019",
	"The Haunt of Hightower (Buy-a-Box)":  "Ravnica Allegiance",

	"Reliquary Tower (Magic League)":       "Core Set 2019 Promos",
	"Nightpack Ambusher (Convention 2019)": "Core Set 2020 Promos",

	"Stocking Tiger - Target":                            "Happy Holidays",
	"Dictate of the Twin Gods - Journey into Nyx Launch": "Journey into Nyx Promos",
}

var card2numberTable = map[string]string{
	// Happy Holidays variants
	"Stocking Tiger - Target": "13†",

	// Unglued variants
	"B.F.M. Big Furry Monster Left":  "28",
	"B.F.M. Big Furry Monster Right": "29",

	// Prerelease variants
	"Lu Bu, Master-at-Arms (Prerelease April 29th)": "6",
	"Lu Bu, Master-at-Arms (Prerelease July 4th)":   "8",

	// Portal variants
	"Blaze (A - Includes flavor text)":             "118†",
	"Elite Cat Warrior (A - Includes flavor text)": "163†",

	// Eldraine variants (even though it's in the previous table)
	"Piper of the Swarm (Bundle)": "392",

	"Phyrexian Colossus (Jon Finkel - 2000)": "jf306a",
}

func (c *ABUCard) parseNumber(setName, specifier string) (number string, numberCheck func(n string) bool) {
	// Function to determine whether we're parsing the card number
	numberCheck = func(n string) bool {
		return n == number
	}

	found := false
	number, found = card2numberTable[c.FullName]
	if found {
		return
	}

	// Look up card number
	number, found = setVariants[setName][c.Name][specifier]
	if found {
		return
	}

	// Arena specifiers might conflict with further checks
	if strings.HasPrefix(setName, "Arena") {
		return
	} else if c.Set == "World Championship" {
		// Stash the original card number, in case it's needed for later
		// (if available at all)
		cardNumber := c.Number
		// Remove any variant letter that is not needed
		if len(cardNumber) > 0 && unicode.IsLetter(rune(cardNumber[len(cardNumber)-1])) {
			cardNumber = cardNumber[:len(cardNumber)-1]
		}

		// Check if card has a SB badge
		sideb := strings.Contains(specifier, "Sideboard")
		specifier = strings.Replace(specifier, " - Sideboard", "", 1)

		// Strip it from the array, to simplify later processing
		specElement := strings.Split(specifier, " - ")

		// See if we can reuse the number from an existing table
		tags := strings.Split(specElement[0], " ")
		if len(tags) > 1 {
			var no string
			var found bool
			switch tags[0] {
			case "Tempest":
				no, found = setVariants[tags[0]][c.Name][tags[1]]
			case "5th":
				no, found = setVariants["Fifth Edition"][c.Name][tags[2]]
			}
			if found {
				cardNumber = no
			}
		}
		number = cardNumber

		// Save the player name to obtain their initials
		playerName := specElement[len(specElement)-2]
		if strings.HasPrefix(c.FullName, "Mountain (4th") {
			playerName = specElement[len(specElement)-1]
		}
		leadingTag := ""
		for _, l := range strings.Split(playerName, " ") {
			leadingTag += strings.ToLower(string(l[0]))
		}
		if leadingTag == "shr" {
			leadingTag = "sr"
		}

		// At this point leadingTag will be initialized, and number may be ready
		// to use or can be overwritten in the later switch at the end.
		switch c.Name {
		// If the card is from Fallen Empires we can reuse the same function for set
		case "Order of the Ebon Hand", "Hymn to Tourach", "Order of Leitbur":
			numberCheck = func(n string) bool {
				check := strings.Contains(n, leadingTag+number)
				return check && strings.HasSuffix(n, strings.ToLower(string(tags[0][0])))
			}
		default:
			numberCheck = func(n string) bool {
				if strings.HasPrefix(c.FullName, "Mountain (6th Edition ") ||
					strings.HasPrefix(c.FullName, "Forest (5th ") {
					//fmt.Println(c.FullName, setName, tags, leadingTag, number, n)
				}
				//
				check := strings.Contains(n, leadingTag+number)

				if sideb {
					// Some cards have even more tags after 'sb'
					// We can't use Contains because it could confict with a leading tag
					check = check && (strings.HasSuffix(n, "sb") || strings.HasSuffix(n[:len(n)-1], "sb"))
				}
				return check
			}
		}
	}

	// Override card number for basic lands and a few other cards
	if c.Name == "Plains" ||
		c.Name == "Island" ||
		c.Name == "Swamp" ||
		c.Name == "Mountain" ||
		c.Name == "Forest" ||
		c.Name == "Wastes" ||
		c.Name == "Solemn Simulacrum" ||
		c.Name == "Temple of the False God" ||
		strings.HasSuffix(c.Name, "Signet") ||
		strings.HasSuffix(c.Name, "Guildgate") {
		// If specifier is a number use it as is
		_, err := strconv.Atoi(specifier)
		if err == nil {
			number = specifier
			return
		}

		// Else strip any content after the space
		s := strings.Split(specifier, " ")
		if len(s) < 2 {
			return
		}

		idx := 0
		// WCD langs have the number tucked away from where we expect
		// Loop over until we find it (skip the last one which is the year)
		if c.Set == "World Championship" {
			for i, tag := range s[:len(s)-1] {
				_, err := strconv.Atoi(tag)
				if err == nil {
					idx = i
					break
				}
			}
		}
		num := strings.TrimLeft(s[idx], "0")

		origNum := num
		// And strip any possible letter after it
		if len(num) > 0 && unicode.IsLetter(rune(num[len(num)-1])) {
			num = num[:len(num)-1]
		}

		// Now check it's number. If it is, use the one with the letter
		// suffix, but without anything coming after the space
		_, err = strconv.Atoi(num)
		if err == nil {
			number = origNum
		}

		// BFZ intro pack lands have a special suffix
		if setName == "Battle for Zendikar" && len(s) > 1 && s[1] == "Intro" {
			number = number + "a"
		}

		return
	}

	switch setName {
	case "Arabian Nights":
		if specifier != "" {
			number = " "
			numberCheck = func(n string) bool {
				check := false
				if specifier == "a Dark" {
					check = !strings.HasSuffix(n, mtgjson.SuffixLightMana)
				} else if specifier == "b Light" {
					check = strings.HasSuffix(n, mtgjson.SuffixLightMana)
				}
				return check
			}
		}
	case "Alliances",
		"Fallen Empires",
		"Homelands",
		"Champions of Kamigawa",
		"Chronicles",
		"Antiquities":
		if specifier != "Prerelease" && specifier != "" {
			number = " "
			numberCheck = func(n string) bool {
				check := strings.HasSuffix(n, strings.ToLower(string(specifier[0])))
				return check
			}
		}
	}

	return
}

func (c *ABUCard) parseSet(specifier string) (setName string, setCheck func(set mtgjson.Set) bool) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Set

	ed, found := setTable[setName]
	if found {
		setName = ed
	} else {
		// Handle "Magic 2010 / M10"
		if strings.Contains(setName, " / ") {
			s := strings.Split(setName, " / ")
			setName = s[0]
			found = true
		} else if setName == "World Championship" {
			s := strings.Split(specifier, " - ")
			year := s[len(s)-1]
			setName = "World Championship Decks " + year
			_, err := strconv.Atoi(year)
			if err != nil || year == "1996" {
				setName = "Pro Tour Collector Set"
			}
			found = true
		}
	}

	if found && specifier == "" && !strings.Contains(c.FullName, " - ") {
		return
	}

	ed, found = card2setTable[c.FullName]
	if found {
		setName = ed
		return
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	switch {
	case specifier == "M20 Promo Pack":
		setName = "Core Set 2020 Promos"
		switch c.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setName = "M20 Promo Packs"
		}
	case strings.HasPrefix(specifier, "Holiday 2"):
		setName = "Happy Holidays"
	case strings.HasPrefix(specifier, "EURO"):
		setName = "European Land Program"
	case strings.HasPrefix(specifier, "APAC"):
		setName = "Asia Pacific Land Program"
	case strings.HasPrefix(specifier, "Convention "):
		setName = "URL/Convention Promos"
	case strings.HasPrefix(specifier, "Champs"):
		setName = "Champs and States"
	case specifier == "B - Includes reminder text" || specifier == "B - No Flavor Text":
		setName = "Portal Demo Game"
	case strings.HasPrefix(specifier, "Duels of the Planeswalker"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
		}
	case strings.HasPrefix(specifier, "Ravnica Weekend"):
		s := strings.Split(specifier, " ")
		switch s[2] {
		case "Azorius", "Gruul", "Orzhov", "Rakdos", "Simic":
			setName = "RNA Ravnica Weekend"
		case "Boros", "Dimir", "Golgari", "Izzet", "Selesnya":
			setName = "GRN Ravnica Weekend"
		}
	case strings.HasPrefix(specifier, "Judge"):
		s := strings.Split(specifier, " ")
		if len(s) > 1 {
			setName = "Judge Gift Cards " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Judge Gift Cards")
			}
		}
	case strings.HasPrefix(specifier, "Arena"):
		s := strings.Split(specifier, " ")
		if len(s) > 1 {
			setName = "Arena League " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Arena League")
			}
		}
	case strings.HasPrefix(specifier, "SDCC"):
		s := strings.Split(specifier, " ")
		if len(s) > 1 {
			setName = "San Diego Comic-Con " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "San Diego Comic-Con")
			}
		}
	case strings.HasPrefix(specifier, "FNM"):
		switch setName {
		case "Promo", "Classic Sixth Edition", "Kaladesh", "Eldritch Moon":
			s := strings.Split(specifier, " ")
			if len(s) > 1 {
				setName = "Friday Night Magic " + s[1]
			} else {
				setCheck = func(set mtgjson.Set) bool {
					return strings.HasPrefix(set.Name, "Friday Night Magic")
				}
			}
		default:
			setName = setName + " Promos"
		}
	case specifier == "IDW Comic":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics")
		}
		//todo: maybe could be merged with origset+promos
	case strings.Contains(specifier, "Game Day") || specifier == "Gameday Extended Art":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
		}
	case strings.HasPrefix(specifier, "WPN") || specifier == "Gateway":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Wizards Play Network") || strings.HasPrefix(set.Name, "Gateway")
		}
	case strings.HasPrefix(specifier, "Player Rewards"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case specifier == "Clash Pack":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Clash Pack")
		}
	case specifier == "Release" || specifier == setName+" Release" || strings.HasSuffix(specifier, "Buy-a-Box"):
		if setName == "Ixalan" {
			setName = "XLN Treasure Chest"
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return set.Name == "Release Events" || set.Name == "Launch Parties" || set.Name == setName+" Promos"
			}
		}
	case specifier == "Prerelease":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Prerelease Events" || set.Name == setName+" Promos"
		}
	case specifier == "Promo Pack":
		setCheck = func(set mtgjson.Set) bool {
			s := strings.Split(set.ReleaseDate, "-")
			setYear, _ := strconv.Atoi(s[0])
			return set.Name == setName+" Promos" || (setYear >= 2019 && set.Name == setName)
		}

	case specifier == "Intro Pack" ||
		specifier == setName+" Intro Pack" ||
		specifier == setName+" Into Pack" ||
		specifier == "Convention" ||
		specifier == "Draft Weekend" ||
		specifier == "Gift Box" ||
		specifier == "Holiday Gift Box" ||
		specifier == "Launch" ||
		specifier == "Magic League" ||
		specifier == "OPEN HOUSE FULL ART" ||
		specifier == "Open House Full Art" ||
		specifier == "Open House" ||
		specifier == "Planeswalker Weekend" ||
		specifier == "Store Championship Extended Art":
		setName = setName + " Promos"
	}

	return
}
