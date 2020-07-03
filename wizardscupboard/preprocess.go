package wizardscupboard

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgdb"
)

var cardTable = map[string]string{
	"Ahn-Crop Chrasher":          "Ahn-Crop Crasher",
	"Coordianted Assault":        "Coordinated Assault",
	"Commencment of Festivities": "Commencement of Festivities",
	"Acolyte of the Affliction":  "Acolyte of Affliction",
	"Evoltuionary Escalation":    "Evolutionary Escalation",
	"Morkut Banshee":             "Morkrut Banshee",
	"Malicious Afflicition":      "Malicious Affliction",
	"Ob Nixillis Reignited":      "Ob Nixilis Reignited",
	"Maelstorm Nexus":            "Maelstrom Nexus",
	"Norin the Way":              "Norin the Wary",
	"Fellhide Minotaur":          "Felhide Minotaur",
	"Phyrexian Ingestor":         "Phyrexian Ingester",
	"Welking Guide":              "Welkin Guide",
	"Dinousaur Stampede":         "Dinosaur Stampede",
	"Zirda, the Dawnmaker":       "Zirda, the Dawnwaker",
	"Huatli, Heart of the Sun":   "Huatli, the Sun's Heart",
	"True-Faith Censor":          "True-Faith Censer",
	"Backwoods Survivlists":      "Backwoods Survivalists",
	"Tolsimir, Friend of Wolves": "Tolsimir, Friend to Wolves",
	"Faithelss Looting":          "Faithless Looting",
	"Wily Bander":                "Wily Bandar",
	"Avacyn's Judgement":         "Avacyn's Judgment",
	"Bloodthristy Blade":         "Bloodthirsty Blade",
	"Burning Vengence":           "Burning Vengeance",
	"Path to Discovery":          "Path of Discovery",
	"Nullmage Adocate":           "Nullmage Advocate",
	"Silhana Edgewalker":         "Silhana Ledgewalker",
	"Prakhata Club Secruity":     "Prakhata Club Security",
	"Holy Justicar":              "Holy Justiciar",
	"Strength of the Fallen":     "Strength from the Fallen",
	"Kozliek's Shrieker":         "Kozilek's Shrieker",
	"Boros Challanger":           "Boros Challenger",
	"Molten Slaghaep":            "Molten Slagheap",
	"Rakdos Guuildgate":          "Rakdos Guildgate",
	"Witch's Vengence":           "Witch's Vengeance",
	"Spawnrithe":                 "Spawnwrithe",
	"Hinterland Drakr":           "Hinterland Drake",
	"Warmonger ":                 "Warmonger",
	"Fierce Impact":              "Fierce Empath",

	"Nighthowler GAME DAY FULL ART": "Nighthowler",

	"The Ultimate Nightmare of WOTC Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",

	"Fire/Ice":                                 "Fire",
	"Who/What/When/Where/Why":                  "Who",
	"Cryptolith Fragment // Aura of Emrakul":   "Cryptolith Fragment // Aurora of Emrakul",
	"Docent of Perfection // Final Iteratioin": "Docent of Perfection // Final Iteration",
}

var promo2variant = map[string]string{
	"Bituminous Blast":   "Textless",
	"Brave the Elements": "Textless",
	"Infest":             "Textless",
	"Oxidize":            "Textless",
	"Terminate":          "Textless",
	"Zombify":            "Textless",
	"Volcanic Fallout":   "Textless",
	"Recollect":          "Textless",
	"Nameless Inversion": "Textless",

	"Experiment One":     "FNM",
	"Flaying Tendrils":   "FNM",
	"Ghor-Clan Rampager": "FNM",
	"Goblin Warchief":    "FNM 2016",
	"Longbow Archer":     "FNM",
	"Magma Spray":        "FNM",
	"Noose Constrictor":  "FNM",
	"Shock":              "FNM",
	"Sparksmith":         "FNM",
	"Teetering Peaks":    "FNM",
	"Mind Warp":          "FNM",
	"Crumbling Vestige":  "FNM",

	"Bloodcrazed Neonate": "WPN",
	"Boneyard Wurm":       "WPN",
	"Circle of Flame":     "WPN",
	"Sprouting Thrinax":   "WPN",
	"Pathrazer of Ulamog": "WPN",
	"Mind Control":        "WPN",
	"Woolly Thoctar":      "WPN",
	"Hellspark Elemental": "WPN",

	"Giant Badger":       "Book",
	"Scent of Cinder":    "Book",
	"Sewers of Estark":   "Book",
	"Warmonger ":         "Book",
	"Windseeker Centaur": "Book",

	"Fated Intervention":  "Clash Pack",
	"Font of Fertility":   "Clash Pack",
	"Sandsteppe Citadel":  "Clash Pack",
	"Reaper of the Wilds": "Clash Pack",

	"Balance":         "Judge",
	"Tradewind Rider": "Judge",

	"Man-o'-War":       "Arena",
	"Uktabi Orangutan": "Arena",

	"Angelic Skirmisher": "Resale Promos",
	"Ghost-Lit Raider":   "Release Promos",
	"Hall of Triumph":    "Hero's Path",
	"Lhurgoyf":           "Deckmasters",
	"Nighthowler":        "Game day Promo",
	"Rhox":               "Alt art",
	"Thran Quarry":       "Junior Super Series",
	"Write Into Being":   "Ugin's Fate",
}

