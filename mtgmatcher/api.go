package mtgmatcher

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/mroth/weightedrand/v2"
)

func (b *Backend) GetUUIDs() []string {
	return b.AllUUIDs
}

func GetUUIDs() []string {
	return defaultBackend.GetUUIDs()
}

func (b *Backend) GetSealedUUIDs() []string {
	return b.AllSealedUUIDs
}

func GetSealedUUIDs() []string {
	return defaultBackend.GetSealedUUIDs()
}

// GetUUIDsInSet returns every non-sealed uuid printed in the given set,
// foil and etched variants included, in sorted order. The result aliases
// the backend index and must not be modified; callers spanning multiple
// sets append the per-set results themselves.
func GetUUIDsInSet(code string) []string {
	return defaultBackend.SetUUIDs[strings.ToUpper(code)]
}

// GetSealedUUIDsInSet is the sealed-product counterpart of GetUUIDsInSet.
func GetSealedUUIDsInSet(code string) []string {
	return defaultBackend.SetSealedUUIDs[strings.ToUpper(code)]
}

func (b *Backend) GetUUID(uuid string) (*CardObject, error) {
	if b.UUIDs == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := b.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownId
	}

	return &co, nil
}

func GetUUID(uuid string) (*CardObject, error) {
	return defaultBackend.GetUUID(uuid)
}

func (b *Backend) GetAllSets() []string {
	return b.AllSets
}

func GetAllSets() []string {
	return defaultBackend.GetAllSets()
}

func (b *Backend) GetSet(code string) (*Set, error) {
	if b.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	set, found := b.Sets[strings.ToUpper(code)]
	if !found {
		return nil, ErrCardNotInEdition
	}

	return set, nil
}

func GetSet(code string) (*Set, error) {
	return defaultBackend.GetSet(code)
}

func (b *Backend) GetSetByName(edition string, flags ...bool) (*Set, error) {
	if b.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	// 1. Check if input is just the set code
	set, err := b.GetSet(edition)
	if err == nil {
		return set, nil
	}

	// 2. Check if input is the full name of the set
	for _, set := range b.Sets {
		if Equals(set.Name, edition) {
			return set, nil
		}
	}

	// 3. Attempt adjusting the edition with a fake card object
	// (skipped when no GameRules are attached, e.g. a hand-built Backend)
	card := &InputCard{
		Edition: edition,
	}
	if len(flags) > 0 {
		card.Foil = flags[0]
	}
	if b.rules != nil {
		b.rules.AdjustEdition(b, card)
	}

	for _, set := range b.Sets {
		if Equals(set.Name, card.Edition) {
			return set, nil
		}
	}

	// 4. We tried
	return nil, ErrCardNotInEdition
}

func GetSetByName(edition string, flags ...bool) (*Set, error) {
	return defaultBackend.GetSetByName(edition, flags...)
}

func (b *Backend) ExternalUUID(id string) string {
	return b.ExternalIdentifiers[id]
}

func ExternalUUID(id string) string {
	return defaultBackend.ExternalUUID(id)
}

func AllPromoTypes() []string {
	return defaultBackend.AllPromoTypes
}

// Return a slice of all names loaded up, in different formats
// normalized, lowercase, canonical, or alternate (normalized)
func AllNames(variant string, sealed bool) []string {
	switch variant {
	case "normalized":
		if sealed {
			return defaultBackend.AllSealed
		}
		return defaultBackend.AllNames
	case "canonical":
		if sealed {
			return defaultBackend.AllCanonicalSealed
		}
		return defaultBackend.AllCanonicalNames
	case "lowercase":
		if sealed {
			return defaultBackend.AllLowerSealed
		}
		return defaultBackend.AllLowerNames
	}
	return nil
}

func (b *Backend) SearchEquals(name string) ([]string, error) {
	if name == "" {
		return b.AllUUIDs, nil
	}

	results, found := b.Hashes[Normalize(name)]
	if !found {
		return nil, ErrCardDoesNotExist
	}

	return results, nil
}

func SearchEquals(name string) ([]string, error) {
	return defaultBackend.SearchEquals(name)
}

