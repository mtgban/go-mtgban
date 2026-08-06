package riftbound

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Riftbound. Like Lorcana, a card
// is identified by name + collector number + foil, with the edition breaking
// ties when it resolves; there are no variant tables or promo types, so most
// hooks are no-ops and the real work is the number disambiguation in
// FilterCards (foil is honored downstream by output).
type Rules struct{}

// Prefilter splits a trailing parenthetical variant off the name before the
// canonical-name lookup — unless the full name is itself a known card: a few
// real names carry a parenthetical ("Recruit (DE)"). Dashes stay intact,
// since they occur in real names too ("Dark Child - Starter").
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
}

// AdjustName provides a prefix fallback: scraper feeds sometimes truncate a
// "Champion, Title" name. When the exact name is unknown, scan for cards
// whose name has the input as a prefix and let the collector number and
// finish narrow them; adopt the name only when exactly one survives. If
// several distinct names survive, the input stays unresolved and Match
// reports an unknown name.
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
		if number != "" && !strings.EqualFold(number, co.Number) {
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

// AdjustEdition normalizes scraper edition strings toward the gallery set
// names: storefronts commonly prefix the game name ("Riftbound: Origins")
// or append catalog suffixes ("... Singles"). An edition that still matches
// no set name simply does not narrow the candidates (the Match skeleton
// falls back to every printing), so trimming can only help.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"Riftbound: League of Legends", "Riftbound", "League of Legends"} {
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

// FilterCards narrows candidates by edition, collector number, and finish,
// mirroring the Lorcana rules: candidates come from the name hash (stable
// load order), the cardSet keys carry the sets matching the input edition
// when one was supplied and resolves (falling back to every printing
// otherwise), and the number comparison is case-insensitive because real
// numbers carry letter affixes ("66a", "T5", "SP3").
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var out []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Foil printings are stored under a "_f"-suffixed uuid; fold them
		// back onto the base card so each candidate appears exactly once.
		// Base uuids never contain underscores, so the first underscore
		// marks the start of the finish suffix.
		base := uuid
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 {
			base = uuid[:idx]
		}
		if seen[base] {
			continue
		}
		seen[base] = true

		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		card := co.Card

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !strings.EqualFold(number, card.Number) {
			continue
		}
		// Every card carries both finishes today, so this never drops a
		// candidate; it stays for symmetry with Lorcana should finish data
		// ever appear in the gallery.
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
// name, so only the first field containing a digit counts — numbers may be
// letter-prefixed ("T1", "SP3"), letter-suffixed ("66a"), or starred
// ("227*"). A full public code ("OGN-066a/298") reduces to its number, and
// the result is canonicalized exactly like the loader's numbers so
// zero-padded feeds compare equal.
func extractNumber(variation string) string {
	number := ""
	for _, field := range strings.Fields(variation) {
		if strings.ContainsAny(field, "0123456789") {
			number = field
			break
		}
	}
	if idx := strings.LastIndexByte(number, '-'); idx >= 0 {
		number = number[idx+1:]
	}
	number = strings.Split(number, "/")[0]
	return canonicalNumber(number)
}
