package mtgseattle

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
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

func preprocess(b *mtgmatcher.Backend, cardName, edition, variant string) (*mtgmatcher.InputCard, error) {
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
	}
	variant = strings.TrimSpace(variant)

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
		case "Sanctum Prelate":
			edition = "MH2"
		case "Gala Greeters":
			if variant == "Box Topper" {
				edition = "SNC"
				variant = "Borderless"
			}
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
	case "Mystery Booster Cards":
		number := mb1PLSTNumber(b, cardName)
		if number != "" {
			edition = "PLST"
			variant = number
		}
	case "Core Set 2021":
		if strings.Contains(variant, "Alternate Art") && mtgmatcher.ExtractNumber(variant) == "" {
			variant = "Borderless"
		}
	case "Adventures in the Forgotten Realms":
		if variant == "Dungeon Module" {
			variant = "Showcase"
		}
	case "Secret Lair Drop Series":
		if variant == "Borderless" && len(b.MatchInSet(cardName, "SLC")) > 0 {
			edition = "SLC"
		}
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}

// mb1PLSTBooster is the Mystery Booster sealed product mb1PLSTNumber reads
// booster contents from.
const mb1PLSTBooster = "Mystery Booster Booster Pack (Retail Edition)"

// mb1PLSTNumber answers which PLST printing Mystery Booster's own booster
// bundles for cardName, or "" when the booster does not name it.
func mb1PLSTNumber(b *mtgmatcher.Backend, cardName string) string {
	for _, uuid := range b.GetSealedUUIDsInSet("MB1") {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Name != mb1PLSTBooster {
			continue
		}

		probs, err := sealed.ProductCounts(b, "MB1", uuid)
		if err != nil {
			return ""
		}
		for _, p := range probs {
			card, err := b.GetUUID(p.UUID)
			if err == nil && card.SetCode == "PLST" && card.Name == cardName {
				return card.Number
			}
		}
		return ""
	}
	return ""
}
