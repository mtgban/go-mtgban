package mintcard

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func setCodeExists(code string) bool {
	_, err := mtgmatcher.GetSet(code)
	return err == nil
}

var nameTable = map[string]string{
	"Godzilla, King of Monsters":             "Godzilla, King of the Monsters",
	"Tavern Ruffian // Tavern Champion":      "Tavern Ruffian // Tavern Smasher",
	"Faithbound Judge // Sinner's Judgement": "Faithbound Judge // Sinner's Judgment",
	"Mothra's Giant Cocoon":                  "Mothra's Great Cocoon",
	"rathi Berserker":                        "Aerathi Berserker",
}

var name2edition = map[string]string{
	"Serra Angel":  "PWOS",
	"Fiendish Duo": "PKHM",
}

func preprocess(cardName, number, finish, langauge, edition, setCode string) (*mtgmatcher.InputCard, error) {
	if setCode == "FWB" {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Count(cardName, "Token") > 1 {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Contains(cardName, "Complete") && strings.Contains(cardName, "Set") {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Contains(cardName, "Signed") {
		return nil, mtgmatcher.ErrUnsupported
	}
	if strings.Contains(cardName, "Graded") {
		return nil, mtgmatcher.ErrUnsupported
	}

	cardName = strings.Replace(cardName, ")(", ") (", -1)
	s := mtgmatcher.SplitVariants(cardName)
	cardName = s[0]
	variant := ""
	if len(s) > 1 {
		variant = strings.Join(s[1:], " ")
	}
	variant = strings.Replace(variant, "HP", "", -1)
	variant = strings.Replace(variant, "MP", "", -1)
	variant = strings.Replace(variant, "DMG", "", -1)
	variant = strings.Replace(variant, "Damaged", "", -1)
	variant = strings.TrimSpace(variant)

	fixup, found := nameTable[cardName]
	if found {
		cardName = fixup
	}

	switch setCode {
	case "PMSC":
		fixup, found := name2edition[cardName]
		if found {
			edition = fixup
		}
	case "PMF":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "PF19"
		}
	case "STA":
		if strings.Contains(variant, "Collector Booster") {
			variant += " Etched"
		}
	case "MYS":
		if variant == "Commander" {
			variant = "Commander 2011"
		}
	case "SLD":
		if len(mtgmatcher.MatchInSet(cardName, "SLC")) == 1 {
			edition = "SLC"
			if len(mtgmatcher.MatchInSet(cardName, "SLD")) > 0 && mtgmatcher.ExtractYear(variant) == "" {
				edition = "SLD"
			}
		}
	default:
		if setCodeExists(setCode) {
			edition = setCode
		}
	}

	foil := strings.Contains(finish, "Foil") || strings.Contains(variant, "Foil")

	if strings.Contains(finish, "Prerelease") {
		variant += " Prerelease"
	}

	if number != "" && len(mtgmatcher.MatchInSetNumber(cardName, setCode, number)) == 1 {
		variant += " " + number
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
		Language:  langauge,
	}, nil
}
