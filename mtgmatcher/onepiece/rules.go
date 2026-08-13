package onepiece

import (
	"regexp"
	"strings"
	"unicode"

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

// dashNumberRe matches the collector number hung inside a name after a
// dash ("Monkey.D.Luffy - P-043").
var dashNumberRe = regexp.MustCompile(`\s+-\s+([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\b`)

// dashTailRe matches the bare number coolstuffinc hangs off a name in
// place of the full code ("Trafalgar Law - 008"). It only matches at the
// end: two cards are named with a dash of their own, and no card name ends
// in one followed by a number.
var dashTailRe = regexp.MustCompile(`\s+-\s+([0-9]+[a-zA-Z]?)$`)

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: storefronts write "Roronoa Zoro (OP01-001) (V.2)",
// "Shanks (001) (Parallel)" and "Monkey.D.Luffy - P-043 (Convention Promo
// 2024)". A full name that is itself canonical stays as it is — the epithet
// parentheticals ("Mr.2.Bon.Kurei (Bentham)") are part of the name.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	split := splitDashNumber(inCard)
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	// The same feed writes its qualifiers behind the number ("Trafalgar Law
	// - 008 (Parallel)"), where the bare tail is no longer at the end of the
	// name to be read; the parentheticals are off by now, so try again, on
	// the same terms - a name that is itself canonical is left alone.
	if split {
		return
	}
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	splitDashNumber(inCard)
}

// splitDashNumber moves the collector number a storefront hangs off a name
// after a dash into the variation, reporting whether it found one. The
// number goes in front of whatever the variation already holds: the retry
// runs with the parenthetical qualifiers moved there, and those carry
// ordinals of their own ("Judge Pack Vol. 5", "1st Anniversary Set") that
// extractNumber's digit-leading fallback would read as the number instead.
func splitDashNumber(inCard *mtgmatcher.InputCard) bool {
	m := dashNumberRe.FindStringSubmatch(inCard.Name)
	if m != nil {
		inCard.Name = strings.Replace(inCard.Name, m[0], "", 1)
		inCard.Variation = strings.TrimSpace(m[1] + " " + inCard.Variation)
		return true
	}
	m = dashTailRe.FindStringSubmatch(inCard.Name)
	if m != nil {
		inCard.Name = strings.TrimSuffix(inCard.Name, m[0])
		inCard.Variation = strings.TrimSpace(m[1] + " " + inCard.Variation)
		return true
	}
	return false
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

// setCodePrefixRe matches a set code worn as an edition prefix: cardtrader
// spells its expansions "OP-01: Romance Dawn", coolstuffinc "OP03 - Pillars
// of Strength", with the compound codes ("OP15-EB04 - ...") in the same
// shape.
var setCodePrefixRe = regexp.MustCompile(`^[A-Za-z]+-?[0-9]+(?:-[A-Za-z]+[0-9]+)?\s*[-:]\s*`)

// AdjustEdition trims the game-name and set-code prefixes storefronts
// decorate set names with. An edition that still matches no set simply does
// not narrow the candidates.
//
// When the rest of the wording asks for a variant printing, the edition
// stops narrowing them: a variant usually shares its collector number with
// the base card while being filed in another set - the alternate arts in
// PRB-01, the event printings in OP-PR - and storefronts name the base
// card's set for all of them. Narrowing on it would delete the printing
// being asked for before FilterCards ever tiers the candidates, pricing a
// promo as the base common. The edition still reaches the tiering as
// wording, so it goes on describing a variant rather than selecting one.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"One Piece Card Game", "One Piece TCG", "One Piece"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	// The code the prefix spells is the storefront saying which set it
	// means, and it survives the trim: it is the only thing telling the
	// members of a set family apart once the shared name is all that is
	// left of the wording.
	code := strings.Trim(setCodePrefixRe.FindString(edition), " -:")
	edition = setCodePrefixRe.ReplaceAllString(edition, "")
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))

	// Storefronts spell a set's name their own way - dropping the ordinal
	// out of "Starter Deck 1: Straw Hat Crew", writing "500 Years into the
	// Future" for the set called "500 Years in the Future" - and a name
	// that is neither equal to nor contained in the datastore's selects
	// nothing at all. Snap it back to the set it describes when one set
	// describes it better than every other, keeping any promo line in
	// front so the event printings are still the ones being named.
	prefix := promoLineRe.FindString(edition)
	canon := canonicalEdition(b, edition[len(prefix):], code)
	if canon != "" {
		edition = prefix + canon
	}
	inCard.Edition = edition

	// PromoWildcard is the flag Match already reads to skip edition
	// selection entirely, both the exact match and the looser one: a
	// storefront's "EB01 - Memorial Collection" does not equal the set's
	// "Extra Booster: Memorial Collection" but is contained in it, and
	// that alone was enough to delete the promo printing.
	if variantPointedAt(b, inCard) {
		inCard.PromoWildcard = true
	}
}

