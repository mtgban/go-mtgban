package strikezone

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

// LUT for SZ sets to MTGJSON sets.
var setTable = map[string]string{
	"10th Edition":                      "Tenth Edition",
	"4th Edition":                       "Fourth Edition",
	"5th Edition":                       "Fifth Edition",
	"6th Edition":                       "Classic Sixth Edition",
	"7th Edition":                       "Seventh Edition",
	"8th Edition":                       "Eighth Edition",
	"9th Edition":                       "Ninth Edition",
	"Classic 6th Edition":               "Classic Sixth Edition",
	"Commander Singles":                 "Commander 2011",
	"Commander 2013 Edition":            "Commander 2013",
	"Commander 2014 Edition":            "Commander 2014",
	"Commander 2016 Edition":            "Commander 2016",
	"Commander":                         "Commander 2011",
	"Futuresight":                       "Future Sight",
	"Hours of Devestation":              "Hour of Devastation",
	"Guilds of Ravnica Mythic Edition":  "Mythic Edition",
	"Mystery Booster Test Print":        "Mystery Booster Playtest Cards",
	"Mystery Booster Test Prints":       "Mystery Booster Playtest Cards",
	"M10 Core Set":                      "Magic 2010",
	"M11 Core Set":                      "Magic 2011",
	"M12 Core Set":                      "Magic 2012",
	"M13 Core Set":                      "Magic 2013",
	"M14 Core Set":                      "Magic 2014",
	"M15 Core Set":                      "Magic 2015",
	"Ravnica Allegiance Mythic Edition": "Mythic Edition",
	"Ravnica":                           "Ravnica: City of Guilds",
	"Revised":                           "Revised Edition",
	"Shadows Over Innistrad":            "Shadows over Innistrad",
	"Time Spiral Time Shifted":          "Time Spiral Timeshifted",
	"Ultimate Box Toppers":              "Ultimate Box Topper",
	"Unlimited":                         "Unlimited Edition",
	"War of the Spark Mythic Edition":   "Mythic Edition",

	"Premium Deck Fire and Lightning": "Premium Deck Series: Fire and Lightning",
	"Premium Deck Graveborn":          "Premium Deck Series: Graveborn",
	"Premium Deck Slivers":            "Premium Deck Series: Slivers",

	"Duel Deck Heros VS Monsters":           "Duel Decks: Heroes vs. Monsters",
	"Duel Decks: Phyrexia vs The Coalition": "Duel Decks: Phyrexia vs. the Coalition",
	"Duel Decks: Kiora vs Elspeth":          "Duel Decks: Elspeth vs. Kiora",
}

var promosetTable = map[string]string{
	"WCQ":                              "World Magic Cup Qualifiers",
	"WMC Promo":                        "World Magic Cup Qualifiers",
	"WMC":                              "World Magic Cup Qualifiers",
	"WMCQ":                             "World Magic Cup Qualifiers",
	"2011 Pro Tour Promo":              "Pro Tour Promos",
	"MCQ Promo":                        "Pro Tour Promos",
	"Players Tour Qualifier PTQ Promo": "Pro Tour Promos",
	"Pro Tour Promo":                   "Pro Tour Promos",
	"RPTQ Promo":                       "Pro Tour Promos",
	"RPTQ":                             "Pro Tour Promos",

	"Convention Foil M19": "Core Set 2019 Promos",

	"Arena IA":            "Arena League 2001",
	"BIBB Alt Art":        "XLN Treasure Chest",
	"Champs Full Art":     "Champs and States",
	"Deckmaster":          "Deckmasters",
	"Deckmasters":         "Deckmasters",
	"GP Promo":            "Grand Prix Promos",
	"Guru":                "Guru",
	"JSS":                 "Junior Super Series",
	"JSS DCI PROMO":       "Junior Super Series",
	"Media Promo":         "Resale Promos",
	"Nationals":           "Nationals Promos",
	"Release Event":       "Release Events",
	"Shooting Star Promo": "2017 Gift Pack",
	"Summer of Magic":     "Summer of Magic",
	"WOTC Employee Card":  "Happy Holidays",

	"MagicFest 2019":                     "MagicFest 2019",
	"FOIL 2019 MF MagicFest GP Promo":    "MagicFest 2019",
	"NONFOIL 2019 MF MagicFest GP Promo": "MagicFest 2019",
	"MagicFest 2020":                     "MagicFest 2020",
	"FOIL 2020 MF MagicFest GP Promo":    "MagicFest 2020",
	"NONFOIL 2020 MF MagicFest GP Promo": "MagicFest 2020",

	"2006 Japanese MPS League Promo": "Magic Premiere Shop 2006",
	"2007 Japanese MPS League Promo": "Magic Premiere Shop 2007",
	"2008 Japanese MPS League Promo": "Magic Premiere Shop 2008",
	"2009 Japanese MPS League Promo": "Magic Premiere Shop 2009",
	"2010 Japanese MPS League Promo": "Magic Premiere Shop 2010",
}

