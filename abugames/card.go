package abugames

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type abuCard struct {
	Name      string `json:"name"`
	Set       string `json:"set"`
	Foil      bool   `json:"foil"`
	Condition string `json:"conditions"`

	BuyPrice     float64 `json:"buy_price"`
	BuyQuantity  int     `json:"buy_quantity"`
	TradePricing float64 `json:"trade_price"`

	SellPrice    float64 `json:"sell_price"`
	SellQuantity int     `json:"sell_quantity"`

	FullName string `json:"full_name"`
	Number   string `json:"number"`
	Layout   string `json:"layout"`
	Id       string `json:"rarity"`
}

var cardTable = map[string]string{
	"B.F.M. Big Furry Monster Left":   "B.F.M.",
	"B.F.M. Big Furry Monster Right":  "B.F.M.",
	"No Name":                         "_____",
	"Rathi Berserker (Aerathi)":       "Aerathi Berserker",
	"Scholar of the Stars":            "Scholar of Stars",
	"Who What When Where Why":         "Who",
	"Absolute Longest Card Name Ever": "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

func (abu *ABUGames) convert(c *abuCard) (*mtgban.Card, error) {
	setName, setCheck := abu.parseSet(c)
	cardName, numberCheck := abu.parseNumber(c, setName)

	cardName = abu.norm.Normalize(cardName)

	// Loop over the DB
	for _, set := range abu.db {
		if setCheck(*set) {
			for _, card := range set.Cards {
				dbCardName := abu.norm.Normalize(card.Name)

				// These sets sometimes have extra stuff that we stripped earlier
				if set.Type == "funny" {
					s := mtgban.SplitVariants(dbCardName)
					dbCardName = s[0]
				}

				// Check card name
				cardCheck := dbCardName == cardName

				// Narrow results with the number callback
				if numberCheck != nil {
					cardCheck = cardCheck && numberCheck(*set, card)
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
