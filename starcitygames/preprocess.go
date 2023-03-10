package starcitygames

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Who / What / When / Where / Why": "Who // What // When // Where // Why",

	"Jushi Apprentice // Tomoya The Revealer // Tomoya The Revealer": " Jushi Apprentice // Tomoya The Revealer",
}

func shouldSkipLang(cardName, edition, variant, language string) bool {
	if mtgmatcher.SkipLanguage(cardName, edition, language) {
		return true
	}

	// Additional language rules
	switch language {
	case "Japanese":
		switch edition {
		case "4th Edition BB":
			if mtgmatcher.IsBasicLand(cardName) {
				return true
			}
		case "War of the Spark",
			"Strixhaven Mystical Archive",
			"Strixhaven Mystical Archive - Foil Etched":
			if variant != "Alternate Art" {
				return true
			}
		}
	case "Italian":
		switch edition {
		case "3rd Edition BB":
			if mtgmatcher.IsBasicLand(cardName) {
				return true
			}
		}
	}

	return false
}

func preprocess(card *SCGCardVariant, edition, language string, foil bool) (*mtgmatcher.Card, error) {
	cardName := strings.Replace(card.Name, "&amp;", "&", -1)

	edition = strings.Replace(edition, "&amp;", "&", -1)
	if strings.HasSuffix(edition, "(Foil)") {
		edition = strings.TrimSuffix(edition, " (Foil)")
		foil = true
	}

	variant := strings.Replace(card.Subtitle, "&amp;", "&", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	vars = mtgmatcher.SplitVariants(edition)
	edition = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	if shouldSkipLang(cardName, edition, variant, language) {
		return nil, errors.New("non-english")
	}

	switch {
	case strings.HasPrefix(cardName, "APAC Land"),
		strings.HasPrefix(cardName, "Euro Land"),
		strings.Contains(cardName, "{") && strings.Contains(cardName, "}"):
		return nil, errors.New("non-single")
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	if mtgmatcher.IsBasicLand(cardName) {
		if strings.Contains(variant, "APAC") {
			edition = "Asia Pacific Land Program"
		} else if strings.Contains(variant, "Euro") {
			edition = "European Land Program"
		}
	}

	// Make sure not to pollute variants with the language otherwise multiple
	// variants may be aliased (ie urzalands)
	switch language {
	case "Japanese":
		if edition == "Chronicles" {
			edition = "Chronicles Japanese"
		} else {
			if variant != "" {
				variant += " "
			}
			variant += language
		}
	case "Italian":
		if edition == "Renaissance" {
			edition = "Rinascimento"
		} else {
			if variant != "" {
				variant += " "
			}
			variant += language
		}
	}

	switch edition {
	case "Promo: General":
		switch cardName {
		case "Water Gun Balloon Game":
			edition = "UNF"
		case "Swiftfoot Boots":
			if variant == "Launch" {
				edition = "PW22"
				variant = ""
			}
		case "Dismember":
			if variant == "Commander Party Phyrexian" {
				edition = "PW22"
			}
		}
	case "Promo: General - Alternate Foil":
		switch cardName {
		case "Arcane Signet":
			if strings.Contains(variant, "Festival") {
				edition = "P30A"
				variant = "1F"
			}
		}
	case "Promo: General - Foil Etched":
		switch cardName {
		case "Arcane Signet":
			if strings.Contains(variant, "Festival") {
				edition = "P30A"
				variant = "1F★"
			}
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
	}, nil
}
