package mtgstocks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Cevill, Bane of Monsters":    "Chevill, Bane of Monsters",
	"Frontland Felidar":           "Frondland Felidar",
	"Ragurin Crystal":             "Raugrin Crystal",
	"Bastion of Rememberance":     "Bastion of Remembrance",
	"Rograkh, Son of Gohgahh":     "Rograkh, Son of Rohgahh",
	"Swords of Plowshares":        "Swords to Plowshares",
	"Kedniss, Emberclaw Familiar": "Kediss, Emberclaw Familiar",

	"Battra, Terror of the City (JP Alternate Art)": "Dirge Bat (Godzilla)",
}

func preprocess(fullName, edition string, foil bool) (*mtgmatcher.Card, error) {
	fullName = strings.Replace(fullName, "[", "(", 1)
	fullName = strings.Replace(fullName, "]", ")", 1)

	if mtgmatcher.IsToken(fullName) ||
		strings.Contains(fullName, "Biography Card") ||
		strings.Contains(fullName, "Ultra Pro Puzzle Quest") ||
		strings.Contains(edition, "Oversize") {
		return nil, errors.New("non single")
	}

	lutName, found := cardTable[fullName]
	if found {
		fullName = lutName
	}

	s := mtgmatcher.SplitVariants(fullName)

	variant := ""
	cardName := s[0]
	if len(s) > 1 {
		variant = strings.Join(s[1:], " ")
	}

	s = strings.Split(cardName, " - ")
	cardName = s[0]
	if len(s) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += s[1]
	}

	if variant == "Welcome Back Promo Hangarback Walker Miscellaneous Promos" {
		cardName = "Hangarback Walker"
		edition = "PLGS"
	}

	lutName, found = cardTable[cardName]
	if found {
		cardName = lutName
	}

	switch edition {
	case "Revised Edition (Foreign White Border)":
		return nil, errors.New("unsupported")
	case "Secret Lair Series":
		if cardName == "Thalia, Guardian of Thraben" && variant == "" {
			variant = "37"
		}
	case "Arabian Nights":
		if variant == "Version 2" {
			variant = "dark"
		} else if variant == "Version 1" {
			variant = "light"
		}
	case "Prerelease Cards":
		variant = edition
	case "JSS/MSS Promos":
		edition = "Junior Super Series"
	case "Media Promos":
		if variant == "" {
			variant = "Book"
		}
	case "Arena Promos":
		if cardName == "Underworld Dreams" {
			edition = "DCI"
		}
	case "WPN & Gateway Promos":
		if cardName == "Deathless Angel" {
			edition = "Rise of the Eldrazi Promos"
		}
	case "Judge Promos":
		switch cardName {
		case "Vampiric Tutor":
			if variant == "" {
				variant = "2000"
			}
		}
	case "Miscellaneous Promos":
		if variant == "Magic Scholarship" {
			edition = "Junior Super Series"
		}
	case "Unglued":
		if strings.HasSuffix(variant, "Right") {
			variant = "29"
		} else if strings.HasSuffix(variant, "Left") {
			variant = "28"
		}
	case "Ikoria: Lair of Behemoths: Extras":
		if variant == "JP Alternate Art" {
			variant = "Godzilla"
		}
		edition = "Ikoria: Lair of Behemoths"
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
	}, nil
}

// Some slug strings are missing quotes and are plain numbers
// use this function to parse it
func getLink(raw interface{}) (string, error) {
	var slug string
	switch v := raw.(type) {
	case string:
		slug = v
	case float64:
		slug = fmt.Sprintf("%.0f", v)
	default:
		return "", errors.New("invalid type")
	}
	return "https://www.mtgstocks.com/prints/" + slug, nil
}
