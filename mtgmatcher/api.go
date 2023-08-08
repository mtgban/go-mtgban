package mtgmatcher

import (
	"regexp"
	"strings"

	"github.com/mroth/weightedrand/v2"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

func GetUUIDs() map[string]CardObject {
	return backend.UUIDs
}

func GetUUID(uuid string) (*CardObject, error) {
	if backend.UUIDs == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := backend.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownId
	}

	return &co, nil
}

func GetSets() map[string]*mtgjson.Set {
	return backend.Sets
}

func GetSet(code string) (*mtgjson.Set, error) {
	if backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	set, found := backend.Sets[strings.ToUpper(code)]
	if !found {
		return nil, ErrCardNotInEdition
	}

	return set, nil
}

func GetSetByName(edition string, flags ...bool) (*mtgjson.Set, error) {
	if backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	card := &Card{
		Edition: edition,
	}
	if len(flags) > 0 {
		card.Foil = flags[0]
	}
	adjustEdition(card)

	for _, set := range backend.Sets {
		if Equals(set.Name, card.Edition) || set.Code == card.Edition {
			return set, nil
		}
	}

	return nil, ErrCardUnknownId
}

func GetSetUUID(uuid string) (*mtgjson.Set, error) {
	if backend.UUIDs == nil || backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := backend.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownId
	}

	set, found := backend.Sets[co.SetCode]
	if !found {
		return nil, ErrCardUnknownId
	}

	return set, nil
}

func Scryfall2UUID(id string) string {
	return backend.Scryfall[id]
}

func Tcg2UUID(id string) string {
	return backend.Tcgplayer[id]
}

func SearchEquals(name string) ([]string, error) {
	if name == "" {
		return backend.AllUUIDs, nil
	}
	results, err := searchEquals(name, backend.AllNames)
	if err != nil {
		return searchEquals(name, backend.AlternateNames)
	}
	return results, nil
}

func SearchSealedEquals(name string) ([]string, error) {
	return searchEquals(name, backend.AllSealed)
}

func searchEquals(name string, slice []string) ([]string, error) {
	name = Normalize(name)
	for i := range slice {
		if slice[i] == name {
			return backend.Hashes[slice[i]], nil
		}
	}
	return nil, ErrCardDoesNotExist
}

func searchFunc(name string, slice []string, f func(string, string) bool) ([]string, error) {
	var hashes []string
	name = Normalize(name)
	for i := range slice {
		if f(slice[i], name) {
			hashes = append(hashes, backend.Hashes[slice[i]]...)
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

func SearchHasPrefix(name string) ([]string, error) {
	if name == "" {
		return backend.AllUUIDs, nil
	}
	results, err := searchFunc(name, backend.AllNames, strings.HasPrefix)
	if err != nil {
		return searchFunc(name, backend.AlternateNames, strings.HasPrefix)
	}
	return results, nil
}

func SearchContains(name string) ([]string, error) {
	results, err := searchFunc(name, backend.AllNames, strings.Contains)
	if err != nil {
		return searchFunc(name, backend.AlternateNames, strings.Contains)
	}
	return results, nil
}

func SearchRegexp(name string) ([]string, error) {
	var hashes []string
	re, err := regexp.Compile(name)
	if err != nil {
		return nil, err
	}
	for i := range backend.AllUUIDs {
		if re.MatchString(backend.UUIDs[backend.AllUUIDs[i]].Name) {
			hashes = append(hashes, backend.AllUUIDs[i])
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

func SearchSealedContains(name string) ([]string, error) {
	return searchFunc(name, backend.AllSealed, strings.Contains)
}

func Printings4Card(name string) ([]string, error) {
	entry, found := backend.Cards[Normalize(name)]
	if !found {
		return nil, ErrCardDoesNotExist
	}
	return entry.Printings, nil
}

func HasExtendedArtPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_effect", mtgjson.FrameEffectExtendedArt, editions...)
}

func HasBorderlessPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "border_color", mtgjson.BorderColorBorderless, editions...)
}

func HasShowcasePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_effect", mtgjson.FrameEffectShowcase, editions...)
}

func HasReskinPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypeGodzilla, editions...)
}

func HasPromoPackPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypePromoPack, editions...)
}

func HasPrereleasePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypePrerelease, editions...)
}

func HasRetroFramePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_version", "1997", editions...)
}

func HasNonfoilPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishNonfoil, editions...)
}

func HasFoilPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishFoil, editions...)
}

func HasEtchedPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishEtched, editions...)
}

var setsWithChineseAltArt = []string{
	"5ED", "6ED", "7ED", "8ED", "9ED", "10E",
	"POR", "USG", "PCY", "INV", "ODY", "DKS",
	"RAV", "DIS",
	"TSP", "FUT", "PLC",
	"LRW", "MOR", "SHM", "EVE",
}