var card2setTable = map[string]string{
	"Serra Angel (Beta Art)":               "Wizards of the Coast Online Store",
	"Vexing Shusher (Release Event)":       "Launch Parties",
	"Nicol Bolas Planeswalker (Archenemy)": "Archenemy: Nicol Bolas",
	"Jace Beleren (Book Promo)":            "Miscellaneous Book Promos",
	"Underworld Dreams (DCI)":              "Two-Headed Giant Tournament",
	"Mutavault (Full Art)":                 "Champs and States",
	"Reliquary Tower (League Promo)":       "Core Set 2019 Promos",

	"Wood Elves (Promo)":              "Gateway 2006",
	"Yixlid Jailer (DCI)":             "Gateway 2007",
	"Wasteland (DCI)":                 "Magic Player Rewards 2001",
	"Voidmage Prodigy (DCI)":          "Magic Player Rewards 2003",
	"Powder Keg (DCI)":                "Magic Player Rewards 2004",
	"Two-Headed Dragon (DCI)":         "Junior Super Series",
	"Crusade (DCI)":                   "Junior Super Series",
	"Lord of Atlantis (DCI)":          "Junior Super Series",
	"Serra Avatar (US)":               "Junior Super Series",
	"Thran Quarry (US)":               "Junior Super Series",
	"Karn, Silver Golem (US)":         "Arena League 1999",
	"Rewind (US)":                     "Arena League 1999",
	"Duress (US)":                     "Arena League 2000",
	"Stupor (6E)":                     "Arena League 2000",
	"Chill (6E)":                      "Arena League 2000",
	"Enlightened Tutor (6E)":          "Arena League 2000",
	"Island (Arena Beta)":             "Arena League 2002",
	"Lightning Bolt (Beta Art Promo)": "Judge Gift Cards 1998",
	"Memory Lapse (6E)":               "Judge Gift Cards 1999",
	"Swords to Plowshares (DCI)":      "Friday Night Magic 2001",
	"Lingering Souls (WPN)":           "Friday Night Magic 2012",
	"Glistener Elf (WPN)":             "Friday Night Magic 2012",
	"Reliquary Tower (FNM)":           "Friday Night Magic 2013",
	"Goblin Warchief (FNM 2016)":      "Friday Night Magic 2016",
	"Incinerate (Book Promo)":         "DCI Legend Membership",
	"Counterspell (Arena Non Foil)":   "DCI Legend Membership",

	"Firesong and Sunspeaker (BIBB)":       "Dominaria",
	"Flusterstorm (BIBB)":                  "Modern Horizons",
	"Impervious Greatwurm (BIBB)":          "Guilds of Ravnica",
	"Nexus of Fate (BIBB Promo)":           "Core Set 2019",
	"The Haunt of Hightower (BIBB)":        "Ravnica Allegiance",
	"Tezzeret Master of the Bridge (BIBB)": "War of the Spark",
	"Rienne Angel of Rebirth (M20 BIBB)":   "Core Set 2020",

	"Ertai, the Corrupted (Alternate Art)":               "Planeshift",
	"Tahngarth, Talruum Hero (Alternate Planeshift Art)": "Planeshift",
	"Skyship Weatherlight (Alternate Art)":               "Planeshift",
}

