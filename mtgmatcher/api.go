package mtgmatcher

import (
	"errors"
	"math/rand"
	"regexp"
	"strings"

	"github.com/mroth/weightedrand/v2"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

func GetUUIDs() []string {
	return backend.AllUUIDs
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

	// 1. Check if input is just the set code
	set, err := GetSet(edition)
	if err == nil {
		return set, nil
	}

	// 2. Check if input is the full name of the set
	for _, set := range backend.Sets {
		if Equals(set.Name, edition) {
			return set, nil
		}
	}

	// 3. Attempt adjusting the edition with a fake card object
	card := &Card{
		Edition: edition,
	}
	if len(flags) > 0 {
		card.Foil = flags[0]
	}
	adjustEdition(card)

	for _, set := range backend.Sets {
		if Equals(set.Name, card.Edition) {
			return set, nil
		}
	}

	// 4. We tried
	return nil, ErrCardNotInEdition
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

func AllPromoTypes() []string {
	return backend.AllPromoTypes
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

func HasSerializedPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypeSerialized, editions...)
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
				uuid, err := MatchId(cardId, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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
				uuid, err := MatchId(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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

func GetPicksForDeck(setCode, deckName string) ([]string, error) {
	var picks []string

	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}

		for _, board := range [][]mtgjson.DeckCard{
			deck.Bonus, deck.Commander, deck.MainBoard, deck.SideBoard,
		} {
			for _, card := range board {
				uuid, err := MatchId(card.UUID, card.IsFoil)
				if err != nil {
					return nil, err
				}

				for i := 0; i < card.Count; i++ {
					picks = append(picks, uuid)
				}
			}
		}
	}

	return picks, nil
}

func GetPicksForSealed(setCode, sealedUUID string) ([]string, error) {
	var picks []string

	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
					uuid, err := MatchId(content.UUID, content.Foil)
					if err != nil {
						return nil, err
					}
					picks = append(picks, uuid)
				case "pack":
					boosterPicks, err := BoosterGen(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					picks = append(picks, boosterPicks...)
				case "sealed":
					for i := 0; i < content.Count; i++ {
						sealedPicks, err := GetPicksForSealed(content.Set, content.UUID)
						if err != nil {
							return nil, err
						}
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := GetPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					picks = append(picks, deckPicks...)
				case "variable":
					variableIndex := rand.Intn(len(content.Configs))
					for _, card := range content.Configs[variableIndex]["card"] {
						uuid, err := MatchId(card.UUID, card.Foil)
						if err != nil {
							return nil, err
						}
						picks = append(picks, uuid)
					}
					for _, booster := range content.Configs[variableIndex]["pack"] {
						boosterPicks, err := BoosterGen(booster.Set, booster.Code)
						if err != nil {
							return nil, err
						}
						picks = append(picks, boosterPicks...)
					}
					for _, sealed := range content.Configs[variableIndex]["sealed"] {
						for i := 0; i < sealed.Count; i++ {
							sealedPicks, err := GetPicksForSealed(sealed.Set, sealed.UUID)
							if err != nil {
								return nil, err
							}
							picks = append(picks, sealedPicks...)
						}
					}
					for _, deck := range content.Configs[variableIndex]["deck"] {
						deckPicks, err := GetPicksForDeck(deck.Set, deck.Name)
						if err != nil {
							return nil, err
						}
						picks = append(picks, deckPicks...)
					}
				case "other":
				default:
					return nil, errors.New("unknown key")
				}
			}
		}
	}

	if len(picks) == 0 {
		return nil, errors.New("nothing was picked")
	}

	return picks, nil
}

func SealedIsRandom(setCode, sealedUUID string) bool {
	set, err := GetSet(setCode)
	if err != nil {
		return false
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		if product.Contents == nil {
			return true
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
				case "pack":
					return true
				case "sealed":
					if SealedIsRandom(content.Set, content.UUID) {
						return true
					}
				case "deck":
				case "variable":
					return true
				case "other":
				default:
					return true
				}
			}
		}
	}

	return false
}

func SealedHasDecklist(setCode, sealedUUID string) bool {
	set, err := GetSet(setCode)
	if err != nil {
		return false
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "sealed":
					if SealedHasDecklist(content.Set, content.UUID) {
						return true
					}
				case "deck":
					return true
				}
			}
		}
	}

	return false
}

type ProductProbabilities struct {
	UUID        string
	Probability float64
}