func (b *Backend) SearchSealedEquals(name string) ([]string, error) {
	return b.searchFunc(name, b.AllSealed, func(a, c string) bool {
		return a == c
	})
}

func SearchSealedEquals(name string) ([]string, error) {
	return defaultBackend.SearchSealedEquals(name)
}

func (b *Backend) searchFunc(name string, slice []string, f func(string, string) bool) ([]string, error) {
	var hashes []string
	name = Normalize(name)
	for i := range slice {
		if f(slice[i], name) {
			hashes = append(hashes, b.Hashes[slice[i]]...)
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

func searchFunc(name string, slice []string, f func(string, string) bool) ([]string, error) {
	return defaultBackend.searchFunc(name, slice, f)
}

func (b *Backend) SearchHasPrefix(name string) ([]string, error) {
	if name == "" {
		return b.AllUUIDs, nil
	}
	return b.searchFunc(name, b.AllNames, strings.HasPrefix)
}

func SearchHasPrefix(name string) ([]string, error) {
	return defaultBackend.SearchHasPrefix(name)
}

func (b *Backend) SearchContains(name string) ([]string, error) {
	return b.searchFunc(name, b.AllNames, strings.Contains)
}

func SearchContains(name string) ([]string, error) {
	return defaultBackend.SearchContains(name)
}

func (b *Backend) SearchRegexp(name string) ([]string, error) {
	var hashes []string
	re, err := regexp.Compile(name)
	if err != nil {
		return nil, err
	}
	for i := range b.AllUUIDs {
		if re.MatchString(b.UUIDs[b.AllUUIDs[i]].Name) {
			hashes = append(hashes, b.AllUUIDs[i])
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

func SearchRegexp(name string) ([]string, error) {
	return defaultBackend.SearchRegexp(name)
}

func (b *Backend) SearchSealedContains(name string) ([]string, error) {
	return b.searchFunc(name, b.AllSealed, strings.Contains)
}

func SearchSealedContains(name string) ([]string, error) {
	return defaultBackend.SearchSealedContains(name)
}

func (b *Backend) Printings4Card(name string) ([]string, error) {
	if b.Hashes == nil {
		return nil, ErrDatastoreEmpty
	}
	uuids, found := b.Hashes[Normalize(name)]
	if !found {
		return nil, ErrCardDoesNotExist
	}
	entry, found := b.UUIDs[uuids[0]]
	if !found {
		return nil, ErrCardDoesNotExist
	}
	return entry.Printings, nil
}

func Printings4Card(name string) ([]string, error) {
	return defaultBackend.Printings4Card(name)
}

func (b *Backend) HasExtendedArtPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "frame_effect", FrameEffectExtendedArt, editions...)
}

func HasExtendedArtPrinting(name string, editions ...string) bool {
	return defaultBackend.HasExtendedArtPrinting(name, editions...)
}

func (b *Backend) HasBorderlessPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "border_color", BorderColorBorderless, editions...)
}

func HasBorderlessPrinting(name string, editions ...string) bool {
	return defaultBackend.HasBorderlessPrinting(name, editions...)
}

func (b *Backend) HasShowcasePrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "frame_effect", FrameEffectShowcase, editions...)
}

func HasShowcasePrinting(name string, editions ...string) bool {
	return defaultBackend.HasShowcasePrinting(name, editions...)
}

func (b *Backend) HasReskinPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "promo_type", PromoTypeGodzilla, editions...)
}

func HasReskinPrinting(name string, editions ...string) bool {
	return defaultBackend.HasReskinPrinting(name, editions...)
}

func (b *Backend) HasPromoPackPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "promo_type", PromoTypePromoPack, editions...)
}

func HasPromoPackPrinting(name string, editions ...string) bool {
	return defaultBackend.HasPromoPackPrinting(name, editions...)
}

func (b *Backend) HasPrereleasePrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "promo_type", PromoTypePrerelease, editions...)
}

func HasPrereleasePrinting(name string, editions ...string) bool {
	return defaultBackend.HasPrereleasePrinting(name, editions...)
}

func (b *Backend) HasSerializedPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "promo_type", PromoTypeSerialized, editions...)
}

