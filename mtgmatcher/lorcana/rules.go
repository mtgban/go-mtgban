package lorcana

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Lorcana. Lorcana has no edition
// aliases, variant tables, or promo types: a card is identified by name +
// collector number + foil. So most hooks are no-ops and the real work is the
// number disambiguation in FilterCards (foil is honored downstream by output).
type Rules struct{}

func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {}

// AdjustName restores the legacy SimpleSearch prefix fallback: scraper feeds
// sometimes truncate the "Character - Title" name, which used to resolve via
// SearchHasPrefix plus number/foil filtering. When the exact name is unknown,
// scan for cards whose name has the input as a prefix and let the collector
// number and finish narrow them; adopt the name only when exactly one
// survives. If several distinct names survive, the input stays unresolved and
// Match reports an unknown name (the legacy path answered with an aliasing
// error here; without a single name there is nothing to hand the pipeline).
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}

	number := extractNumber(inCard.Variation)
	var match string
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		if number != "" && number != co.Number {
			continue
		}
		if inCard.Foil && !co.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !co.HasFinish(mtgmatcher.FinishNonfoil) {
			continue
		}
		if match != "" && match != co.Name {
			// Different names survive the filters: genuinely ambiguous.
			return
		}
		match = co.Name
	}
	if match != "" {
		inCard.Name = match
	}
}

func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {}

func (Rules) FilterPrintings(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, editions []string) []string {
	return editions
}

func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) IsSpecificUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

// FilterCards narrows candidates by collector number and finish, mirroring
// the legacy SimpleSearch: candidates come from the name hash rather than the
// edition-keyed cardSet, so case-variant spellings that normalize to the same
// canonical name stay reachable (three real pairs exist in the data) and
// iteration follows stable load order instead of random map order. Lorcana
// identifies a card by name + collector number + finish; the edition is not
// part of the contract, so the cardSet narrowing is deliberately ignored.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var out []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Foil printings are stored under an extra suffixed uuid; fold them
		// back onto the base card so each candidate appears exactly once.
		uuid = strings.TrimSuffix(uuid, suffixFoil)
		if seen[uuid] {
			continue
		}
		seen[uuid] = true

		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		card := co.Card

		if number != "" && number != card.Number {
			continue
		}
		// Drop candidates that cannot satisfy the requested finish, the way
		// the old SimpleSearch filtered uuids by foil status. A card that
		// has both finishes passes either way; output() picks the uuid.
		if inCard.Foil && !card.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !card.HasFinish(mtgmatcher.FinishNonfoil) {
			continue
		}
		out = append(out, card)
	}
	return out
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation. Core Match may append parenthetical chunks split off the input
// name ("205 Enchanted", or just "Enchanted" when no number was supplied),
// so only the first digit-leading field counts. The number is the part
// before '/' with leading zeros stripped — except an all-zero number stays
// "0", so the genuine 0-numbered promo is reachable and a wrong '0' input
// errors instead of silently disabling the filter.
func extractNumber(variation string) string {
	number := ""
	for _, field := range strings.Fields(variation) {
		if field[0] >= '0' && field[0] <= '9' {
			number = field
			break
		}
	}
	number = strings.Split(number, "/")[0]
	trimmed := strings.TrimLeft(number, "0")
	if trimmed == "" && number != "" {
		trimmed = "0"
	}
	return trimmed
}
