package mythicmtg

import (
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Archangel Avacyn | Avacyn, the Purifer":     "Archangel Avacyn // Avacyn, the Purifier",
	"Aberrant Researcher | Perfect Form":         "Aberrant Researcher // Perfected Form",
	"Duskwatch Recruiter | Krallenhorde Brawler": "Duskwatch Recruiter | Krallenhorde Howler",
	"Golden Guardian | Gold-Forged Garrison":     "Golden Guardian | Gold-Forge Garrison",
	"Grizzled Angler | Grizzled Anglerfish":      "Grizzled Angler // Grisly Anglerfish",
	"Jushi Apprentice | Tomoya the Reveler":      "Jushi Apprentice // Tomoya the Revealer",

	"Wretched, The": "The Wretched",
	"Finality":      "Find // Finality",
}

func preprocess(cardName, edition string) (*mtgmatcher.Card, error) {
	variant := ""
	variants := strings.Split(cardName, " - ")
	cardName = variants[0]
	if len(variants) > 1 {
		variant = variants[1]
	}

	if strings.HasSuffix(edition, "En") {
		edition = strings.TrimSuffix(edition, " En")
	}

	switch edition {
	case "Kaldheim":
		if variant == "Extended" {
			switch cardName {
			case "Alrund's Epiphany",
				"Battle Mammoth",
				"Haunting Voyage",
				"Quakebringer",
				"Starnheim Unleashed":
				variant = "Borderless"
			default:
				variant = "Extended Art"
			}
		}
	case "Double Masters":
		if mtgmatcher.IsBasicLand(cardName) {
			variant = "Unglued"
		}
	}

	isFoil := false
	variants = mtgmatcher.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += variants[1]
		if strings.ToLower(variant) == "foil" {
			isFoil = true
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
		Foil:      isFoil,
	}, nil
}