func HasSerializedPrinting(name string, editions ...string) bool {
	return defaultBackend.HasSerializedPrinting(name, editions...)
}

func (b *Backend) HasRetroFramePrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "frame_version", "1997", editions...)
}

func HasRetroFramePrinting(name string, editions ...string) bool {
	return defaultBackend.HasRetroFramePrinting(name, editions...)
}

func (b *Backend) HasNonfoilPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishNonfoil, editions...)
}

func HasNonfoilPrinting(name string, editions ...string) bool {
	return defaultBackend.HasNonfoilPrinting(name, editions...)
}

func (b *Backend) HasFoilPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishFoil, editions...)
}

func HasFoilPrinting(name string, editions ...string) bool {
	return defaultBackend.HasFoilPrinting(name, editions...)
}

func (b *Backend) HasEtchedPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishEtched, editions...)
}

func HasEtchedPrinting(name string, editions ...string) bool {
	return defaultBackend.HasEtchedPrinting(name, editions...)
}

func (b *Backend) hasPrinting(name, field, value string, editions ...string) bool {
	if b.Sets == nil {
		return false
	}

	var checkFunc func(Card, string) bool
	switch field {
	case "promo_type":
		checkFunc = func(card Card, value string) bool {
			return card.HasPromoType(value)
		}
	case "frame_effect":
		checkFunc = func(card Card, value string) bool {
			return card.HasFrameEffect(value)
		}
	case "border_color":
		checkFunc = func(card Card, value string) bool {
			return card.BorderColor == value
		}
	case "frame_version":
		checkFunc = func(card Card, value string) bool {
			return card.FrameVersion == value
		}
	case "finish":
		checkFunc = func(card Card, value string) bool {
			return card.HasFinish(value)
		}
	case "field":
		switch value {
		case "attractionLights":
			checkFunc = func(card Card, value string) bool {
				return card.AttractionLights != nil
			}
		default:
			return false
		}
	default:
		return false
	}

	printings, err := b.Printings4Card(name)
	if err != nil {
		if b.rules == nil {
			return false
		}
		cc := &InputCard{
			Name: name,
		}
		b.rules.AdjustName(b, cc)
		name = cc.Name
		printings, err = b.Printings4Card(name)
		if err != nil {
			return false
		}
	}
	for _, code := range printings {
		var set *Set
		if len(editions) > 0 {
			set = b.Sets[editions[0]]
			if set == nil {
				set, _ = b.GetSetByName(editions[0])
			}
		}
		if set == nil {
			set = b.Sets[code]
			if set == nil {
				continue
			}
		}
		for _, in := range set.Cards {
			if Equals(name, in.Name) && checkFunc(in, value) {
				return true
			}
		}
	}

	return false
}

func HasPrinting(name, field, value string, editions ...string) bool {
	return defaultBackend.hasPrinting(name, field, value, editions...)
}

const maxRerollThreshold = 50

