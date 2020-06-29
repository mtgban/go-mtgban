package facetoface

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgdb"
)

type ftfCard struct {
	URLId      string
	Key        string
	Name       string
	Edition    string
	Foil       bool
	Conditions string
	Price      float64
	Quantity   int
}

var tagsTable = []string{
	"2018 Grand Prix Promo",
	"Alternate Art Foil",
	"Arena 2001",
	"Bundle Promo",
	"Buy-a-Box Promo",
	"DCI Judge Promo",
	"Dark Frame Promo",
	"Draft Weekend Promo",
	"FNM2012",
	"Foil Beta Picture",
	"Game Day Promo",
	"IDW",
	"Intro Pack Promo",
	"JSS Promo",
	"Judge Academy Promo",
	"Judge Promo",
	"Magic League Promo",
	"Open House Promo",
	"Planechase 2016 Promo",
	"Planeswalker Deck Exclusive",
	"Planeswalker Weekend Promo",
	"Prerelease Promo",
	"Store Championship Promo",
	"Textless Player Rewards",
	"WMCQ Promo",
	"WPN",
	"Worlds 1999 Promo",
}

var cardTable = map[string]string{
	// Typos
	"Moutain (41)":                            "Mountain (41)",
	"Morbid Curiousity":                       "Morbid Curiosity",
	"Pir, Imaginitive Rascal (Release Promo)": "Pir, Imaginative Rascal (Release Promo)",
	"Questing Pheldagrif (PreRelease GREEK)":  "Questing Phelddagrif (Prerelease)",
	"Essence Symbiont":                        "Essence Symbiote",
	"Quartzwood Crusher":                      "Quartzwood Crasher",
	"Quartzwood Crusher (Extended Art)":       "Quartzwood Crasher (Extended Art)",

	"Battra, Terror of the City (Dirge Bat JP Alternate Art)": "Dirge Bat (Godzilla)",

	// Funny cards
	"_________":                    "_____",
	"Who/What/When/Where/Why":      "Who",
	"B.F.M. 1 (Big Furry Monster)": "B.F.M. (28)",
	"B.F.M. 2 (Big Furry Monster)": "B.F.M. (29)",
	"Our Market Research Shows That Players Like Really Long Card Names So We Make This Card to Have the Absolute Longest Card Name E": "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",

	// Lands
	"Plains (Portal 1)": "Plains (Portal)",

	// Adjust promos
	"Corpse Knight (2/3 Misprint)": "Corpse Knight (Misprint)",
	"Doubling Season (JR)":         "Doubling Season (Judge)",
	"Underworld Dreams Promo":      "Underworld Dreams (2HG)",

	"Sylvan Ranger (WPN)":          "Sylvan Ranger (WPN 2010)",
	"Sylvan Ranger (astral)":       "Sylvan Ranger (WPN 2011)",
	"Fling (WPN DCI)":              "Fling (WPN 2010)",
	"Fling (WPN Star)":             "Fling (WPN 2011)",
	"Shrapnel Blast (FNM 2009)":    "Shrapnel Blast (FNM 2008)",
	"Elvish Mystic (FNM 2013)":     "Elvish Mystic (FNM 2014)",
	"Warleader's Helix (FNM 2013)": "Warleader's Helix (FNM 2014)",

	"Ajani Steadfast (SDCC Exclusive Promo M15)":            "Ajani Steadfast (SDCC 2014)",
	"Garruk, Apex Predator (SDCC Exclusive Promo M15)":      "Garruk, Apex Predator (SDCC Exclusive Promo 2014)",
	"Jace, the Living Guildpact (SDCC Exclusive Promo M15)": "Jace, the Living Guildpact (SDCC 2014)",
	"Liliana Vess (SDCC Exclusive Promo M15)":               "Liliana Vess (SDCC 2014)",
	"Nissa, Worldwaker (SDCC Exclusive Promo M15)":          "Nissa, Worldwaker (SDCC 2014)",

	"Lavinia, Azorius Renegade Ravnica Weekend (Store Championship Promo)": "Lavinia, Azorius Renegade (Store Championship Promo)",
}

var card2setTable = map[string]string{
	"Demonic Tutor (DCI Judge Promo)":             "Judge Gift Cards 2008",
	"Demonic Tutor (Judge Academy Promo)":         "Judge Gift Cards 2020",
	"Vampiric Tutor (DCI Judge Promo Old Border)": "Judge Gift Cards 2000",
	"Hall of Triumph (Game Day Promo)":            "Journey into Nyx Hero's Path",
	"Warmonger (Player's Guide)":                  "Magazine Inserts",
	"Cast Down (Japanese Promo)":                  "Magazine Inserts",

	"Kenrith, the Returned King (Collector Pack Exclusive)": "Throne of Eldraine",
	"Pristine Talisman (Mirrodin Pure Preview Promo)":       "New Phyrexia Promos",
}

