package onepiece

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for One Piece. A card is identified
// by name + collector number, with the variant label separating the
// printings that share a number (alternate arts, parallels, event
// printings). Foil never gates anything: a product is one printing whatever
// its stamping, and both flag values resolve to the same uuid.
type Rules struct{}

// fullNumberRe matches the game's collector number shapes: "OP01-001",
// "ST01-001", "EB01-023", "P-001", with an optional letter tail
// (cardtrader suffixes alternate arts "OP01-001a").
var fullNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*$`)

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: storefronts write "Roronoa Zoro (OP01-001) (V.2)"
// and "Shanks (001) (Parallel)". A full name that is itself canonical stays
// as it is — the epithet parentheticals ("Mr.2.Bon.Kurei (Bentham)") are
// part of the name.
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

// AdjustName provides a prefix fallback for truncated feeds, adopting the
// one name among the prefix matches that carries the input's number. Names
// compare normalized, so punctuation variants of one name are not read as
// ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}

	number := extractNumber(inCard.Variation)
	var match, matchNorm string
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed {
			continue
		}
		if number != "" && !numberMatches(number, co.Number) {
			continue
		}
		norm := mtgmatcher.Normalize(co.Name)
		if match != "" && matchNorm != norm {
			return
		}
		match, matchNorm = co.Name, norm
	}
	if match != "" {
		inCard.Name = match
	}
}

// AdjustEdition trims the game-name prefixes storefronts decorate set names
// with. An edition that still matches no set simply does not narrow the
// candidates.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"One Piece Card Game", "One Piece TCG", "One Piece"} {
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

// FilterCards narrows candidates by edition, collector number and variant.
// The variant tiering mirrors the number sharing in the data: when the
// input's wording describes a variant label, the printings it describes
// win; a bare input keeps the base printing when one exists; cardmarket's
// positional "(V.n)" wording keeps the base for V.1 and the variants
// otherwise.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var candidates []mtgmatcher.Card
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !numberMatches(number, card.Number) {
			continue
		}
		candidates = append(candidates, card)
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// The letter tail cardtrader appends to a number ("OP01-001a") means a
	// variant printing without saying which; the V.n index says the same.
	// Either demand drops the base printing from consideration.
	described, base, variants := tierByVariant(inCard, candidates, number)
	if len(described) > 0 {
		return described
	}
	if wantsVariant(inCard, number) {
		if len(variants) > 0 {
			return variants
		}
		return candidates
	}
	if len(base) > 0 {
		return base
	}
	return candidates
}

// tierByVariant splits the candidates into the ones whose variant label the
// input's wording describes, the base printings, and the variant printings.
func tierByVariant(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card, number string) (described, base, variants []mtgmatcher.Card) {
	wording := strings.ToLower(inCard.Variation + " " + inCard.Edition)
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		variants = append(variants, card)
		if variantDescribed(wording, card.PromoTypes[0], number) {
			described = append(described, card)
		}
	}
	return
}

// variantDescribed reports whether the input's wording mentions every word
// of the printing's variant label, the number and positional tokens aside.
func variantDescribed(wording, variant, number string) bool {
	words := strings.Fields(strings.ToLower(variant))
	if len(words) == 0 {
		return false
	}
	for _, word := range words {
		if !strings.Contains(wording, word) {
			return false
		}
	}
	return true
}

// wantsVariant reports whether the input demands some variant printing
// without describing which: cardmarket's "(V.n)" with n past 1, or a letter
// tail on the collector number ("OP01-001a").
func wantsVariant(inCard *mtgmatcher.InputCard, number string) bool {
	if number != "" && number[len(number)-1] >= 'a' {
		return true
	}
	for _, field := range strings.Fields(strings.ToLower(inCard.Variation)) {
		if strings.HasPrefix(field, "v.") && field != "v.1" {
			return true
		}
	}
	return false
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation: the full "OP01-001" shape when present, else the first
// digit-leading field ("001", "001/121").
func extractNumber(variation string) string {
	fields := strings.Fields(variation)
	for _, field := range fields {
		if fullNumberRe.MatchString(field) {
			return strings.Split(field, "/")[0]
		}
	}
	for _, field := range fields {
		if field[0] >= '0' && field[0] <= '9' {
			return strings.Split(field, "/")[0]
		}
	}
	return ""
}

// numberMatches compares an input number against a printing's full
// collector number: equal full codes, or a bare input matching the code's
// numeric tail. Leading zeros never decide ("1" matches "OP01-001"), and a
// letter tail on the input ("OP01-001a") matches its base number.
func numberMatches(input, full string) bool {
	if strings.EqualFold(input, full) {
		return true
	}
	input = strings.TrimRight(input, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if strings.EqualFold(input, full) {
		return true
	}
	tail := full
	if idx := strings.LastIndexByte(full, '-'); idx >= 0 {
		tail = full[idx+1:]
	}
	if idx := strings.LastIndexByte(input, '-'); idx >= 0 {
		input = input[idx+1:]
	}
	return input != "" && canonicalTail(input) == canonicalTail(tail)
}

// canonicalTail strips leading zeros from a bare number, an all-zero run
// staying "0".
func canonicalTail(number string) string {
	trimmed := strings.TrimLeft(number, "0")
	if trimmed == "" && number != "" {
		return "0"
	}
	return trimmed
}