func (b *Backend) BoosterGen(setCode, boosterType string) ([]string, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}
	if set.Booster == nil {
		return nil, fmt.Errorf("%s is missing booster information", strings.ToUpper(setCode))
	}
	_, found := set.Booster[boosterType]
	if !found {
		return nil, fmt.Errorf("%s has no booster named '%s'", strings.ToUpper(setCode), boosterType)
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
	for sheetName, count := range contents {
		// Grab the sheet
		sheet := set.Booster[boosterType].Sheets[sheetName]

		if sheet.Fixed {
			// Fixed means there is no randomness, just pick the cards as listed
			for cardId, subcount := range sheet.Cards {
				// Convert to custom IDs
				uuid, err := MatchId(cardId, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
				if err != nil {
					return nil, err
				}
				for j := 0; j < subcount; j++ {
					picks = append(picks, uuid)
				}
			}
		} else {
			var duplicated map[string]bool
			var balancedSheets map[string][]weightedrand.Choice[string, int]

			// Prepare maps to keep track of duplicates and balaced colors if necessary
			if !sheet.AllowDuplicates {
				duplicated = map[string]bool{}
			}

			// This is an approximation of the actual algorithm since we don't
			// have precise print sheet information availabe.
			// The first N cards (where N is the number of colors) get picked
			// from these special sheets.
			// See https://github.com/taw/magic-search-engine/blob/master/search-engine/lib/color_balanced_card_sheet.rb
			if sheet.BalanceColors {
				balancedSheets = map[string][]weightedrand.Choice[string, int]{}

				// Rescale weights of the subsheets
				mult := 1
				for _, weight := range sheet.Cards {
					mult = LCM(mult, weight)
				}

				// Create subsheets for each color (multi color gets included
				// multiple times)
				for cardId, weight := range sheet.Cards {
					co, found := b.UUIDs[cardId]
					if !found {
						return nil, fmt.Errorf("sheet '%s' contains an unknown id (%s)", sheetName, cardId)
					}

					choice := weightedrand.NewChoice(cardId, weight*mult)
					for _, color := range co.ColorIdentity {
						balancedSheets[color] = append(balancedSheets[color], choice)
					}
					if len(co.ColorIdentity) < 1 && !slices.Contains(co.Types, "Land") {
						balancedSheets["C"] = append(balancedSheets["C"], choice)
					}
				}

				// Sanity check
				if count < len(balancedSheets) {
					return nil, fmt.Errorf("fewer slots (%d) than colors (%d) for %s", count, len(balancedSheets), sheetName)
				}

				// Prefill the balanced slots
				for _, cardChoices := range balancedSheets {
					cardChooser, err := weightedrand.NewChooser(cardChoices...)
					if err != nil {
						return nil, err
					}
					item := cardChooser.Pick()

					// Convert to custom IDs
					uuid, err := MatchId(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
					if err != nil {
						return nil, err
					}

					// Add to what's found
					picks = append(picks, uuid)

					// One slot was filled, reduce the number of remaining ones
					count--
				}
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

			// Pick a card uuid as many times as defined by its count
			// (count may have been adjusted due to balanceColors)
			for j := 0; j < count; j++ {
				var uuid string
				var e int

				// Repeat rerolls up to the specified threshold
				for e = 0; e < maxRerollThreshold; e++ {
					item := cardChooser.Pick()

					// Validate card exists (ie in case of online-only printing)
					_, found := b.UUIDs[item]
					if !found {
						return nil, fmt.Errorf("sheet '%s' contains an unknown id (%s)", sheetName, item)
					}

					// Check if the sheet allows duplicates, and, if not, pick again
					// in case the uuid was already picked
					if !sheet.AllowDuplicates {
						if duplicated[item] {
							continue
						}
						duplicated[item] = true
					}

					// Convert to custom IDs
					uuid, err = MatchId(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
					if err != nil {
						return nil, err
					}

					// Gotem
					break
				}
				if e == maxRerollThreshold {
					return nil, errors.New("reroll threshold reached")
				}

				picks = append(picks, uuid)
			}
		}
	}

	return picks, nil
}

func BoosterGen(setCode, boosterType string) ([]string, error) {
	return defaultBackend.BoosterGen(setCode, boosterType)
}

func (b *Backend) GetPicksForDeck(setCode, deckName string) ([]string, error) {
	var picks []string

	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}

		for i, board := range [][]DeckCard{
			deck.Commander,
			deck.DisplayCommander,
			deck.MainBoard,
			deck.Planes,
			deck.Schemes,
			deck.SideBoard,
			deck.Tokens,
		} {
			for _, card := range board {
				uuid, err := MatchId(card.UUID, card.IsFoil, card.IsEtched)
				if err != nil {
					// XXX: Tokens are not fully loaded so don't error out if one is missing
					if i == 6 {
						continue
					}
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

func GetPicksForDeck(setCode, deckName string) ([]string, error) {
	return defaultBackend.GetPicksForDeck(setCode, deckName)
}

func (b *Backend) GetDecklist(setCode, sealedUUID string) ([]string, error) {
	var picks []string

	if !b.SealedHasDecklist(setCode, sealedUUID) {
		return nil, errors.New("product does not have a decklist")
	}

	set, err := b.GetSet(setCode)
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
				case "sealed":
					for i := 0; i < content.Count; i++ {
						// Content of sealed is unpredictable, so ignore errors
						sealedPicks, _ := b.GetDecklist(content.Set, content.UUID)
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := b.GetPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}

					// This set data cannot be represented in mtgjson data without
					// breaking the output format, instead hack things here
					if content.Set == "slc" {
						for i := 0; i < len(deckPicks)-1; i++ {
							n := rand.Intn(10)
							if n < 3 {
								uuidFoil, err := MatchId(deckPicks[i], true)
								if err != nil {
									continue
								}
								deckPicks[i] = uuidFoil
							}
						}
					}

					picks = append(picks, deckPicks...)
				}
			}
		}
	}

	return picks, nil
}

func GetDecklist(setCode, sealedUUID string) ([]string, error) {
	return defaultBackend.GetDecklist(setCode, sealedUUID)
}

func (b *Backend) GetPicksForSealed(setCode, sealedUUID string) ([]string, error) {
	var picks []string

	set, err := b.GetSet(setCode)
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
					boosterPicks, err := b.BoosterGen(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					picks = append(picks, boosterPicks...)
				case "sealed":
					for i := 0; i < content.Count; i++ {
						sealedPicks, err := b.GetPicksForSealed(content.Set, content.UUID)
						if err != nil {
							// Ignore errors from this type of product as it doesn't
							// change ev much, and hides relevant results
							if strings.Contains(content.Name, "Sample Pack") {
								continue
							}
							return nil, err
						}
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := b.GetPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}

					// This set data cannot be represented in mtgjson data without
					// breaking the output format, instead hack things here
					if content.Set == "slc" {
						for i := 0; i < len(deckPicks)-1; i++ {
							n := rand.Intn(10)
							if n < 3 {
								uuidFoil, err := MatchId(deckPicks[i], true)
								if err != nil {
									continue
								}
								deckPicks[i] = uuidFoil
							}
						}
					}

					picks = append(picks, deckPicks...)
				case "variable":
					// Use weightedrand to pick a configuration for us
					var choices []weightedrand.Choice[map[string][]SealedContent, int]
					for _, config := range content.Configs {
						weightedConfigs, found := config["variable_config"]
						if !found {
							weightedConfigs = append(weightedConfigs, SealedContent{
								Chance: 1,
								Weight: len(content.Configs),
							})
						}
						choices = append(choices, weightedrand.NewChoice(config, weightedConfigs[0].Chance))
					}

					variableChooser, err := weightedrand.NewChooser(choices...)
					if err != nil {
						return nil, err
					}
					config := variableChooser.Pick()

					for _, card := range config["card"] {
						uuid, err := MatchId(card.UUID, card.Foil)
						if err != nil {
							return nil, err
						}
						picks = append(picks, uuid)
					}
					for _, booster := range config["pack"] {
						boosterPicks, err := b.BoosterGen(booster.Set, booster.Code)
						if err != nil {
							return nil, err
						}
						picks = append(picks, boosterPicks...)
					}
					for _, sealed := range config["sealed"] {
						for i := 0; i < sealed.Count; i++ {
							sealedPicks, err := b.GetPicksForSealed(sealed.Set, sealed.UUID)
							if err != nil {
								return nil, err
							}
							picks = append(picks, sealedPicks...)
						}
					}
					for _, deck := range config["deck"] {
						deckPicks, err := b.GetPicksForDeck(deck.Set, deck.Name)
						if err != nil {
							return nil, err
						}
						picks = append(picks, deckPicks...)
					}
				}
			}
		}
	}

	return picks, nil
}

