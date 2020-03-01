package channelfireball

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

var promosetTable = map[string]string{
	"2017 WMCQ Promo":                     "World Magic Cup Qualifiers",
	"WMCQ Foil":                           "World Magic Cup Qualifiers",
	"Mythic Championship Qualifier Promo": "Pro Tour Promos",
	"Pro Tour Foil":                       "Pro Tour Promos",
	"Regional PTQ Promo Foil":             "Pro Tour Promos",
	"15th Anniversary Foil":               "15th Anniversary Cards",
	"15th Anniversary Promo":              "15th Anniversary Cards",
	"Extended Art":                        "Champs and States",
	"Extended Art Foil":                   "Champs and States",
	"PAX Prime Promo":                     "URL/Convention Promos",
	"2012 Convention Promo":               "URL/Convention Promos",

	"2018 Nationals Promo":                 "Nationals Promos",
	"2HG Foil":                             "Two-Headed Giant Tournament",
	"Deckmaster Promo":                     "Deckmasters",
	"Foil Beta Picture":                    "Wizards of the Coast Online Store",
	"Hascon 2017 Promo":                    "HasCon 2017",
	"Japanese Glossy Gotta Magazine Promo": "Magazine Inserts",
	"Journey into Nyx Game Day":            "Journey into Nyx Hero's Path",
	"Junior APAC Series Promos":            "Junior APAC Series",
	"Stained Glass":                        "Secret Lair Drop Promos",
	"Standard Series Promo":                "BFZ Standard Series",
	"Summer of Magic Promo":                "Summer of Magic",
	"Tarkir Dragonfury Promo":              "Tarkir Dragonfury",
	"Treasure Map Promo":                   "XLN Treasure Chest",
	"Treasure Map":                         "XLN Treasure Chest",

	"Welcome Deck 2016":      "Welcome Deck 2016",
	"Welcome Deck 2017":      "Welcome Deck 2017",
	"2017 Standard Showdown": "XLN Standard Showdown",
	"2018 Standard Showdown": "M19 Standard Showdown",
	"Gift Pack 2017":         "2017 Gift Pack",
	"Gift Pack 2018":         "M19 Gift Pack",

	"Resale Foil":  "Resale Promos",
	"Resale Promo": "Resale Promos",
}

var card2setTable = map[string]string{
	"Counterspell (Arena 1996)": "DCI Legend Membership",
	"Incinerate (Arena 1996)":   "DCI Legend Membership",

	"Dictate of Kruphix (Journey into Nyx Game Day)": "Journey into Nyx Promos",
	"Squelching Leeches (Journey into Nyx Game Day)": "Journey into Nyx Promos",

	"Firesong and Sunspeaker (Buy-A-Box Promo)":        "Dominaria",
	"Flusterstorm (Buy-a-Box Promo)":                   "Modern Horizons",
	"Nexus of Fate (Buy-a-Box Promo)":                  "Core Set 2019",
	"Impervious Greatwurm (Buy-a-Box Promo)":           "Guilds of Ravnica",
	"The Haunt of Hightower (Buy-a-Box Promo)":         "Ravnica Allegiance",
	"Tezzeret, Master of the Bridge (Buy-a-Box Promo)": "War of the Spark",
	"Rienne, Angel of Rebirth (Buy-A-Box Promo)":       "Core Set 2020",
	"Kenrith, the Returned King (Buy-a-Box Promo)":     "Throne of Eldraine",
	"Athreos, Shroud-Veiled (Buy-a-Box Promo)":         "Theros Beyond Death",

	"Balduvian Horde (Judge Foil)":                           "World Championship Promos",
	"Dauntless Dourbark (2008 States Foil)":                  "Gateway 2007",
	"Dreg Mangler (Holiday Box 2012 Promo)":                  "Return to Ravnica Promos",
	"Karametra's Acolyte (Holiday 2013 Gift Box Promo Foil)": "Theros Promos",
	"Rhox (Alternate Art Foil)":                              "Starter 2000",

	"Cast Down (Japanese Promo)":       "Magazine Inserts",
	"Steward of Valeron (Media Promo)": "URL/Convention Promos",
	"Kor Skyfisher (Media Promo)":      "URL/Convention Promos",

	"Tahngarth, Talruum Hero (Alternate Art Foil)": "Planeshift",
	"Skyship Weatherlight (Alternate Art Foil)":    "Planeshift",
	"Ertai, the Corrupted (Alternate Art Foil)":    "Planeshift",

	"Armored Pegasus (Flavor Text + Reminder Text)": "Portal Demo Game",
	"Bull Hippo (Reminder Text Only)":               "Portal Demo Game",
	"Cloud Pirates (Reminder Text Only)":            "Portal Demo Game",
	"Feral Shadow (Flavor Text + Reminder Text)":    "Portal Demo Game",
	"Snapping Drake (Flavor Text + Reminder Text)":  "Portal Demo Game",
	"Storm Crow (Flavor Text + Reminder Text)":      "Portal Demo Game",

	"Knight of New Alara (Alara Reborn Release)":                          "Launch Parties",
	"Ass Whuppin' (Unhinged Prerelease)":                                  "Release Events",
	"Ajani Vengeant (Shards of Alara Release)":                            "Prerelease Events",
	"Lu Bu, Master-at-Arms (Portal Three Kingdoms Prerelease April 29th)": "Prerelease Events",
	"Lu Bu, Master-at-Arms (July 4, 1999)":                                "Prerelease Events",

	"Death Baron (Extended Art) (Convention 2018) (Core Set 2019 Symbol)": "Core Set 2019 Promos",

	"Demonic Tutor (Judge Foil)":          "Judge Gift Cards 2008",
	"Demonic Tutor (Judge Academy Promo)": "Judge Gift Cards 2020",
}