// These don't have any variant, but this table only applies to Promo edition
var promo2setTable = map[string]string{
	"Show and Tell":             "Judge Gift Cards 2013",
	"Overwhelming Forces":       "Judge Gift Cards 2013",
	"Nekusar the Mindrazer":     "Judge Gift Cards 2014",
	"Hanna Ship s Navigator":    "Judge Gift Cards 2014",
	"Riku of Two Reflections":   "Judge Gift Cards 2014",
	"Feldon of the Third Path":  "Judge Gift Cards 2015",
	"Yuriko the Tiger s Shadow": "Judge Gift Cards 2019",
	"Dismember":                 "Friday Night Magic 2012",
	"Ancient Grudge":            "Friday Night Magic 2012",
	"Nalathni Dragon":           "Dragon Con",
	"Mishra s Toy Workshop":     "Happy Holidays",
	"Scavenging Ooze":           "Duels of the Planeswalkers 2013 Promos",
	"Bonescythe Sliver":         "Duels of the Planeswalkers 2013 Promos",
}

var card2number = map[string]string{
	"Wastes (Full Art 184)":          "184",
	"B.F.M. (Big Furry Monster)":     "28",
	"B.F.M. (Big Furry Monster) (b)": "29",
	"Goblin Grenade (1)":             "56b",
	"Goblin Grenade (2)":             "56c",
}

