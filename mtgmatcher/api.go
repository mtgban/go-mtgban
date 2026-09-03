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

// GetUUIDs returns every non-sealed uuid in the datastore. The result aliases
// the backend index and must not be modified.
func (b *Backend) GetUUIDs() []string {
	return b.AllUUIDs
}

// GetUUIDs returns every non-sealed uuid in the default datastore.
func GetUUIDs() []string {
	return defaultBackend.GetUUIDs()
}

// GetSealedUUIDs returns every sealed uuid in the datastore. The result
// aliases the backend index and must not be modified.
func (b *Backend) GetSealedUUIDs() []string {
	return b.AllSealedUUIDs
}

// GetSealedUUIDs returns every sealed uuid in the default datastore.
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

// GetUUID returns the card object stored for the given uuid. The object
// is shared across all callers and must not be modified.
func (b *Backend) GetUUID(uuid string) (*CardObject, error) {
	if b.UUIDs == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := b.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownID
	}

	return co, nil
}

// GetUUID returns the card object for the uuid, from the default datastore.
// The object is shared and must not be modified.
func GetUUID(uuid string) (*CardObject, error) {
	return defaultBackend.GetUUID(uuid)
}

// GetAllSets returns every set code in the datastore.
func (b *Backend) GetAllSets() []string {
	return b.AllSets
}

// GetAllSets returns every set code in the default datastore.
func GetAllSets() []string {
	return defaultBackend.GetAllSets()
}

// GetSet returns the set with this code, matched case-insensitively. The set
// is shared and must not be modified.
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

// GetSet returns the set with this code, from the default datastore.
func GetSet(code string) (*Set, error) {
	return defaultBackend.GetSet(code)
}

// GetSetByName returns the set an edition string names, trying the set code
// first, then the full name, then the many spellings storefronts use for it.
// The set is shared and must not be modified.
func (b *Backend) GetSetByName(edition string) (*Set, error) {
	if b.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	// 1. Check if input is just the set code
	set, err := b.GetSet(edition)
	if err == nil {
		return set, nil
	}

	// 2. Check if input is the full name of the set
	set, found := b.NormalizedSets[Normalize(edition)]
	if found {
		return set, nil
	}

	// 3. Ask the game to spell the edition the way the datastore does
	// (skipped when no GameRules are attached, e.g. a hand-built Backend)
	if b.rules != nil {
		set, found = b.NormalizedSets[Normalize(b.rules.AliasEdition(b, edition))]
		if found {
			return set, nil
		}
	}

	// 4. We tried
	return nil, ErrCardNotInEdition
}

// GetSetByName returns the set an edition string names, from the default
// datastore.
func GetSetByName(edition string) (*Set, error) {
	return defaultBackend.GetSetByName(edition)
}

// AllPromoTypes returns every promo type present in the default datastore.
func AllPromoTypes() []string {
	return defaultBackend.AllPromoTypes
}

// AllNames returns every card or sealed name in the default datastore, in the
// requested form: normalized, lowercase, or canonical. An unknown form returns
// nothing.
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

// SearchEquals returns the uuids of every printing whose name matches exactly,
// ignoring case and punctuation. An empty name returns everything.
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

// SearchEquals searches the default datastore by exact name.
func SearchEquals(name string) ([]string, error) {
	return defaultBackend.SearchEquals(name)
}

// SearchSealedEquals is the sealed-product counterpart of SearchEquals.
func (b *Backend) SearchSealedEquals(name string) ([]string, error) {
	return b.searchFunc(name, b.AllSealed, func(a, c string) bool {
		return a == c
	})
}

// SearchSealedEquals searches the default datastore's sealed products by exact
// name.
func SearchSealedEquals(name string) ([]string, error) {
	return defaultBackend.SearchSealedEquals(name)
}

