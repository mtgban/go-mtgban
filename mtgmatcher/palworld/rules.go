package palworld

import (
	"regexp"
	"strings"
	"sync"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Palworld. A card is identified
// by name + collector number alone: this game numbers a parallel printing
// apart from the card it parallels, the rarity's code being the number's
// tail, so no two printings of the game share a number. What the rules have
// to do is read that tail out of a storefront's wording where the number
// was written without it - TCGplayer names the products that way too,
// "Grizzbolt - Rumbling Tank (TSR)" for ETD01-001TSR.
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes, the rarity tail
// included: "EBP01-001", "ETD01-001TSR", "EPR-004". Every number the
// datastore carries matches it.
var fullNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[A-Za-z]*$`)

// numberTailRe splits a collector number into the run's number and the
// rarity code it ends in, the tail empty for a plain printing.
var numberTailRe = regexp.MustCompile(`^([A-Za-z]+[0-9]*-[0-9]+)([A-Za-z]*)$`)

// trailingCodeRe matches the collector number a storefront writes behind a
// name in parentheses of its own ("Grizzbolt - Rumbling Tank (ETD01-001)").
var trailingCodeRe = regexp.MustCompile(`\s*\(([A-Za-z]+[0-9]*-[0-9]+[A-Za-z]*)\)$`)

// rarityTails is every rarity code a number ends in, mapped to the rarity
// the catalog spells it out as. TCGplayer writes the code in parentheses
// after the name as well, so it reaches the matcher from either side.
var rarityTails = map[string]string{
	"SR":  "Super Rare",
	"OSR": "Over Super Rare",
	"SP":  "Super Parallel",
	"SSP": "Super Special Parallel",
	"TSR": "Trial Deck Super Deck Rare",
	"TSP": "Trial Deck Super Parallel",
}

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup. The card names of this game carry a dash of their
// own - every Pal is "Name - Epithet" - so nothing is split on one; only
// the parentheses are read, and only when the whole name is not itself a
// card.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	// The number is the one parenthetical never part of a name, so peel it
	// alone first and leave the name untouched unless what is left is a card.
	if fields := trailingCodeRe.FindStringSubmatch(inCard.Name); fields != nil {
		peeled := strings.TrimSuffix(inCard.Name, fields[0])
		if _, found := b.CanonicalNames[mtgmatcher.Normalize(peeled)]; found {
			inCard.Name = peeled
			inCard.AddToVariant(fields[1])
			return
		}
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
}

// AdjustName provides a prefix fallback for the feeds that truncate a name,
// accepting the head they kept when exactly one canonical name starts with
// it. It only runs once the canonical lookup has failed, so a name that
// resolves on its own is never touched.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	prefix := mtgmatcher.Normalize(inCard.Name)
	if len(prefix) < 4 {
		return
	}
	var match string
	for normalized, canonical := range b.CanonicalNames {
		if !strings.HasPrefix(normalized, prefix) {
			continue
		}
		if match != "" && match != canonical {
			return
		}
		match = canonical
	}
	if match != "" {
		inCard.Name = match
	}
}

// AliasEdition resolves the headings storefronts file this game under onto
// the set names the datastore carries. The catalog names a set by its code
// and title both ("BP01: Dawn of Palpagos"), which a storefront usually
// writes one half of. The whole fixup reads only the edition, so it is the
// fixup entire and AdjustEdition delegates to it.
func (Rules) AliasEdition(b *mtgmatcher.Backend, edition string) string {
	edition = strings.TrimSpace(edition)
	// An edition already naming a set verbatim needs no normalization, and
	// must not be trimmed out of matching: the promo set is named "Palworld
	// Promo Cards" and would lose its heading below.
	for _, set := range b.Sets {
		if mtgmatcher.Equals(set.Name, edition) {
			return edition
		}
	}
	for _, prefix := range []string{"Palworld OFFICIAL CARD GAME", "Palworld Official Card Game", "Palworld"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	// A heading naming a set's code alone reaches the set that carries it,
	// the datastore keying its sets by that code.
	set, found := b.Sets[strings.ToUpper(edition)]
	if found {
		return set.Name
	}
	if edition != "" && (mtgmatcher.IsPromoHeading(edition) || endsInPromo(edition)) {
		edition = "Promos"
	}
	return edition
}

// AdjustEdition normalizes the edition a storefront published toward a set
// name; the fixup reads nothing but the edition, so the alias is all of it.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	inCard.Edition = (Rules{}).AliasEdition(b, inCard.Edition)
}

// endsInPromo reports whether a heading ends in the game's own word for a
// promotional run, which is what a storefront writes where the datastore
// says "Palworld Promo Cards".
func endsInPromo(edition string) bool {
	lower := strings.ToLower(strings.TrimSpace(edition))
	return strings.HasSuffix(lower, "promo") || strings.HasSuffix(lower, "promos") ||
		strings.HasSuffix(lower, "promo cards")
}

// CanonicalFinish adds nothing to the shared vocabulary: this game sells a
// printing plain or foil, and the catalog already spells those the way the
// matcher does.
func (Rules) CanonicalFinish(name string) string {
	return mtgmatcher.CanonicalFinish(name)
}

// plainNumberTail are the treatment codes a number is suffixed with.
const plainNumberTail = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// PlainNumber implements mtgmatcher.GameRules. The letters behind a number
// name the treatment rather than number it - EBP01-001OSR and EBP01-001SSP
// are the Jormuntide Ignis that EBP01-001 is - so the number they carry is
// the plain one. Every one of the 89 stands beside its base number.
func (Rules) PlainNumber(number string) string {
	plain := strings.TrimRight(number, plainNumberTail)
	if plain == "" {
		return number
	}
	return plain
}

// FilterCards narrows candidates by edition and collector number. The
// number's tail is the rarity's code and the whole of what tells a parallel
// from the card it parallels, so the run's number narrows first and the
// tail chooses: written onto the number ("ETD01-001TSR"), written beside it
// ("ETD01-001 TSR"), or spelled out as the rarity ("Trial Deck Super Deck
// Rare"). A wording naming no tail means the plain card.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		base := strings.TrimSuffix(uuid, finishSuffix)
		if seen[base] {
			continue
		}
		seen[base] = true

		card := co.Card
		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		candidates = append(candidates, card)
	}

	if number == "" {
		return candidates
	}

	run, writtenTail := splitNumber(number)
	var sameRun []mtgmatcher.Card
	for _, card := range candidates {
		if cardRun, _ := splitNumber(card.Number); cardRun != "" && strings.EqualFold(run, cardRun) {
			sameRun = append(sameRun, card)
		}
	}
	if len(sameRun) == 0 {
		return nil
	}
	candidates = sameRun
	if len(candidates) == 1 {
		return candidates
	}

	// A tail written onto the number says which printing outright.
	if writtenTail != "" {
		return withTail(candidates, writtenTail)
	}
	// Otherwise the wording may name one beside the number, by the code or
	// by the rarity it stands for.
	if tail := tailSaid(inCard.Variation); tail != "" {
		return withTail(candidates, tail)
	}
	// Saying nothing means the plain card: the parallels are the printings
	// a storefront names.
	return withTail(candidates, "")
}

// withTail keeps the candidates whose number ends in the given rarity code,
// the empty code naming the plain printings. It returns nothing rather than
// everything when no candidate carries the tail, so a code that names no
// printing of this run is a miss rather than a wrong answer.
func withTail(cards []mtgmatcher.Card, tail string) []mtgmatcher.Card {
	var out []mtgmatcher.Card
	for _, card := range cards {
		if _, cardTail := splitNumber(card.Number); strings.EqualFold(cardTail, tail) {
			out = append(out, card)
		}
	}
	return out
}

// tailSaid reads the rarity code a wording names beside the number, by the
// code itself or by the words it stands for, and answers "" when it names
// none. The comparison is word-bounded on the wording as written: "SR" is a
// substring of "TSR" and of any number ending in one, so a normalized
// contains would read every trial-deck listing as a Super Rare.
//
// A wording naming two codes names neither, there being no saying which was
// meant.
func tailSaid(variation string) string {
	if strings.TrimSpace(variation) == "" {
		return ""
	}
	var found string
	for tail, rarity := range rarityTails {
		if saysWord(variation, tail) || saysWord(variation, rarity) {
			if found != "" && found != tail {
				return ""
			}
			found = tail
		}
	}
	return found
}

// wordPatterns caches one matcher per spelling.
var wordPatterns sync.Map

// saysWord reports whether the wording carries the given spelling as a word
// of its own rather than inside a longer one.
func saysWord(variation, word string) bool {
	if word == "" {
		return false
	}
	cached, found := wordPatterns.Load(word)
	if !found {
		pattern := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(word) + `(?:$|[^A-Za-z0-9])`)
		cached, _ = wordPatterns.LoadOrStore(word, pattern)
	}
	return cached.(*regexp.Regexp).MatchString(variation)
}

// splitNumber cuts a collector number into the run's number and the rarity
// code it ends in. A number the game does not shape this way - the one
// printing carrying none - answers with two empty strings.
func splitNumber(number string) (string, string) {
	m := numberTailRe.FindStringSubmatch(number)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

// extractNumber reads the collector number out of a storefront's wording,
// taking the game's full code where one is written and nothing otherwise: a
// bare number names no set here, every collector number carrying the code
// of the run it belongs to.
func extractNumber(variation string) string {
	for _, field := range strings.Fields(variation) {
		field = strings.Trim(field, "()[],.")
		if fullNumberRe.MatchString(field) {
			return field
		}
	}
	return ""
}
