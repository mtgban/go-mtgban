package cardshark

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Eight and a Half Tails": "Eight-and-a-Half-Tails",

	"B.F.M.  1": "B.F.M. (Big Furry Monster)",
	"B.F.M.  2": "B.F.M. (Big Furry Monster)",
}

var editionTable = map[string]string{
	"Eighth Edition Core Set Pack Cards": "Eighth Edition",
	"From the Vault Annihilation (2014)": "From the Vault Annihilation",
	"Ugin's Fate promos":                 "Ugin's Fate",
}

var promoTable = map[string]string{
	// Prerelease Stamped
	"Obelisk of Alara": "Launch Parties",

	// Promotional Gateway
	"Emeria Angel":        "Zendikar Promos",
	"Hada Freeblade":      "Worldwake Promos",
	"Staggershock":        "Rise of the Eldrazi Promos",
	"Hellspark Elemental": "Wizards Play Network 2009",
	"Marisi's Twinclaws":  "Wizards Play Network 2009",
	"Woolly Thoctar":      "Wizards Play Network 2008",
	"Naya Sojourners":     "Magic 2010 Promos",
	"Reckless Wurm":       "Gateway 2007",

	// Promotional Other
	"Sprouting Thrinax":   "Wizards Play Network 2008",
	"Imperious Perfect":   "Champs and States",
	"Scent of Cinder":     "Magazine Inserts",
	"Fireball":            "Magazine Inserts",
	"Treetop Village":     "Summer of Magic",
	"Storm Entity":        "Release Events",
	"Azorius Guildmage":   "Release Events",
	"Ghost-Lit Raider":    "Release Events",
	"Hedge Troll":         "Release Events",
	"Shriekmaw":           "Release Events",
	"Rukh Egg":            "Release Events",
	"Force of Nature":     "Release Events",
	"Earwig Squad":        "Launch Parties",
	"Figure of Destiny":   "Launch Parties",
	"Knight of New Alara": "Launch Parties",

	// Promotional Friday Night Magic
	"Warleader's Helix": "Friday Night Magic 2014",
}

// Some notes make no sense, try to normalize them
var customNotesTable = map[string]string{
	"Foil NOT the promo just foil":  "Foil",
	"extended art Promo non foil":   "Promo Pack",
	"welcome deck 16 promo":         "Welcome Deck 2016",
	"Foil Jan 22 Promo":             "Foil Release Promo",
	"FNM promo pack stamp non foil": "Promo Pack",
}

// Some of these are hardcoded with whatever notes where available
var notesTable = map[string]string{
	"Kamahl, Pit Fighter":     "15th Anniversary Cards",
	"Ancient Crab":            "Amonkhet",
	"Immaculate Magistrate":   "Duels of the Planeswalkers",
	"Savage Lands":            "Friday Night Magic 2011",
	"Acidic Slime":            "Friday Night Magic 2012",
	"Despise":                 "Friday Night Magic 2012",
	"Dimir Charm":             "Friday Night Magic 2013",
	"Suspension Field":        "Friday Night Magic 2015",
	"Orator of Ojutai":        "Friday Night Magic 2015",
	"Frost Walker":            "Friday Night Magic 2015",
	"Blighted Fen":            "Friday Night Magic 2016",
	"Nissa's Pilgrimage":      "Friday Night Magic 2016",
	"Clash of Wills":          "Friday Night Magic 2016",
	"Smash to Smithereens":    "Friday Night Magic 2016",
	"Fortune's Favor":         "Friday Night Magic 2017",
	"Figure of Destiny":       "Launch Parties",
	"Magister of Worth":       "Launch Parties",
	"Hero's Downfall":         "Fate Reforged Clash Pack",
	"Warmonger":               "Magazine Inserts",
	"Lightning Hounds":        "Magazine Inserts",
	"Fated Intervention":      "Magic 2015 Clash Pack",
	"Font of Fertility":       "Magic 2015 Clash Pack",
	"Prophet of Kruphix":      "Magic 2015 Clash Pack",
	"Prognostic Sphinx":       "Magic 2015 Clash Pack",
	"Hydra Broodmaster":       "Magic 2015 Clash Pack",
	"Valorous Stance":         "Magic Origins Clash Pack",
	"Phyrexian Ingester":      "New Phyrexia",
	"Wren's Run Packmaster":   "Prerelease Events",
	"Retaliator Griffin":      "Resale Promos",
	"Genesis Hydra":           "Resale Promos",
	"Jaya Ballard, Task Mage": "Release Events",
	"Force of Nature":         "Release Events",
	"Debilitating Injury":     "Ugin's Fate",
	"Sultai Emissary":         "Ugin's Fate",
	"Dragon Fodder":           "Tarkir Dragonfury",
	"Kor Duelist":             "Wizards Play Network 2009",
	"Curse of Wizardry":       "Wizards Play Network 2010",
	"Golem's Heart":           "Wizards Play Network 2010",
	"Tormented Soul":          "Wizards Play Network 2011",
	"Fling":                   "Wizards Play Network 2011",
}