func (b *Backend) searchFunc(name string, slice []string, f func(string, string) bool) ([]string, error) {
	var hashes []string
	name = Normalize(name)
	// One printing can answer to several entries - a game may index a card
	// under a qualified spelling as well as its bare name, and a prefix
	// match reaches both - so the buckets overlap and a caller would be
	// handed the same card twice.
	seen := map[string]bool{}
	for i := range slice {
		if !f(slice[i], name) {
			continue
		}
		for _, uuid := range b.Hashes[slice[i]] {
			if seen[uuid] {
				continue
			}
			seen[uuid] = true
			hashes = append(hashes, uuid)
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

// SearchHasPrefix returns the uuids of every printing whose name starts with
// the input, which is how a truncated listing is recovered.
func (b *Backend) SearchHasPrefix(name string) ([]string, error) {
	if name == "" {
		return b.AllUUIDs, nil
	}
	return b.searchFunc(name, b.AllNames, strings.HasPrefix)
}

// SearchHasPrefix searches the default datastore by name prefix.
func SearchHasPrefix(name string) ([]string, error) {
	return defaultBackend.SearchHasPrefix(name)
}

// SearchContains returns the uuids of every printing whose name contains the
// input.
func (b *Backend) SearchContains(name string) ([]string, error) {
	return b.searchFunc(name, b.AllNames, strings.Contains)
}

// SearchContains searches the default datastore by substring.
func SearchContains(name string) ([]string, error) {
	return defaultBackend.SearchContains(name)
}

// SearchRegexp returns the uuids of every printing whose name matches the
// expression.
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

// SearchRegexp searches the default datastore by regular expression.
func SearchRegexp(name string) ([]string, error) {
	return defaultBackend.SearchRegexp(name)
}

// SearchSealedContains is the sealed-product counterpart of SearchContains.
func (b *Backend) SearchSealedContains(name string) ([]string, error) {
	return b.searchFunc(name, b.AllSealed, strings.Contains)
}

// SearchSealedContains searches the default datastore's sealed products by
// substring.
func SearchSealedContains(name string) ([]string, error) {
	return defaultBackend.SearchSealedContains(name)
}

// entry4Name returns the bucket entry actually named this way.
//
// Cards are deliberately hashed under their face, flavor and printed
// names as well, so that a query naming a single face still finds the
// card. One bucket therefore holds several distinct cards, each carrying
// the properties of its own card: "Servo" hashes both the Servo token and
// "Servo // Thopter".
//
// Lookups pick between them in this order: the card whose name matches
// verbatim, then any whose name normalizes the same, then the first entry
// in the bucket.
//
// The first case is what tells apart the cards that share a bucket outright:
// normalization folds case and punctuation, so "Mr. 1 (Daz.Bonez)" and
// "Mr.1 (Daz.Bonez)" are two cards under one key. The second prefers an
// entry that owns the name over one the bucket holds only as an alias, for
// a query spelled unlike any of them. The last covers buckets reached only
// by an alias, such as a flavor name.
func (b *Backend) entry4Name(name string) (*CardObject, bool) {
	norm := Normalize(name)
	uuids, found := b.Hashes[norm]
	if !found {
		return nil, false
	}
	var normalized *CardObject
	for _, uuid := range uuids {
		entry, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		if strings.EqualFold(entry.Name, name) {
			return entry, true
		}
		if normalized == nil && Normalize(entry.Name) == norm {
			normalized = entry
		}
	}
	if normalized != nil {
		return normalized, true
	}
	entry, found := b.UUIDs[uuids[0]]
	return entry, found
}

// NameIsToken reports whether the card actually named this way is a token.
// Exported because the only caller now lives in the per-game rules packages.
func (b *Backend) NameIsToken(name string) bool {
	entry, found := b.entry4Name(name)
	return found && entry.Layout == "token"
}

// Printings4Card returns the set codes a card by this name was printed in.
func (b *Backend) Printings4Card(name string) ([]string, error) {
	if b.Hashes == nil {
		return nil, ErrDatastoreEmpty
	}
	entry, found := b.entry4Name(name)
	if !found {
		return nil, ErrCardDoesNotExist
	}
	return entry.Printings, nil
}

// Printings4Card returns the sets a card was printed in, from the default
// datastore.
func Printings4Card(name string) ([]string, error) {
	return defaultBackend.Printings4Card(name)
}

// HasNonfoilPrinting reports whether the card was ever sold nonfoil, narrowed
// to the given editions when any are named.
func (b *Backend) HasNonfoilPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishNonfoil, editions...)
}

// HasFoilPrinting reports whether the named card carries the foil slot. A
// game whose finish is not foilness answers yes throughout: Yu-Gi-Oh's
// treatment is the rarity, so every printing points both flag slots at its
// default print run and none of them is a foil anybody sells.
func (b *Backend) HasFoilPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishFoil, editions...)
}

