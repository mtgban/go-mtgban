package miniaturemarket

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	// Typos
	"Asylum Visitior":           "Asylum Visitor",
	"Fiesty Stegosaurus":        "Feisty Stegosaurus",
	"Torban, Thane of Red Fell": "Torbran, Thane of Red Fell",

	"Conspicious Snoop (Extended Art)": "Conspicuous Snoop (Extended Art)",

	"Cunning Bandit /Azamuki, Treachery Incarnate": "Cunning Bandit / Azamuki, Treachery Incarnate",

	// Funny cards
	"Who / What / When / Where / Why":       "Who",
	"'Rumors of My Death. . .''":            "\"Rumors of My Death . . .\"",
	"B.F.M. (Big Furry Monster Left Side)":  "B.F.M. (28)",
	"B.F.M. (Big Furry Monster Right Side)": "B.F.M. (29)",

	"The Ultimate Nightmare of Wizards of the Coast(R) Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",

	// Hero's path cards tagged as prerelease
	"Axe of the Warmonger (Pre-Release)": "Axe of the Warmonger (Hero's Path)",
	"Hall of Triumph (Pre-Release)":      "Hall of Triumph (Hero's Path)",
	"Lash of the Tyrant (Pre-Release)":   "Lash of the Tyrant (Hero's Path)",
	"The Avenger (Pre-Release)":          "The Avenger (Hero's Path)",
	"The Champion (Pre-Release)":         "The Champion (Hero's Path)",
	"The Destined (Pre-Release)":         "The Destined (Hero's Path)",
	"The Explorer (Pre-Release)":         "The Explorer (Hero's Path)",
	"The Harvester (Pre-Release)":        "The Harvester (Hero's Path)",
	"The Philosopher (Pre-Release)":      "The Philosopher (Hero's Path)",
	"The Slayer (Pre-Release)":           "The Slayer (Hero's Path)",
	"The Vanquisher (Pre-Release)":       "The Vanquisher (Hero's Path)",
	"The Warrior (Pre-Release)":          "The Warrior (Hero's Path)",

	// Promos
	"Sol Ring (Commander)":           "Sol Ring (MagicFest 2019)",
	"Stocking Tiger (Repack Insert)": "Stocking Tiger (misprint)",
}

var card2setTable = map[string]string{
	"Angelic Guardian (Gift Box)":     "M19 Gift Pack",
	"Immortal Phoenix (Gift Box)":     "M19 Gift Pack",
	"Rukh Egg (MTG 10th Anniversary)": "Release Events",
	"Serra Angel (DCI)":               "Wizards of the Coast Online Store",

	"Forest (Gift Box)":   "2017 Gift Pack",
	"Island (Gift Box)":   "2017 Gift Pack",
	"Mountain (Gift Box)": "2017 Gift Pack",
	"Plains (Gift Box)":   "2017 Gift Pack",
	"Swamp (Gift Box)":    "2017 Gift Pack",

	"Celestine Reef (Pre-Release)":             "Promotional Planes",
	"Horizon Boughs (WPN)":                     "Promotional Planes",
	"Mirrored Depths (WPN)":                    "Promotional Planes",
	"Tember City (WPN)":                        "Promotional Planes",
	"Stairs to Infinity (Launch Party)":        "Promotional Planes",
	"Tazeem (Launch Party)":                    "Promotional Planes",
	"Drench the Soil in Their Blood (WPN)":     "Promotional Schemes",
	"Imprison This Insolent Wretch (WPN)":      "Promotional Schemes",
	"Perhaps You've Met My Cohort (WPN)":       "Promotional Schemes",
	"Plots That Span Centuries (Launch Party)": "Promotional Schemes",
	"Your Inescapable Doom (WPN)":              "Promotional Schemes",

	"Demonic Tutor (Judge Rewards Anna Steinbauer)": "Judge Gift Cards 2020",
	"Demonic Tutor (Judge Rewards Daarken)":         "Judge Gift Cards 2008",
	"Vampiric Tutor (Judge Rewards Gary Leach)":     "Judge Gift Cards 2000",
	"Vampiric Tutor (Judge Rewards Lucas Graciano)": "Judge Gift Cards 2018",
	"Vindicate (Judge Rewards Karla Ortiz)":         "Judge Gift Cards 2013",
	"Vindicate (Judge Rewards Mark Zug)":            "Judge Gift Cards 2007",
	"Wasteland (Judge Rewards Carl Critchlow)":      "Judge Gift Cards 2010",
	"Wasteland (Judge Rewards Steve Belledin)":      "Judge Gift Cards 2015",
}