func preprocess(cardName, edition, number, notes string) (*mtgmatcher.Card, error) {
	variant := number

	if mtgmatcher.IsToken(cardName) {
		return nil, errors.New("not single")
	}

	switch cardName {
	// No way to tell these apart
	case "Brothers Yamazaki",
		"Amateur Auteur",
		"Beast in Show",
		"Everythingamajig",
		"Extremely Slow Zombie",
		"Garbage Elemental",
		"Ineffable Blessing",
		"Knight of the Kitchen Sink",
		"Novellamental",
		"Secret Base",
		"Sly Spy",
		"Target Minotaur",
		"Very Cryptic Command":
		return nil, errors.New("dupe")
	// CS grabs card names from Gatherer which treats splits cards differently
	// so just ignore the back side.
	case "Cooperate",
		"Dawn",
		"Conduit of Emrakul",
		"Dronepack Kindred",
		"Erupting Dreadwolf",
		"Extricator of Flesh",
		"Grisly Anglerfish",
		"Ulvenwald Abomination",
		"Jace, Telepath Unbound",
		"Chandra, Roaring Flame",
		"Burn",
		"Concoct",
		"Statue",
		"Finality",
		"Deploy",
		"Carnage",
		"Incongruity",
		"Replicate",
		"Warden",
		"Atzal, Cave of Eternity",
		"Ancient of the Equinox",
		"Branded Howler",
		"Lambholt Butcher",
		"Chop Down",
		"Dizzying Swoop",
		"Heart's Desire",
		"Battle Display",
		"Seasonal Ritual",
		"Rage of Winter",
		"Rider in Need",
		"Boulder Rush",
		"Haggle",
		"Alter Fate",
		"Curry Favor",
		"Treats to Share",
		"Venture Deeper",
		"On Alert",
		"Harvest Fear",
		"Oaken Boon",
		"Shield's Might",
		"Murkwater Pathway (Alternate Art)":
		return nil, errors.New("split")
	}

	fields := mtgmatcher.SplitVariants(cardName)
	cardName = fields[0]
	extra := ""
	if len(fields) > 1 {
		extra = fields[1]
	}
	// Numbers are surprisingly accurate, so assign this field only when
	// number is not available
	if number == "" {
		variant = extra
	}

	switch extra {
	case "Complete Set":
		return nil, errors.New("non mtg")
	case "Jr Super Series":
		variant = "JSS"
	case "Nyx Game Day Promo":
		variant = "Hero's Path"
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	lutName, found = editionTable[edition]
	if found {
		edition = lutName
	}

	switch edition {
	case "Arabian Nights":
		if variant == "Version 2" {
			variant = "dark"
		}
	case "European Lands":
		variant = strings.Replace(variant, "Euro 1", "", 1)
		variant = strings.Replace(variant, "Euro 2", "", 1)
		variant = strings.Replace(variant, "Euro 3", "", 1)
	case "Portal":
		if variant == "Version 2" {
			variant = "reminder text"
		}
	case "Promotional Arena League":
		variant = extra
	case "Promotional DCI Judge":
		switch cardName {
		case "Vindicate":
			if variant == "DCI Judge v2" {
				variant = "Judge Gift Cards 2013"
			}
		}
	case "Promotional Gateway",
		"Promotional Friday Night Magic",
		"Promotional Other",
		"Prerelease Stamped":
		switch cardName {
		case "Fling",
			"Sylvan Ranger":
			return nil, errors.New("dupe")
		case "Goblin Warchief":
			variant = extra
		}

		lutName, found = promoTable[cardName]
		if found {
			edition = lutName
		}
	}

	lutName, found = customNotesTable[notes]
	if found {
		notes = lutName
	}

	foil := mtgmatcher.Contains(notes, "Foil") && !mtgmatcher.Contains(notes, "non")

	if mtgmatcher.Contains(notes, "Mystery Booster") {
		edition = "Mystery Booster"
	}

	isPromo := mtgmatcher.Contains(notes, "Promo") ||
		mtgmatcher.Contains(notes, "Prerelease") ||
		mtgmatcher.Contains(notes, "Bundle") ||
		mtgmatcher.Contains(notes, "FNM")

	isCond := mtgmatcher.Contains(notes, "NM") ||
		mtgmatcher.Contains(notes, "very good") ||
		mtgmatcher.Contains(notes, "otherwise")

	if isPromo {
		if (notes == "Foil Promo" && cardName == "Sword-Point Diplomacy") ||
			(notes == "promo stamp" && cardName == "Thassa's Oracle") {
			return nil, errors.New("dupe")
		}

		lutName, found = notesTable[cardName]
		if found {
			edition = lutName
		} else {
			if !isCond {
				edition = "Promos"
			}
			if cardName == "Noosegraf Mob" && notes == "Foil Promo Pack" {
				notes = "Intro Pack Promo"
			}
			variant = strings.Replace(notes, "FMN", "FNM", 1)
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
	}, nil
}
