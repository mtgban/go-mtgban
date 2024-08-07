package ninetyfive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

var mediaTable = map[string]string{
	"Arcbound Ravager": "PPRO",
	"Disenchant":       "PARL",
}

func preprocess(product *NFProduct) (*mtgmatcher.InputCard, error) {
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

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += vars[1]
	}

	switch edition {
	case "Friday Night Magic",
		"Grand Prix",
		"Happy Holidays",
		"Judge Gift Program",
		"Magic Game Day",
		"Media Inserts",
		"Prerelease Events",
		"Promotional Schemes",
		"World Magic Cup Qualifiers":
		// Drop any number information
		variant = ""
		ed, found := mediaTable[cardName]
		if found {
			edition = ed
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
	case "Warhammer 40,000":
		variant = strings.Replace(variant, "u", mtgjson.SuffixSpecial, 1)
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      product.Foil == 1,
	}, nil
}
