package abugames

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

var promosetTable = map[string]string{
	"15th Anniversary":        "15th Anniversary Cards",
	"2HG":                     "Two-Headed Giant Tournament",
	"Armada Comics":           "Magazine Inserts",
	"Alternate Art Duelist":   "Magazine Inserts",
	"Book":                    "HarperPrism Book Promos",
	"Dragon'Con 1994":         "Dragon Con",
	"Dragonfury":              "Tarkir Dragonfury",
	"HASCON 2017":             "HasCon 2017",
	"Redemption Original Art": "Wizards of the Coast Online Store",
	"Standard Series":         "BFZ Standard Series",

	"MagicFest":          "MagicFest 2019",
	"MagicFest 2019":     "MagicFest 2019",
	"MagicFest 2020":     "MagicFest 2020",
	"MagicFest Textless": "MagicFest 2020",
	"Judge Extended Art": "Judge Gift Cards 2014",
	"States 2008":        "Gateway 2007",

	"MCQ":      "Pro Tour Promos",
	"Pro Tour": "Pro Tour Promos",
	"RPTQ":     "Pro Tour Promos",
	"Players Tour Qualifier": "Pro Tour Promos",
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
	"World Championship":       "World Championship Promos",
	"Mystery Booster":          "Mystery Booster",
}

var guildkitTable = map[string]string{
	"Guild Kit: Boros":    "GRN Guild Kit",
	"Guild Kit: Dimir":    "GRN Guild Kit",
	"Guild Kit: Golgari":  "GRN Guild Kit",
	"Guild Kit: Izzet":    "GRN Guild Kit",
	"Guild Kit: Selesnya": "GRN Guild Kit",
	"Guild Kit: Azorius":  "RNA Guild Kit",
	"Guild Kit: Gruul":    "RNA Guild Kit",
	"Guild Kit: Orzhov":   "RNA Guild Kit",
	"Guild Kit: Rakdos":   "RNA Guild Kit",
	"Guild Kit: Simic":    "RNA Guild Kit",
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
	"Balduvian Horde (Judge)": "World Championship Promos",

	"Treetop Village (Gateway)": "Summer of Magic",
	"Faerie Conclave (Gateway)": "Summer of Magic",

	"Lightning Bolt (MagicFest Textless)": "MagicFest 2019",
	"Sol Ring (Commander)":                "MagicFest 2019",

	"Ass Whuppin' (Prerelease)": "Release Events",
	"Rukh Egg (Prerelease)":     "Release Events",

	"Pristine Talisman (Game Day Mirrodin Pure Preview)": "New Phyrexia Promos",
	"Jace Beleren (Book Agents of Artiface)":             "Miscellaneous Book Promos",
	"Rhox (Alternate Art)":                               "Starter 2000",
	"Astral Drift (Preview)":                             "Modern Horizons Promos",
	"Hall of Triumph (Game Day)":                         "Journey into Nyx Hero's Path",

	"Sylvan Ranger (WPN #51)": "Wizards Play Network 2010",
	"Sylvan Ranger (WPN #70)": "Wizards Play Network 2011",

	"Chandra's Regulator (Bundle)": "Core Set 2020 Promos",

	"Burning Sun's Avatar (Buy-a-Box)":    "Ixalan Promos",
	"Honor of the Pure (Buy-a-Box)":       "Magic 2010 Promos",
	"Firesong and Sunspeaker (Buy-a-Box)": "Dominaria",
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
	"Blaze (A - Includes flavor text)":             "118",
	"Blaze (B - No flavor text)":                   "118†",
	"Elite Cat Warrior (A - Includes flavor text)": "163†",

	"Piper of the Swarm (Bundle)": "392",

	"Phyrexian Colossus (Jon Finkel - 2000)": "jf305",
}

