package cardsphere

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var promoTable = map[string]string{
	"Reliquary Tower":   "PLGS",
	"Hangarback Walker": "PLGS",
	"Lightning Bolt":    "MagicFest 2019",
	"Sol Ring":          "MagicFest 2019",

	"Pristine Talisman":  "New Phyrexia Promos",
	"Dauntless Dourbark": "Gateway 2007",
}

var tagsTable = map[string]string{
	"Extended Art":    "Extended Art",
	"Etched Foil":     "Etched Foil",
	"Alternate Art":   "Borderless",
	"Godzilla Series": "Godzilla",
	"Alternate Frame": "Showcase",
	"Showcase":        "Showcase",
}

func preprocess(cardName, edition string) (*mtgmatcher.Card, error) {
	if mtgmatcher.IsToken(cardName) {
		return nil, errors.New("not singles")
	}
	switch cardName {
	case "Adaptive Enchantment",
		"Faceless Menace",
		"Exquisite Invention",
		"Merciless Rage",
		"Nature's Vengeance",
		"Mystic Intellect",
		"Subjective Reality",
		"Primal Genesis",
		"Angel | Demon (Double-Sided)",
		"Complete Set",
		"Guild Kit":
		return nil, errors.New("not singles")
	}

	variant := ""
	fields := mtgmatcher.SplitVariants(cardName)
	cardName = fields[0]
	if len(fields) > 1 {
		variant = strings.Join(fields[1:], " ")
	}

	switch variant {
	case "sealed":
		return nil, errors.New("non single")
	}

	if mtgmatcher.IsBasicLand(cardName) {
		switch edition {
		case "Battle for Zendikar", "Oath of the Gatewatch":
			variant += "a"
		case "Core 2021 - Promo Pack":
			return nil, errors.New("not exist")
		case "European Lands":
			variant = strings.Replace(variant, "UK", "U.K.", 1)
		case "Core 2019",
			"Core 2020",
			"Ixalan":
			if strings.HasSuffix(variant, "Full Art") {
				variant = strings.TrimSuffix(variant, " Full Art")
			}
		case "Showdown Promos":
			if variant == "" {
				edition = "XLN Standard Showdown"
			} else if variant == "2018" {
				edition = "M19 Standard Showdown"
			}
		}
	}

	for tag, repl := range tagsTable {
		if strings.HasSuffix(edition, tag) {
			if variant != "" {
				variant += " "
			}
			variant += repl
			edition = strings.TrimSuffix(edition, " - "+tag)
			break
		}
	}

	switch edition {
	case "Core 2019":
		if len(mtgmatcher.MatchInSet(cardName, "G18")) > 0 {
			edition = "G18"
		}
	case "Duels of the Planeswalkers Game Promos":
		switch cardName {
		case "Ascendant Evincar",
			"Immaculate Magistrate",
			"Verdant Force":
			return nil, errors.New("dupe")
		}
	case "Intro Packs":
		for _, code := range []string{"PORI", "PKTK", "PFRF", "PDTK", "PBFZ", "POGW", "PSOI", "PEMN"} {
			if len(mtgmatcher.MatchInSet(cardName, code)) > 0 {
				edition = code
				break
			}
		}
	case "Ikoria: Lair of Behemoths":
		switch cardName {
		case "Lukka, Coppercoat Outcast",
			"Narset of the Ancient Way",
			"Vivien, Monsters' Advocate":
		default:
			if variant == "Borderless" {
				variant = "Showcase"
			}
		}
	case "Judge Promos":
		if variant == "" {
			switch cardName {
			case "Demonic Tutor":
				variant = "2008"
			case "Vampiric Tutor":
				variant = "2000"
			}
		}
	case "Miscellaneous Promos",
		"Mirrodin Pure Preview",
		"States 2008":
		ed, found := promoTable[cardName]
		if found {
			edition = ed
		}
	case "SDCC":
		if variant == "" {
			switch cardName {
			case "Chandra, Torch of Defiance",
				"Gideon of the Trials",
				"Jace, Unraveler of Secrets":
				variant = "2017"
			}
		}
	case "Unglued":
		if cardName == "B.F.M. (Big Furry Monster)" {
			if variant == "left" {
				variant = "28"
			} else if variant == "right" {
				variant = "29"
			}
		}
	case "War of the Spark":
		if variant == "Alternate Art" {
			variant = "Japanese"
		}
	case "The List - Zendikar Rising":
		edition = "The List"
	case "Mystery Booster":
		if cardName == "In Oketra's Name" {
			return nil, errors.New("not exist")
		}
	case "Theros: Beyond Death - Promo Pack":
		if cardName == "Heroic Intervention" {
			variant = "Promo Pack"
			edition = "PAER"
		}
	case "Prerelease Promos":
		if strings.HasPrefix(cardName, "Temple of") {
			variant = "Prerelease"
			edition = "Core Set 2020 Promos"
		} else if cardName == "Heroic Intervention" {
			variant = "Prerelease"
			edition = "PAER"
		}
	case "Core 2021 - Prerelease Promos":
		variant = "Prerelease"
		edition = "Core Set 2021 Promos"
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
	}, nil
}