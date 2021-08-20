package ninetyfive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var mediaTable = map[string]string{
	"Arcbound Ravager (2019)": "PPRO",
}

func preprocess(product *NFProduct) (*mtgmatcher.Card, error) {
	card := product.Card
	edition := product.Set.Name
	variant := ""
	if edition == "" {
		edition = product.Card.Set.Name
	}
	cardName := card.Name

	switch product.Language.Code {
	case "en":
	case "it":
		switch edition {
		case "Legends", "The Dark":
			variant = "Italian"
		case "Rinascimento":
		default:
			return nil, errors.New("non-english")
		}
	case "jp":
		switch edition {
		case "WAR Alt-art Promos":
			// IKO cards are listed English
		default:
			return nil, errors.New("non-english")
		}
	default:
		return nil, errors.New("non-english")
	}

	// card.Number is an int so it's missing the letter variations, so the variant
	// in the name is more accurate. Prevent adding both which would confuse the matcher
	if card.Number != 0 && !strings.Contains(card.Name, fmt.Sprint(card.Number)) {
		if variant != "" {
			variant += " "
		}
		variant += fmt.Sprint(card.Number)
	}

	if mtgmatcher.IsToken(cardName) ||
		strings.Contains(edition, "Art Series") {
		return nil, errors.New("token")
	}

	switch edition {
	case "Fourth Edition (Alt)":
		return nil, errors.New("unsupported")
	case "Promotional Schemes":
		if cardName == "Disenchant (6)" {
			cardName = "Disenchant"
			edition = "PARL"
			variant = ""
		}
	case "Friday Night Magic",
		"Grand Prix",
		"Happy Holidays",
		"Judge Gift Program",
		"Magic Game Day",
		"Media Inserts",
		"Prerelease Events",
		"World Magic Cup Qualifiers":
		// Drop any number information
		variant = ""
		ed, found := mediaTable[cardName]
		if found {
			edition = ed
		} else {
			for _, ed = range []string{"CP1", "CP2", "CP3"} {
				if len(mtgmatcher.MatchInSet(cardName, ed)) != 0 {
					edition = ed
					break
				}
			}
		}
	case "Signature Spellbook 1: Jace":
		edition = "Signature Spellbook: Jace"
	case "Signature Spellbook 2: Gideon":
		edition = "Signature Spellbook: Gideon"
	case "Champions of Kamigawa":
		if !mtgmatcher.IsBasicLand(cardName) {
			variant = ""
		}
	case "Deckmasters":
		if !mtgmatcher.IsBasicLand(cardName) {
			variant = ""
		}
	case "Battle for Zendikar":
		edition = map[string]string{
			"Canopy Vista (234)":     "PSS1",
			"Cinder Glade (235)":     "PSS1",
			"Prairie Stream (241)":   "PSS1",
			"Smoldering Marsh (247)": "PSS1",
			"Sunken Hollow (249)":    "PSS1",
		}[cardName]
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      product.Foil == 1,
	}, nil
}