// HasFoilPrinting queries the default datastore.
func HasFoilPrinting(name string, editions ...string) bool {
	return defaultBackend.HasFoilPrinting(name, editions...)
}

// HasEtchedPrinting reports whether the card was ever sold etched, narrowed to
// the given editions when any are named.
func (b *Backend) HasEtchedPrinting(name string, editions ...string) bool {
	return b.hasPrinting(name, "finish", FinishEtched, editions...)
}

// HasEtchedPrinting queries the default datastore.
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

	// Resolve which real card name the query means, the way Printings4Card
	// does: the case-exact entry when one exists, the first normalized
	// match otherwise. The hash bucket conflates names that normalize the
	// same but belong to different cards ("Mr. 1 (Daz.Bonez)" beside
	// "Mr.1 (Daz.Bonez)"), and the printings of one must never answer for
	// the other.
	entry, found := b.entry4Name(name)
	if !found {
		if b.rules == nil {
			return false
		}
		cc := &InputCard{
			Name: name,
		}
		b.rules.AdjustName(b, cc)
		entry, found = b.entry4Name(cc.Name)
		if !found {
			return false
		}
	}
	canonicalName := entry.Name
	uuids := b.Hashes[Normalize(canonicalName)]

	// A pinned edition narrows the check to that set alone; when it cannot
	// be resolved, every printing is checked, like the set loop used to.
	var pinnedCode string
	if len(editions) > 0 {
		set := b.Sets[editions[0]]
		if set == nil {
			set, _ = b.GetSetByName(editions[0])
		}
		if set != nil {
			pinnedCode = set.Code
		}
	}

	// The hash bucket holds every card whose name (or alias) normalizes the
	// same, so this visits exactly the cards the old per-set scans compared
	// with Equals - minus the scans: iterating full sets for every printing
	// made this function the dominant cost of Match for widely printed
	// cards. The name check stays because aliases (flavor or printed names)
	// hash into the same bucket but never matched the scans' Equals against
	// the real card name.
	for _, uuid := range uuids {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		if pinnedCode != "" && co.SetCode != pinnedCode {
			continue
		}
		if strings.EqualFold(co.Name, canonicalName) && checkFunc(co.Card, value) {
			return true
		}
	}

	return false
}

// HasPrinting reports whether any printing of the card carries this value in
// the named field, narrowed to the given editions when any are named.
func HasPrinting(name, field, value string, editions ...string) bool {
	return defaultBackend.hasPrinting(name, field, value, editions...)
}

const maxRerollThreshold = 50

