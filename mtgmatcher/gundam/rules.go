package gundam

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for the Gundam Card Game. A card is
// identified by name + collector number, and where several printings share
// one number it is the rarity that tells them apart: this game marks a
// parallel run by suffixing the rarity rather than by lettering the number
// or renaming the card. 382 of the game's numbers are shared that way, and
// every one of those groups differs in rarity and in nothing else.
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes: "GD01-001",
// "ST01-011", "EB01-023", "RP-001", "T-024". Every number the datastore
// carries matches it.
var fullNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*$`)

// trailingCodeRe matches the collector number a storefront writes behind a
// name in parentheses of its own ("Gundam (GD01-001)"). It only matches at
// the end, and only a full code: a parenthetical anywhere else may be part
// of the name.
var trailingCodeRe = regexp.MustCompile(`\s*\(([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\)$`)

// dashNumberRe matches the collector number hung inside a name after a dash
// ("Wing Gundam - GD01-001").
var dashNumberRe = regexp.MustCompile(`\s+-\s+([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\b`)

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: storefronts write "Gundam (GD01-001) (SP)" and
// "Char Aznable - GD01-093". A full name that is itself canonical stays as
// it is, since a card of this game may carry a parenthetical of its own.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		// A name that is a card may still be the head of another: "Delta
		// Plus" is GD01-006, and "Delta Plus" beside "Waverider Mode" is
		// GD02-017.
		rejoinName(b, inCard)
		return
	}
	if m := dashNumberRe.FindStringSubmatch(inCard.Name); m != nil {
		inCard.Name = strings.Replace(inCard.Name, m[0], "", 1)
		inCard.Variation = strings.TrimSpace(m[1] + " " + inCard.Variation)
	}
	// A card whose own name ends in a parenthetical wears two of them once
	// the storefront writes the number behind it, and the split below cannot
	// tell which is which. The number is the one parenthetical never part of
	// a name, so peel that alone and ask again, leaving the name untouched
	// unless the answer is yes.
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
	rejoinName(b, inCard)
}

// rejoinName gives a name back the parenthetical that was split off it. The
// datastore names a card the way the game's list does - "Delta Plus
// (Waverider Mode)", "Freedom Gundam (METEOR)", "Elan Ceres (Enhanced
// Person Number 4)" - and a storefront writes the head plain with the rest
// beside it as a qualifier, which the split above produces too. The head is
// a card of this game on its own often enough (GD01-006 is a Delta Plus)
// that the canonical lookup would stop there and read the qualifier as a
// variant no printing of it has; and where the head is no card at all the
// qualifier was still left behind once AdjustName found the spelling.
//
// The collector numbers in the variation are set aside first, since a
// listing writes its number wherever it likes and no name holds one. Then
// the leading words go back onto the name, longest run first, where the two
// spell a canonical name, and what is left stays the variation: "Freedom
// Gundam" and "GD03-076 Meteor Boost Kit 01" is "Freedom Gundam (METEOR)"
// at GD03-076 from the Boost Kit. Where the card puts its parenthetical in
// the middle - "Guncannon (108) & Guncannon (109)", "Zaku (Four Snake
// Eyes') [YETI] (GQ)" - the split hands the words back in another order, so
// the one canonical name holding every word of the head and made only of
// words the listing wrote is taken, and the words it did not use - the
// rarity of a parallel - stay the variation.
//
// Only the canonical names are consulted, never the qualified spellings a
// printing is searchable under: "Gundam" beside "SP" must stay the suit
// and its rarity, which FilterCards reads, and not become a name.
func rejoinName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if inCard.Variation == "" {
		return
	}
	var numbers, words []string
	for _, word := range strings.Fields(inCard.Variation) {
		if fullNumberRe.MatchString(word) {
			numbers = append(numbers, word)
		} else {
			words = append(words, word)
		}
	}
	if len(words) == 0 || headHoldsNumber(b, inCard.Name, numbers) {
		return
	}
	variation := func(left []string) string {
		return strings.Join(append(slices.Clone(numbers), left...), " ")
	}
	for i := len(words); i > 0; i-- {
		key := mtgmatcher.Normalize(inCard.Name + " " + strings.Join(words[:i], " "))
		if canonical, found := b.CanonicalNames[key]; found {
			inCard.Name = canonical
			inCard.Variation = variation(words[i:])
			return
		}
	}
	head := bagOfWords(inCard.Name)
	have := bagOfWords(inCard.Name + " " + strings.Join(words, " "))
	var match string
	var leftover map[string]int
	for _, canonical := range b.CanonicalNames {
		if !strings.ContainsAny(canonical, "([") {
			continue
		}
		bag := bagOfWords(canonical)
		if _, holdsHead := subtract(bag, head); !holdsHead {
			continue
		}
		left, spelled := subtract(have, bag)
		if !spelled {
			continue
		}
		if match != "" && match != canonical {
			return
		}
		match, leftover = canonical, left
	}
	if match == "" {
		return
	}
	var left []string
	for _, word := range words {
		if key := mtgmatcher.Normalize(word); leftover[key] > 0 {
			leftover[key]--
			left = append(left, word)
		}
	}
	inCard.Name = match
	inCard.Variation = variation(left)
}

// headHoldsNumber reports whether the name as written is a card with a
// printing at one of the numbers the listing carries. Then the listing means
// that card and the words beside it are its own qualifier, not a name's:
// "GN Armor" at GD03-057 beside "Type-E" is the card the datastore files
// under that head, wherever it also files a "GN Armor Type-E". A listing
// carrying no number says nothing here.
func headHoldsNumber(b *mtgmatcher.Backend, name string, numbers []string) bool {
	if len(numbers) == 0 {
		return false
	}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(name)] {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		for _, number := range numbers {
			if mtgmatcher.Equals(co.Number, number) {
				return true
			}
		}
	}
	return false
}

// bagOfWords is the words of a name, normalized one by one and sorted, so
// two spellings that place a parenthetical differently compare equal.
func bagOfWords(s string) []string {
	var out []string
	for _, field := range strings.Fields(s) {
		if word := mtgmatcher.Normalize(field); word != "" {
			out = append(out, word)
		}
	}
	sort.Strings(out)
	return out
}

// subtract takes one bag of words out of another, and says whether every
// word was there to take. What is left is counted by word.
func subtract(have, take []string) (map[string]int, bool) {
	left := map[string]int{}
	for _, word := range have {
		left[word]++
	}
	for _, word := range take {
		if left[word] == 0 {
			return nil, false
		}
		left[word]--
	}
	return left, true
}

// AdjustName reaches the printings the catalog files under a qualified
// spelling the storefront wrote plain, and provides a prefix fallback for
// the feeds that truncate. Both only run once the canonical lookup has
// already failed, so a name that resolves on its own is never touched.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	// The qualified spellings are in the name index, so a storefront writing
	// "Gundam SP" reaches "Gundam (SP)" once the qualifier is a variation.
	needle := mtgmatcher.Normalize(inCard.Name + " " + inCard.Variation)
	for _, name := range b.AllNames {
		if mtgmatcher.Normalize(name) == needle {
			inCard.Name = b.CanonicalNames[mtgmatcher.Normalize(name)]
			return
		}
	}
	// A truncated feed keeps the head of the name; accept it when exactly
	// one canonical name starts with what was written. Names compare
	// normalized, so punctuation variants are not read as ambiguity.
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
// the set names the datastore carries. The catalog names a set for the
// product line it belongs to ("Newtype Rising", "Starter Deck 01: Heroic
// Beginnings"), and a storefront usually writes the game in front of it.
// The whole fixup reads only the edition, so it is the fixup entire and
// AdjustEdition delegates to it.
func (Rules) AliasEdition(b *mtgmatcher.Backend, edition string) string {
	edition = strings.TrimSpace(edition)
	// An edition already naming a set verbatim needs no normalization, and
	// must not be trimmed out of matching: the promotional sets are named
	// "Gundam Promotional Cards" and would lose their heading below.
	for _, set := range b.Sets {
		if mtgmatcher.Equals(set.Name, edition) {
			return edition
		}
	}
	for _, prefix := range []string{"Gundam Card Game", "Gundam"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	// Every promo heading collapses onto one, the way the other games do it:
	// the heading spans several promotional sets and agrees with none of
	// their names, so the tiers in FilterCards choose rather than whichever
	// set name happens to contain the heading's words.
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
// says "Gundam Promotional Cards".
func endsInPromo(edition string) bool {
	lower := strings.ToLower(strings.TrimSpace(edition))
	return strings.HasSuffix(lower, "promo") || strings.HasSuffix(lower, "promos") ||
		strings.HasSuffix(lower, "promotional cards")
}

// CanonicalFinish names the one finish this game has of its own. The catalog
// calls a stamped printing "Holofoil", which is this game's standard foil
// and its only one, so it and the vendor spellings of it reach FinishFoil.//
// The vocabulary is open past that. TCGplayer adds a printing to a category
// when it likes and the builder carries it under its own name rather than
// dropping it, so a name neither vocabulary places is normalized and handed
// back - it reaches a uuid only if the printing's own finishes hold one, and
// a list kept here would refuse a real priced printing the day one arrives.
func (Rules) CanonicalFinish(name string) string {
	switch mtgmatcher.NormalizeFinish(name) {
	case "holofoil", "holo", "holographic":
		return mtgmatcher.FinishFoil
	}
	if finish := mtgmatcher.CanonicalFinish(name); finish != "" {
		return finish
	}
	return mtgmatcher.NormalizeFinish(name)
}

// PlainNumber implements mtgmatcher.GameRules. This game writes its collector
// numbers plainly, so a number is its own plain form.
func (Rules) PlainNumber(number string) string {
	return number
}

// FilterCards narrows candidates by edition, collector number and then by
// rarity, which is what tells this game's parallel runs apart. The variant
// label carries the event printings and the alternate forms, and is tiered
// first where the storefront's wording describes one.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		// A dual-printing product files both finish uuids under the name
		// bucket; fold them onto the product they print so each candidate
		// appears once, and let output() pick the finish.
		base := productKeyOf(co.Card.Identifiers, uuid)
		if seen[base] {
			continue
		}
		seen[base] = true

		card := co.Card
		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !strings.EqualFold(number, card.Number) {
			continue
		}
		candidates = append(candidates, card)
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// The variant label is the storefront's own words where it carries one.
	// The labels nest - "World Championship Regionals 26-27 Season 1
	// Finalist" says everything "... Season 1" does - so the longest label
	// the wording describes is the one it means, the way the rarities are
	// read below. Taking every label described would call the two an
	// ambiguity and price neither.
	var described []mtgmatcher.Card
	longest := 0
	for _, card := range candidates {
		if len(card.PromoTypes) > 0 && wordsDescribe(inCard.Variation, card.PromoTypes) &&
			labelLength(card.PromoTypes) > longest {
			longest = labelLength(card.PromoTypes)
		}
	}
	for _, card := range candidates {
		if len(card.PromoTypes) > 0 && labelLength(card.PromoTypes) == longest &&
			wordsDescribe(inCard.Variation, card.PromoTypes) {
			described = append(described, card)
		}
	}
	if len(described) > 0 {
		candidates = described
	} else if plain := unlabelled(candidates); len(plain) > 0 {
		// A wording naming no event means the printing that was handed out
		// at none. The promo set reprints a card once per event it appeared
		// in, all at the main set's number and rarity, and one of those
		// reprints carries no event of its own - that is the one a bare
		// listing names.
		candidates = plain
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// Then the rarity. A wording naming one picks the printings carrying it;
	// a wording naming none keeps the base run, since the plain rarity is
	// what a listing without a qualifier means. Both tiers only narrow -
	// a rarity nothing carries leaves the candidates as they were, so an
	// ambiguity is reported rather than silently resolved.
	if named := rarityNamed(inCard.Variation, candidates); len(named) > 0 {
		candidates = named
	} else if base := baseRarity(candidates); len(base) > 0 {
		candidates = base
	}
	return candidates
}

// unlabelled keeps the printings carrying no variant label, for the wording
// that described none. It narrows or it does nothing: where every candidate
// is labelled, or none is, there is nothing here to choose on.
func unlabelled(cards []mtgmatcher.Card) []mtgmatcher.Card {
	var out []mtgmatcher.Card
	for _, card := range cards {
		if len(card.PromoTypes) == 0 {
			out = append(out, card)
		}
	}
	if len(out) == len(cards) {
		return nil
	}
	return out
}

// rarityNamed keeps the candidates whose rarity the wording spells, by the
// catalog's own name for it. The comparison is word-bounded on the wording
// as written rather than on a normalized form: normalizing drops the "+"
// that marks a parallel, and "C+" reduced to "c" is a substring of very
// nearly every listing there is.
//
// The longest spelling wins, because the rarity names nest - "Rare" is a
// tail of "Legend Rare", and a listing saying the latter says the former
// too.
func rarityNamed(variation string, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if strings.TrimSpace(variation) == "" {
		return nil
	}
	longest := 0
	for _, card := range cards {
		if saysRarity(variation, card.Rarity) && len(card.Rarity) > longest {
			longest = len(card.Rarity)
		}
	}
	if longest == 0 {
		return nil
	}
	var out []mtgmatcher.Card
	for _, card := range cards {
		if len(card.Rarity) == longest && saysRarity(variation, card.Rarity) {
			out = append(out, card)
		}
	}
	// A tier that keeps everything has chosen nothing.
	if len(out) == len(cards) {
		return nil
	}
	return out
}

// rarityPatterns caches one matcher per rarity spelling. The boundaries are
// written out rather than left to \\b, which does not fire after a "+": the
// class excludes "+" on both sides so "C+" cannot match inside "C++".
var rarityPatterns sync.Map

// saysRarity reports whether the wording names this rarity as a word of its
// own. An empty rarity is named by nothing.
func saysRarity(variation, rarity string) bool {
	if rarity == "" {
		return false
	}
	cached, found := rarityPatterns.Load(rarity)
	if !found {
		pattern := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9+])` + regexp.QuoteMeta(rarity) + `(?:$|[^A-Za-z0-9+])`)
		cached, _ = rarityPatterns.LoadOrStore(rarity, pattern)
	}
	return cached.(*regexp.Regexp).MatchString(variation)
}

// baseRarity keeps the printings of the plain run: the rarities that carry
// no parallel marker. It is what a listing saying nothing but the number
// means, the parallels being the printings a storefront names.
func baseRarity(cards []mtgmatcher.Card) []mtgmatcher.Card {
	var out []mtgmatcher.Card
	for _, card := range cards {
		if !strings.HasSuffix(card.Rarity, "+") {
			out = append(out, card)
		}
	}
	if len(out) == len(cards) {
		return nil
	}
	return out
}

// labelLength is how much of a wording a printing's labels account for,
// which is what makes the longest of two nested labels the one meant.
func labelLength(promoTypes []string) int {
	var total int
	for _, promoType := range promoTypes {
		total += len(promoType)
	}
	return total
}

// wordsDescribe reports whether the storefront's wording says every promo
// type the printing carries. The containment runs that way round so a
// wording naming one event cannot answer for a printing of another.
func wordsDescribe(wording string, promoTypes []string) bool {
	said := mtgmatcher.Normalize(wording)
	if said == "" {
		return false
	}
	for _, promoType := range promoTypes {
		label := mtgmatcher.Normalize(promoType)
		if label == "" || !strings.Contains(said, label) {
			return false
		}
	}
	return true
}

// extractNumber reads the collector number out of a storefront's wording,
// taking the game's full code where one is written and nothing otherwise: a
// bare number names no set here, every collector number of this game
// carrying the code of the run it belongs to.
func extractNumber(variation string) string {
	for _, field := range strings.Fields(variation) {
		field = strings.Trim(field, "()[],.")
		if fullNumberRe.MatchString(field) {
			return field
		}
	}
	return ""
}
