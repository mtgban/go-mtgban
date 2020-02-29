package channelfireball

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type cfbCard struct {
	Key        string
	Name       string
	Edition    string
	Foil       bool
	Conditions string
	Price      float64
	Quantity   int
	Id         string
}

var cardTable = map[string]string{
	"Morbid Curiousity": "Morbid Curiosity",
	"Poison Tip-Archer": "Poison-Tip Archer",

	"Forest 015 (Arena 2000)": "Forest (Arena 2000)",
	"Plains 011 (Arena 2000)": "Plains (Arena 2000)",

	"Pir, Imaginitive Rascal (Release Promo)": "Pir, Imaginative Rascal (Release Promo)",

	"_________":               "_____",
	"Pang Tong, ":             "Pang Tong, \"Young Phoenix\"",
	"Kongming, ":              "Kongming, \"Sleeping Dragon\"",
	"Rumors of My Death...":   "\"Rumors of My Death . . .\"",
	"Who/What/When/Where/Why": "Who",
	"Our Market Research Shows That Players Like Really Long Card Names So We Make This Card to Have the Absolute Longest Card Name E": "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

func (cfb *Channelfireball) convert(c *cfbCard) (*mtgban.Card, error) {
	setName, setCheck := cfb.parseSet(c)
	cardName, numberCheck := cfb.parseNumber(c, setName)

	cardName = cfb.norm.Normalize(cardName)

	// Loop over the DB
	for _, set := range cfb.db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := cfb.norm.Normalize(card.Name)

				// These sets sometimes have extra stuff that we stripped earlier
				if set.Type == "funny" {
					s := mtgban.SplitVariants(dbCardName)
					dbCardName = s[0]
				}

				// Check card name
				cardCheck := dbCardName == cardName

				// Narrow results with the number callback
				if numberCheck != nil {
					cardCheck = cardCheck && numberCheck(set, card)
				}

				if cardCheck {
					foil := c.Foil
					if !foil && !card.HasNonFoil {
						cfb.printf("Fixing up foil for %s (%s)", card.Name, set.Name)
						foil = true
					}
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: foil,
					}
					if (card.HasNonFoil || card.IsAlternative) && foil {
						ret.Id += "_f"
					}
					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) [%v]", c.Name, cardName, c.Edition, setName, c.Foil, c)
}