var card2numberTable = map[string]string{
	"B.F.M. (Big Furry Monster) (Left side)":  "28",
	"B.F.M. (Big Furry Monster) (Right side)": "29",

	"Blaze (Flavor Text Only)":   "118",
	"Blaze (Reminder Text Only)": "118†",

	"Kaya, Ghost Assassin":                                "75",
	"Kaya, Ghost Assassin (222/221) (Alternate Art Foil)": "222",

	"Stocking Tiger (2013 Holiday Foil)":       "13",
	"Stocking Tiger (No Stamp) (Holiday 2013)": "13†",
}

func (cfb *Channelfireball) parseSet(c *cfbCard) (setName string, setCheck mtgban.SetCheckFunc) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Edition

	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}

	// Contrary to most scrapers, we check this table first, since there is
	// aliasing between "Portal 1" and "Portal Demo Game"
	ed, found := card2setTable[c.Name]
	if found {
		setName = ed
		return
	}

	ed, found = mtgban.EditionTable[setName]
	if found {
		setName = ed
		return
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	switch setName {
	case "Secret Lair Drop Series":
		setName = "Secret Lair Drop"
		if specifier == "Stained Glass" {
			setName += " Promos"
		}

	case "Duel Decks Anthology":
		if specifier != "" && len(variants[len(variants)-1]) > 3 {
			deckVariant := strings.Replace(variants[len(variants)-1], " vs ", " vs. ", 1)
			if deckVariant == "Goblins vs. Elves" {
				deckVariant = "Elves vs. Goblins"
			}
			setName = "Duel Decks Anthology: " + deckVariant
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Duel Decks Anthology")
			}
		}
	case "Promos: Intro/Clash Pack":
		if strings.Contains(specifier, "Clash Pack") {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Clash Pack")
			}
			return
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
		}
	case "Promos: Arena":
		for _, spec := range variants {
			if strings.Contains(spec, "Arena") {
				fields := strings.Fields(spec)
				for _, field := range fields {
					_, err := strconv.Atoi(field)
					if err == nil {
						setName = "Arena League " + field
						return
					}
				}
			}
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Arena League")
		}
	case "Promos: MPS Lands":
		for _, spec := range variants {
			if strings.HasPrefix(spec, "MPS") {
				fields := strings.Fields(spec)
				for _, field := range fields {
					_, err := strconv.Atoi(field)
					if err == nil {
						setName = "Magic Premiere Shop " + field
						return
					}
				}
			}
		}
		if strings.HasPrefix(specifier, "The ") || strings.HasSuffix(specifier, "Japanese") ||
			specifier == "MPS Cult of Rakdos" || specifier == "MPS Golgari Swarm" ||
			specifier == "MPS House Dimir" || specifier == "MPS Orzhov Syndicate" {
			setName = "Magic Premiere Shop 2005"
			return
		}
		setName = "UNKNOWN: " + strings.Join(variants, ":")
	case "Promos: Book Inserts":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics") ||
				set.Name == "Magazine Inserts" ||
				set.Name == "HarperPrism Book Promos" ||
				set.Name == "Miscellaneous Book Promos" ||
				set.Name == "Dragon Con"
		}
	case "Promos: FNM":
		fields := strings.Fields(specifier)
		for _, field := range fields {
			num, err := strconv.Atoi(field)
			if err == nil && num != 2019 {
				setName = "Friday Night Magic " + field
				return
			}
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Friday Night Magic") ||
				strings.HasSuffix(set.Name, "Promos")
		}
	case "Promos: MPR":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case "Promos: Buy a Box":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
		}
	case "Promos: Game Day":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
		}
	case "Promo Pack":
		switch variants[0] {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setName = "M20 Promo Packs"
			return
		}
		if len(variants) > 2 {
			setName = specifier + " Promos"
			return
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos") || set.Type == "expansion"
		}
	case "Promos: Judge Rewards":
		fields := strings.Fields(specifier)
		for _, field := range fields {
			_, err := strconv.Atoi(field)
			if err == nil {
				setName = "Judge Gift Cards " + field
				return
			}
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Judge Gift Cards")
		}
	case "Promos: WPN":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Wizards Play Network") || strings.HasPrefix(set.Name, "Gateway")
		}
	case "Promos: JSS":
		switch {
		case strings.HasPrefix(specifier, "Junior Series E") ||
			strings.HasPrefix(specifier, "European JSS") ||
			strings.HasPrefix(specifier, "JSS Foil E") ||
			strings.Contains(specifier, "ESS"):
			setName = "Junior Series Europe"
		default:
			setName = "Junior Super Series"
		}
	case "Promos: Release":
		if strings.Contains(specifier, "Prerelease") {
			if strings.HasPrefix(specifier, "XLN") || strings.HasPrefix(specifier, "ELD") {
				fields := strings.Fields(specifier)
				setName = cfb.db[fields[0]].Name + " Promos"
				return
			}
			setCheck = func(set mtgjson.Set) bool {
				return set.Name == "Prerelease Events" || strings.HasSuffix(set.Name, "Promos")
			}
			return
		}
		if specifier == "WPN Foil" {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Wizards Play Network")
			}
			return
		}

		// One-off card without variant
		if c.Name == "Hamletback Goliath" {
			setName = "Resale Promos"
		}

		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Promos")
		}

	case "Promos: Miscellaneous":
		switch {
		case len(specifier) == 3 &&
			unicode.IsLetter(rune(specifier[0])) && unicode.IsDigit(rune(specifier[2])):
			if specifier[0] == 'A' {
				setName = "GRN Ravnica Weekend"
			} else if specifier[0] == 'B' {
				setName = "RNA Ravnica Weekend"
			}
		case strings.Contains(specifier, "SDCC"):
			fields := strings.Fields(specifier)
			for _, field := range fields {
				_, err := strconv.Atoi(field)
				if err == nil {
					setName = "San Diego Comic-Con " + field
					return
				}
			}
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "San Diego Comic-Con")
			}
		case strings.Contains(specifier, "MagicFest"):
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "MagicFest")
			}
		case strings.Contains(specifier, "of the Plane"):
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
			}
		case strings.Contains(specifier, "Gateway") || specifier == "Euro Promo":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Gateway")
			}
		case strings.Contains(specifier, "Holiday"):
			setName = "Happy Holidays"
		case strings.HasPrefix(specifier, "Standard Showdown 2017"):
			setName = "XLN Standard Showdown"
		case strings.HasPrefix(specifier, "Standard Showdown 2018"):
			setName = "M19 Standard Showdown"
		case specifier == "Media Promo":
			setName = "Resale Promos"
		default:
			if c.Name == "Decorated Knight // Present Arms" ||
				c.Name == "Stocking Tiger (No Stamp) (Holiday 2013)" {
				setName = "Happy Holidays"
				return
			}
			setCheck = func(set mtgjson.Set) bool {
				return set.Name == "Release Events" ||
					set.Name == "Launch Parties" ||
					strings.HasSuffix(set.Name, "Promos")
			}
		}
	}

	return
}