// variantPointedAt reports whether the input's own wording asks for a
// variant printing: a letter tail or "(V.n)" index, or words naming the
// variant label of some printing of this card. Only the variation is read,
// never the edition - the edition is what this decides whether to trust.
func variantPointedAt(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	number := extractNumber(inCard.Variation)
	if wantsVariant(inCard, number) {
		return true
	}
	wording := strings.ToLower(inCard.Variation)
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || len(co.PromoTypes) == 0 {
			continue
		}
		// Only a printing of the number being asked about can say that this
		// input wants a variant. A label another number wears - the card
		// reprinted elsewhere under its own - would otherwise unpin the
		// edition for an input the edition was answering perfectly, and the
		// widened pool then hands back the base printing.
		if number != "" && !numberMatches(number, co.Number) {
			continue
		}
		if variantDescribed(wording, co.PromoTypes[0], number) {
			return true
		}
	}
	return false
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
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		// A dual-printing product files both its finish uuids under the
		// name bucket; fold the foil one back onto the bare id so each
		// candidate appears exactly once, and output() picks the finish
		// afterwards. The base id's own underscores rule out the first-
		// underscore cut riftbound uses, but the foil suffix is fixed.
		base := strings.TrimSuffix(uuid, "_foil")
		if seen[base] {
			continue
		}
		seen[base] = true

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
		return editionTiebreak(b, inCard, described)
	}
	if wantsVariant(inCard, number) {
		if len(variants) > 0 {
			return editionTiebreak(b, inCard, variants)
		}
		return candidates
	}
	if len(base) > 0 {
		return editionTiebreak(b, inCard, base)
	}
	return editionTiebreak(b, inCard, candidates)
}

// editionTiebreak narrows a tier still holding several printings to the
// ones filed in the set the edition describes, with the exact name match
// preferred over the contained-in one, mirroring how Match itself selects
// editions. PromoWildcard trades the edition's pre-filtering for a wider
// pool so a cross-set variant survives to the tiering; this hands the
// edition back its say once the wording has done its work, so the same
// label printed in several sets - the starter-deck and Premium Booster
// Reprints share one number and one label - resolves instead of aliasing.
// An edition matching none of the tier keeps the whole tier: storefronts
// name the base card's set for printings filed elsewhere, and failing safe
// on genuine ambiguity is the point of the tiering.
func editionTiebreak(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if len(cards) <= 1 {
		return cards
	}
	if inCard.Edition == "" {
		return cards
	}

	// A storefront files a set's event printings under that set's promo
	// line: cardmarket's "Promos: Pillars of Strength" is the set the
	// datastore calls "Pillars of Strength Pre-Release Cards". Neither name
	// contains the other, so the marker has to be read as the selection it
	// is or the whole promo line prices as the base commons.
	// A marker with nothing behind it - a bare "Promos:" bucket - names no
	// set, and every set name contains the empty string: reading it as a
	// selection would hand back the event printing of whatever card it
	// decorated. Only a line naming a set selects one.
	base := promoLineRe.ReplaceAllString(inCard.Edition, "")
	if base != "" && base != inCard.Edition {
		var event []mtgmatcher.Card
		for _, card := range cards {
			set, found := b.Sets[card.SetCode]
			if !found || !isEventSet(card.SetCode) {
				continue
			}
			if mtgmatcher.Contains(set.Name, base) {
				event = append(event, card)
			}
		}
		if len(event) > 0 {
			return event
		}
	}

	var equal, contained []mtgmatcher.Card
	for _, card := range cards {
		set, found := b.Sets[card.SetCode]
		if !found {
			continue
		}
		if mtgmatcher.Equals(set.Name, inCard.Edition) {
			equal = append(equal, card)
		} else if mtgmatcher.Contains(set.Name, inCard.Edition) {
			contained = append(contained, card)
		}
	}
	if len(equal) > 0 {
		return equal
	}
	if len(contained) > 0 {
		return contained
	}

	// The event printings a set's cards receive are filed in a set of their
	// own instead of wearing a variant label, so they share a name, a
	// number and an empty label with the card they reprint and nothing but
	// the set name tells them apart. A storefront names an event printing
	// when it stocks one, so an edition matching neither set is still a
	// claim that excludes them - unlike no edition at all, which claims
	// nothing and is left to alias.
	var regular []mtgmatcher.Card
	for _, card := range cards {
		if !isEventSet(card.SetCode) {
			regular = append(regular, card)
		}
	}
	if len(regular) > 0 {
		return regular
	}
	return cards
}

// promoLineRe matches the promo-line prefix a storefront hangs a set name
// off to name that set's event printings.
var promoLineRe = regexp.MustCompile(`(?i)^promos?\s*[:-]\s*`)

