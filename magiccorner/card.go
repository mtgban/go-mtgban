package magiccorner

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type mcCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Id     int    `json:"id"`
	Number string `json:"number"`

	setCode string
	extra   string
	orig    string
}

var cardTable = map[string]string{
	"Ambitions's Cost":        "Ambition's Cost",
	"Fire (Fire/Ice) ":        "Fire",
	"Fire (Fire/Ice)":         "Fire",
	"Fire/Ice":                "Fire",
	"Kingsbaile Skirmisher":   "Kinsbaile Skirmisher",
	"Ral, Caller of Storm":    "Ral, Caller of Storms",
	"Rough (Rough/Tumble)":    "Rough",
	"Time Shrike":             "Tine Shrike",
	"Treasure Find":           "Treasured Find",
	"Who/What/When/Where/Why": "Who",
	"Who,What,When,Where,Why": "Who",

	"Elspeth, Knight Errant (Mythic Edition)": "Elspeth, Knight-Errant",
}

func (mc *Magiccorner) Convert(c *mcCard) (*mtgban.Card, error) {
	setName, setCheck := mc.parseSet(c)
	cardName, numberCheck := mc.parseNumber(c, setName)

	cardName = mc.norm.Normalize(cardName)

	// Loop over the DB
	for _, set := range mc.db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := mc.norm.Normalize(card.Name)

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

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) [%v]", c.Name, cardName, c.Set, setName, c.Foil, c)
}
