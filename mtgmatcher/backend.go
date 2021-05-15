package mtgmatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

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
	Sealed  bool
}

// Card implements the Stringer interface
func (c CardObject) String() string {
	return fmt.Sprintf("%s|%s-%s|%s|%v", c.Name, c.SetCode, c.Edition, c.Number, c.Foil)
}

var backend struct {
	Sets  map[string]*mtgjson.Set
	Cards map[string]cardinfo
	UUIDs map[string]CardObject

	Scryfall map[string]string
}

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids := map[string]CardObject{}
	cards := map[string]cardinfo{}
	scryfall := map[string]string{}

	for code, set := range ap.Data {
		// Skip a set with a single foreign card, and the celebratory printings
		switch code {
		case "PRED", "PCEL":
			delete(ap.Data, code)
			continue
		}
		// Skip online sets, and any token-based sets
		if set.IsOnlineOnly || set.Type == "token" {
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

			// Skip duplicate cards that cause trouble down the road
			switch set.Code {
			case "INV", "USG", "POR", "7ED":
				if strings.HasSuffix(card.Number, "s") {
					continue
				}
			case "STA":
				if strings.HasSuffix(card.Number, "e") {
					card.FrameEffects = []string{mtgjson.FrameEffectFoilEtched}
				}
			}

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

			// Now assign the card to the list of cards to be saved
			filteredCards = append(filteredCards, card)

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

			scryfallId, found := card.Identifiers["scryfallId"]
			if found {
				scryfall[scryfallId] = card.UUID
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

		// Adjust the setBaseSize to take into account the cards with
		// the same name in the same set (also make sure that it is
		// correctly initialized)
		setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
		if err != nil {
			continue
		}
		if setDate.After(PromosForEverybodyYay) {
			for i, card := range set.Cards {
				if card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
					set.BaseSetSize = i + 1
					break
				}
			}
		}

		for _, product := range set.SealedProduct {
			uuids[product.UUID] = CardObject{
				Card: mtgjson.Card{
					UUID:        product.UUID,
					Name:        product.Name,
					SetCode:     code,
					Identifiers: product.Identifiers,
					Rarity:      "Product",
				},
				Sealed:  true,
				Edition: set.Name,
			}
		}
	}

	duplicate(ap.Data, cards, uuids, "Legends Italian", "LEG", "ITA", "1995-04-01")
	duplicate(ap.Data, cards, uuids, "The Dark Italian", "DRK", "ITA", "1995-07-01")
	duplicate(ap.Data, cards, uuids, "Chronicles Japanese", "CHR", "JPN", "1995-07-01")

	backend.Sets = ap.Data
	backend.Cards = cards
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
}

func duplicate(sets map[string]*mtgjson.Set, cards map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := mtgjson.Set{}
	dup = *sets[code]

	// Update with new info
	dup.Name = name
	dup.Code = code + tag
	dup.ReleaseDate = date

	// Copy card information
	dup.Cards = make([]mtgjson.Card, len(sets[code].Cards))
	for i := range sets[code].Cards {
		// Update printings for the original set
		printings := append(sets[code].Cards[i].Printings, dup.Code)
		sets[code].Cards[i].Printings = printings

		// Loop through all other sets mentioned
		for _, setCode := range printings {
			// Skip the set being added, there might be cards containing
			// the set code being processed due to variants
			if setCode == dup.Code {
				continue
			}
			for j := range sets[setCode].Cards {
				// Name match, can't break after the first because there could
				// be other variants
				if sets[setCode].Cards[j].Name == sets[code].Cards[i].Name {
					sets[setCode].Cards[j].Printings = printings
				}
			}
		}

		// Update with new info
		dup.Cards[i] = sets[code].Cards[i]
		dup.Cards[i].UUID += "_" + strings.ToLower(tag)
		dup.Cards[i].SetCode = dup.Code

		// Update printings for the CardInfo map
		ci := cards[Normalize(dup.Cards[i].Name)]
		ci.Printings = printings
		cards[Normalize(dup.Cards[i].Name)] = ci

		// Add the new uuid to the UUID map
		uuids[dup.Cards[i].UUID] = CardObject{
			Card:    dup.Cards[i],
			Edition: name,
		}
	}

	sets[dup.Code] = &dup
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