func SealedBoosterProbabilities(setCode, boosterType string) ([]ProductProbabilities, error) {
	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}

	boosterConfig, found := set.Booster[boosterType]
	if !found {
		return nil, errors.New("booster not found")
	}

	tmp := map[string]float64{}
	for _, booster := range boosterConfig.Boosters {
		for sheetName, count := range booster.Contents {
			probs, err := SealedSheetProbabilities(setCode, boosterType, sheetName)
			if err != nil {
				return nil, err
			}

			// Add to the map in case a card appears in different slots/sheets
			// (very common in old boosters, and crazy modern boosters)
			for i := range probs {
				tmp[probs[i].UUID] += probs[i].Probability * float64(count) * float64(booster.Weight)
			}
		}
	}

	// Normalize booster weight with the provided totals
	var probabilities []ProductProbabilities
	for uuid, probability := range tmp {
		probabilities = append(probabilities, ProductProbabilities{
			UUID:        uuid,
			Probability: probability / float64(boosterConfig.BoostersTotalWeight),
		})
	}
	return probabilities, nil
}

func SealedSheetProbabilities(setCode, boosterType, sheetName string) ([]ProductProbabilities, error) {
	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}

	sheet, found := set.Booster[boosterType].Sheets[sheetName]
	if !found {
		return nil, errors.New("sheet not found")
	}

	isEtched := strings.Contains(strings.ToLower(sheetName), "etched")
	var probs []ProductProbabilities

	for cardId, frequency := range sheet.Cards {
		uuid, err := MatchId(cardId, sheet.Foil, isEtched)
		if err != nil {
			return nil, err
		}
		probability := float64(frequency) / float64(sheet.TotalWeight)
		probs = append(probs, ProductProbabilities{
			UUID:        uuid,
			Probability: probability,
		})
	}

	return probs, nil
}

func GetProbabilitiesForSealed(setCode, sealedUUID string) ([]ProductProbabilities, error) {
	set, err := GetSet(setCode)
	if err != nil {
		return nil, err
	}

	var probs []ProductProbabilities

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
					uuid, err := MatchId(content.UUID, content.Foil)
					if err != nil {
						return nil, err
					}
					probs = append(probs, ProductProbabilities{
						UUID:        uuid,
						Probability: 1,
					})
				case "pack":
					boosterProbabilities, err := SealedBoosterProbabilities(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					probs = append(probs, boosterProbabilities...)
				case "sealed":
					sealedProbabilities, err := GetProbabilitiesForSealed(content.Set, content.UUID)
					if err != nil {
						return nil, err
					}
					for i := range sealedProbabilities {
						sealedProbabilities[i].Probability *= float64(content.Count)
					}
					probs = append(probs, sealedProbabilities...)
				case "deck":
					deckPicks, err := GetPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					for _, uuid := range deckPicks {
						probs = append(probs, ProductProbabilities{
							UUID:        uuid,
							Probability: 1,
						})
					}
				case "variable":
					variableIndex := rand.Intn(len(content.Configs))
					for _, card := range content.Configs[variableIndex]["card"] {
						uuid, err := MatchId(card.UUID, card.Foil)
						if err != nil {
							return nil, err
						}
						probs = append(probs, ProductProbabilities{
							UUID:        uuid,
							Probability: 1,
						})
					}
					for _, booster := range content.Configs[variableIndex]["pack"] {
						boosterProbabilities, err := SealedBoosterProbabilities(booster.Set, booster.Code)
						if err != nil {
							return nil, err
						}
						probs = append(probs, boosterProbabilities...)
					}
					for _, sealed := range content.Configs[variableIndex]["sealed"] {
						sealedProbabilities, err := GetProbabilitiesForSealed(sealed.Set, sealed.UUID)
						if err != nil {
							return nil, err
						}
						for i := range sealedProbabilities {
							sealedProbabilities[i].Probability *= float64(sealed.Count)
						}
						probs = append(probs, sealedProbabilities...)
					}
					for _, deck := range content.Configs[variableIndex]["deck"] {
						deckPicks, err := GetPicksForDeck(deck.Set, deck.Name)
						if err != nil {
							return nil, err
						}
						for _, uuid := range deckPicks {
							probs = append(probs, ProductProbabilities{
								UUID:        uuid,
								Probability: 1,
							})
						}
					}
				case "other":
				default:
					return nil, errors.New("unknown key")
				}
			}
		}
	}

	if len(probs) == 0 {
		return nil, errors.New("nothing was probs")
	}

	return probs, nil
}