func parseConditions(notes string) (string, error) {
	notes = strings.Replace(notes, " ", " ", -1)

	switch notes {
	case "",
		"soul separater",
		"foil",
		"card #141 of 140. only exists in foil.",
		"starter deck only",
		"4 versions":
		notes = "nm"
	case "line",
		"fine",
		"fine growth":
		notes = "fine best"
	case "f/g best",
		"good/ wavy only":
		notes = "fine good"
	}

	switch {
	case strings.Contains(notes, "in plastic"),
		strings.Contains(notes, "in celephane"),
		strings.Contains(notes, "in cellophane"):
		notes = "nm"
	case strings.Contains(notes, "white"),
		strings.Contains(notes, "blue"),
		strings.Contains(notes, "black"),
		strings.Contains(notes, "red"),
		strings.Contains(notes, "green"),
		strings.Contains(notes, "artifact"),
		strings.Contains(notes, "varients"),
		strings.Contains(notes, "multicolor"),
		strings.Contains(notes, "colorless"):
		notes = "nm"
	case (strings.Contains(notes, "german") ||
		strings.Contains(notes, "asian") ||
		strings.Contains(notes, "spanish") ||
		strings.Contains(notes, "italian") ||
		strings.Contains(notes, "french") ||
		strings.Contains(notes, "russian") ||
		strings.Contains(notes, "chinese")) &&
		(strings.Contains(notes, "also") ||
			strings.Contains(notes, "1") ||
			strings.Contains(notes, "2")):
		notes = "nm"
	case strings.Contains(notes, "double sided"),
		strings.Contains(notes, "meld partner"),
		strings.Contains(notes, "even distribution"),
		strings.Contains(notes, "good also"),
		strings.Contains(notes, "fine also"):
		notes = "nm"
	}

	conditions := ""
	switch {
	case strings.Contains(notes, "m/nm"),
		strings.Contains(notes, "nm"),
		strings.Contains(notes, "near mint"):
		conditions = "NM"
	case strings.Contains(notes, "fine best"),
		(strings.Contains(notes, "fine") && strings.Contains(notes, "better")),
		(strings.Contains(notes, "fine") && strings.Contains(notes, "best")),
		(strings.Contains(notes, "fine") && strings.Contains(notes, "good")):
		conditions = "SP"
	case (strings.Contains(notes, "good") && strings.Contains(notes, "best")),
		(strings.Contains(notes, "good") && strings.Contains(notes, "better")),
		strings.Contains(notes, "good/poor best"):
		conditions = "MP"
	case strings.Contains(notes, "poor/marked"),
		strings.Contains(notes, "binder bend"),
		strings.Contains(notes, "water damage"),
		strings.Contains(notes, "creased"),
		strings.Contains(notes, "stained"),
		strings.Contains(notes, "poor"),
		strings.Contains(notes, "signed"):
		conditions = "HP"
	default:
		if !strings.Contains(notes, "also") {
			return "", errors.New(notes)
		}
		conditions = "NM"
	}
	return conditions, nil
}

