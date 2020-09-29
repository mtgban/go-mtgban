package mtgmatcher

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

type cardobject struct {
	mtgjson.Card
	Edition string
	Foil    bool
}

var backend struct {
	Sets  map[string]mtgjson.Set
	Cards map[string]cardinfo
	UUIDs map[string]cardobject
}

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids := map[string]cardobject{}
	cards := map[string]cardinfo{}

	for code, set := range ap.Data {
		if set.IsOnlineOnly || code == "PRED" {
			delete(ap.Data, code)
			continue
		}
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
			// Shared card object
			co := cardobject{
				Card:    card,
				Edition: set.Name,
			}
			// If card is foil, check whether it has a non-foil counterpart
			if card.HasFoil {
				uuid := card.UUID
				// If it has, save the nonfoil cardobject, and change hash
				if card.HasNonFoil {
					uuids[uuid] = co
					uuid += "_f"
				}
				// Regardless of above, set the the foil status
				co.Foil = true
				uuids[uuid] = co
			} else {
				// If it's non-foil, use as-is
				uuids[card.UUID] = co
			}
		}
	}

	backend.Sets = ap.Data
	backend.Cards = cards
	backend.UUIDs = uuids
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
