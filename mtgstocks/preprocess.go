package mtgstocks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Cevill, Bane of Monsters":    "Chevill, Bane of Monsters",
	"Frontland Felidar":           "Frondland Felidar",
	"Ragurin Crystal":             "Raugrin Crystal",
	"Bastion of Rememberance":     "Bastion of Remembrance",
	"Rograkh, Son of Gohgahh":     "Rograkh, Son of Rohgahh",
	"Swords of Plowshares":        "Swords to Plowshares",
	"Kedniss, Emberclaw Familiar": "Kediss, Emberclaw Familiar",
	"Gilanra, Caller or Wirewood": "Gilanra, Caller of Wirewood",
	"Rengade Tactics":             "Renegade Tactics",
	"Iona's Judgement":            "Iona's Judgment",
	"Axguard Armory":              "Axgard Armory",
	"Immersturn Raider":           "Immersturm Raider",
	"Artifact of Mishra":          "Ankh of Mishra",
	"Rosethron Acolyte":           "Rosethorn Acolyte",
	"Blackboom Rogue":             "Blackbloom Rogue",

	"Nezumi Shortfang // Nezumi Shortfang":                 "Nezumi Shortfang",
	"Corpse Knight (2/3 Misprint)":                         "Corpse Knight (Misprint)",
	"Subira, Tulzidi Caravaneer (Extended Art)":            "Subira, Tulzidi Caravanner (Extended Art)",
	"Fiendish Duo (JP Exclusive Store Support Promo)":      "Fiendish Duo (PKHM)",
	"Wind Drake (17/264)":                                  "Wind Drake (Intro)",
	"Valakut Awakening // Valakut Stoneforge (Borderless)": "Valakut Awakening (Extended Art)",

	"Darkbore Pathway (Extended Art)":    "Darkbore Pathway // Slitherbore Pathway (Borderless)",
	"Hengegate Pathway (Extended Art)":   "Hengegate Pathway // Mistgate Pathway (Borderless)",
	"Blightstep Pathway (Extended Art)":  "Blightstep Pathway // Searstep Pathway (Borderless)",
	"Barkchannel Pathway (Extended Art)": "Barkchannel Pathway // Tidechannel Pathway (Borderless)",

	"Haunting Voyage (Extended Art)":                "Haunting Voyage (Borderless)",
	"Quakebringer (Extended Art)":                   "Quakebringer (Borderless)",
	"Tevesh Szat, Doom of Fools (Extended Art)":     "Tevesh Szat, Doom of Fools (Borderless)",
	"Battra, Terror of the City (JP Alternate Art)": "Dirge Bat (Godzilla)",
}

var promoTable = map[string]string{
	"Crucible of Worlds":         "PWOR",
	"Mana Crypt":                 "PHPR",
	"Fireball":                   "PMEI",
	"Loam Lion":                  "PRES",
	"Oran-Rief, the Vastwood":    "PRES",
	"Treasure Hunt":              "PIDW",
	"Reliquary Tower":            "PLG20",
	"Flooded Strand":             "PNAT",
	"Serra Avatar":               "PDP13",
	"Corrupt":                    "PI13",
	"Duress":                     "PI14",
	"Electrolyze":                "PIDW",
	"Jaya Ballard, Task Mage":    "PRES",
	"Liliana Vess":               "PDP10",
	"Llanowar Elves":             "PDOM",
	"Noble Hierarch":             "PPRO",
	"Cryptic Command":            "PPRO",
	"Chandra, Torch of Defiance": "Q06",
}

func preprocess(fullName, edition string, foil bool) (*mtgmatcher.InputCard, error) {
	fullName = strings.Replace(fullName, "[", "(", 1)
	fullName = strings.Replace(fullName, "]", ")", 1)

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

	lutName, found = cardTable[cardName]
	if found {
		cardName = lutName
	}

	switch edition {
	case "Oversize Cards":
		if !strings.Contains(variant, "Planechase") {
			return nil, errors.New("unsupported")
		}
		if cardName == "Stairs to Infinity" && variant == "Planechase 2012" {
			return nil, errors.New("does not exist")
		}
	case "Secret Lair Series":
		switch cardName {
		case "Thalia, Guardian of Thraben":
			if variant == "" {
				variant = "37"
			}
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
	case "Arena Promos":
		if cardName == "Underworld Dreams" {
			edition = "DCI"
		}
	case "WPN & Gateway Promos":
		if cardName == "Deathless Angel" {
			edition = "Rise of the Eldrazi Promos"
		}
	case "Launch Party & Release Event Promos":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		}
	case "Judge Promos":
		switch cardName {
		case "Vampiric Tutor":
			if variant == "" {
				variant = "2000"
			}
		case "Demonic Tutor":
			if variant == "" {
				variant = "2008"
			}
		case "Wasteland":
			if variant == "" {
				variant = "2010"
			}
		}
	case "Miscellaneous Promos",
		"Media Promos",
		"Open House Promos",
		"Pro Tour Promos":
		if variant == "Magic Scholarship" {
			edition = "Junior Super Series"
		} else {
			ed, found := promoTable[cardName]
			if found {
				edition = ed
			}
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

	return &mtgmatcher.InputCard{
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
