package mtgmatcher

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

type cardobject struct {
	mtgjson.Card
	SetCode string
	Edition string
}

var sets map[string]mtgjson.Set
var cards map[string]cardinfo
var uuids map[string]cardobject

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids = map[string]cardobject{}
	cards = map[string]cardinfo{}
	sets = ap.Data

	for code, set := range ap.Data {
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
			uuids[card.UUID] = cardobject{
				Card:    card,
				Edition: set.Name,
				SetCode: code,
			}
		}
	}
}

func LoadDatastore(reader io.Reader) error {
	allprints, err := mtgjson.LoadAllPrintings(reader)
	if err != nil {
		return err
	}

	NewDatastore(allprints)
	return nil
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