// canonicalEdition returns the name of the one set an edition describes
// better than every other, or "" when the wording picks no clear winner.
//
// The wording has to leave nothing of the set name unaccounted for, or
// share three words with it when it does: two words in common is the
// coincidence a vendor bucket like "One Piece Products" lands on, and
// answering that with a set would price a whole shelf as one card.
//
// The set code the storefront wore in front of the name overrides the
// wording: a family shares its name across volumes ("Premium Booster -The
// Best-" and its Vol. 2), so the code is the only thing that says which
// one the listing is filed in.
func canonicalEdition(b *mtgmatcher.Backend, edition, code string) string {
	want := editionTokens(edition)
	if len(want) == 0 {
		return ""
	}
	wantCode := foldSetCode(code)
	var best, runner, coded editionScore
	for setCode, set := range b.Sets {
		// A set already naming itself needs no rewriting - unless the code
		// names another one, which is the storefront saying which member of
		// a family it means where the wording no longer can: PRB-02's name
		// begins with the whole of PRB-01's, so a truncating storefront
		// spells one name and the other code.
		if mtgmatcher.Equals(set.Name, edition) && (wantCode == "" || foldSetCode(setCode) == wantCode) {
			return ""
		}
		cur := scoreEdition(want, set.Name)
		cur.event = isEventSet(setCode)
		if cur.shared < 2 || cur.missing > 1 || (cur.missing > 0 && cur.shared < 3) {
			continue
		}
		if wantCode != "" && foldSetCode(setCode) == wantCode {
			coded = cur
		}
		if best.name == "" || cur.beats(best) {
			runner, best = best, cur
		} else if runner.name == "" || cur.beats(runner) {
			runner = cur
		}
	}
	if coded.name != "" {
		return coded.name
	}
	if best.name == "" || !best.beats(runner) {
		return ""
	}
	return best.name
}

// foldSetCode drops the punctuation a storefront spells a set code with:
// cardtrader writes "OP-01" for the datastore's "OP01", coolstuffinc
// "EB01" for its "EB-01". The event sets keep their marker ("OP03 PRE"
// folds to OP03PRE), which no storefront prefix spells.
func foldSetCode(code string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, code)
}

// editionScore counts how much of a set name an edition accounts for: the
// words the two share and the ones the set name carries that the edition
// left out, with the event sets flagged for the tie those two cannot
// break. Nothing else orders two sets - the words a set name spells beyond
// the wording say nothing about which one was meant, and answering a
// family's shared name with its shortest member invents a volume the
// storefront never wrote.
type editionScore struct {
	name            string
	shared, missing int
	event           bool
}

func scoreEdition(want map[string]bool, name string) editionScore {
	have := editionTokens(name)
	cur := editionScore{name: name}
	for token := range want {
		if have[token] {
			cur.shared++
		} else {
			cur.missing++
		}
	}
	return cur
}

func (s editionScore) beats(other editionScore) bool {
	if s.shared != other.shared {
		return s.shared > other.shared
	}
	if s.missing != other.missing {
		return s.missing < other.missing
	}
	// An event set wears its base set's whole name with a marker appended,
	// so a wording spelling the base name alone accounts for both exactly
	// as well and they would tie forever. A wording that spells the marker
	// too wins the event set the counts above, so a tie left here is the
	// marker gone unspelled, and the base set is the one being named.
	return !s.event && other.event
}

// editionTokens splits a set name into the words that carry its identity.
// Normalize is no help here: it drops the spaces the words are counted by.
func editionTokens(edition string) map[string]bool {
	out := map[string]bool{}
	fields := strings.FieldsFunc(strings.ToLower(edition), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, field := range fields {
		out[field] = true
	}
	return out
}

// isEventSet reports whether a set code files the event printings of
// another set's cards. The datastore spells them as the base code with a
// marker appended - "OP03 PRE" for the pre-release cards, "OP10 RE" for the
// release event ones, "OP05 ANN" for the anniversary tournament ones - and
// those are the only set codes carrying a space.
func isEventSet(code string) bool {
	fields := strings.Fields(code)
	if len(fields) < 2 {
		return false
	}
	switch fields[len(fields)-1] {
	case "PRE", "RE", "ANN":
		return true
	}
	return false
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
	inSet, inTail := splitNumber(input)
	fullSet, fullTail := splitNumber(full)
	// An input spelling its own set code has already named which printing's
	// number it is, and only that code may answer it: dropping to the tail
	// alone would let "OP07-002" take every card numbered -002 in the game,
	// aliasing a set's card against its every same-numbered reprint. A bare
	// input carries no code to disagree with, so it still matches on the
	// tail: that is the whole identification cardmarket's number field has.
	if inSet != "" && !strings.EqualFold(inSet, fullSet) {
		return false
	}
	return inTail != "" && canonicalTail(inTail) == canonicalTail(fullTail)
}

// splitNumber cuts a collector number into its set code and numeric tail,
// a number without a code being all tail.
func splitNumber(number string) (set, tail string) {
	idx := strings.LastIndexByte(number, '-')
	if idx >= 0 {
		return number[:idx], number[idx+1:]
	}
	return "", number
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