func (cfb *Channelfireball) parseNumber(c *cfbCard, setName string) (cardName string, numberCheck mtgban.NumberCheckFunc) {
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
				return card.Number == number
			}
		}

		variants = mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		if strings.Contains(cardName, " // ") {
			s := strings.Split(cardName, " // ")
			cardName = s[0]
		}
	}()

	// Look up card number from every detail
	no, found := card2numberTable[c.Name]
	if found {
		number = no
		return
	}

	// Look up card number from every detail
	no, found = mtgban.VariantsTable[setName][cardName][specifier]
	if found {
		number = no
		return
	}

	// Override card number for basic lands and a few other cards
	if cardName == "Plains" ||
		cardName == "Island" ||
		cardName == "Swamp" ||
		cardName == "Mountain" ||
		cardName == "Forest" ||
		cardName == "Wastes" ||
		cardName == "Solemn Simulacrum" ||
		cardName == "Temple of the False God" ||
		cardName == "Serum Visions" ||
		strings.HasSuffix(cardName, "Signet") ||
		strings.HasSuffix(cardName, "Guildgate") {

		if strings.HasSuffix(setName, "Ravnica Weekend") {
			number = specifier
			return
		}

		for _, field := range variants {
			// Fate Reforged lands
			if strings.Contains(field, "/") {
				pos := strings.Index(field, "/")
				field = field[:pos]
			}

			num := strings.TrimLeft(field, "0")
			n, err := strconv.Atoi(num)
			// This magic number is to prevent using a year as variant
			if err == nil && n < 1000 {
				number = num
				return
			}
		}
	}

	if strings.Contains(specifier, "Prerelease") {
		specifier = "Prerelease"
	}

	switch specifier {
	case "July 4, 1999":
		number = "8"
	case "Portal Three Kingdoms Prerelease April 29th":
		number = "6"

	case "Prerelease":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if set.Name == "Prerelease Events" {
				return true
			}
			switch card.Name {
			case "Moonsilver Spear",
				"Astral Drift",
				"Mayor of Avabruck",
				"Bloodlord of Vaasgoth",
				"Xathrid Gorgon":
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
			if !card.HasFrameEffect(mtgjson.FrameEffectInverted) && strings.HasSuffix(set.Name, "Promos") {
				return strings.Contains(card.Number, "p")
			}
			return card.HasFrameEffect(mtgjson.FrameEffectInverted)
		}
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
	case "Japanese Alternate Art":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	default:
		switch setName {
		case "Portal":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				if specifier == "Reminder Text Only" {
					return strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
				}
				return !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
			}
		case "Promos: FNM":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				if strings.HasSuffix(set.Name, "Promos") {
					return card.HasFrameEffect(mtgjson.FrameEffectInverted)
				}
				if number != "" {
					return card.Number == number
				}
				return strings.HasPrefix(set.Name, "Friday")
			}
		case "Arabian Nights":
			if specifier != "" {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					check := false
					if specifier == "Dark" {
						check = !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					} else if specifier == "Light" {
						check = strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					}
					return check
				}
			}
		case "Fallen Empires", "Asia Pacific Land Program":
			if specifier != "" {
				fields := strings.Fields(specifier)
				artist := fields[0]
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.Contains(card.Artist, artist)
				}
			}
		case "European Land Program":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return card.FlavorText == specifier
			}
		case "Homelands":
			if strings.HasPrefix(specifier, "Quote") {
				author := strings.Replace(specifier, "Quote ", "", 1)
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasSuffix(card.FlavorText, author)
				}
			}
		case "Unstable":
			if specifier != "" && specifier != "Used" {
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
