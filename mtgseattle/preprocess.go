package mtgseattle

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Pir, Imaginitive Rascal":      "Pir, Imaginative Rascal",
	"Battra, Terror of the City":   "Dirge Bat",
	"Mechagodzilla":                "Crystalline Giant",
	"B.F.M. 2 (Big Furry Monster)": "B.F.M. (Big Furry Monster Right)",
	"B.F.M. 1 (Big Furry Monster)": "B.F.M. (Big Furry Monster Left)",

	"_________": "_____",
}

var promoTags = []string{
	"2012 Holiday Promo",
	"Alternate Art Foil",
	"Buy-a-Box Promo",
	"Foil Beta Picture",
	"Game Day Promo",
	"SDCC 2019 Exclusive",
}

func preprocess(cardName, edition, variant string) (*mtgmatcher.Card, error) {
	if mtgmatcher.IsToken(cardName) ||
		strings.Contains(variant, "Token") {
		return nil, errors.New("not single")
	}

	if strings.HasPrefix(cardName, "Complete") && strings.HasSuffix(cardName, "Set") {
		return nil, errors.New("incomplete")
	}

	s := mtgmatcher.SplitVariants(cardName)
	cardName = s[0]
	if len(s) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(s[1:], " ")
	}

	if strings.Contains(variant, "Oversized") {
		return nil, errors.New("unsupported")
	}

	isFoil := strings.Contains(variant, "Foil")
	if isFoil {
		variant = strings.Replace(variant, "Foil", "", 1)
		variant = strings.TrimSpace(variant)
	}

	switch edition {
	case "French Revised FBB",
		"German Revised FBB",
		"German Revised FWB",
		"German Renaissance":
		return nil, errors.New("foreign")
	case "Italian Renaissance":
		if cardName == "Blood Moon" {
			return nil, errors.New("does not exist")
		}
	case "Ikoria: Lair of Behemoths":
		if strings.HasSuffix(variant, "JP Alternate Art") {
			variant = "Godzilla"
		}
	case "Commander Anthology Vol. II":
		if cardName == "Bonehoard" {
			return nil, errors.New("does not exist")
		}
	case "Starter 2000":
		switch cardName {
		case "Spined Wurm", "Counterspell", "Shock", "Llanowar Elves":
			return nil, errors.New("does not exist")
		}
	case "Portal 1":
		if variant == "2" {
			variant = "reminder text"
		}
	case "Pre-Release Promos":
		switch cardName {
		case "Pir, Imaginitive Rascal":
			variant = ""
		case "In Garruk's Wake":
			edition = "PM15"
		}
		if strings.HasSuffix(cardName, "Foil") {
			cardName = strings.TrimSuffix(cardName, " Foil")
			isFoil = true
		}
	case "FNM Promos":
		switch cardName {
		case "Elvish Mystic":
			variant = "2014"
		case "Shrapnel Blast":
			variant = "2008"
		case "Chandra's Fury":
			edition = "URL/Convention Promos"
		}
	case "Judge Rewards Promos",
		"Judge Academy Promo":
		switch cardName {
		case "Demonic Tutor":
			if variant == "DCI Judge Promo" {
				variant = "2008"
			} else {
				variant = "2020"
			}
		case "Vampiric Tutor":
			if variant == "DCI Judge Promo" {
				variant = "2000"
			} else {
				variant = "2018"
			}
		case "Wasteland":
			if variant == "DCI Judge Promo" {
				variant = "2010"
			} else {
				variant = "2015"
			}
		}
	case "Unique & Misc Promos":
		switch cardName {
		case "1996 World Champion",
			"Fraternal Exaltation",
			"Splendid Genesis":
			return nil, errors.New("not tracked")
		case "Reliquary Tower":
			edition = "PLGS"
		case "Hydra Broodmaster",
			"Prophet of Kruphix",
			"Temple of Mystery":
			edition = "CP1"
		case "Tember City":
			edition = "PHOP"
		case "Hall of Triumph":
			edition = "THP3"
		case "Lavinia, Azorius Renegade":
			edition = "PRNA"
			variant = ""
		case "Warmonger":
			edition = "PMEI"
		}
		for _, tag := range promoTags {
			if strings.HasSuffix(cardName, tag) {
				cardName = strings.TrimSuffix(cardName, " "+tag)
				variant = tag
			}
		}
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