func preprocess(title, sku string) (*mtgmatcher.Card, error) {
	fields := strings.Split(title, " - ")
	cardName := fields[0]
	edition := fields[1]
	if strings.Contains(edition, " (") {
		if edition == "4th Edition (Alternate)" {
			return nil, errors.New("untracked edition")
		} else if strings.Contains(edition, "(Preorder)") {
			return nil, errors.New("too soon")
		}
		fields = mtgmatcher.SplitVariants(edition)
		edition = fields[0]
	}

	// Skip non-singles cards
	switch cardName {
	case "City's Blessing",
		"Companion",
		"Energy Reserve",
		"Experience Counter",
		"Manifest",
		"Morph",
		"On an Adventure",
		"Poison Counter",
		"The Monarch":
		return nil, errors.New("non-single card")
	default:
		switch {
		case strings.Contains(cardName, "Token"),
			strings.Contains(cardName, "Emblem"),
			strings.Contains(cardName, "Checklist Card"),
			strings.Contains(cardName, "Punch Card"),
			strings.Contains(cardName, "Oversized"):
			return nil, errors.New("non-single card")
		case strings.HasPrefix(cardName, "Mana Crypt") &&
			strings.Contains(cardName, "(Media Insert)") &&
			!strings.Contains(cardName, "(English)"):
			return nil, errors.New("non-english card")
		}
	}

	switch edition {
	case "Planechase 2009":
		set, err := mtgmatcher.GetSet("OHOP")
		if err != nil {
			return nil, err
		}
		for _, card := range set.Cards {
			if mtgmatcher.Equals(card.Name, cardName) {
				edition = "Planechase Planes"
				break
			}
		}
	case "Modern Horizons Art Series":
		return nil, errors.New("untracked edition")
	case "Legends":
		if strings.Contains(cardName, "Italian") {
			return nil, errors.New("non-english edition")
		}
	case "Portal Three Kingdoms":
		if strings.Contains(cardName, "Chinese") || strings.Contains(cardName, "Japanese") {
			return nil, errors.New("non-english edition")
		}
	case "Duel Decks: Jace vs. Chandra":
		if strings.Contains(cardName, "Japanese") {
			return nil, errors.New("non-english edition")
		}
	}

	if strings.Contains(cardName, " [") && strings.Contains(cardName, "]") {
		cardName = strings.Replace(cardName, "[", "(", 1)
		cardName = strings.Replace(cardName, "]", ")", 1)
	}

	cardName = strings.Replace(cardName, ") (", " ", -1)

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	ed, found := card2setTable[cardName]
	if found {
		edition = ed
	}

	variant := ""
	variants := mtgmatcher.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		variant = variants[1]
	}

	if strings.Contains(title, "(Collector Edition)") && variant == "Alternate Art" {
		variant = "Borderless"
	}

	// Need to discern duplicates of these cards
	if edition == "Promo" || edition == "Promo Pack" {
		sku = strings.TrimSuffix(sku, "-1NM")
		sku = strings.TrimSuffix(sku, "-3F")
		switch sku {
		case "M-650-124", "M-660-012": // Sorcerous Spyglass
			variant += " XLN"
		case "M-650-176", "M-660-016", // Sorcerous Spyglass
			"M-665-026", "M-655-153": // Fabled Passage
			variant += " ELD"
		case "M-664-053", "M-654-231": // Heroic Intervention
			variant += " AER"
		case "M-665-015", "M-665-016", "M-665-017", "M-665-018", "M-665-019", // Temples
			"M-655-142", "M-655-143", "M-655-144", "M-655-145", "M-655-146":
			variant += " M20"
		case "M-665-053", "M-655-169", // Fabled Passage
			"M-665-054", "M-665-055", "M-665-056", "M-665-057", "M-665-058", // Temples
			"M-655-170", "M-655-171", "M-655-172", "M-655-173", "M-655-174",
			"M-664-083", "M-654-377": // Heroic Intervention
			variant += " M21"
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
