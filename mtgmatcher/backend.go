package mtgmatcher

import (
	"io/ioutil"
	"log"

	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

var sets map[string]mtgjson.Set
var cards map[string]cardinfo

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func NewDatastore(ap mtgjson.AllPrintings) {
	cards = map[string]cardinfo{}
	sets = ap.Data

	for _, set := range ap.Data {
		for _, card := range set.Cards {
			norm := Normalize(card.Name)
			_, found := cards[norm]
			if !found {
				cards[norm] = cardinfo{
					Name:      card.Name,
					Printings: card.Printings,
					Layout:    card.Layout,
				}
			}
		}
	}
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
