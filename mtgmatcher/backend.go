package mtgmatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

// CardObject is an extension of mtgjson.Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	mtgjson.Card
	Edition string
	Foil    bool
}

// Card implements the Stringer interface
func (c CardObject) String() string {
	return fmt.Sprintf("%s|%s-%s|%s|%v", c.Name, c.SetCode, c.Edition, c.Number, c.Foil)
}

var backend struct {
	Sets  map[string]*mtgjson.Set
	Cards map[string]cardinfo
	UUIDs map[string]CardObject
}

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids := map[string]CardObject{}
	cards := map[string]cardinfo{}

	for code, set := range ap.Data {
		if set.IsOnlineOnly || code == "PRED" {
			delete(ap.Data, code)
			continue
		}

		var filteredCards []mtgjson.Card
		for _, card := range set.Cards {
			// MTGJSON v5 contains duplicated card info for each face, and we do
			// not need that level of detail, so just skip any extra side.
			if card.Side != "" && card.Side != "a" {
				continue
			}
			filteredCards = append(filteredCards, card)

			// Filter out unneeded printings
			var printings []string
			for i := range card.Printings {
				subset, found := ap.Data[card.Printings[i]]
				// If not found it means the set was already deleted above
				if !found || subset.IsOnlineOnly {
					continue
				}
				printings = append(printings, card.Printings[i])
			}
			card.Printings = printings

			// Quick dictionary of valid card names and their printings
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
			co := CardObject{
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

		// Replace the original array with the filtered one
		set.Cards = filteredCards
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

func LoadDatastoreFile(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	return LoadDatastore(reader)
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