func (sz *Strikezone) parseSet(c *szCard) (setName string, setCheck mtgban.SetCheckFunc) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Edition

	// Look up the Set
	ed, found := setTable[setName]
	if found {
		setName = ed
		return
	}

	if strings.HasPrefix(setName, "Duel Deck ") {
		setName = strings.Replace(setName, "Deck", "Decks:", 1)
	}
	if strings.HasPrefix(setName, "Duel Decks ") {
		setName = strings.Replace(setName, "Decks", "Decks:", 1)
	}
	if strings.HasPrefix(setName, "Duel Decks: ") {
		setName = strings.Replace(setName, " VS ", " vs ", 1)
		setName = strings.Replace(setName, " vs ", " vs. ", 1)
		return
	}
	if strings.HasPrefix(setName, "Magic 20") {
		setName = strings.Replace(setName, " Core Set", "", 1)

		// Handle the post-Origins core sets
		s := strings.Fields(setName)
		if len(s) > 1 {
			year, _ := strconv.Atoi(s[1])
			if year >= 2019 {
				setName = fmt.Sprintf("Core Set %d", year)
				return
			}
			setName = s[0] + " " + s[1]
		}
		return
	}
	if setName == "Secret Lair" {
		setName = "Secret Lair Drop"
		if strings.HasSuffix(c.Name, "Stained Glass") {
			setName += " Promos"
		}
		return
	}

	ed, found = card2setTable[c.Name]
	if found {
		setName = ed
		return
	}

	if setName == "Promotional Cards" {
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
	tmpName := variants[0]

	// XLN treasure chest cards, strip the duplicated card name from the specifier
	if strings.Contains(specifier, "BIBB Alt Art") {
		specifier = strings.Replace(specifier, c.Name+" ", "", 1)
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	// The SDCC year is a random position every time :(
	switch {
	case strings.Contains(specifier, "SDCC"):
		fields := strings.Fields(specifier)
		year := ""
		for _, field := range fields {
			_, err := strconv.Atoi(field)
			if err == nil {
				year = field
				break
			}
		}
		setName = "San Diego Comic-Con " + year
		return
	case strings.HasPrefix(specifier, "The "):
		setName = "Magic Premiere Shop 2005"
		return
	case strings.Contains(specifier, "Holiday"):
		setName = "Happy Holidays"
		return
	case specifier == "US":
		switch tmpName {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setName = "Arena League 1999"
			return
		}
	case strings.Contains(c.Name, "APAC"):
		setName = "Asia Pacific Land Program"
		return
	case strings.Contains(c.Name, "EURO"):
		setName = "European Land Program"
		return
	case strings.Contains(c.Name, "Standard Showdown 2018"):
		setName = "M19 Standard Showdown"
		return
	case strings.Contains(c.Name, "Ravnica Weekend A"):
		setName = "GRN Ravnica Weekend"
		return
	case strings.Contains(c.Name, "Ravnica Weekend B"):
		setName = "RNA Ravnica Weekend"
		return
	}

	// Wrap various tags in a single one
	if strings.Contains(specifier, "Judge") {
		fields := strings.Fields(specifier)
		for _, year := range fields {
			_, err := strconv.Atoi(year)
			if err == nil {
				setName = "Judge Gift Cards " + year
				return
			}
		}
		specifier = "Judge"
	} else if strings.Contains(specifier, "Arena") {
		fields := strings.Fields(specifier)
		for _, year := range fields {
			_, err := strconv.Atoi(year)
			if err == nil {
				setName = "Arena League " + year
				return
			}
		}
		specifier = "Arena"
	} else if strings.Contains(specifier, "Prerelease") {
		specifier = "Prerelease"
	}

	switch specifier {
	case "DCI", "VI DCI":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Judge Gift Cards") || strings.HasPrefix(set.Name, "Friday Night Magic") || strings.HasPrefix(set.Name, "Arena League")
		}
	case "Judge", "US":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Judge Gift Cards")
		}
	case "Prerelease", "Pre Release":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Prerelease Events" || strings.HasSuffix(set.Name, "Promos")
		}
	case "FNM", "6E", "FNM Promo", "FNM Foil":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Friday Night Magic") || strings.HasSuffix(set.Name, "Promos")
		}
	case "Textless", "Textless Player Rewards", "FOIL Textless Player Rewards":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case "Clash Pack Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Clash Pack")
		}
	case "IDW":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics")
		}
	case "Arena":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Arena League")
		}
	case "Duel of the Planeswalkers",
		"Duels of the Planeswalkers - PC",
		"Playstation", "PS3 Promo", "DotP 2012 - Xbox", "X Box Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
		}
	case "WPN", "WPN Foil", "WPN Promo", "WPN Foil Promo":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Wizards Play Network")
		}
	case "Gateway":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Gateway")
		}
	default:
		switch setName {
		case "Promotional Cards":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Promos")
			}
		case "Promo Pack":
			switch c.Name {
			case "Plains", "Island", "Swamp", "Mountain", "Forest":
				setName = "M20 Promo Packs"
			default:
				setCheck = func(set mtgjson.Set) bool {
					return strings.HasSuffix(set.Name, "Promos") || set.Type == "expansion"
				}
			}
		}
	}

	return
}

