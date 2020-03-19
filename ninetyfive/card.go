package ninetyfive

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type nfCard struct {
	Name   string
	Code   string
	IsFoil bool
}

func (nf *Ninetyfive) convert(c *nfCard) (*mtgban.Card, error) {
	cardName := nf.norm.Normalize(c.Name)

	// Loop over the DB
	for _, card := range nf.db[c.Code].Cards {
		dbCardName := nf.norm.Normalize(card.Name)

		if dbCardName == cardName {
			ret := mtgban.Card{
				Id:   card.UUID,
				Name: card.Name,
				Set:  nf.db[c.Code].Name,
				Foil: c.IsFoil,
			}
			if (card.HasNonFoil || card.IsAlternative) && c.IsFoil {
				ret.Id += "_f"
			}
			return &ret, nil
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) %q", c.Name, cardName, c.Code, c.IsFoil, c)
}