func HasChineseAltArtPrinting(name string, editions ...string) bool {
	if len(editions) > 0 {
		return hasPrinting(name, "promo_type", mtgjson.PromoTypeSChineseAltArt, editions...)
	}
	return hasPrinting(name, "promo_type", mtgjson.PromoTypeSChineseAltArt, setsWithChineseAltArt...)
}

func hasPrinting(name, field, value string, editions ...string) bool {
	if backend.Sets == nil {
		return false
	}

	var checkFunc func(mtgjson.Card, string) bool
	switch field {
	case "promo_type":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasPromoType(value)
		}
	case "frame_effect":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasFrameEffect(value)
		}
	case "border_color":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.BorderColor == value
		}
	case "frame_version":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.FrameVersion == value
		}
	case "finish":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasFinish(value)
		}
	default:
		return false
	}

	card, found := backend.Cards[Normalize(name)]
	if !found {
		cc := &Card{
			Name: name,
		}
		adjustName(cc)
		card, found = backend.Cards[Normalize(cc.Name)]
		if !found {
			return false
		}
	}
	for _, code := range card.Printings {
		var set *mtgjson.Set
		if len(editions) > 0 {
			set, found = backend.Sets[editions[0]]
			if !found {
				set, _ = GetSetByName(editions[0])
			}
		}
		if set == nil {
			set, found = backend.Sets[code]
			if !found {
				continue
			}
		}
		for _, in := range set.Cards {
			if (card.Name == in.Name) && checkFunc(in, value) {
				return true
			}
		}
	}

	return false
}

func BoosterGen(setCode, boosterType string) ([]string, error) {
	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}
	if set.Booster == nil {
		return nil, ErrEditionNoSealed
	}
	_, found := set.Booster[boosterType]
	if !found {
		return nil, ErrEditionNoBoosterSheet
	}

	// Pick a rarity distribution as defined in Contents at random using their weight
	var choices []weightedrand.Choice[map[string]int, int]
	for _, booster := range set.Booster[boosterType].Boosters {
		choices = append(choices, weightedrand.NewChoice(booster.Contents, booster.Weight))
	}
	sheetChooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return nil, err
	}

	contents := sheetChooser.Pick()

	var picks []string
	// For each sheet, pick a card at random using the weight
	for sheetName, frequency := range contents {
		// Grab the sheet
		sheet := set.Booster[boosterType].Sheets[sheetName]

		if sheet.Fixed {
			// Fixed means there is no randomness, just pick the cards as listed
			for cardId, frequency := range sheet.Cards {
				// Convert to custom IDs
				uuid, err := MatchId(cardId, sheet.Foil, strings.Contains(sheetName, "etched"))
				if err != nil {
					return nil, ErrCardUnknownId
				}
				for j := 0; j < frequency; j++ {
					picks = append(picks, uuid)
				}
			}
		} else {
			var duplicated map[string]bool
			var balanced map[string]bool

			// Prepare maps to keep track of duplicates and balaced colors if necessary
			if !sheet.AllowDuplicates {
				duplicated = map[string]bool{}
			}
			if sheet.BalanceColors {
				balanced = map[string]bool{}
			}

			// Move sheet data into randutil data type
			var cardChoices []weightedrand.Choice[string, int]
			for cardId, weight := range sheet.Cards {
				cardChoices = append(cardChoices, weightedrand.NewChoice(cardId, weight))
			}

			cardChooser, err := weightedrand.NewChooser(cardChoices...)
			if err != nil {
				return nil, err
			}

			// Pick a card uuid as many times as defined by its frequency
			// Note that it's ok to pick the same card from the same sheet multiple times
			for j := 0; j < frequency; j++ {
				item := cardChooser.Pick()
				// Validate card exists (ie in case of online-only printing)
				co, found := backend.UUIDs[item]
				if !found {
					j--
					continue
				}

				// Check if we need to reroll due to BalanceColors
				if sheet.BalanceColors && frequency > 4 && j < 5 {
					// Reroll for the first five cards, the first 5 cards cannot be multicolor or colorless
					if len(co.Colors) != 1 {
						j--
						continue
					}
					// Reroll if one of the single colors was already found
					if balanced[co.Colors[0]] {
						j--
						continue
					}
					// Found!
					balanced[co.Colors[0]] = true
				}

				// Check if the sheet allows duplicates, and, if not, pick again
				// in case the uuid was already picked
				if !sheet.AllowDuplicates {
					if duplicated[item] {
						j--
						continue
					}
					duplicated[item] = true
				}

				// Convert to custom IDs
				uuid, err := MatchId(item, sheet.Foil, strings.Contains(sheetName, "etched"))
				if err != nil {
					j--
					continue
				}

				picks = append(picks, uuid)
			}
		}
	}

	return picks, nil
}