func (sz *Strikezone) parseNumber(c *szCard, setName string) (cardName string, numberCheck mtgban.NumberCheckFunc) {
	cardName = c.Name
	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}
	cardName = variants[0]
	number := ""

	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				// Reduce aliasing by making sure that only "<xxx> Promos" (except
				// Resale) have numbers with 'p' or 's' at the end.
				if (set.Name == "Resale Promos" || !strings.HasSuffix(set.Name, "Promos")) && (strings.HasSuffix(card.Number, "p") || strings.Contains(card.Number, "s")) {
					return false
				}
				return card.Number == number
			}
		}

		if setName == "Promotional Cards" && specifier != "Clash Pack Promo" && specifier != "Duel of the Planeswalkers" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return card.IsPromo
			}
		}

		variants = mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		switch {
		case strings.Contains(cardName, " | "):
			cn := strings.Split(cardName, " | ")
			cardName = cn[0]
		case strings.Contains(cardName, " // "):
			cn := strings.Split(cardName, " // ")
			cardName = cn[0]
		case strings.Contains(cardName, " / "):
			cn := strings.Split(cardName, " / ")
			cardName = cn[0]
		case strings.Contains(cardName, " to ") && (setName == "Hour of Devastation" || setName == "Amonkhet"):
			cn := strings.Split(cardName, " to ")
			cardName = cn[0]
		}
	}()

	no, found := card2number[c.Name]
	if found {
		number = no
		return
	}

	if cardName == "Kaya, Ghost Assassin" {
		number = "75"
		if c.IsFoil {
			number = "222"
		}
		return
	}

	no, found = setVariants[setName][cardName][specifier]
	if found {
		number = no
		return
	}

	if strings.Contains(specifier, "Prerelease") {
		specifier = "Prerelease"
	}

	switch specifier {
	case "Prerelease":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if set.Name == "Prerelease Events" {
				return true
			}
			// These cards are tagged as Prerelease even though they are Release,
			// just passthrough them, or they don't have the 's' suffix
			switch card.Name {
			case "Bloodlord of Vaasgoth", //m12
				"Xathrid Gorgon",    //m13
				"Mayor of Avabruck", //inn
				"Moonsilver Spear",  //avr
				"Astral Drift",      //mh1
				"Reya Dawnbringer",  //10e
				"Celestine Reef",    //hop
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
		return
	case "JPN Alternate Art":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
		return
	default:
		if specifier != "" {
			switch cardName {
			case "Plains", "Island", "Swamp", "Mountain", "Forest":
				_, err := strconv.Atoi(specifier)
				if err == nil {
					number = specifier
					return
				}
			}
		}
	}

	switch setName {
	case "Promo Pack":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if !card.HasFrameEffect(mtgjson.FrameEffectInverted) && strings.HasSuffix(set.Name, "Promos") {
				return strings.Contains(card.Number, "p")
			}
			return card.HasFrameEffect(mtgjson.FrameEffectInverted)
		}
	case "GRN Ravnica Weekend", "RNA Ravnica Weekend":
		fields := strings.Fields(cardName)
		cardName = fields[0]
		number = fields[3]
	case "M19 Standard Showdown":
		fields := strings.Fields(cardName)
		cardName = fields[0]
	case "Secret Lair Drop Promos":
		cardName = strings.TrimSuffix(cardName, " Stained Glass")
	case "Secret Lair Drop":
		if strings.HasPrefix(cardName, "Serum Visions") {
			fields := strings.Fields(cardName)
			number = fields[2]
			cardName = strings.Replace(cardName, " "+number, "", 1)
		}
	case "FNM", "6E", "FNM Promo", "FNM Foil":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if strings.HasSuffix(set.Name, "Promos") {
				return card.HasFrameEffect(mtgjson.FrameEffectInverted)
			}
			if number != "" {
				return card.Number == number
			}
			return strings.HasPrefix(set.Name, "Friday")
		}
	case "Asia Pacific Land Program", "European Land Program":
		fields := strings.Fields(cardName)
		cardName = fields[0]
		no, found = setVariants[setName][cardName][specifier]
		if found {
			number = no
			return
		}
		fields = strings.Fields(specifier)
		location := fields[0]
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			return strings.Contains(card.FlavorText, location)
		}
	case "Throne of Eldraine", "Theros Beyond Death":
		fakePromo := false
		switch {
		case strings.HasPrefix(cardName, "Borderless"):
			cardName = strings.TrimPrefix(cardName, "Borderless ")
			fakePromo = true
		case strings.HasPrefix(cardName, "Showcase"):
			cardName = strings.TrimPrefix(cardName, "Showcase ")
			fakePromo = true
		case strings.HasPrefix(cardName, "Extended Art"):
			cardName = strings.TrimPrefix(cardName, "Extended Art ")
			fakePromo = true
		}

		if fakePromo {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				num, err := strconv.Atoi(card.Number)
				if err != nil {
					return false
				}
				return num > set.BaseSetSize
			}
			return

		}

		// Skip the various variants from the normal set
		switch setName {
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

	case "Arabian Nights":
		if specifier != "" {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				check := false
				if specifier == "dark circle" {
					check = !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
				} else if specifier == "light circle" {
					check = strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
				}
				return check
			}
		}
	}

	return
}
