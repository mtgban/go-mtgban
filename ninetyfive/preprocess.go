package ninetyfive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"B.F.M. (Big Furry Monster Left)":  "B.F.M. (Big Furry Monster)",
	"B.F.M. (Big Furry Monster Right)": "B.F.M. (Big Furry Monster)",
}

var mediaTable = map[string]string{
	"Rakdos Firewheeler": "PRNA",

	"Hall of Triumph": "THP3",

	"Canopy Vista":     "PSS1",
	"Cinder Glade":     "PSS1",
	"Prairie Stream":   "PSS1",
	"Smoldering Marsh": "PSS1",
	"Sunken Hollow":    "PSS1",
}

func preprocess(card NFCard, edition string, foil bool) (*mtgmatcher.Card, error) {
	cardName := card.Name
	variant := ""
	if card.Number != 0 {
		variant = fmt.Sprint(card.Number)
	}

	if mtgmatcher.IsToken(cardName) {
		return nil, errors.New("token")
	}

	switch edition {
	case "Friday Night Magic",
		"Grand Prix",
		"Happy Holidays",
		"Judge Gift Program",
		"Magic Game Day",
		"Media Inserts",
		"Prerelease Events":
		// Drop any number information
		variant = ""
		// See if it's a known wrong card or a judge promo
		switch cardName {
		case "Ajani Steadfast",
			"Gideon, Ally of Zendikar",
			"Nissa, Worldwaker":
			return nil, errors.New("does not exist")
		case "Demonic Tutor":
			variant = "2008"
		case "Vindicate":
			variant = "2007"
		case "Wasteland":
			variant = "2010"
		default:
			ed, found := mediaTable[cardName]
			if found {
				edition = ed
			}
		}
	case "Arena League":
		switch cardName {
		case "Evolving Wilds",
			"Reliquary Tower":
			return nil, errors.New("does not exist")
		case "Unclaimed Territory":
			edition = "PXLN"
		}
	case "WAR Alt-art Promos":
		edition = "WAR"
		variant = "Japanese"
		// The site is buggy and lists cards that should not be in this set
		if len(mtgmatcher.MatchInSet(cardName, "WAR")) != 2 {
			return nil, errors.New("does not exist")
		}
	case "PW Stamped Cards ":
		edition = "ignored"
		variant = "Promo Pack"
	case "Signature Spellbook 1: Jace":
		edition = "Signature Spellbook: Jace"
	case "Signature Spellbook 2: Gideon":
		edition = "Signature Spellbook: Gideon"
	case "Champions of Kamigawa":
		if !mtgmatcher.IsBasicLand(cardName) {
			variant = ""
		}
		if cardName == "Brothers Yamazaki" {
			return nil, errors.New("dupe")
		}
	case "Deckmasters":
		if !mtgmatcher.IsBasicLand(cardName) {
			variant = ""
		}
	}

	// Boosterfun stuff is relagated to a Promos tag
	if strings.HasSuffix(edition, "Promos") {
		edition = strings.TrimSuffix(edition, " Promos")
		// Drop incorrect BaB/BAB tags
		if strings.Contains(cardName, "(") {
			vars := mtgmatcher.SplitVariants(cardName)
			cardName = vars[0]
		}
		// Only set that outlines the promos like this
		if edition == "Core Set 2020" {
			edition = "Core Set 2020 Promos"
			variant = "Promo Pack"
		}
	}

	if strings.HasSuffix(cardName, "BaB") {
		cardName = strings.TrimSuffix(cardName, " BaB")
		variant = "BaB"
	}

	if strings.Contains(cardName, "(") {
		vars := mtgmatcher.SplitVariants(cardName)
		cardName = vars[0]
		if len(vars) > 1 {
			switch edition {
			case "Commander Anthology 2018",
				"Guilds of Ravnica",
				"Ravnica Allegiance":
			default:
				variant = vars[1]
			}
		}
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}