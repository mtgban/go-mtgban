package mtgseattle

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
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
	s := mtgmatcher.SplitVariants(cardName)
	cardName = s[0]
	if len(s) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(s[1:], " ")
	}

	if strings.Contains(variant, "BGS") ||
		mtgmatcher.Contains(cardName, "Deprecated") ||
		strings.Contains(cardName, "Does not exist") {
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
	case "9th Edition":
		if cardName == "Goblin Raider" && isFoil {
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
		var year string
		switch cardName {
		case "Demonic Tutor":
			if variant == "DCI Judge Promo" {
				year = "2008"
			} else if variant == "Judge Academy Promo" {
				year = "2020"
			}
		case "Vampiric Tutor":
			year = mtgmatcher.ExtractYear(variant)
			if year == "" {
				year = "2000"
			}
		case "Wasteland":
			year = mtgmatcher.ExtractYear(variant)
		}
		if year != "" {
			variant = year
		}
	case "Unique & Misc Promos":
		switch cardName {
		case "1996 World Champion",
			"Fraternal Exaltation",
			"Splendid Genesis":
			edition = "Special Occasion"
		case "Reliquary Tower":
			edition = "PLG20"
		case "Tember City":
			edition = "PHOP"
		case "Lavinia, Azorius Renegade":
			edition = "PRNA"
			variant = ""
		case "Warmonger":
			edition = "PMEI"
		case "Stonecoil Serpent",
			"Vito, Thorn of the Dusk Rose":
			edition = "PRES"
		case "Sanctum Prelate":
			edition = "MH2"
		}
		for _, tag := range promoTags {
			if strings.HasSuffix(cardName, tag) {
				cardName = strings.TrimSuffix(cardName, " "+tag)
				variant = tag
			}
		}
		if variant == "Winner" {
			return nil, errors.New("unsupported")
		}
	case "Core Set 2021":
		if strings.Contains(variant, "Alternate Art") && mtgmatcher.ExtractNumber(variant) == "" {
			variant = "Borderless"
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