var promocard2setTable = map[string]string{
	"Aegis Angel":         "Welcome Deck 2016",
	"Air Servant":         "Welcome Deck 2016",
	"Borderland Marauder": "Welcome Deck 2016",
	"Cone of Flame":       "Welcome Deck 2016",
	"Disperse":            "Welcome Deck 2016",
	"Incremental Growth":  "Welcome Deck 2016",
	"Marked by Honor":     "Welcome Deck 2016",
	"Mind Rot":            "Welcome Deck 2016",
	"Walking Corpse":      "Welcome Deck 2016",

	"Air Elemental":     "Welcome Deck 2017",
	"Bloodhunter Bat":   "Welcome Deck 2017",
	"Divine Verdict":    "Welcome Deck 2017",
	"Falkenrath Reaver": "Welcome Deck 2017",
	"Glory Seeker":      "Welcome Deck 2017",
	"Rootwalla":         "Welcome Deck 2017",
	"Shivan Dragon":     "Welcome Deck 2017",
	"Sphinx of Magosi":  "Welcome Deck 2017",
	"Standing Troops":   "Welcome Deck 2017",
	"Thundering Giant":  "Welcome Deck 2017",
	"Untamed Hunger":    "Welcome Deck 2017",
	"Victory's Herald":  "Welcome Deck 2017",
	"Wing Snare":        "Welcome Deck 2017",

	"Ainok Tracker":            "Ugin's Fate",
	"Altar of the Brood":       "Ugin's Fate",
	"Arashin War Beast":        "Ugin's Fate",
	"Arc Lightning":            "Ugin's Fate",
	"Briber's Purse":           "Ugin's Fate",
	"Debilitating Injury":      "Ugin's Fate",
	"Dragonscale Boon":         "Ugin's Fate",
	"Fierce Invocation":        "Ugin's Fate",
	"Formless Nurturing":       "Ugin's Fate",
	"Ghostfire Blade":          "Ugin's Fate",
	"Hewed Stone Retainers":    "Ugin's Fate",
	"Mystic of the Hidden Way": "Ugin's Fate",
	"Reality Shift":            "Ugin's Fate",
	"Grim Haruspex":            "Ugin's Fate",
	"Ruthless Ripper":          "Ugin's Fate",
	"Smite the Monstrous":      "Ugin's Fate",
	"Soul Summons":             "Ugin's Fate",
	"Sultai Emissary":          "Ugin's Fate",
	"Ugin's Construct":         "Ugin's Fate",
	"Ugin, the Spirit Dragon":  "Ugin's Fate",
	"Watcher of the Roost":     "Ugin's Fate",

	"Go for the Throat":  "Friday Night Magic 2011",
	"Encroaching Wastes": "Friday Night Magic 2014",
	"Selkie Hedge-Mage":  "Gateway 2008",
	"Gaze of Granite":    "IDW Comics 2013",
}

