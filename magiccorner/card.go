package magiccorner

import (
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type MCCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Pricing   float64 `json:"price"`
	Qty       int     `json:"quantity"`
	Condition string  `json:"conditions"`

	Id     int    `json:"id"`
	Number string `json:"number"`

	setCode string
	extra   string
	orig    string
}

var cardTable = map[string]string{
	"Ambitions's Cost":        "Ambition's Cost",
	"Elspeth, Knight Errant":  "Elspeth, Knight-Errant",
	"Fire (Fire/Ice) ":        "Fire",
	"Fire (Fire/Ice)":         "Fire",
	"Fire/Ice":                "Fire",
	"Kingsbaile Skirmisher":   "Kinsbaile Skirmisher",
	"Ral, Caller of Storm":    "Ral, Caller of Storms",
	"Rough (Rough/Tumble)":    "Rough",
	"Time Shrike":             "Tine Shrike",
	"Treasure Find":           "Treasured Find",
	"Who/What/When/Where/Why": "Who",
}

func (c *MCCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	variants := mtgban.SplitVariants(c.Name)

	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}
	cname := variants[0]

	lutName, found := cardTable[cname]
	if found {
		cname = lutName
	}

	newName, setName, setCheck := c.parseSet(db, cname, specifier)
	cardName, number, numberCheck := c.parseNumber(newName, setName, specifier)

	variants = mtgban.SplitVariants(cardName)
	cardName = variants[0]

	// Only keep one of the split cards
	switch {
	// Only keep one of the split/ transform cards
	case strings.Contains(cardName, " // "):
		cn := strings.Split(cardName, " // ")
		cardName = cn[0]
	// Only keep one of the transform cards
	case strings.Contains(cardName, " / "):
		cn := strings.Split(cardName, " / ")
		cardName = cn[0]
	}

	n := mtgban.NewNormalizer()
	cardName = n.Normalize(cardName)

	// Loop over the DB
	for _, set := range db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := n.Normalize(card.Name)

				// These sets sometimes have extra stuff that we stripped earlier
				if set.Type == "funny" {
					s := mtgban.SplitVariants(dbCardName)
					dbCardName = s[0]
				}

				// Check card name
				cardCheck := dbCardName == cardName

				// If card number is available, use it to narrow results
				if numberCheck != nil {
					cardCheck = cardCheck && numberCheck(set, card.Number)
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

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) {%s} [%v]", c.Name, cardName, c.Set, setName, c.Foil, number, c)
}

func (c *MCCard) Conditions() string {
	cond := c.Condition
	if cond == "NM/M" {
		cond = "NM"
	}
	return cond
}
func (c *MCCard) Market() string {
	return "Magic Corner"
}
func (c *MCCard) Price() float64 {
	return c.Pricing
}
func (c *MCCard) TradePrice() float64 {
	return 0
}
func (c *MCCard) Quantity() int {
	return c.Qty
}