func preprocess(cardName, edition, notes string) (*mtgdb.Card, error) {
	ogName := cardName

	switch {
	case (strings.Contains(notes, "asian") ||
		strings.Contains(notes, "russian") ||
		strings.Contains(notes, "german") ||
		strings.Contains(notes, "italian") ||
		strings.Contains(notes, "itailian") ||
		strings.Contains(notes, "japanese")) &&
		(strings.Contains(notes, "only") ||
			!strings.Contains(notes, "also")):
		return nil, errors.New("non english")
	case strings.Contains(notes, "these cards are not normal size") ||
		strings.Contains(notes, "token"),
		strings.Contains(notes, "oversized"):
		return nil, errors.New("non mtg")
	case strings.Contains(cardName, " Punch Out"),
		strings.Contains(cardName, "Check List"),
		strings.Contains(cardName, "Checklist"),
		strings.Contains(cardName, "Construct // Clue"),
		strings.Contains(cardName, "Emblem"),
		strings.Contains(cardName, "Ixalan Jace Lands"),
		strings.Contains(cardName, "Zombie // Gold"),
		strings.Contains(strings.ToLower(cardName), "token"),
		strings.HasPrefix(cardName, "Art Series"):
		return nil, errors.New("non mtg")
	case strings.Contains(cardName, "Brothers Yamazaki"):
		return nil, errors.New("dupe")
	case strings.Contains(cardName, "ther Adept"),
		strings.HasPrefix(cardName, "Lim") && strings.HasSuffix(cardName, "'s Vault"):
		return nil, errors.New("unicode")
	}

	isFoil := strings.Contains(cardName, " (Foil)") || strings.Contains(notes, "foil")
	if isFoil {
		cardName = strings.Replace(cardName, " (Foil)", "", 1)
		cardName = strings.Replace(cardName, " foil", "", 1)
	}

	variant := ""
	vars := mtgdb.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		variant = vars[1]
	}
	vars = strings.Split(cardName, " - ")
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += vars[1]
	}

	if strings.HasSuffix(edition, "Extended Art") {
		edition = strings.TrimSuffix(edition, " Extended Art")
	}

	switch edition {
	case "Arabian Nights":
		if variant == "b" {
			variant = "light"
		} else if variant == "a" {
			variant = "dark"
		}
	case "Antiquities":
		if strings.Contains(cardName, ", ") {
			s := strings.Split(cardName, ", ")
			cardName = s[0]
			if len(s) > 1 {
				variant = s[len(s)-1]
			}
		}
	case "Ikoria Showcase":
		edition = "Ikoria: Lair of Behemoths"
		if strings.Contains(cardName, " - ") {
			s := strings.Split(cardName, " - ")
			cardName = s[0]
		}
	case "Portal":
		if strings.ToLower(variant) == "version 1" {
			variant = "No Flavor Text"
		} else if strings.ToLower(variant) == "version 2" {
			variant = "No Reminder Text"
		}
	case "Alliances":
		for _, num := range mtgdb.VariantsTable[edition][cardName] {
			if (variant == "" && strings.HasSuffix(num, "a")) ||
				(variant == "v. 2" && strings.HasSuffix(num, "b")) {
				variant = num
				break
			}
		}
	case "Unglued":
		if ogName == "B.F.M. (Big Furry Monster) Left" {
			cardName = "B.F.M."
			variant = "28"
		} else if ogName == "B.F.M. (Big Furry Monster) Right" {
			cardName = "B.F.M."
			variant = "29"
		}
	case "Unhinged":
		if cardName == "Erase" {
			cardName = "Erase (Not the Urza's Legacy One)"
		}
	case "Unstable":
		cardName = strings.Replace(cardName, "|", " ", -1)
	case "Guilds of Ravnica: Guild Kit":
		switch cardName {
		case "Archon of the Triumvirate",
			"Azorius Chancery",
			"Azorius Charm",
			"Azorius Guildmage",
			"Azorius Herald",
			"Dovescape",
			"Stoic Ephemera",
			"Windreaver",
			"Court Hussar",
			"Isperia, Supreme Judge":
			edition = "RNA Guild Kit"
		}
	case "Guilds of Ravnica", "Ravnica Allegiance":
		if strings.Contains(cardName, "Guildgate") {
			return nil, errors.New("dupe")
		}
	case "Chronicles":
		switch cardName {
		case "Urza's Power Plant", "Urza's Tower", "Urza's Mine":
			return nil, errors.New("dupe")
		}
	case "Oversize":
		return nil, errors.New("not single")
	case "Foreign BB":
		return nil, errors.New("not english")
	case "Promo Cards (Prerelease)":
		variant = "Prerelease"
	case "Promo Cards":
		if variant == "" {
			variant = notes
		}

		maybeVariant, found := promo2variant[cardName]
		if found {
			variant = maybeVariant
		} else {
			switch cardName {
			case "Llanowar Elves":
				if variant == "DCI Foil" {
					variant = "FNM"
				}
			case "Mana Crypt":
				if variant == "Arena" {
					variant = "Book Promo"
				}
			case "Fling":
				if variant == "DCI Promo" {
					variant = "WPN 2010"
				}
			case "Sorcerous Spyglass":
				return nil, errors.New("dupe")
			}
		}
	}

	switch variant {
	case "2 versions", "3 versions", "4 versions",
		"3 Versions", "4 Versions":
		return nil, errors.New("not unique")
	case "6th Prerelease":
		variant = "World Championship Foil"
	case "DCI", "FMN":
		variant = "FNM"
	case "15 Anniversary", "15th anniversary":
		variant = "15th Anniversary"
	case "Ravnica PRERE":
		variant = "Prerelease"
	case "Promo Promo":
		variant = "Promo Pack"
	case "WPN & Gateway":
		variant = "WPN 2010"
	case "Gift Pack":
		edition = "M19 Gift Pack"
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	return &mtgdb.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