func GetPicksForSealed(setCode, sealedUUID string) ([]string, error) {
	return defaultBackend.GetPicksForSealed(setCode, sealedUUID)
}

func (b *Backend) SealedIsRandom(setCode, sealedUUID string) bool {
	set, err := b.GetSet(setCode)
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
					if b.SealedIsRandom(content.Set, content.UUID) {
						return true
					}
				case "deck":
					// This set data cannot be represented in mtgjson data without
					// breaking the output format, instead hack things here
					if content.Set == "slc" {
						return true
					}
				case "variable":
					return true
				case "other":
				}
			}
		}
	}

	return false
}

func SealedIsRandom(setCode, sealedUUID string) bool {
	return defaultBackend.SealedIsRandom(setCode, sealedUUID)
}

func (b *Backend) SealedCardUnit(setCode, sealedUUID string) int {
	var result int

	set, err := b.GetSet(setCode)
	if err != nil {
		return 0
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
					result += 1
				case "pack",
					"deck":
					result += product.CardCount
				case "sealed":
					result += b.SealedCardUnit(content.Set, content.UUID) * content.Count
				case "variable":
				}
			}
		}
	}

	return result
}

func SealedCardUnit(setCode, sealedUUID string) int {
	return defaultBackend.SealedCardUnit(setCode, sealedUUID)
}

