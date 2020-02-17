package miniaturemarket

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type mmCard struct {
	Name    string
	Edition string
	Foil    bool
}

var cardTable = map[string]string{
	"Asylum Visitior":           "Asylum Visitor",
	"Bloodmist":                 "Blood Mist",
	"Fiesty Stegosaurus":        "Feisty Stegosaurus",
	"Torban, Thane of Red Fell": "Torbran, Thane of Red Fell",

	"Cunning Bandit /Azamuki, Treachery Incarnate": "Cunning Bandit / Azamuki, Treachery Incarnate",

	"Who / What / When / Where / Why": "Who",
	"'Rumors of My Death. . .''":      "\"Rumors of My Death . . .\"",

	"The Ultimate Nightmare of Wizards of the Coast(R) Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",
}

func (mm *Miniaturemarket) convert(c *mmCard) (*mtgban.Card, error) {
	setName, setCheck := mm.parseSet(c)
	cardName, numberCheck := mm.parseNumber(c, setName)

	cardName = mm.norm.Normalize(cardName)

	// Loop over the DB
	for _, set := range mm.db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := mm.norm.Normalize(card.Name)

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
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: c.Foil,
					}
					if (card.HasNonFoil || card.IsAlternative) && c.Foil {
						ret.Id += "_f"
					}
					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) [%v]", c.Name, cardName, c.Edition, setName, c.Foil, c)
}