// BoosterGen opens one booster of the given type, drawing from the set's
// sheets with the weights the real product uses, and returns what came out.
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
			for cardID, subcount := range sheet.Cards {
				// Convert to custom IDs
				uuid, err := MatchID(cardID, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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
					uuid, err := MatchID(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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
					uuid, err = MatchID(item, sheet.Foil, strings.Contains(strings.ToLower(sheetName), "etched"))
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

// BoosterGen opens a booster from the default datastore.
func BoosterGen(setCode, boosterType string) ([]string, error) {
	return defaultBackend.BoosterGen(setCode, boosterType)
}

// GetPicksForDeck returns the uuids a preconstructed deck contains.
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
				uuid, err := MatchID(card.UUID, card.IsFoil, card.IsEtched)
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

// GetDecklist returns the uuids of the fixed decks a sealed product contains,
// for the products whose contents are known rather than drawn.
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
					uuid, err := MatchID(content.UUID, content.Foil)
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
								uuidFoil, err := MatchID(deckPicks[i], true)
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

// GetDecklist queries the default datastore.
func GetDecklist(setCode, sealedUUID string) ([]string, error) {
	return defaultBackend.GetDecklist(setCode, sealedUUID)
}

// GetPicksForSealed opens a sealed product once, resolving its packs and decks
// and drawing whatever it leaves to chance.
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
					uuid, err := MatchID(content.UUID, content.Foil)
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
								uuidFoil, err := MatchID(deckPicks[i], true)
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
						uuid, err := MatchID(card.UUID, card.Foil)
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

// GetPicksForSealed opens a product from the default datastore.
func GetPicksForSealed(setCode, sealedUUID string) ([]string, error) {
	return defaultBackend.GetPicksForSealed(setCode, sealedUUID)
}

// SealedIsRandom reports whether opening the product twice can give different
// cards, which is what separates a booster from a fixed deck.
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

// SealedIsRandom queries the default datastore.
func SealedIsRandom(setCode, sealedUUID string) bool {
	return defaultBackend.SealedIsRandom(setCode, sealedUUID)
}

// SealedCardUnit returns how many cards the product holds in total.
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
					result++
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

// SealedHasDecklist reports whether the product contains a fixed deck whose
// contents are known.
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

// SealedHasDecklist queries the default datastore.
func SealedHasDecklist(setCode, sealedUUID string) bool {
	return defaultBackend.SealedHasDecklist(setCode, sealedUUID)
}

// ProductProbabilities is one uuid and how likely opening a product is to
// yield it.
type ProductProbabilities struct {
	UUID        string
	Probability float64
}

// SealedBoosterProbabilities returns how likely each card is to appear in one
// booster of the given type.
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

// SealedSheetProbabilities returns how likely each card on one sheet is to be
// drawn from it.
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

	for cardID, count := range sheet.Cards {
		uuid, err := MatchID(cardID, sheet.Foil, isEtched)
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

// GetProbabilitiesForSealed returns how likely each card is to appear when the
// whole product is opened, across every pack and deck it contains.
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
					uuid, err := MatchID(content.UUID, content.Foil)
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

							uuidFoil, err := MatchID(uuid, true)
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
							uuid, err := MatchID(card.UUID, card.Foil)
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

// GetProbabilitiesForSealed queries the default datastore.
func GetProbabilitiesForSealed(setCode, sealedUUID string) ([]ProductProbabilities, error) {
	return defaultBackend.GetProbabilitiesForSealed(setCode, sealedUUID)
}

// BuildSealedProductMap indexes the sealed products by one of their outside
// identifiers, skipping any product that does not carry it. A slice usually
// holds a single uuid, but an id shared by a foil and a nonfoil product holds
// both, foil last.
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
			// there isn't any drop containing non-Textured foil versions of the cards
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

// BuildSealedProductMap indexes the default datastore's sealed products by an
// outside identifier.
func BuildSealedProductMap(idName string) map[int][]string {
	return defaultBackend.BuildSealedProductMap(idName)
}

// PromoTypeSlug renders a promo type as the single token that identifies it:
// lower case, with everything that is not a letter or a digit dropped. This
// is the form the games store, so that a promo type is one word wherever it
// is read - a search query is split on whitespace, and Magic's own types
// have always been single words for exactly that reason ("boosterfun").
func PromoTypeSlug(promoType string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(promoType) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// SlugDescribes reports whether a storefront's wording says the words a promo
// type's slug was made from, as a run of whole words.
//
// The slug has lost the spaces, so a plain comparison cannot be used and a
// substring test would be too generous - "metal" is inside "metallic". Words
// are joined back up a run at a time instead, which asks the question the
// slug's own spelling asks: does this wording name this promo type.
func SlugDescribes(wording, slug string) bool {
	if slug == "" {
		return false
	}
	words := strings.Fields(strings.ToLower(wording))
	for i := range words {
		var joined string
		for j := i; j < len(words); j++ {
			joined += PromoTypeSlug(words[j])
			if joined == slug {
				return true
			}
			if len(joined) >= len(slug) {
				break
			}
		}
	}
	return false
}

// containingLabel settles a tie between printings whose labels a catalog
// writes as one phrase rather than as separate tags, and leaves every other
// tie alone.
//
// TCGplayer sells "Monkey.D.Luffy (Super Alternate Art)" beside
// "Monkey.D.Luffy (Red Super Alternate Art)", one parenthetical each, so the
// printings wear one tag apiece and one tag contains the other whole. Both
// are named by a wording that spells the longer out, both name one tag and
// have none left over, and the count cannot separate them - which aliases
// away a wording that could not have been more specific. Containment is what
// the two labels share and what settles it: the tag that spells the other
// out is the one the wording was about.
//
// Nothing settles a tie between labels that are merely different. Flesh and
// Blood writes the finish into the same wording as the variant, so "UPR043
// Cold Foil" on a card whose name says "(Marvel)" describes a cold foil
// label and a marvel one at once - neither containing the other, and the
// longer of them not the answer. That tie is the caller's to break, by
// asking again without the words its finish already consumed.
func containingLabel(wording string, cards []Card) []Card {
	if len(cards) < 2 {
		return cards
	}
	named := make([][]string, len(cards))
	for i, card := range cards {
		for _, promoType := range card.PromoTypes {
			if SlugDescribes(wording, promoType) {
				named[i] = append(named[i], promoType)
			}
		}
	}
	for i := range cards {
		var wider bool
		beats := true
		for j := range cards {
			if i == j {
				continue
			}
			// Containment has to run one way only. Two printings wearing
			// the same label spell each other out, and answering with
			// whichever came first would settle by position what the
			// labels do not settle at all - the edition the printings sit
			// in is what tells those apart, further down.
			if !labelsContain(named[i], named[j]) || labelsContain(named[j], named[i]) {
				beats = false
				break
			}
			wider = true
		}
		if beats && wider {
			return []Card{cards[i]}
		}
	}
	return cards
}

// labelsContain reports whether every label in inner is spelled out by one of
// the labels in outer, which is true of a label against itself.
func labelsContain(outer, inner []string) bool {
	for _, in := range inner {
		if !slices.ContainsFunc(outer, func(out string) bool {
			return strings.Contains(out, in)
		}) {
			return false
		}
	}
	return true
}

// SlugDescribesAny reports whether a wording names any of the promo types a
// printing wears. Every one is asked rather than the first alone: a printing
// carrying two tags is named by either of them.
func SlugDescribesAny(wording string, slugs []string) bool {
	for _, slug := range slugs {
		if SlugDescribes(wording, slug) {
			return true
		}
	}
	return false
}

// DescribedVariants keeps the printings a wording names best, and nothing
// when it names none of them.
//
// A printing is asked about every promo type it wears rather than its first,
// which is what makes the rest of them reachable: the three alternate arts
// of one Yu-Gi-Oh number are told apart by the colour they wear beside the
// shared "Alternate Art", and the only extended art of a Flesh and Blood
// number wears those words behind a colour.
//
// Best is the most tags named, and among those the printing wearing the
// fewest the wording said nothing about. The count is what lets a fuller
// wording win - "Alternate Art Blue" names the printing wearing both over the
// ones wearing either - and the tie-break is what keeps a narrow wording
// narrow, so "Blue" still names the plain blue printing rather than the
// alternate-art one that merely contains the word.
//
// The third is for the labels a catalog writes as one phrase rather than as
// separate tags. TCGplayer sells "Monkey.D.Luffy (Super Alternate Art)"
// beside "Monkey.D.Luffy (Red Super Alternate Art)", one parenthetical
// each, so the printings wear one tag apiece and one tag contains the other
// whole. Both are named by a wording that spells the longer out, both name
// one tag and have none left over, and the first two rules call that a tie -
// which aliases away a wording that could not have been more specific.
// Containment settles it the way specificity does, and only where it runs
// one way: the tag that spells the other out is the one the wording was
// about, while two printings wearing the same label say nothing about which
// was meant and are left as they are. See containingLabel. A game whose
// catalog splits the same distinction into separate tags never reaches
// there, the count having already answered.
func DescribedVariants(wording string, cards []Card) []Card {
	var best []Card
	var most, unnamed int
	for _, card := range cards {
		var named int
		for _, promoType := range card.PromoTypes {
			if SlugDescribes(wording, promoType) {
				named++
			}
		}
		if named == 0 {
			continue
		}
		rest := len(card.PromoTypes) - named
		if named > most || (named == most && rest < unnamed) {
			best, most, unnamed = nil, named, rest
		}
		if named == most && rest == unnamed {
			best = append(best, card)
		}
	}
	best = containingLabel(wording, best)
	return best
}

// PromoTypeLabel spells a promo type the way it was written before it became
// a token, falling back on the token itself where no fuller spelling was
// kept. Callers displaying a promo type should ask for this rather than
// title-casing the token, which cannot put back the spaces it dropped.
func (b *Backend) PromoTypeLabel(promoType string) string {
	if label := b.PromoTypeLabels[promoType]; label != "" {
		return label
	}
	return Title(promoType)
}

// PromoTypeLabel spells a promo type as PromoTypeLabel does, in the default
// backend.
func PromoTypeLabel(promoType string) string {
	return defaultBackend.PromoTypeLabel(promoType)
}
