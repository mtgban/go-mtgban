package bigorbitcards

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func preprocess(cardName, edition string) (*mtgmatcher.Card, error) {
	fields := mtgmatcher.SplitVariants(cardName)
	cardName = fields[0]
	variant := ""
	if len(fields) > 1 {
		variant = strings.Join(fields[1:], " ")
	}
	foil := strings.Contains(variant, "Foil")

	if variant == "Art" {
		return nil, errors.New("unsupported")
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
	}, nil
}
