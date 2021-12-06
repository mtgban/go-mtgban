package mtgmatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
	Flavor    string
}

// CardObject is an extension of mtgjson.Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	mtgjson.Card
	Edition string
	Foil    bool
	Etched  bool
	Sealed  bool
}

// Card implements the Stringer interface
func (co CardObject) String() string {
	finish := "nonfoil"
	if co.Etched {
		finish = "etched"
	} else if co.Foil {
		finish = "foil"
	}
	return fmt.Sprintf("%s|%s", co.Card, finish)
}

var backend struct {
	Sets  map[string]*mtgjson.Set
	Cards map[string]cardinfo
	UUIDs map[string]CardObject

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every uniquely normalized product name
	AllSealed []string
	// Map of normalized names to slice of uuids
	Hashes map[string][]string

	Scryfall  map[string]string
	Tcgplayer map[string]string
}

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

func skipSet(set *mtgjson.Set) bool {
	// Skip unsupported sets
	switch set.Code {
	case "PRED", // a single foreign card
		"PCEL", // celebratory printings
		"OLGC", // oversize
		"FJMP": // jumpstart front cards
		return true
	}
	// Skip online sets, and any token-based sets
	if set.IsOnlineOnly ||
		set.Type == "token" ||
		strings.HasSuffix(set.Name, "Art Series") ||
		strings.HasSuffix(set.Name, "Minigames") ||
		strings.Contains(set.Name, "Heroes of the Realm") {
		return true
	}
	return false
}

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids := map[string]CardObject{}
	cards := map[string]cardinfo{}
	scryfall := map[string]string{}
	tcgplayer := map[string]string{}

	for code, set := range ap.Data {
		if skipSet(set) {
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
			}

			// Filter out unneeded printings
			var printings []string
			for i := range card.Printings {
				subset, found := ap.Data[card.Printings[i]]
				// If not found it means the set was already deleted above
				if !found || skipSet(subset) {
					continue
				}
				printings = append(printings, card.Printings[i])
			}
			// Sort printings by most recent sets first
			sort.Slice(printings, func(i, j int) bool {
				setDateI, errI := time.Parse("2006-01-02", ap.Data[printings[i]].ReleaseDate)
				setDateJ, errJ := time.Parse("2006-01-02", ap.Data[printings[j]].ReleaseDate)
				if errI != nil || errJ != nil {
					return false
				}

				return setDateI.After(setDateJ)

			})
			card.Printings = printings

			// Now assign the card to the list of cards to be saved
			filteredCards = append(filteredCards, card)

			// Quick dictionary of valid card names and their printings
			for i, name := range []string{card.Name, card.FaceName, card.FlavorName, card.FaceFlavorName} {
				// Skip empty entries
				if name == "" {
					continue
				}
				// Skip faces of DFCs with same names, so that faces don't pollute
				// the main dictionary with a wrong rename
				if i != 0 && set.Code == "SLD" && strings.Contains(card.Name, "//") {
					continue
				} else if i == 1 {
					// Skip FaceName entries that could be aliased
					// ie 'Start' could be Start//Finish and Start//Fire
					switch name {
					case "Bind",
						"Smelt",
						"Start":
						continue
					}
				}
				norm := Normalize(name)
				_, found := cards[norm]
				if !found {
					cards[norm] = cardinfo{
						Name:      card.Name,
						Printings: card.Printings,
						Layout:    card.Layout,
						Flavor:    card.FlavorName,
					}
				}
			}

			// Initialize custom lookup tables
			scryfallId, found := card.Identifiers["scryfallId"]
			if found {
				scryfall[scryfallId] = card.UUID
			}
			for _, tag := range []string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
				tcgplayerId, found := card.Identifiers[tag]
				if found {
					tcgplayer[tcgplayerId] = card.UUID
				}
			}

			// Shared card object
			co := CardObject{
				Card:    card,
				Edition: set.Name,
			}

			// Save the original uuid
			co.Identifiers["mtgjsonId"] = card.UUID

			// Append "_f" and "_e" to uuids, unless etched is the only printing.
			// If it's not etched, append "_f", unless foil is the only printing.
			// Leave uuids unchanged, if there is a single printing of any kind.
			if card.HasFinish(mtgjson.FinishEtched) {
				uuid := card.UUID

				// Etched + Nonfoil [+ Foil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co
				}

				// Etched + Foil
				if card.HasFinish(mtgjson.FinishFoil) {
					// Set the main property
					co.Foil = true
					// Make sure "_f" is appended if a different version exists
					if card.HasFinish(mtgjson.FinishNonfoil) {
						uuid = card.UUID + suffixFoil
						co.UUID = uuid
					}
					// Save the card object
					uuids[uuid] = co
				}

				// Etched
				// Set the main properties
				co.Foil = false
				co.Etched = true
				// If there are alternative finishes, always append the suffix
				if card.HasFinish(mtgjson.FinishNonfoil) || card.HasFinish(mtgjson.FinishFoil) {
					uuid = card.UUID + suffixEtched
					co.UUID = uuid
				}
				// Save the card object
				uuids[uuid] = co
			} else if card.HasFinish(mtgjson.FinishFoil) {
				uuid := card.UUID

				// Foil [+ Nonfoil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co

					// Update the uuid for the *next* finish type
					uuid = card.UUID + suffixFoil
					co.UUID = uuid
				}

				// Foil
				co.Foil = true
				// Save the card object
				uuids[uuid] = co
			} else {
				// Single printing, use as-is
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

	// XXX: maybe FaceName cause trouble when searching prefix?
	hashes := map[string][]string{}
	names := make([]string, 0, len(cards))
	var sealed []string
	for uuid, card := range uuids {
		norm := Normalize(card.Name)
		_, found := hashes[norm]
		if !found {
			if card.Sealed {
				sealed = append(sealed, norm)
			} else {
				names = append(names, norm)
			}
		}
		hashes[norm] = append(hashes[norm], uuid)
	}

	backend.Hashes = hashes
	backend.AllNames = names
	backend.AllSealed = sealed
	backend.Sets = ap.Data
	backend.Cards = cards
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
	backend.Tcgplayer = tcgplayer
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
			_, found := sets[setCode]
			if !found {
				continue
			}
			if skipSet(sets[setCode]) {
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
