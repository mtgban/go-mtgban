package cardkingdom

import (
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

var promosetTable = map[string]string{
	"2017 Gift Pack":   "G17",
	"2018 Gift Pack":   "G18",
	"Convention Foil":  "PURL",
	"Dragonfury Promo": "PTKDF",
	"Resale Foil":      "PRES",
	"Store Foil":       "PRES",
	"Ugin's Fate":      "UGIN",
}

var codeFixupTable = map[string]string{
	// planeshift
	"PPLS": "PLS",
	// magic fests
	"F19": "PF19",
	"F20": "PF20",
	"P20": "PF20",
	// arena league 96
	"PAL96": "PARL",
	// shadowmoor prerelease
	"PSHM": "PPRE",
	// new phyrexia promos
	"PNHP": "PNPH",
	// jss
	"PJJT": "PSUS",
	// duels of the pw
	"D15": "PDP14",
	// resale
	"PPRM": "PRES",
}

func (ck *Cardkingdom) parseSetCode(c *ckCard) (setCode string) {
	setCode = c.SetCode

	// Update the edition field when the code is found
	defer func() {
		c.Edition = ck.db[setCode].Name
	}()

	if c.Edition == "Ultimate Box Topper" {
		setCode = "PUMA"
		return
	}
	if c.Edition == "Mystery Booster" {
		setCode = "MB1"
		if c.Variation == "Not Tournament Legal" {
			setCode = "CMB1"
		}
		return
	}
	if setCode == "SLD" {
		num, _ := strconv.Atoi(c.Number)
		if num >= 500 {
			setCode = "PSLD"
		}
		return
	}
	if strings.HasPrefix(c.Edition, "Promo Pack") {
		switch c.Name {
		// These are their own set
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setCode = "PPP1"
			return
		// These are the cards with a special frame, but no pw stamp
		case "Negate", "Disfigure", "Flame Sweep", "Thrashing Brontodon", "Corpse Knight":
			setCode = "PM20"
			if strings.HasSuffix(c.Number, "p") {
				c.Number = c.Number[:len(c.Number)-1]
			}
		// These are the ELD promo cards counted as being part of the main set
		case "Inspiring Veteran", "Kenrith's Transformation", "Glass Casket",
			"Slaying Fire", "Improbable Alliance":
			if strings.HasSuffix(c.Number, "p") {
				c.Number = c.Number[:len(c.Number)-1]
			}
		// Same for TBD
		case "Alseid of Life's Bounty", "Thirst for Meaning", "Gray Merchant of Asphodel",
			"Thrill of Possibility", "Wolfwillow Haven":
			setCode = "THB"

		// Numbers are simply off for these two
		case "Giant Killer":
			c.Number = "14p"
		case "Fae of Wishes":
			c.Number = "44p"
		}

		// Make sure that we are in the 'Promos' version of their own sets when
		// the collector number ends with a promo suffix
		if len(setCode) == 3 && strings.HasSuffix(c.Number, "p") {
			setCode = "P" + setCode
		}

		return
	}

	switch {
	// Fixup some unhinged marked as unglues
	case setCode == "UGL" && strings.HasPrefix(c.Edition, "Unhinged"):
		setCode = "UNH"
	// Give precedence to Variation field for this case, too many codes otherwise
	case c.Variation == "Holiday Foil":
		setCode = "HHO"
	// Bundles are part of the normal set, just drop the P if present
	// except for a single one card
	case c.Variation == "Bundle Foil" && strings.HasPrefix(setCode, "P") && c.Name != "Chandra's Regulator":
		setCode = setCode[1:]
	// The GameDay cards need to be part of Promos from their current set,
	// also the collector number needs to be numeric only
	case (c.Variation == "Game Day Extended Art" ||
		c.Variation == "Game Day Extended" ||
		c.Variation == "Gameday Extended Art" ||
		c.Variation == "Game Day Promo") && strings.HasSuffix(c.Number, "p"):
		if len(setCode) == 3 {
			setCode = "P" + setCode
		}
		c.Number = c.Number[:len(c.Number)-1]
	// Ravnica weekend 1
	case strings.HasPrefix(c.Variation, "Ravnica Weekend - A"):
		setCode = "PRWK"
	// Ravnica weekend 2
	case strings.HasPrefix(c.Variation, "Ravnica Weekend - B"):
		setCode = "PRW2"
	// Ixalan Treasure Chest
	case setCode == "PXLN" && c.Variation == "Buy-a-Box Foil":
		setCode = "PXTC"
	// AltArt fast lands
	case setCode == "PBFZ" && c.Variation == "Alternate Art":
		setCode = "PSS1"
	// As last resort, we could have a simple replacement
	default:
		code, found := codeFixupTable[setCode]
		if found {
			setCode = code
		}
	}

	switch c.Name {
	case "Crucible of Worlds":
		if setCode == "" {
			setCode = "PWOR"
		}
	// These are the buy-a-box cards that are part of the original set, dropping
	// the P would enough, but the Variation field is too generic, so we need to
	// list them all unfortunately
	case "Firesong and Sunspeaker",
		"Nexus of Fate",
		"The Haunt of Hightower",
		"Impervious Greatwurm",
		"Tezzeret, Master of the Bridge",
		"Rienne, Angel of Rebirth":
		setCode = setCode[1:]
	// These are convention promos that are part of the Promo set, but since they
	// were distributed at Convention, the table promosetTable would incorrectly
	// assign a "Convention Promo" edition
	case "Deeproot Champion",
		"Death Baron",
		"Nightpack Ambusher":
	// For all other cases, lookup this table for any remaining substitution
	default:
		code, found := promosetTable[c.Variation]
		if found {
			setCode = code
		}
	}

	return
}

func (ck *Cardkingdom) parseNumber(c *ckCard) (cardName string, numberCheck mtgban.NumberCheckFunc) {
	cardName = c.Name
	setName := c.Edition
	number := ""

	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return strings.ToLower(card.Number) == number
			}
		}

		// Needed for some funny cards
		variants := mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		switch {
		case strings.Contains(cardName, " // "):
			cn := strings.Split(cardName, " // ")
			cardName = cn[0]
		}
	}()

	switch setName {
	case "Unhinged",
		// These editions feature a different collector number syntax
		// (star)(number) but only for some special cards
		"Magic 2010 Promos",
		"Magic 2011 Promos",
		"Magic 2012 Promos",
		"Magic 2013 Promos",
		"Zendikar Promos",
		"Rise of the Eldrazi Promos",
		"Mirrodin Besieged Promos",
		"Scars of Mirrodin Promos",
		"New Phyrexia Promos",
		"Dark Ascension Promos",
		"Avacyn Restored Promos",
		"Return to Ravnica Promos",
		"Dragon's Maze Promos",
		"Gatecrash Promos",
		"Theros Promos",
		"Born of the Gods Promos",
		"Journey into Nyx Promos",

		// The following editions have random letters or years in the number
		"Ultimate Box Topper",

		"Mystery Booster",
		"Mystery Booster Playtest Cards",

		"Junior Super Series",
		"Junior Series Europe",
		"Junior APAC Series",

		"MagicFest 2019",
		"MagicFest 2020",
		"Judge Gift Cards 2014",
		"Judge Gift Cards 2019",

		"Pro Tour Promos",
		"Grand Prix Promos",
		"World Magic Cup Qualifiers",
		"Nationals Promos",

		"Resale Promos",
		"Prerelease Events",
		"Happy Holidays",
		"URL/Convention Promos",
		"Tarkir Dragonfury",
		"Ugin's Fate",
		"M20 Promo Packs",
		"Duels of the Planeswalkers 2014 Promos",

		"Collectors’ Edition", "Battlebond Promos",
		"Intl. Collectors’ Edition":
		// Ignore the number entirely
	case "Classic Sixth Edition", "Eighth Edition", "Ninth Edition", "Deckmasters",
		"Duel Decks: Elves vs. Goblins", "Coldsnap Theme Decks":
		// Ignore the number only if it's not a land
		switch cardName {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			number = c.Number
		}
	// The Meld cards have some additional suffices while CK has not
	case "Eldritch Moon",
		"From the Vault: Transform",
		"Eldritch Moon Promos":
		number = c.Number
		// Remove any prefix from the number
		number = strings.Replace(number, "a", "", 1)
		number = strings.Replace(number, "b", "", 1)
		// Prepare a special function to check these cards
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			check := strings.HasPrefix(card.Number, number)
			if strings.Contains(c.Variation, "Prerelease") {
				check = check && strings.HasSuffix(card.Number, "s")
			}
			return check
		}
		return
	case "Unsanctioned":
		// Plains has a wrong SKU
		switch c.Variation {
		case "087 A":
			number = "87"
		case "088 - B":
			number = "88"
		default:
			number = c.Number
		}
	case "War of the Spark", "War of the Spark Promos", "Duel Decks: Jace vs. Chandra":
		number = c.Number
		number = strings.Replace(number, "a", "s", 1)
		number = strings.Replace(number, "jp", mtgjson.SuffixSpecial, 1)
	case "Modern Horizons", "Modern Horizons Promos":
		number = c.Number
		number = strings.Replace(number, "p", "", 1)
		number = strings.Replace(number, "b", "", 1)
	case "Magic 2015 Promos":
		number = c.Number
		number = strings.Replace(number, "s", "", 1)
		number = strings.Replace(number, "a", "", 1)
		number = strings.Replace(number, "b", "", 1)
	case "Hour of Devastation Promos":
		number = c.Number
		number = strings.Replace(number, "a", "s", 1)
		number = strings.Replace(number, "b", "", 1)
	default:
		number = c.Number
	}

	if strings.Contains(c.Variation, "Prerelease") {
		number = strings.Replace(number, "a", "s", 1)
		if c.Variation == "July 4 Prerelease" {
			number = strings.Replace(c.Number, "a", "", 1)
		}
		return
	}
	if strings.Contains(c.Variation, "Launch") {
		number = strings.Replace(number, "b", "", 1)
		return
	}
	if strings.HasPrefix(setName, "World Championship Decks") ||
		setName == "Pro Tour Collector Set" {
		// Wrong number is wrong
		if c.Name == "Volrath's Stronghold" {
			number = "143"
		}
		// Use a very relaxed way to check the number
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			return strings.Contains(card.Number, number)
		}
		return
	}
	if setName == "Guilds of Ravnica" && strings.Contains(c.Name, "Guildgate") {
		fields := strings.Fields(c.Variation)
		number = fields[0]
		return
	}

	no, found := setVariants[setName][c.Name][c.Variation]
	if found {
		number = no
		return
	}

	return
}
