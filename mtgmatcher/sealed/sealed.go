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
//
// Each draw walks the weights against their total, so opening a booster
// builds nothing per call. A weight below one is never drawn.
func BoosterPicks(b *mtgmatcher.Backend, setCode, boosterType string) ([]string, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}
	if set.Booster == nil {
		return nil, fmt.Errorf("%s is missing booster information", strings.ToUpper(setCode))
	}
	booster, found := set.Booster[boosterType]
	if !found {
		return nil, fmt.Errorf("%s has no booster named '%s'", strings.ToUpper(setCode), boosterType)
	}

	// Pick a rarity distribution as defined in Contents at random using their weight
	contents, err := drawContents(booster, boosterType)
	if err != nil {
		return nil, err
	}

	var picks []string
	// For each sheet, pick a card at random using the weight
	for sheetName, count := range contents {
		// Grab the sheet
		sheet := booster.Sheets[sheetName]
		isEtched := strings.Contains(strings.ToLower(sheetName), "etched")

		if sheet.Fixed {
			// Fixed means there is no randomness, just pick the cards as listed
			for cardID, subcount := range sheet.Cards {
				// Convert to custom IDs
				uuid, err := b.MatchID(cardID, sheet.Foil, isEtched)
				if err != nil {
					return nil, err
				}
				for range subcount {
					picks = append(picks, uuid)
				}
			}
			continue
		}

		// This is an approximation of the actual algorithm since we don't
		// have precise print sheet information available.
		// The first N cards (where N is the number of colors) get picked
		// from these special sheets.
		// See https://github.com/taw/magic-search-engine/blob/master/search-engine/lib/color_balanced_card_sheet.rb
		if sheet.BalanceColors {
			before := len(picks)
			picks, err = drawBalanced(b, picks, sheet, sheetName, count)
			if err != nil {
				return nil, err
			}
			// Each slot filled reduces the number of remaining ones
			count -= len(picks) - before
		}

		total := sheetWeight(sheet.Cards, nil)
		if total <= 0 {
			return nil, fmt.Errorf("sheet '%s' has no weight to draw by", sheetName)
		}

		// Cards already drawn, where the sheet allows no duplicates
		var drawn []string

		// Pick a card uuid as many times as defined by its count
		// (count may have been adjusted due to balanceColors)
		for j := 0; j < count; j++ {
			var uuid string
			var e int

			// Repeat rerolls up to the specified threshold
			for e = 0; e < maxRerollThreshold; e++ {
				item := drawCard(sheet.Cards, total, nil)

				// Validate card exists (ie in case of online-only printing)
				_, found := b.UUIDs[item]
				if !found {
					return nil, fmt.Errorf("sheet '%s' contains an unknown id (%s)", sheetName, item)
				}

				// Check if the sheet allows duplicates, and, if not, pick again
				// in case the uuid was already picked
				if !sheet.AllowDuplicates {
					if slices.Contains(drawn, item) {
						continue
					}
					drawn = append(drawn, item)
				}

				// Convert to custom IDs
				uuid, err = b.MatchID(item, sheet.Foil, isEtched)
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

	return picks, nil
}

// drawContents picks one of a booster type's slot layouts by its weight.
func drawContents(booster mtgmatcher.Booster, boosterType string) (map[string]int, error) {
	var total int
	for _, layout := range booster.Boosters {
		total += max(layout.Weight, 0)
	}
	if total <= 0 {
		return nil, fmt.Errorf("booster '%s' has no weight to draw by", boosterType)
	}
	r := rand.Intn(total)
	for _, layout := range booster.Boosters {
		if r < layout.Weight {
			return layout.Contents, nil
		}
		r -= max(layout.Weight, 0)
	}
	return nil, fmt.Errorf("booster '%s' drew past its weight", boosterType)
}

// sheetWeight is the weight of the cards keep accepts, or of all of them
// where keep is nil, a weight below one counting as none.
func sheetWeight(cards map[string]int, keep func(cardID string) bool) int {
	var total int
	for cardID, weight := range cards {
		if weight > 0 && (keep == nil || keep(cardID)) {
			total += weight
		}
	}
	return total
}

// drawCard picks one card id from a sheet by weight, total being what
// sheetWeight gives for the same keep, and above zero. Walking a map in
// whatever order it iterates is still an exact weighted draw.
func drawCard(cards map[string]int, total int, keep func(cardID string) bool) string {
	r := rand.Intn(total)
	for cardID, weight := range cards {
		if weight <= 0 || (keep != nil && !keep(cardID)) {
			continue
		}
		if r < weight {
			return cardID
		}
		r -= weight
	}
	return ""
}

// inBalance reports whether a card belongs to the subsheet of one color of
// its identity, or, for color "", to the colorless subsheet, which holds no
// lands.
func inBalance(co *mtgmatcher.CardObject, color string) bool {
	if color == "" {
		return len(co.ColorIdentity) < 1 && !slices.Contains(co.Types, "Land")
	}
	return slices.Contains(co.ColorIdentity, color)
}

// drawBalanced fills one slot of a color-balanced sheet from each color on
// it, appending to picks. A multicolor card stands in every color of its
// identity, at its own weight.
func drawBalanced(b *mtgmatcher.Backend, picks []string, sheet mtgmatcher.Sheet, sheetName string, count int) ([]string, error) {
	var colors []string
	var totals []int
	add := func(color string, weight int) {
		idx := slices.Index(colors, color)
		if idx < 0 {
			colors = append(colors, color)
			totals = append(totals, 0)
			idx = len(colors) - 1
		}
		totals[idx] += weight
	}
	for cardID, weight := range sheet.Cards {
		co, found := b.UUIDs[cardID]
		if !found {
			return nil, fmt.Errorf("sheet '%s' contains an unknown id (%s)", sheetName, cardID)
		}
		weight = max(weight, 0)
		for _, color := range co.ColorIdentity {
			add(color, weight)
		}
		if inBalance(co, "") {
			add("", weight)
		}
	}

	// Sanity check
	if count < len(colors) {
		return nil, fmt.Errorf("fewer slots (%d) than colors (%d) for %s", count, len(colors), sheetName)
	}

	isEtched := strings.Contains(strings.ToLower(sheetName), "etched")
	for i, color := range colors {
		if totals[i] <= 0 {
			return nil, fmt.Errorf("sheet '%s' has no weight to draw color '%s' by", sheetName, color)
		}
		item := drawCard(sheet.Cards, totals[i], func(cardID string) bool {
			return inBalance(b.UUIDs[cardID], color)
		})

		// Convert to custom IDs
		uuid, err := b.MatchID(item, sheet.Foil, isEtched)
		if err != nil {
			return nil, err
		}
		picks = append(picks, uuid)
	}
	return picks, nil
}

// DeckCards returns the uuids a preconstructed deck contains.
//
// A deck listing more than maxOpened cards is refused, like a product.
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

				if card.Count > maxOpened-len(picks) {
					return nil, errTooLarge
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

			if card.Count > maxOpened-len(picks) {
				return nil, errTooLarge
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

// DeckIsRandom reports whether an opened copy of a deck from the set can hold
// other cards than its list, as a Countdown Kit's foil roll does.
func DeckIsRandom(deckSet string) bool {
	return deckFoilTenths(deckSet) > 0
}

// sealedKinds are the kinds of entry a walk reads from a product's contents,
// in the order it reads them, so the same product answers the same way every
// time. "other" lists what holds no card and is never read.
var sealedKinds = []string{"card", "pack", "deck", "sealed", "variable"}

// maxOpened is the most one walk opens: products entered, held copies and
// uuids yielded together. The largest real products, a case of six 36-pack
// boxes, come to a few thousand; past a million the data is corrupt, and
// opening it copy by copy would spin or exhaust memory.
const maxOpened = 1 << 20

var errTooLarge = fmt.Errorf("sealed product opens more than %d items", maxOpened)

// walk is one pass over a product and everything it holds. It tracks the
// products on the path from the root, not every one seen, so a product held
// twice is opened twice while one holding itself is refused, and it spends
// from the maxOpened budget.
type walk struct {
	path  map[string]bool
	spent int
}

func newWalk() *walk {
	return &walk{path: map[string]bool{}}
}

// spend charges n items to the walk, failing once it is past maxOpened; a
// negative count opens nothing and refunds nothing.
func (w *walk) spend(n int) error {
	n = max(n, 0)
	if n > maxOpened-w.spent {
		w.spent = maxOpened
		return errTooLarge
	}
	w.spent += n
	return nil
}

// enter puts a product on the walk's path, failing if it is already there or
// the budget is spent; a nil error is paired with a call to leave.
func (w *walk) enter(setCode, sealedUUID string) error {
	err := w.spend(1)
	if err != nil {
		return err
	}
	key := strings.ToUpper(setCode) + "/" + sealedUUID
	if w.path[key] {
		return fmt.Errorf("sealed product %s holds itself", sealedUUID)
	}
	w.path[key] = true
	return nil
}

func (w *walk) leave(setCode, sealedUUID string) {
	delete(w.path, strings.ToUpper(setCode)+"/"+sealedUUID)
}

// ProductDecklist returns the uuids of the fixed decks a sealed product
// contains, for the products whose contents are known rather than drawn.
//
// Fixed means fixed: asking twice answers twice the same. Where a product
// draws for something - the Countdown Kits upgrade cards to foil at a chance
// the data cannot express - that belongs to opening a copy rather than to the
// product, and lives in ProductPicks and in the counts ProductCounts carries.
func ProductDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string) ([]string, error) {
	return productDecklist(b, setCode, sealedUUID, newWalk())
}

func productDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string, w *walk) ([]string, error) {
	var picks []string

	if !hasDecklist(b, setCode, sealedUUID, w) {
		return nil, errors.New("product does not have a decklist")
	}

	err := w.enter(setCode, sealedUUID)
	if err != nil {
		return nil, err
	}
	defer w.leave(setCode, sealedUUID)

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
					err = w.spend(1)
					if err != nil {
						return nil, err
					}
					picks = append(picks, uuid)
				case "sealed":
					err := w.spend(content.Count)
					if err != nil {
						return nil, err
					}
					for i := 0; i < content.Count; i++ {
						// Content of sealed is unpredictable, so ignore errors
						// but for a walk that has opened too much
						sealedPicks, err := productDecklist(b, content.Set, content.UUID, w)
						if errors.Is(err, errTooLarge) {
							return nil, err
						}
						picks = append(picks, sealedPicks...)
					}
				case "deck":
					deckPicks, err := DeckCards(b, content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					err = w.spend(len(deckPicks))
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
	return productPicks(b, setCode, sealedUUID, newWalk())
}

func productPicks(b *mtgmatcher.Backend, setCode, sealedUUID string, w *walk) ([]string, error) {
	var picks []string

	err := w.enter(setCode, sealedUUID)
	if err != nil {
		return nil, err
	}
	defer w.leave(setCode, sealedUUID)

	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}
		contentPicks, err := pickContents(b, product.Contents, productNamesEtched(product.Name), w)
		if err != nil {
			return nil, err
		}
		picks = append(picks, contentPicks...)
	}

	return picks, nil
}

// pickContents opens every entry of a product's contents, or of the variable
// config a copy holds; etched is what the product's name says.
func pickContents(b *mtgmatcher.Backend, contents map[string][]mtgmatcher.SealedContent, etched bool, w *walk) ([]string, error) {
	var picks []string

	for _, kind := range sealedKinds {
		for _, content := range contents[kind] {
			switch kind {
			case "card":
				uuid, err := b.MatchID(content.UUID, content.Foil, etched)
				if err != nil {
					return nil, err
				}
				err = w.spend(1)
				if err != nil {
					return nil, err
				}
				picks = append(picks, uuid)
			case "pack":
				boosterPicks, err := BoosterPicks(b, content.Set, content.Code)
				if err != nil {
					return nil, err
				}
				err = w.spend(len(boosterPicks))
				if err != nil {
					return nil, err
				}
				picks = append(picks, boosterPicks...)
			case "sealed":
				err := w.spend(content.Count)
				if err != nil {
					return nil, err
				}
				for i := 0; i < content.Count; i++ {
					sealedPicks, err := productPicks(b, content.Set, content.UUID, w)
					if err != nil {
						if optionalSealed(content.Name) && !errors.Is(err, errTooLarge) {
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
				err = w.spend(len(deckPicks))
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

				configPicks, err := pickContents(b, variableChooser.Pick(), etched, w)
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
// holds, as Chance in Weight; a config stating none, or publishing an empty
// list of them, is one of n equals.
func configOdds(config map[string][]mtgmatcher.SealedContent, n int) mtgmatcher.SealedContent {
	odds := config["variable_config"]
	if len(odds) == 0 {
		return mtgmatcher.SealedContent{Chance: 1, Weight: n}
	}
	return odds[0]
}

// IsRandom reports whether opening the product twice can give different
// cards, which is what separates a booster from a fixed deck.
//
// A product held inside itself adds nothing past its first copy on the path.
func IsRandom(b *mtgmatcher.Backend, setCode, sealedUUID string) bool {
	return isRandom(b, setCode, sealedUUID, newWalk())
}

func isRandom(b *mtgmatcher.Backend, setCode, sealedUUID string, w *walk) bool {
	err := w.enter(setCode, sealedUUID)
	if err != nil {
		return false
	}
	defer w.leave(setCode, sealedUUID)

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
					if isRandom(b, content.Set, content.UUID, w) {
						return true
					}
				case "deck":
					if DeckIsRandom(content.Set) {
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
//
// A product held inside itself adds nothing past its first copy on the path.
func HasDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string) bool {
	return hasDecklist(b, setCode, sealedUUID, newWalk())
}

func hasDecklist(b *mtgmatcher.Backend, setCode, sealedUUID string, w *walk) bool {
	err := w.enter(setCode, sealedUUID)
	if err != nil {
		return false
	}
	defer w.leave(setCode, sealedUUID)

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
					if hasDecklist(b, content.Set, content.UUID, w) {
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
// stands for, and Total what the whole product yields.
type Count struct {
	UUID          string
	ExpectedCount float64
	Copies        int
}

// Total is how many copies of the uuid the whole product yields on average,
// across every copy of the product the entry stands for: ExpectedCount alone
// is the chance in one copy, and undercounts a product held several times.
func (c Count) Total() float64 {
	return c.ExpectedCount * float64(c.Copies)
}

// newCount is an entry for one copy of the product the card comes from.
func newCount(uuid string, expectedCount float64) Count {
	return Count{UUID: uuid, ExpectedCount: expectedCount, Copies: 1}
}

// BoosterCounts returns how many copies of each card one booster of the given
// type holds on average, by the same weights BoosterPicks draws by.
func BoosterCounts(b *mtgmatcher.Backend, setCode, boosterType string) ([]Count, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	boosterConfig, found := set.Booster[boosterType]
	if !found {
		return nil, fmt.Errorf("booster '%s' not found", boosterType)
	}

	var total int
	for _, booster := range boosterConfig.Boosters {
		total += max(booster.Weight, 0)
	}
	if total <= 0 {
		return nil, fmt.Errorf("booster '%s' has no weight to draw by", boosterType)
	}

	// Sheets are read in order, so a card on several sheets sums to the same
	// bits on every call.
	tmp := map[string]float64{}
	for _, booster := range boosterConfig.Boosters {
		if booster.Weight <= 0 {
			continue
		}
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
		counts = append(counts, newCount(uuid, tmp[uuid]/float64(total)))
	}
	return counts, nil
}

// SheetCounts returns how likely each card on one sheet is to be drawn from
// it, which for a single draw is also the copies it yields on average, by the
// same weights BoosterPicks draws by.
func SheetCounts(b *mtgmatcher.Backend, setCode, boosterType, sheetName string) ([]Count, error) {
	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	sheet, found := set.Booster[boosterType].Sheets[sheetName]
	if !found {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}

	total := sheetWeight(sheet.Cards, nil)
	if total <= 0 {
		return nil, fmt.Errorf("sheet '%s' has no weight to draw by", sheetName)
	}

	isEtched := strings.Contains(strings.ToLower(sheetName), "etched")
	var counts []Count

	for cardID, count := range sheet.Cards {
		if count <= 0 {
			continue
		}
		uuid, err := b.MatchID(cardID, sheet.Foil, isEtched)
		if err != nil {
			return nil, err
		}
		probability := float64(count) / float64(total)
		counts = append(counts, newCount(uuid, probability))
	}

	return counts, nil
}

// ProductCounts returns how many copies of each card opening the product
// yields on average, across every pack and deck it contains: per copy of the
// product each entry comes from, held Copies times.
func ProductCounts(b *mtgmatcher.Backend, setCode, sealedUUID string) ([]Count, error) {
	return productCounts(b, setCode, sealedUUID, newWalk())
}

func productCounts(b *mtgmatcher.Backend, setCode, sealedUUID string, w *walk) ([]Count, error) {
	err := w.enter(setCode, sealedUUID)
	if err != nil {
		return nil, err
	}
	defer w.leave(setCode, sealedUUID)

	set, err := b.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	var counts []Count

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}
		found, err := contentCounts(b, product.Contents, productNamesEtched(product.Name), w)
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
func contentCounts(b *mtgmatcher.Backend, contents map[string][]mtgmatcher.SealedContent, etched bool, w *walk) ([]Count, error) {
	var counts []Count

	for _, kind := range sealedKinds {
		for _, content := range contents[kind] {
			switch kind {
			case "card":
				uuid, err := b.MatchID(content.UUID, content.Foil, etched)
				if err != nil {
					return nil, err
				}
				err = w.spend(1)
				if err != nil {
					return nil, err
				}
				counts = append(counts, newCount(uuid, 1))
			case "pack":
				boosterCounts, err := BoosterCounts(b, content.Set, content.Code)
				if err != nil {
					return nil, err
				}
				err = w.spend(len(boosterCounts))
				if err != nil {
					return nil, err
				}
				counts = append(counts, boosterCounts...)
			case "sealed":
				heldCounts, err := productCounts(b, content.Set, content.UUID, w)
				if err != nil {
					if optionalSealed(content.Name) && !errors.Is(err, errTooLarge) {
						continue
					}
					return nil, err
				}
				// Each level multiplies its count onto what the held product
				// returned: 6 boxes of 36 packs hold each pack 216 times.
				// No entry stands for more copies than a walk may open.
				for i := range heldCounts {
					if content.Count > 0 && heldCounts[i].Copies > maxOpened/content.Count {
						return nil, errTooLarge
					}
					heldCounts[i].Copies *= content.Count
				}
				counts = append(counts, heldCounts...)
			case "deck":
				deckPicks, err := DeckCards(b, content.Set, content.Name)
				if err != nil {
					return nil, err
				}
				err = w.spend(len(deckPicks))
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

					variableCounts, err := contentCounts(b, config, etched, w)
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