func preprocess(cardName, edition string) (string, string, error) {
	// Skip oversized card sets and unsupported ones
	switch {
	case strings.Contains(edition, "Oversized"):
		return "", "", fmt.Errorf("skipping oversized card set")
	default:
		switch edition {
		case "4th Black Border",
			"Alternate 4th Edition",
			"DD: Japanese Jace vs Chandra",
			"Foreign Limited - FBB",
			"Italian Legends",
			"Commander 2020",
			"MTG Arena Unused Codes",
			"Starter 2000", // scryfall missing too many cards
			"WCD: Blank Cards":
			return "", "", fmt.Errorf("skipping untracked set")
		}
	}

	// Quotes are not escaped
	if cardName == "" || strings.HasSuffix(cardName, ", ") {
		return "", "", fmt.Errorf("empty card name")
	}

	// Skip tokens and similar cards
	switch cardName {
	case "Lu Bu, Master at Arms - Foil - Prerelease Promo - 4/29/1999",
		"Legends Rules Card", "Energy Reserve", "Faerie Rogue",
		"Experience Counter", "Experience Card", "Poison Counter",
		"Goblin", "Pegasus", "Sheep", "Soldier", "Squirrel", "Zombie":
		return "", "", fmt.Errorf("not a real card")
	case "Mana Crypt - Book Promo (White Border Version)":
		return "", "", fmt.Errorf("non-english card")
	default:
		if strings.Contains(strings.ToLower(cardName), "token") ||
			strings.Contains(cardName, "Checklist") ||
			strings.Contains(cardName, "Oversized") ||
			strings.Contains(cardName, "Sealed Pack") ||
			strings.Contains(cardName, "Unopened") ||
			strings.Contains(cardName, "Salvat Promo") ||
			strings.Contains(cardName, "Chinese") ||
			strings.Contains(cardName, "Ultra Pro Puzzle Cards") ||
			strings.Contains(cardName, "Art Series") ||
			strings.Contains(strings.ToLower(cardName), "(scan ") ||
			strings.Contains(cardName, "Pre-release Guild Card") ||
			strings.Contains(cardName, "DEPRECATED") ||
			strings.Contains(cardName, "Emblem") {
			return "", "", fmt.Errorf("not a real card")
		}
		// Skip non-english versions of this card
		if strings.Contains(cardName, "Ajani Goldmane") &&
			strings.Contains(cardName, "Japanese") {
			return "", "", fmt.Errorf("non-english card")
		}
	}

	if cardName != "Sealed Fate" {
		if strings.Contains(cardName, " Sealed") {
			return "", "", fmt.Errorf("sealed card")
		} else if strings.Contains(cardName, "Sealed ") {
			return "", "", fmt.Errorf("sealed card")
		}
	}

	// Convert UTF-8 dash in ASHII dash
	cardName = strings.Replace(cardName, "–", "-", -1)

	// Drop pointeless tags
	cardName = strings.Replace(cardName, "-Foil", "", 1)

	// Replace any square parenthesis with proper ones
	cardName = strings.Replace(cardName, "[", "(", 1)
	cardName = strings.Replace(cardName, "]", ")", 1)

	// Make sure that variants are separated from the name
	parIndex := strings.Index(cardName, "(")
	if parIndex-1 > 0 && parIndex-1 < len(cardName) && cardName[parIndex-1] != ' ' {
		cardName = strings.Replace(cardName, "(", " (", 1)
	}

	// Split by -, rebuild the cardname in a standardized way
	variation := ""
	vars := strings.Split(cardName, " - ")
	cardName = vars[0]
	if len(vars) > 1 {
		variation = strings.Join(vars[1:], " ")
	}

	// Split by ()
	vars = mtgdb.SplitVariants(cardName)
	if vars[0] != "B.F.M." {
		cardName = vars[0]
		if len(vars) > 1 {
			if variation != "" {
				variation += " "
			}
			variation += strings.Join(vars[1:], " ")
		}
	}

	for _, tag := range []string{"FNM", "SDCC", "Holiday"} {
		cuts := mtgdb.Cut(cardName, tag)
		cardName = cuts[0]
		if len(cuts) > 1 {
			variation += " " + cuts[1]
		}
	}

	// Correctly put variants in the correct tag (within parenthesis)
	for _, tag := range tagsTable {
		cuts := mtgdb.Cut(cardName, tag)
		cardName = cuts[0]
		if len(cuts) > 1 {
			variation += " " + cuts[1]
		}
	}

	if strings.HasSuffix(cardName, "-") {
		cardName = cardName[:len(cardName)-1]
	}

	if cardName == "Sorcerous Spyglass" {
		if edition == "Promo Pack: Core Set 2020" {
			edition = "Promo Pack: Ixalan"
		}
	}

	if edition == "Zendikar" {
		switch cardName {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			if !strings.Contains(variation, "Foil") &&
				!strings.Contains(variation, "Extended Art") {
				variation += " non-full art"
			}
		}
	}

	// Flatten the variants
	if variation != "" {
		variation = strings.Replace(variation, "(", "", -1)
		variation = strings.Replace(variation, ")", "", -1)
		variation = strings.Replace(variation, "Foil ", "", 1)
		variation = strings.Replace(variation, " Foil", "", 1)
		variation = strings.Replace(variation, "Foil", "", 1)
		if variation != "" {
			cardName = cardName + " (" + strings.TrimSpace(variation) + ")"
		}
		variation = ""
	}

	// Fixup any expected errors
	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}
	ed, found := card2setTable[cardName]
	if found {
		edition = ed
	}

	// Split again by () to catch the tags above
	vars = mtgdb.SplitVariants(cardName)
	if len(vars) > 1 {
		variation = strings.Join(vars[1:], " ")
	}

	if variation == "" &&
		(edition == "Non-Foil Promos" || edition == "Foil Promos") {
		ed, found = promocard2setTable[cardName]
		if found {
			edition = ed
		}
	}

	// Variants like extended art have a specifier in the edition
	edVars := mtgdb.SplitVariants(edition)
	edition = edVars[0]

	if strings.HasPrefix(edition, "Box Sets - ") {
		edition = strings.Replace(edition, "Box Sets - ", "", 1)
	} else if strings.HasPrefix(edition, "Box Set - ") {
		edition = strings.Replace(edition, "Box Set - ", "", 1)
	}
	if strings.HasPrefix(edition, "FTV:") {
		edition = strings.Replace(edition, "FTV", "From the Vault", 1)
	}

	// Only pick one WCD per variant
	if strings.Contains(edition, "WCD") && strings.Contains(variation, "Version") {
		vars := mtgdb.SplitVariants(cardName)
		cardName = vars[0]
	}
	if strings.Contains(edition, "WCD") && unicode.IsDigit(rune(cardName[len(cardName)-1])) {
		cardName = cardName[:len(cardName)-1]
		variation += " " + string(cardName[len(cardName)-1])
	}

	if (cardName == "Liberate" && edition == "Invasion") ||
		(cardName == "Bind // Liberate" && edition == "Mystery Booster Playtest Cards") ||
		(cardName == "Bind" && edition == "Invasion") ||
		(cardName == "Shivan Dragon (Japanese Magazine Promo)" && edition == "Non-Foil Promos") ||
		(cardName == "Start // Finish" && edition == "Amonkhet") {
		return "", "", fmt.Errorf("card cannot be represented in mtgjson")
	}

	return cardName, edition, nil
}