func (b *Backend) SealedHasDecklist(setCode, sealedUUID string) bool {
	set, err := b.GetSet(setCode)
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
					if b.SealedHasDecklist(content.Set, content.UUID) {
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

func SealedHasDecklist(setCode, sealedUUID string) bool {
	return defaultBackend.SealedHasDecklist(setCode, sealedUUID)
}

type ProductProbabilities struct {
	UUID        string
	Probability float64
}

func (b *Backend) SealedBoosterProbabilities(setCode, boosterType string) ([]ProductProbabilities, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	boosterConfig, found := set.Booster[boosterType]
	if !found {
		return nil, fmt.Errorf("booster '%s' not found", boosterType)
	}

	tmp := map[string]float64{}
	for _, booster := range boosterConfig.Boosters {
		for sheetName, count := range booster.Contents {
			probs, err := b.SealedSheetProbabilities(setCode, boosterType, sheetName)
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

func SealedBoosterProbabilities(setCode, boosterType string) ([]ProductProbabilities, error) {
	return defaultBackend.SealedBoosterProbabilities(setCode, boosterType)
}

func (b *Backend) SealedSheetProbabilities(setCode, boosterType, sheetName string) ([]ProductProbabilities, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	sheet, found := set.Booster[boosterType].Sheets[sheetName]
	if !found {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}

	isEtched := strings.Contains(strings.ToLower(sheetName), "etched")
	var probs []ProductProbabilities

	for cardId, count := range sheet.Cards {
		uuid, err := MatchId(cardId, sheet.Foil, isEtched)
		if err != nil {
			return nil, err
		}
		probability := float64(count) / float64(sheet.TotalWeight)
		probs = append(probs, ProductProbabilities{
			UUID:        uuid,
			Probability: probability,
		})
	}

	return probs, nil
}

func SealedSheetProbabilities(setCode, boosterType, sheetName string) ([]ProductProbabilities, error) {
	return defaultBackend.SealedSheetProbabilities(setCode, boosterType, sheetName)
}

func (b *Backend) GetProbabilitiesForSealed(setCode, sealedUUID string) ([]ProductProbabilities, error) {
	set, err := b.GetSet(setCode)
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
					boosterProbabilities, err := b.SealedBoosterProbabilities(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					probs = append(probs, boosterProbabilities...)
				case "sealed":
					sealedProbabilities, err := b.GetProbabilitiesForSealed(content.Set, content.UUID)
					if err != nil {
						// Ignore errors from this type of product as it doesn't
						// change ev much, and hides relevant results
						if strings.Contains(content.Name, "Sample Pack") {
							continue
						}
						return nil, err
					}
					for i := range sealedProbabilities {
						sealedProbabilities[i].Probability *= float64(content.Count)
					}
					probs = append(probs, sealedProbabilities...)
				case "deck":
					deckPicks, err := b.GetPicksForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					for _, uuid := range deckPicks {
						// This set data cannot be represented in mtgjson data without
						// breaking the output format, instead hack things here
						if content.Set == "slc" {
							probNF := ProductProbabilities{
								UUID:        uuid,
								Probability: 0.7,
							}
							probs = append(probs, probNF)

							uuidFoil, err := MatchId(uuid, true)
							if err != nil {
								continue
							}
							probF := ProductProbabilities{
								UUID:        uuidFoil,
								Probability: 0.3,
							}
							probs = append(probs, probF)
						} else {
							probs = append(probs, ProductProbabilities{
								UUID:        uuid,
								Probability: 1,
							})
						}
					}
				case "variable":
					for _, config := range content.Configs {
						// Retrieve the variable configuration and compute the chance of getting this config
						weightedConfigs, found := config["variable_config"]
						if !found {
							weightedConfigs = append(weightedConfigs, SealedContent{
								Chance: 1,
								Weight: len(content.Configs),
							})
						}
						variableChance := float64(weightedConfigs[0].Chance) / float64(weightedConfigs[0].Weight)

						var variableProbs []ProductProbabilities
						for _, card := range config["card"] {
							uuid, err := MatchId(card.UUID, card.Foil)
							if err != nil {
								return nil, err
							}
							variableProbs = append(variableProbs, ProductProbabilities{
								UUID:        uuid,
								Probability: 1,
							})
						}
						for _, booster := range config["pack"] {
							boosterProbabilities, err := b.SealedBoosterProbabilities(booster.Set, booster.Code)
							if err != nil {
								return nil, err
							}
							variableProbs = append(variableProbs, boosterProbabilities...)
						}
						for _, sealed := range config["sealed"] {
							sealedProbabilities, err := b.GetProbabilitiesForSealed(sealed.Set, sealed.UUID)
							if err != nil {
								return nil, err
							}
							for i := range sealedProbabilities {
								sealedProbabilities[i].Probability *= float64(sealed.Count)
							}
							variableProbs = append(variableProbs, sealedProbabilities...)
						}
						for _, deck := range config["deck"] {
							deckPicks, err := b.GetPicksForDeck(deck.Set, deck.Name)
							if err != nil {
								return nil, err
							}
							for _, uuid := range deckPicks {
								variableProbs = append(variableProbs, ProductProbabilities{
									UUID:        uuid,
									Probability: 1,
								})
							}
						}

						// Modify the retrieved probability according to the chance of this configuration
						for i := range variableProbs {
							variableProbs[i].Probability *= variableChance
						}
						// Update output probabilities
						probs = append(probs, variableProbs...)
					}
				}
			}
		}
	}

	return probs, nil
}

func GetProbabilitiesForSealed(setCode, sealedUUID string) ([]ProductProbabilities, error) {
	return defaultBackend.GetProbabilitiesForSealed(setCode, sealedUUID)
}

// Provide a map of ids with a slice of uuids
// For most cases the slice will be of size one, but some ids may hold
// a second uuid representing the foil version of the product
func (b *Backend) BuildSealedProductMap(idName string) map[int][]string {
	productMap := map[int][]string{}
	for _, uuid := range b.AllSealedUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		id := co.Identifiers[idName]

		// Some products do not carry an id because they are already assigned
		// For specific cases, look for them since we have the canonical number
		if id == "" && co.SetCode == "SLD" && strings.HasSuffix(co.Name, " Foil") {
			name := co.Name

			// This list of tags represents products with separate entries, but
			// with the same listing. For example, there is no Textured because
			// there isn't any drop containing non-Texured foil versions of the cards
			for _, tag := range []string{"Foil", "Rainbow", "Galaxy", "Confetti"} {
				name = strings.TrimSuffix(name, tag)
				name = strings.TrimSpace(name)

				uuids, err := b.SearchSealedEquals(name)
				if err != nil {
					continue
				}
				subco, found := b.UUIDs[uuids[0]]
				if !found {
					continue
				}
				id = subco.Identifiers[idName]
			}
		}

		idNum, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		productMap[idNum] = append(productMap[idNum], uuid)

		// Preserve Foil variant at the end of the slice
		sort.Slice(productMap[idNum], func(i, j int) bool {
			coI := b.UUIDs[productMap[idNum][i]]
			coJ := b.UUIDs[productMap[idNum][j]]
			return coI.Name < coJ.Name
		})
	}
	return productMap
}

func BuildSealedProductMap(idName string) map[int][]string {
	return defaultBackend.BuildSealedProductMap(idName)
}
