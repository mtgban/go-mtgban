// Package sealed opens sealed product: what one copy of a product, a
// booster or a deck holds, drawn at random where the product leaves it to
// chance, and how many copies of each card a product holds on average.
package sealed

import (
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"strings"

	"github.com/mroth/weightedrand/v2"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const maxRerollThreshold = 50

// BoosterPicks opens one booster of the given type, drawing from the set's
// sheets with the weights the real product uses, and returns what came out.
func BoosterPicks(b *mtgmatcher.Backend, setCode, boosterType string) ([]string, error) {
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
			for cardID, subcount := range sheet.Cards {
				// Convert to custom IDs
				uuid, err := b.MatchID(cardID, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
				if err != nil {
					return nil, err
				}
				for range subcount {
					picks = append(picks, uuid)
				}
			}
		} else {
			var duplicated map[string]bool
			var balancedSheets map[string][]weightedrand.Choice[string, int]

			// Prepare maps to keep track of duplicates and balanced colors if necessary
			if !sheet.AllowDuplicates {
				duplicated = map[string]bool{}
			}

			// This is an approximation of the actual algorithm since we don't
			// have precise print sheet information available.
			// The first N cards (where N is the number of colors) get picked
			// from these special sheets.
			// See https://github.com/taw/magic-search-engine/blob/master/search-engine/lib/color_balanced_card_sheet.rb
			if sheet.BalanceColors {
				balancedSheets = map[string][]weightedrand.Choice[string, int]{}

				// Rescale weights of the subsheets
				mult := 1
				for _, weight := range sheet.Cards {
					mult = leastCommonMultiple(mult, weight)
				}

				// Create subsheets for each color (multi color gets included
				// multiple times)
				for cardID, weight := range sheet.Cards {
					co, found := b.UUIDs[cardID]
					if !found {
						return nil, fmt.Errorf("sheet '%s' contains an unknown id (%s)", sheetName, cardID)
					}

					choice := weightedrand.NewChoice(cardID, weight*mult)
					for _, color := range co.ColorIdentity {
						balancedSheets[color] = append(balancedSheets[color], choice)
					}
					if len(co.ColorIdentity) < 1 && !slices.Contains(co.Types, "Land") {
						balancedSheets["colorless"] = append(balancedSheets["colorless"], choice)
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
					uuid, err := b.MatchID(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
					if err != nil {
						return nil, err
					}

					// Add to what's found
					picks = append(picks, uuid)

					// One slot was filled, reduce the number of remaining ones
					count--
				}
			}

			// Move sheet data into weightedrand choices
			var cardChoices []weightedrand.Choice[string, int]
			for cardID, weight := range sheet.Cards {
				cardChoices = append(cardChoices, weightedrand.NewChoice(cardID, weight))
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
					uuid, err = b.MatchID(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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

// DeckCards returns the uuids a preconstructed deck contains.
func DeckCards(b *mtgmatcher.Backend, setCode, deckName string) ([]string, error) {
	var picks []string

	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}

		for _, board := range [][]mtgmatcher.DeckCard{
			deck.Commander,
			deck.DisplayCommander,
			deck.MainBoard,
			deck.Planes,
			deck.Schemes,
			deck.SideBoard,
		} {
			for _, card := range board {
				uuid, err := b.MatchID(card.UUID, card.IsFoil, card.IsEtched)
				if err != nil {
					return nil, err
				}

				for range card.Count {
					picks = append(picks, uuid)
				}
			}
		}

		// The datastore does not hold every token a deck lists, and a token
		// it lacks is left out rather than failing the deck.
		for _, card := range deck.Tokens {
			uuid, err := b.MatchID(card.UUID, card.IsFoil, card.IsEtched)
			if err != nil {
				continue
			}

			for range card.Count {
				picks = append(picks, uuid)
			}
		}
	}

	return picks, nil
}

// productNamesEtched reports whether a sealed product's name says the cards
// it holds are the etched printings. mtgjson has no field for it: a Secret
// Lair sold etched is a separate product whose card contents carry the plain
// foil flag, because etched is the only foil those cards come in.
//
// Read a word at a time rather than by substring or suffix: a Foundations
// commander deck called "Wretched Ranks" holds nothing etched, and "Secret
// Lair Drop The Tokyo Lands Etched Foil" does even though the name does not
// end there.
func productNamesEtched(name string) bool {
	for _, word := range strings.Fields(name) {
		if strings.EqualFold(word, "Etched") {
			return true
		}
	}
	return false
}

// optionalSealed reports whether a nested product that cannot be opened is
// left out of the product holding it rather than failing it: a sample pack
// barely moves the value and would hide everything else.
func optionalSealed(name string) bool {
	return strings.Contains(name, "Sample Pack")
}

// deckFoilTenths returns how many cards in ten of a deck from the set come
// foil in an opened copy, a roll mtgjson has no field for: three in ten for
// the Secret Lair Countdown Kits.
func deckFoilTenths(deckSet string) int {
	if deckSet == "slc" {
		return 3
	}
	return 0
}

// sealedKinds are the kinds of entry a walk reads from a product's contents,
// in the order it reads them, so the same product answers the same way every
// time. "other" lists what holds no card and is never read.
var sealedKinds = []string{"card", "pack", "deck", "sealed", "variable"}

// ProductDecklist returns the uuids of the fixed decks a sealed product
// contains, for the products whose contents are known rather than drawn.
//
// Fixed means fixed: asking twice answers twice the same. Where a product
// draws for something - the Countdown Kits upgrade cards to foil at a chance
// the data cannot express - that belongs to opening a copy rather than to the
// product, and lives in ProductPicks and in the counts ProductCounts carries.
func ProductDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string) ([]string, error) {
	var picks []string

	if !HasDecklist(b, setCode, sealedUUID) {
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
		etched := productNamesEtched(product.Name)

		for _, kind := range sealedKinds {
			for _, content := range product.Contents[kind] {
				switch kind {
				case "card":
					uuid, err := b.MatchID(content.UUID, content.Foil, etched)
					if err != nil {
						return nil, err
					}
					picks = append(picks, uuid)
				case "sealed":
					for i := 0; i < content.Count; i++ {
						// Content of sealed is unpredictable, so ignore errors
						sealedPicks, _ := ProductDecklist(b, content.Set, content.UUID)
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := DeckCards(b, content.Set, content.Name)
					if err != nil {
						return nil, err
					}

					picks = append(picks, deckPicks...)
				}
			}
		}
	}

	return picks, nil
}

// ProductPicks opens a sealed product once, resolving its packs and decks
// and drawing whatever it leaves to chance.
func ProductPicks(b *mtgmatcher.Backend, setCode, sealedUUID string) ([]string, error) {
	var picks []string

	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}
		contentPicks, err := pickContents(b, product.Contents, productNamesEtched(product.Name))
		if err != nil {
			return nil, err
		}
		picks = append(picks, contentPicks...)
	}

	return picks, nil
}

// pickContents opens every entry of a product's contents, or of the variable
// config a copy holds; etched is what the product's name says.
func pickContents(b *mtgmatcher.Backend, contents map[string][]mtgmatcher.SealedContent, etched bool) ([]string, error) {
	var picks []string

	for _, kind := range sealedKinds {
		for _, content := range contents[kind] {
			switch kind {
			case "card":
				uuid, err := b.MatchID(content.UUID, content.Foil, etched)
				if err != nil {
					return nil, err
				}
				picks = append(picks, uuid)
			case "pack":
				boosterPicks, err := BoosterPicks(b, content.Set, content.Code)
				if err != nil {
					return nil, err
				}
				picks = append(picks, boosterPicks...)
			case "sealed":
				for i := 0; i < content.Count; i++ {
					sealedPicks, err := ProductPicks(b, content.Set, content.UUID)
					if err != nil {
						if optionalSealed(content.Name) {
							continue
						}
						return nil, err
					}
					picks = append(picks, sealedPicks...)
				}
			case "deck":
				deckPicks, err := DeckCards(b, content.Set, content.Name)
				if err != nil {
					return nil, err
				}

				tenths := deckFoilTenths(content.Set)
				if tenths > 0 {
					for i := range deckPicks {
						n := rand.Intn(10)
						if n < tenths {
							uuidFoil, err := b.MatchID(deckPicks[i], true)
							if err != nil {
								continue
							}
							deckPicks[i] = uuidFoil
						}
					}
				}

				picks = append(picks, deckPicks...)
			case "variable":
				var choices []weightedrand.Choice[map[string][]mtgmatcher.SealedContent, int]
				for _, config := range content.Configs {
					odds := configOdds(config, len(content.Configs))
					choices = append(choices, weightedrand.NewChoice(config, odds.Chance))
				}

				variableChooser, err := weightedrand.NewChooser(choices...)
				if err != nil {
					return nil, err
				}

				configPicks, err := pickContents(b, variableChooser.Pick(), etched)
				if err != nil {
					return nil, err
				}
				picks = append(picks, configPicks...)
			}
		}
	}

	return picks, nil
}

// configOdds returns the odds a variable entry's config is the one a copy
// holds, as Chance in Weight; a config stating none is one of n equals.
func configOdds(config map[string][]mtgmatcher.SealedContent, n int) mtgmatcher.SealedContent {
	odds, found := config["variable_config"]
	if !found {
		return mtgmatcher.SealedContent{Chance: 1, Weight: n}
	}
	return odds[0]
}

// IsRandom reports whether opening the product twice can give different
// cards, which is what separates a booster from a fixed deck.
func IsRandom(b *mtgmatcher.Backend, setCode, sealedUUID string) bool {
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

		for _, kind := range sealedKinds {
			for _, content := range product.Contents[kind] {
				switch kind {
				case "card":
				case "pack":
					return true
				case "sealed":
					if IsRandom(b, content.Set, content.UUID) {
						return true
					}
				case "deck":
					if deckFoilTenths(content.Set) > 0 {
						return true
					}
				case "variable":
					return true
				}
			}
		}
	}

	return false
}

// HasDecklist reports whether the product contains a fixed deck whose
// contents are known.
func HasDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string) bool {
	set, err := b.GetSet(setCode)
	if err != nil {
		return false
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for _, kind := range sealedKinds {
			for _, content := range product.Contents[kind] {
				switch kind {
				case "sealed":
					if HasDecklist(b, content.Set, content.UUID) {
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

// Count is one uuid and how many copies of it one copy of the product it
// comes from yields on average, which can be more than one for a card drawn
// from several slots. Copies is how many copies of that product the entry
// stands for, so the whole product yields ExpectedCount * Copies.
type Count struct {
	UUID          string
	ExpectedCount float64
	Copies        int
}

// newCount is an entry for one copy of the product the card comes from.
func newCount(uuid string, expectedCount float64) Count {
	return Count{UUID: uuid, ExpectedCount: expectedCount, Copies: 1}
}

// BoosterCounts returns how many copies of each card one booster of the given
// type holds on average.
func BoosterCounts(b *mtgmatcher.Backend, setCode, boosterType string) ([]Count, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	boosterConfig, found := set.Booster[boosterType]
	if !found {
		return nil, fmt.Errorf("booster '%s' not found", boosterType)
	}

	// Sheets are read in order, so a card on several sheets sums to the same
	// bits on every call.
	tmp := map[string]float64{}
	for _, booster := range boosterConfig.Boosters {
		for _, sheetName := range slices.Sorted(maps.Keys(booster.Contents)) {
			count := booster.Contents[sheetName]
			sheetCounts, err := SheetCounts(b, setCode, boosterType, sheetName)
			if err != nil {
				return nil, err
			}

			// Add to the map in case a card appears in different slots/sheets
			// (very common in old boosters, and crazy modern boosters)
			for i := range sheetCounts {
				tmp[sheetCounts[i].UUID] += sheetCounts[i].ExpectedCount * float64(count) * float64(booster.Weight)
			}
		}
	}

	// Normalize booster weight with the provided totals
	var counts []Count
	for _, uuid := range slices.Sorted(maps.Keys(tmp)) {
		counts = append(counts, newCount(uuid, tmp[uuid]/float64(boosterConfig.BoostersTotalWeight)))
	}
	return counts, nil
}

// SheetCounts returns how likely each card on one sheet is to be drawn from
// it, which for a single draw is also the copies it yields on average.
func SheetCounts(b *mtgmatcher.Backend, setCode, boosterType, sheetName string) ([]Count, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	sheet, found := set.Booster[boosterType].Sheets[sheetName]
	if !found {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}

	isEtched := strings.Contains(strings.ToLower(sheetName), "etched")
	var counts []Count

	for cardID, count := range sheet.Cards {
		uuid, err := b.MatchID(cardID, sheet.Foil, isEtched)
		if err != nil {
			return nil, err
		}
		probability := float64(count) / float64(sheet.TotalWeight)
		counts = append(counts, newCount(uuid, probability))
	}

	return counts, nil
}

// ProductCounts returns how many copies of each card opening the product
// yields on average, across every pack and deck it contains: per copy of the
// product each entry comes from, held Copies times.
func ProductCounts(b *mtgmatcher.Backend, setCode, sealedUUID string) ([]Count, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	var counts []Count

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}
		found, err := contentCounts(b, product.Contents, productNamesEtched(product.Name))
		if err != nil {
			return nil, err
		}
		counts = append(counts, found...)
	}

	return counts, nil
}

// contentCounts returns how many copies of each card the entries of a
// product's contents, or of one of its variable configs, yield on average;
// etched is what the product's name says.
func contentCounts(b *mtgmatcher.Backend, contents map[string][]mtgmatcher.SealedContent, etched bool) ([]Count, error) {
	var counts []Count

	for _, kind := range sealedKinds {
		for _, content := range contents[kind] {
			switch kind {
			case "card":
				uuid, err := b.MatchID(content.UUID, content.Foil, etched)
				if err != nil {
					return nil, err
				}
				counts = append(counts, newCount(uuid, 1))
			case "pack":
				boosterCounts, err := BoosterCounts(b, content.Set, content.Code)
				if err != nil {
					return nil, err
				}
				counts = append(counts, boosterCounts...)
			case "sealed":
				heldCounts, err := ProductCounts(b, content.Set, content.UUID)
				if err != nil {
					if optionalSealed(content.Name) {
						continue
					}
					return nil, err
				}
				// Each level multiplies its count onto what the held product
				// returned: 6 boxes of 36 packs hold each pack 216 times.
				for i := range heldCounts {
					heldCounts[i].Copies *= content.Count
				}
				counts = append(counts, heldCounts...)
			case "deck":
				deckPicks, err := DeckCards(b, content.Set, content.Name)
				if err != nil {
					return nil, err
				}
				tenths := deckFoilTenths(content.Set)
				for _, uuid := range deckPicks {
					if tenths > 0 {
						countNF := newCount(uuid, float64(10-tenths)/10)
						counts = append(counts, countNF)

						uuidFoil, err := b.MatchID(uuid, true)
						if err != nil {
							continue
						}
						countF := newCount(uuidFoil, float64(tenths)/10)
						counts = append(counts, countF)
					} else {
						counts = append(counts, newCount(uuid, 1))
					}
				}
			case "variable":
				for _, config := range content.Configs {
					odds := configOdds(config, len(content.Configs))
					variableChance := float64(odds.Chance) / float64(odds.Weight)

					variableCounts, err := contentCounts(b, config, etched)
					if err != nil {
						return nil, err
					}

					// Scale by the chance a copy holds this config
					for i := range variableCounts {
						variableCounts[i].ExpectedCount *= variableChance
					}
					counts = append(counts, variableCounts...)
				}
			}
		}
	}

	return counts, nil
}

// greatestCommonDivisor exists for the multiple below.
func greatestCommonDivisor(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

// leastCommonMultiple is folded over a color-balanced sheet's weights:
// scaled by it, every subsheet keeps its proportions in integers.
func leastCommonMultiple(a, b int) int {
	return a * b / greatestCommonDivisor(a, b)
}
