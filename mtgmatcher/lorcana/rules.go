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

// Prefilter splits a trailing parenthetical variant off the name before the
// canonical-name lookup. Unlike Magic it leaves " - " intact, since Lorcana
// names are "Character - Title".
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
}

// AdjustName provides a prefix fallback: scraper feeds sometimes truncate the
// "Character - Title" name. When the exact name is unknown, scan for cards
// whose name has the input as a prefix and let the collector number and finish
// narrow them; adopt the name only when exactly one survives. If several
// distinct names survive, the input stays unresolved and Match reports an
// unknown name (without a single name there is nothing to hand the pipeline).
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

// AdjustEdition normalizes scraper edition strings toward LorcanaJSON set
// names: storefronts commonly prefix the game name ("Disney Lorcana: Rise of
// the Floodborn") or append catalog suffixes ("... Singles"). An edition that
// still matches no set name simply does not narrow the candidates (the Match
// skeleton falls back to every printing), so trimming can only help.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"Disney Lorcana", "Lorcana"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	inCard.Edition = edition
}

func (Rules) FilterPrintings(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, editions []string) []string {
	return editions
}

func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) IsSpecificUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) MissingPromoTag(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, co *mtgmatcher.CardObject) bool {
	return false
}

// FilterCards narrows candidates by edition, collector number, and finish:
// candidates come from the name hash rather than the edition-keyed cardSet
// values, so case-variant spellings that normalize to the same canonical name
// stay reachable (three real pairs exist in the data) and iteration follows
// stable load order instead of random map order. The cardSet keys still
// matter: the Match skeleton fills them with the sets matching the input
// edition when one was supplied and resolves — falling back to every printing
// otherwise — so honoring them disambiguates a name+number shared across sets
// ("Let It Go" #163) while a missing or unrecognized edition changes nothing.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var out []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Foil printings (the primary "_f" and every foil sub-type suffix) are
		// stored under a suffixed uuid; fold them back onto the base card so
		// each candidate appears exactly once. Base uuids are numeric, so the
		// first underscore marks the start of any finish suffix.
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 {
			uuid = uuid[:idx]
		}
		if seen[uuid] {
			continue
		}
		seen[uuid] = true

		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		card := co.Card

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && number != card.Number {
			continue
		}
		// Drop candidates that cannot satisfy the requested finish, filtering
		// uuids by foil status. A card that has both finishes passes either
		// way; output() picks the uuid.
		if inCard.Foil && !card.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !card.HasFinish(mtgmatcher.FinishNonfoil) {
			continue
		}
		// A variation naming a foil sub-type re-keys the copy's FoilUUIDs so
		// the flag-driven resolution downstream lands on that sub-type's uuid
		// instead of the primary foil's.
		if finish := selectFinish(inCard, &card); finish != "" {
			foilUUIDs := make(map[string]string, len(card.FoilUUIDs))
			for k, v := range card.FoilUUIDs {
				foilUUIDs[k] = v
			}
			foilUUIDs[mtgmatcher.FinishFoil] = card.FoilUUIDs[finish]
			card.FoilUUIDs = foilUUIDs
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

// selectFinish maps a storefront's foil sub-type wording onto one of the
// card's stored finishes, so sub-typed printings resolve to their own uuid
// instead of folding onto the primary foil. A direct mention of the exported
// sub-type name wins; TCGplayer instead calls every sub-type past the primary
// cold foil "Holofoil", so when the card stores exactly one such sub-type,
// that is the one. Anything else keeps the flag-driven resolution.
func selectFinish(inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	variation := strings.ToLower(strings.ReplaceAll(inCard.Variation, " ", ""))

	var extra string
	extras := 0
	for finish := range card.FoilUUIDs {
		if finish == mtgmatcher.FinishNonfoil || finish == mtgmatcher.FinishFoil {
			continue
		}
		if strings.Contains(variation, finish) {
			return finish
		}
		extra = finish
		extras++
	}
	if extras == 1 && strings.Contains(variation, "holofoil") {
		return extra
	}
	return ""
}
