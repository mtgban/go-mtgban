package cardkingdom

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type ckCard struct {
	Id      string
	Name    string
	Edition string
	Foil    bool

	SetCode   string
	Variation string
	Number    string
}

var cardTable = map[string]string{
	"BFM Left":  "B.F.M. (Big Furry Monster)",
	"BFM Right": "B.F.M. (Big Furry Monster)", // (b)

	"Surgeon Commander":         "Surgeon ~General~ Commander",
	"\"Rumors of My Death...\"": "\"Rumors of My Death . . .\"",

	"The Ultimate Nightmare of WotC Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",
	"Our Market Research":                             "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

var skuFixupTable = map[string]string{
	"013A":      "FEM-013A",
	"F18-060":   "PDOM-060",
	"F18-081":   "PDOM-081",
	"F18-204":   "PDOM-204",
	"F18-029":   "PM19-029",
	"F18-110":   "PM19-110",
	"F18-180":   "PM19-180",
	"F18-006":   "PGRN-006",
	"F18-054":   "PGRN-054",
	"F18-206":   "PGRN-206",
	"F19-107":   "PRNA-107",
	"F19-178":   "PRNA-178",
	"F19-192":   "PRNA-192",
	"F19-171":   "PWAR-171",
	"F19-193":   "PWAR-193",
	"FODY-666":  "PPRE-15",
	"PIMA-094A": "PM15-101",
	"PIMA-166A": "PJOU-126",
	"PUMA-050A": "PKTK-036A",
	"PUMA-062A": "PDTK-061A",
	"PUMA-117A": "PFRF-087A",
	"PRM-31483": "P05-002",
	"PEMN-089A": "PEMN-098A",
	"PM20-009":  "PM20-009A",
	"DOM-269P":  "PDOM-001P",
	"FDOM-269P": "PDOM-001P",
	"ELD-P":     "ELD-036P",
	"WAR-270P":  "WAR-220P",
	"ATQ-080A":  "ATQ-080C",
	"ATQ-080B":  "ATQ-080A",
	"ATQ-080C":  "ATQ-080B",
	"ATQ-080D":  "ATQ-080D",
	"BFZ-247P":  "PSS1-247",
}

func (ck *Cardkingdom) convert(c *ckCard) (*mtgban.Card, error) {
	setCode := ck.parseSetCode(c)
	cardName, numberCheck := ck.parseNumber(c)

	set := ck.db[setCode]
	cardName = ck.norm.Normalize(cardName)

	// Loop over the DB
	for _, card := range set.Cards {
		dbCardName := ck.norm.Normalize(card.Name)

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

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v, number=%s) [%v]", c.Name, cardName, c.Edition, setCode, c.Foil, c.Number, c)
}