func (abu *ABUGames) parseSet(c *abuCard) (setName string, setCheck mtgban.SetCheckFunc) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Set

	variants := mtgban.SplitVariants(c.FullName)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}

	// Append the year to WCD sets
	if setName == "World Championship" {
		s := strings.Split(specifier, " - ")
		year := s[len(s)-1]
		setName = "World Championship Decks " + year
		num, err := strconv.Atoi(year)
		if err != nil || num == 1996 {
			setName = "Pro Tour Collector Set"
		}
		return
	}
	// Post m10 core sets have an extra suffix, just drop it
	if strings.Contains(setName, " / ") {
		fields := strings.Split(setName, " / ")
		setName = fields[0]
	}
	// Separate the secret editions
	if setName == "Secret Lair" {
		setName = "Secret Lair Drop"
		if specifier == "Stained Glass" {
			setName += " Promos"
		}
	}
	// Parse Ravnica Guild Kits, but don't return right away because
	// there might be Ravnica Weekend cards mixed in
	if strings.HasPrefix(setName, "Guild Kit") {
		ed, found := guildkitTable[setName]
		if found {
			setName = ed
		}
	}

	ed, found := card2setTable[c.FullName]
	if found {
		setName = ed
		return
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	ed, found = mtgban.EditionTable[setName]
	if found {
		setName = ed
		return
	}

	// Set not yet found, let's check the specifier for more details
	switch {
	case specifier == "M20 Promo Pack":
		setName = "Core Set 2020 Promos"
		switch c.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setName = "M20 Promo Packs"
		}
	case specifier == "Stained Glass":
		setName = "Secret Lair Drop Promos"
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
	case specifier == "IDW Comic":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics")
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
			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
			return set.Name == setName+" Promos" || (setDate.Year() >= 2019 && set.Name == setName)
		}
	case strings.HasPrefix(specifier, "Duels of the Planeswalker"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
		}
	case strings.HasPrefix(specifier, "Ravnica Weekend"):
		s := strings.Fields(specifier)
		if len(s) < 3 {
			setName = "UNKNOWN: " + specifier
			return
		}
		switch s[2] {
		case "Azorius", "Gruul", "Orzhov", "Rakdos", "Simic":
			setName = "RNA Ravnica Weekend"
		case "Boros", "Dimir", "Golgari", "Izzet", "Selesnya":
			setName = "GRN Ravnica Weekend"
		default:
			setName = "UNKNOWN: " + s[2]
		}
	case strings.HasPrefix(specifier, "Judge"):
		s := strings.Fields(specifier)
		if len(s) > 1 {
			setName = "Judge Gift Cards " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Judge Gift Cards")
			}
			// Override the set name as it may interfere later
			setName = "Promo"
		}
	case strings.HasPrefix(specifier, "Arena"):
		s := strings.Fields(specifier)
		if len(s) > 1 {
			setName = "Arena League " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Arena League")
			}
		}
	case strings.HasPrefix(specifier, "SDCC"):
		s := strings.Fields(specifier)
		if len(s) > 1 {
			setName = "San Diego Comic-Con " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "San Diego Comic-Con")
			}
		}
	case strings.HasPrefix(specifier, "FNM"):
		// FNM is sometimes used incorrectly to mark an edition promo,
		// so only list those that indicate a real FNM card
		switch setName {
		case "Promo", "Classic Sixth Edition", "Kaladesh", "Eldritch Moon":
			s := strings.Fields(specifier)
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
	case strings.HasPrefix(specifier, "WPN") || specifier == "Gateway":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Wizards Play Network") || strings.HasPrefix(set.Name, "Gateway")
		}
	case strings.HasPrefix(specifier, "Player Rewards"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case strings.Contains(specifier, "Game Day") || specifier == "Gameday Extended Art":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
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

func (abu *ABUGames) parseNumber(c *abuCard, setName string) (cardName string, numberCheck mtgban.NumberCheckFunc) {
	cardName = c.Name
	variants := mtgban.SplitVariants(c.FullName)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}

	number := ""

	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return card.Number == number
			}
		}

		variants = mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		if strings.Contains(cardName, " / ") {
			s := strings.Split(cardName, " / ")
			cardName = s[0]
		}
		if c.Layout != "" {
			if strings.Contains(cardName, " and ") {
				s := strings.Split(cardName, " and ")
				cardName = s[0]
			} else if strings.Contains(cardName, " to ") {
				s := strings.Split(cardName, " to ")
				cardName = s[0]
			}
		}

	}()

	// Look up card number from the full name only
	no, found := card2numberTable[c.FullName]
	if found {
		number = no
		return
	}

	// Look up card number from every detail
	no, found = setVariants[setName][c.Name][specifier]
	if found {
		number = no
		return
	}

	if c.Set == "World Championship" {
		// Stash the original card number, in case it's needed for later
		// (if available at all)
		cardNumber := c.Number
		// Remove any variant letter that is not needed
		if len(cardNumber) > 0 && unicode.IsLetter(rune(cardNumber[len(cardNumber)-1])) {
			cardNumber = cardNumber[:len(cardNumber)-1]
		}

		// Check if card has a SB badge, then drop it from the specifier
		sideb := strings.Contains(specifier, "Sideboard")
		specifier = strings.Replace(specifier, " - Sideboard", "", 1)

		// Strip it from the array, to simplify later processing
		specElement := strings.Split(specifier, " - ")

		// See if we can reuse the number from an existing table
		tags := strings.Fields(specElement[0])
		origEd := tags[0]
		if len(tags) > 1 {
			var no string
			var found bool
			switch {
			case strings.HasPrefix(origEd, "Tempest"):
				no, found = setVariants["Tempest"][c.Name][tags[1]]
				// There might be a variant letter in the edition name
				fields := strings.Fields(origEd)
				if len(fields) > 1 {
					no += strings.ToLower(fields[1])
				}
			case origEd == "5th":
				no, found = setVariants["Fifth Edition"][c.Name][tags[2]]
			}
			if found {
				cardNumber = no
			}
		}

		// Save the player name to obtain their initials in leadingTag
		playerName := specElement[len(specElement)-2]
		// Its position may very though, try looking for any 2+ word combination
		// that is not in the first poisition (which usually contains edition name)
		for _, element := range specElement[1:] {
			if len(strings.Fields(element)) > 1 {
				playerName = element
				break
			}
		}

		leadingTag := ""
		for _, l := range strings.Fields(playerName) {
			leadingTag += strings.ToLower(string(l[0]))
		}

		// cardNumber may be empty at this point, but it is fine
		number = leadingTag + cardNumber

		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			check := strings.HasPrefix(card.Number, number)
			if sideb {
				// There might be a variant letter, so check with and without
				check = check && (strings.HasSuffix(card.Number, "sb") || strings.HasSuffix(card.Number[:len(card.Number)-1], "sb"))
			}
			switch card.Name {
			// These cards have a variant letter appended to the number
			case "Order of Leitbur", "Order of the Ebon Hand", "Hymn to Tourach":
				suffix := strings.ToLower(tags[0])
				if sideb {
					suffix += "sb"
				}

				check = check && strings.HasSuffix(card.Number, suffix)
			}
			return check
		}

		return
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

		// Strip any content after the space
		s := strings.Fields(specifier)
		if len(s) < 2 {
			// There was no other content, so nothing else can be done
			return
		}

		num := strings.TrimLeft(s[0], "0")
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

	switch specifier {
	case "Showcase":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			num, _ := strconv.Atoi(card.Number)
			return card.HasFrameEffect(mtgjson.FrameEffectShowcase) || num > set.BaseSetSize
		}
	case "Extended Art":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			num, _ := strconv.Atoi(card.Number)
			return card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) || num > set.BaseSetSize
		}
	case "Borderless":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			num, _ := strconv.Atoi(card.Number)
			return card.BorderColor == mtgjson.BorderColorBorderless || num > set.BaseSetSize
		}
	case "Prerelease":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if set.Name == "Prerelease Events" {
				return true
			}
			// These cards are tagged as Prerelease even though they are Release,
			// just passthrough them
			switch card.Name {
			case "Bloodlord of Vaasgoth", //m12
				"Xathrid Gorgon",    //m13
				"Mayor of Avabruck", //inn
				"Moonsilver Spear",  //avr
				"Reya Dawnbringer",  //10e
				"Rukh Egg":          //8ed
				return true
			}

			if strings.HasSuffix(set.Name, "Promos") {
				setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
				if setDate.After(mtgban.NewPrereleaseDate) {
					return strings.Contains(card.Number, "s")
				}
			}

			return card.IsDateStamped
		}
	case "Promo Pack":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if card.Name != "Corpse Knight" && strings.HasSuffix(set.Name, "Promos") {
				return strings.Contains(card.Number, "p")
			}
			return card.HasFrameEffect(mtgjson.FrameEffectInverted)
		}
	default:
		switch setName {
		// Distinguish the light/dark mana symbols
		case "Arabian Nights":
			if specifier != "" {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					check := false
					if specifier == "a Dark" {
						check = !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					} else if specifier == "b Light" {
						check = strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					}
					return check
				}
			}
		// Use the first letter of the specifier as suffix for the collector number
		case "Alliances",
			"Antiquities",
			"Champions of Kamigawa",
			"Chronicles",
			"Deckmasters",
			"Fallen Empires",
			"Homelands",
			"Unstable":
			if specifier != "" {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasSuffix(card.Number, strings.ToLower(string(specifier[0])))
				}
			}
		// Skip the various variants from the normal set
		case "Throne of Eldraine":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				num, _ := strconv.Atoi(card.Number)
				return num <= set.BaseSetSize || (num >= 303 && num <= 333)
			}
		case "Theros Beyond Death":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				num, _ := strconv.Atoi(card.Number)
				return num <= set.BaseSetSize || (num >= 269 && num <= 297)
			}
		}
	}

	return
}
