package mtgmatcher

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Sealed-name resolution: storefronts name sealed products in their own
// vocabulary, and only Magic gets marketplace ids from its datastore, so
// the other games resolve a vendor's product name against the sealed
// namespace. Tuned against the real Cardmarket and StarCityGames catalogs
// for Riftbound and Lorcana; every rule below exists because its absence
// mismatched a real product, and the discipline throughout is unique or
// nothing: a wrong sealed match silently reroutes a product's whole price
// history, while a dropped one only loses coverage.

var sealedTokenRe = regexp.MustCompile(`[a-z0-9]+`)
var sealedCountRe = regexp.MustCompile(`^\d+x?$`)

// sealedMultiplierRe matches the form a storefront writes a bare multiplier
// in. The trailing "x" is the whole of what it says.
var sealedMultiplierRe = regexp.MustCompile(`^\d+x$`)

var sealedNumberRe = regexp.MustCompile(`^\d+$`)

// sealedParenRe matches the parentheticals a storefront appends to a name.
var sealedParenRe = regexp.MustCompile(`\(([^)]*)\)`)

// sealedContainerWords name the thing a lot is a lot of, and so close the
// parenthetical a storefront says a quantity in.
var sealedContainerWords = map[string]bool{
	"booster": true, "boosters": true, "box": true, "boxes": true,
	"pack": true, "packs": true, "deck": true, "decks": true,
	"tin": true, "tins": true, "case": true, "cases": true,
	"card": true, "cards": true, "display": true, "displays": true,
	"tuckbox": true, "tuckboxes": true,
}

// sealedQuantityTokens returns the numbers a storefront's name says as a count
// rather than as part of what the product is called. A bare number in the body
// of a name is usually the product's own - "Hidden Arsenal 5" and "Duelist
// Pack: Yusei 2" are sequels, not five and two of anything - so a number is
// read as a count only where the name says so: with the multiplier's own "x",
// or opening a parenthetical that a container closes, "(18 Booster)" and "(8
// Structure Decks)".
//
// A count says how many of a thing the named product holds, so the rest of
// the name has to name the product: "Millennium Pack Booster Box (18 Booster)"
// is a booster box and holds eighteen boosters, and "Everfest Case (4 Booster
// Boxes)" is a case and holds four boxes. Where the rest of the name says no
// product at all - "Wild Survivors (12 Booster Boxes)" says nothing but the
// set - the parenthetical is not the contents, it is the product: twelve
// booster boxes are a case, and reading the twelve as a count lands the case's
// price on a single booster box.
func sealedQuantityTokens(name string) map[string]bool {
	out := map[string]bool{}
	lower := asciiReplacer.Replace(strings.ToLower(name))
	for _, tok := range sealedTokenRe.FindAllString(lower, -1) {
		if sealedMultiplierRe.MatchString(tok) {
			out[tok] = true
		}
	}
	var named bool
	for _, tok := range sealedTokenRe.FindAllString(sealedParenRe.ReplaceAllString(lower, " "), -1) {
		if sealedContainerWords[tok] {
			named = true
			break
		}
	}
	for _, match := range sealedParenRe.FindAllStringSubmatch(lower, -1) {
		toks := sealedTokenRe.FindAllString(match[1], -1)
		if len(toks) < 2 || !sealedNumberRe.MatchString(toks[0]) {
			continue
		}
		if named && sealedContainerWords[toks[len(toks)-1]] {
			out[toks[0]] = true
		}
	}
	return out
}

// sealedFiller are the words that carry no product identity: articles and
// the game's own name, which storefronts prepend freely.
var sealedFiller = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "and": true,
	"disney": true, "lorcana": true, "riftbound": true, "league": true,
	"legends": true, "tcg": true, "trading": true, "game": true,
	"card": true, "cards": true, "one": true, "piece": true,
	"bandai": true, "pokemon": true,
}

// sealedFold folds the marketplace vocabularies together - TCGplayer's
// "Booster Display" is everyone else's "Booster Box", a bare "Booster" is a
// pack - and the plurals onto their singulars.
func sealedFold(tok string) string {
	switch tok {
	case "box", "boxes":
		return "display"
	case "booster", "boosters", "packs":
		return "pack"
	case "decks":
		return "deck"
	case "blisters":
		return "blister"
	case "versus":
		return "vs"
	case "volume":
		return "vol"
	}
	return tok
}

// sealedTokens reduces a product name to its canonical identity tokens:
// lowercased, deduplicated, sorted, with the filler dropped and the
// marketplace vocabularies folded together - TCGplayer's "Booster Display"
// is everyone else's "Booster Box", and a bare "Booster" is a pack.
func sealedTokens(name string) []string {
	set := map[string]bool{}
	// The token pattern is plain ASCII, so an accented letter reads as a
	// separator rather than a letter: "Pokémon" splits into "pok" and "mon"
	// and matches nothing the catalog spells without the accent. Card names
	// are folded the same way before they are looked up. The fold knows only
	// the lowercase letters, so it has to come after the lowercasing.
	name = asciiReplacer.Replace(strings.ToLower(name))
	for _, tok := range sealedTokenRe.FindAllString(name, -1) {
		if sealedFiller[tok] {
			continue
		}
		set[sealedFold(tok)] = true
	}
	out := make([]string, 0, len(set))
	for tok := range set {
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

// sealedOuterWords are the words a storefront names the box a set's packs come
// in, on either side of the vocabulary the fold above runs together.
var sealedOuterWords = map[string]string{
	"box": "box", "boxes": "box",
	"display": "display", "displays": "display",
}

// sealedOuterSaid reports which of those words a name says, before the fold
// runs them together.
func sealedOuterSaid(name string) (box, display bool) {
	for _, tok := range sealedTokenRe.FindAllString(strings.ToLower(name), -1) {
		switch sealedOuterWords[tok] {
		case "box":
			box = true
		case "display":
			display = true
		}
	}
	return box, display
}

// sealedOuterAgreement picks the one candidate among equally exact matches
// that says the same outer container the vendor said. A vendor naming
// neither, or both, has not chosen between them and gets no answer here.
func sealedOuterAgreement(name string, entries []*sealedEntry) string {
	box, display := sealedOuterSaid(name)
	if box == display {
		return ""
	}
	var chosen string
	for _, entry := range entries {
		if entry.boxOnly != box || entry.dispOnly != display {
			continue
		}
		if chosen != "" {
			return ""
		}
		chosen = entry.uuid
	}
	return chosen
}

// sealedNamesDisplayOnly reports whether a name says Display and never Box.
func sealedNamesDisplayOnly(name string) bool {
	box, display := sealedOuterSaid(name)
	return display && !box
}

// sealedNamesBoxOnly reports whether a name says Box and never Display.
func sealedNamesBoxOnly(name string) bool {
	box, display := sealedOuterSaid(name)
	return box && !display
}

// sealedTokenCounts is sealedTokens without the deduplication: how many times
// a name says each of its words.
func sealedTokenCounts(name string) map[string]int {
	out := map[string]int{}
	for _, tok := range sealedTokenRe.FindAllString(strings.ToLower(asciiReplacer.Replace(name)), -1) {
		if sealedFiller[tok] {
			continue
		}
		out[sealedFold(tok)]++
	}
	return out
}

// sealedSaysTwice reports whether the vendor says a word of the candidate's
// set name more times than the candidate's name and the set name can account
// for between them.
//
// The set name is free against every product on its shelf, because storefronts
// file products under the set they belong to - but they write it once, ahead of
// the product's own name. A set that names two sides writes both of them onto
// the shelf, and a storefront filing the deck as `EX Team Magma vs Team Aqua:
// Team Aqua Theme Deck` said Aqua twice: once for the shelf, once for the deck.
// The second one is the deck's own name and the Team Magma deck is not it -
// which is the whole of what stops a Magma deck's uuid taking an Aqua deck's
// price, the datastore holding no Aqua row to reach instead.
//
// Only the set's own words are counted this way. A storefront repeats plenty of
// others for reasons of its own - `Hoenn Collection: Primal Groudon Collection`
// is one collection - and the shelf is the only word whose second saying means
// the storefront moved on to naming the product.
func sealedSaysTwice(vendorCounts, account map[string]int, setWords map[string]bool) bool {
	for tok, said := range vendorCounts {
		if setWords[tok] && said > 1 && said > account[tok] {
			return true
		}
	}
	return false
}

// sealedExtrasSafe reports whether every vendor token missing from the
// candidate is harmless: a count, a set-name word (storefronts file
// products under the set they belong to), or noise. Anything else means
// the vendor is naming a different product - "Prerelease Pack" and
// "Participation Booster" must not resolve to the plain Booster Pack.
//
// The candidate's own set name is checked before anything else, because a
// word the shelf itself is called cannot be the word telling its products
// apart - every product on it may be spelled with the set's name. What the
// rest of the game's set names free is checked after: a word that names
// something more specific on this shelf is not harmless there, however
// freely another set's name spells it.
func sealedExtrasSafe(vendor []string, candSet, setTokens, ownSetTokens, narrower map[string]bool) bool {
	for _, tok := range vendor {
		if candSet[tok] || ownSetTokens[tok] || sealedCountRe.MatchString(tok) {
			continue
		}
		if narrower[tok] {
			return false
		}
		if setTokens[tok] {
			continue
		}
		switch tok {
		case "s", "en", "english":
			continue
		}
		return false
	}
	return true
}

func tokensEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// bracketRe matches the bracketed groups a catalog name decorates a product
// with.
var bracketRe = regexp.MustCompile(`\[([^\]]*)\]`)

// sealedExclusiveRe matches the parenthetical a catalog marks a storefront's
// own edition of a product with. It says who sold the product rather than
// what it is, and only when it says nothing else: "(Retail Exclusive)" and
// "(EU Exclusive)" name which storefront, and two of those are two products.
var sealedExclusiveRe = regexp.MustCompile(`(?i)\(\s*exclusive\s*\)`)

// sealedPrintRunWords are the words a bracket uses to name which run of a
// product it is. They are the whole of what such a bracket says: strip them
// and two runs of one product read as one product.
var sealedPrintRunWords = map[string]bool{
	"1st": true, "unlimited": true, "limited": true, "edition": true,
}

// sealedPrintRunBracket reports whether a bracket says nothing but which print
// run this is. A bracket that names a run alongside anything else - a region,
// a year - is not one of these: the rest of it is identity we cannot forgive
// without knowing what it means.
func sealedPrintRunBracket(inner string) bool {
	toks := sealedTokens(inner)
	if len(toks) == 0 {
		return false
	}
	for _, tok := range toks {
		if !sealedPrintRunWords[tok] {
			return false
		}
	}
	return true
}

// sealedQualifierTokens returns the words a catalog name carries only inside
// brackets, or in the one parenthetical that behaves like one, and that a
// storefront may leave unsaid. Those are of three kinds.
//
// The first is decoration: what a product is pictured with rather than what it
// is. The deck the catalog files as `Theme Deck - "Storm Rider" [Zapdos]` is
// the Storm Rider deck whatever Zapdos is doing on the box, and a storefront
// naming it says nothing about Zapdos.
//
// The second is the print run. Two runs of one product are two products and
// the bracket is the only thing telling them apart, so a vendor that names
// neither run must reach neither - but that is the ambiguity rule's job, and
// keeping the run unforgivable here instead means a vendor cannot reach the
// run even when the datastore carries exactly one. Forgiven, `Burst Protocol
// Booster Box` reaches the single `[1st Edition]` row and still stalls on
// `Pharaonic Guardian`, where both runs exist and neither is more likely.
//
// The third is which storefront sold it. A catalog marks its own edition of a
// product `(Exclusive)`, and a storefront that only ever sold its own does not
// repeat it: the Pokemon Center's Elite Trainer Boxes are named for their set
// everywhere but the catalog. Only the bare word counts - `(Retail
// Exclusive)` and `(EU Exclusive)` say which storefront, and two of those are
// two products.
//
// A word the name also carries outside these groups is left in, since there it
// is doing the product's own work.
//
// The words come back one bracket at a time, because the ranking counts how
// many brackets a storefront left unsaid rather than how many words: two runs
// of a product are equally unsaid whether the catalog spells one `[Unlimited]`
// and the other `[1st Edition]`, and counting words would hand the tie to
// whichever the catalog spelled shorter.
func (b *Backend) sealedQualifierGroups(name string) []map[string]bool {
	var inners []string
	for _, match := range bracketRe.FindAllStringSubmatch(name, -1) {
		// Everything else a bracket holds is the product's identity - the
		// number of copies, the placing a promo was handed out for - and
		// forgiving those would merge products that differ by nothing else.
		_, isCard := b.CanonicalNames[Normalize(match[1])]
		if !isCard && !sealedPrintRunBracket(match[1]) {
			continue
		}
		inners = append(inners, match[1])
	}
	for range sealedExclusiveRe.FindAllString(name, -1) {
		inners = append(inners, "exclusive")
	}
	if len(inners) == 0 {
		return nil
	}

	outside := map[string]bool{}
	bare := sealedExclusiveRe.ReplaceAllString(bracketRe.ReplaceAllString(name, " "), " ")
	for _, tok := range sealedTokens(bare) {
		outside[tok] = true
	}

	var groups []map[string]bool
	for _, inner := range inners {
		inside := map[string]bool{}
		for _, tok := range sealedTokens(inner) {
			if outside[tok] {
				continue
			}
			inside[tok] = true
		}
		if len(inside) > 0 {
			groups = append(groups, inside)
		}
	}
	return groups
}

// sealedListedRe splits what a bracket lists into the things it lists.
var sealedListedRe = regexp.MustCompile(`\s*(?:,|&|/)\s*`)

// sealedQualifierContradicts reports whether the vendor named part of what a
// bracket lists and left the rest of the list out.
//
// Forgiving a bracket is forgiving the whole of it: a storefront that names
// the deck without the Zapdos on its box said nothing about Zapdos, and that
// silence is what makes the bracket decoration. A storefront that names one
// of three Pokemon a blister pictures did not go silent - it said which
// product it means, and it is not this one. Reading that as decoration is how
// `2-Pack Blister: Raikou` lands on the blister picturing Raikou, Entei and
// Suicune while the Raikou blisters sit unreached beside it.
//
// A list item counts as named on one word, because a storefront routinely
// drops the card type a catalog spells: the Elite Trainer Box a storefront
// files under `Iron Leaves` is the one the catalog brackets `[Iron Leaves
// ex]`.
func sealedQualifierContradicts(listed [][][]string, said map[string]bool) bool {
	for _, items := range listed {
		var named, silent int
		for _, toks := range items {
			var spoken bool
			for _, tok := range toks {
				if said[tok] {
					spoken = true
					break
				}
			}
			if spoken {
				named++
			} else {
				silent++
			}
		}
		if named > 0 && silent > 0 {
			return true
		}
	}
	return false
}

// sealedQualifierTokens is sealedQualifierGroups with the brackets run
// together, for the containment test that only asks whether a word is free.
func sealedQualifierTokens(groups []map[string]bool) map[string]bool {
	if len(groups) == 0 {
		return nil
	}
	inside := map[string]bool{}
	for _, group := range groups {
		for tok := range group {
			inside[tok] = true
		}
	}
	return inside
}

// sealedUnsaidGroups counts the brackets the vendor named nothing out of.
func sealedUnsaidGroups(groups []map[string]bool, vendor []string) int {
	said := map[string]bool{}
	for _, tok := range vendor {
		said[tok] = true
	}
	var n int
	for _, group := range groups {
		for tok := range group {
			if !said[tok] {
				n++
				break
			}
		}
	}
	return n
}

// tokensSubsetModulo reports whether every word of sub is one that super says
// or one of the free words beside it, and returns how many of sub's words
// super actually said. The floor matters: a candidate forgiven down to nothing
// would answer for every name in the game.
func tokensSubsetModulo(sub, super []string, free map[string]bool) (bool, int) {
	supSet := map[string]bool{}
	for _, tok := range super {
		supSet[tok] = true
	}
	var matched int
	for _, tok := range sub {
		switch {
		case supSet[tok]:
			matched++
		case free[tok]:
		default:
			return false, 0
		}
	}
	return matched > 0, matched
}

// ResolveSealed returns the uuid of the single sealed product the given
// storefront name describes. An exact token match wins; failing that, a
// candidate fully contained in the vendor's wording wins when its extras
// are safe and it is the most specific such candidate. No match, or more
// than one equally good, is ErrCardDoesNotExist: unique or nothing.
func (b *Backend) ResolveSealed(name string) (string, error) {
	return b.resolveSealed(name, "")
}

// ResolveSealedWithHint is ResolveSealed with a second phrase the storefront
// files the product under - CardTrader's expansion name - to settle a tie the
// product name alone cannot. The hint is never read as more product words:
// appending a storefront's context wholesale loses matches, because the words
// it adds are extras the candidate cannot explain. It only chooses among the
// candidates the name already reached, so a hint can turn a refusal into an
// answer and can never turn one answer into another.
func (b *Backend) ResolveSealedWithHint(name, hint string) (string, error) {
	return b.resolveSealed(name, hint)
}

// sealedEntry is everything the resolver reads off one sealed product that
// does not depend on what a storefront called it.
type sealedEntry struct {
	uuid     string
	tokens   []string
	said     map[string]bool
	account  map[string]int
	groups   []map[string]bool
	free     map[string]bool
	listed   [][][]string
	setWords map[string]bool
	narrower map[string]bool
	boxOnly  bool
	dispOnly bool
}

// sealedIndex is the sealed namespace read the way the resolver reads it. It
// depends on nothing but the datastore, and building it per lookup made every
// resolution walk every product of every set again - a scraper asks for
// thousands of names against tens of thousands of products.
type sealedIndex struct {
	pooled  map[string]bool
	entries []*sealedEntry
}

// buildSealedIndex reads the sealed namespace once.
func (b *Backend) buildSealedIndex() *sealedIndex {
	idx := &sealedIndex{pooled: map[string]bool{}}
	setWords := map[string]map[string]bool{}
	setCounts := map[string]map[string]int{}
	for code, set := range b.Sets {
		own := map[string]bool{}
		for _, tok := range sealedTokens(set.Name) {
			idx.pooled[tok] = true
			own[tok] = true
		}
		setWords[code] = own
		setCounts[code] = sealedTokenCounts(set.Name)
	}

	shelves := map[string][]*sealedEntry{}
	for _, uuid := range b.AllSealedUUIDs {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		entry := &sealedEntry{
			uuid:     uuid,
			tokens:   sealedTokens(co.Name),
			said:     map[string]bool{},
			account:  sealedTokenCounts(co.Name),
			groups:   b.sealedQualifierGroups(co.Name),
			setWords: setWords[co.SetCode],
			boxOnly:  sealedNamesBoxOnly(co.Name),
			dispOnly: sealedNamesDisplayOnly(co.Name),
		}
		for _, tok := range entry.tokens {
			entry.said[tok] = true
		}
		for tok, n := range setCounts[co.SetCode] {
			entry.account[tok] += n
		}
		entry.free = sealedQualifierTokens(entry.groups)
		for _, match := range bracketRe.FindAllStringSubmatch(co.Name, -1) {
			if _, isCard := b.CanonicalNames[Normalize(match[1])]; !isCard {
				continue
			}
			var items [][]string
			for _, item := range sealedListedRe.Split(match[1], -1) {
				if toks := sealedTokens(item); len(toks) > 0 {
					items = append(items, toks)
				}
			}
			if len(items) > 1 {
				entry.listed = append(entry.listed, items)
			}
		}
		if entry.setWords == nil {
			entry.setWords = map[string]bool{}
		}
		idx.entries = append(idx.entries, entry)
		shelves[co.SetCode] = append(shelves[co.SetCode], entry)
	}

	for _, shelf := range shelves {
		for _, entry := range shelf {
			entry.narrower = sealedNarrowerWords(shelf, entry)
		}
	}
	return idx
}

// sealedLookup returns the index SortSealed built, or builds one for a
// datastore that never filed a sealed product through it.
func (b *Backend) sealedLookup() *sealedIndex {
	if b.sealedIdx != nil {
		return b.sealedIdx
	}
	return b.buildSealedIndex()
}

func (b *Backend) resolveSealed(name, hint string) (string, error) {
	if b.UUIDs == nil {
		return "", ErrDatastoreEmpty
	}
	idx := b.sealedLookup()

	counts := sealedQuantityTokens(name)
	vendor := sealedTokens(name)
	vendorCounts := sealedTokenCounts(name)
	displayOnly := sealedNamesDisplayOnly(name)
	said := map[string]bool{}
	for _, tok := range vendor {
		said[tok] = true
	}
	var exact, contained []string
	var exactEntries []*sealedEntry
	shared := map[string]int{}
	unsaid := map[string]int{}
	unexplained := map[string]int{}
	forgiven := map[string]bool{}
	qualifiers := map[string]map[string]bool{}
	for _, entry := range idx.entries {
		if displayOnly && entry.boxOnly {
			continue
		}
		if tokensEqual(entry.tokens, vendor) {
			exact = append(exact, entry.uuid)
			exactEntries = append(exactEntries, entry)
			continue
		}
		if sealedQualifierContradicts(entry.listed, said) {
			continue
		}
		if sealedSaysTwice(vendorCounts, entry.account, entry.setWords) {
			continue
		}
		subset, matched := tokensSubsetModulo(entry.tokens, vendor, entry.free)
		if subset && sealedExtrasSafe(vendor, entry.said, idx.pooled, entry.setWords, entry.narrower) {
			uuid := entry.uuid
			contained = append(contained, uuid)
			// Specificity counts the words the vendor and the candidate
			// actually share. A forgiven word is one the vendor never said,
			// so letting it lengthen the candidate would rank a product by
			// how much of it went unsaid.
			shared[uuid] = matched
			unsaid[uuid] = sealedUnsaidGroups(entry.groups, vendor)
			unexplained[uuid] = unexplainedTokens(vendor, entry.said, entry.setWords, counts)
			forgiven[uuid] = matched < len(entry.tokens)
			qualifiers[uuid] = entry.free
		}
	}

	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		// The fold lets a vendor's Box reach a catalog's Display, so a
		// catalog holding both under one set answers twice to the same
		// name. The candidate spelling the outer container the way the
		// vendor spelled it is the one being sold; without that the pair
		// cancel out and a product the catalog names word for word goes
		// unpriced.
		if agreed := sealedOuterAgreement(name, exactEntries); agreed != "" {
			return agreed, nil
		}
		return "", ErrCardDoesNotExist
	}
	if len(contained) > 0 {
		// The candidate that accounts for most of what the vendor said
		// wins: first by how many vendor words neither it nor its own set
		// explains, then by how many words it shares with the vendor, then
		// by how few brackets the vendor left unsaid. Counting shared words
		// alone reads a product's own identity as filler when its words
		// happen to be a rearrangement of another's - "Paramount War Box
		// Promotion Booster" names the Box Promotion Pack, but has every
		// word of the Paramount War Booster Box too, and that one is
		// longer. A tie on all three stays unresolved.
		sort.Slice(contained, func(i, j int) bool {
			left, right := contained[i], contained[j]
			if unexplained[left] != unexplained[right] {
				return unexplained[left] < unexplained[right]
			}
			if shared[left] != shared[right] {
				return shared[left] > shared[right]
			}
			if unsaid[left] != unsaid[right] {
				return unsaid[left] < unsaid[right]
			}
			return left < right
		})
		var answer string
		if len(contained) == 1 ||
			unexplained[contained[0]] < unexplained[contained[1]] ||
			shared[contained[0]] > shared[contained[1]] ||
			unsaid[contained[0]] < unsaid[contained[1]] {
			answer = contained[0]
		} else {
			answer, _ = sealedHintBreaksTie(hint, contained, unexplained, shared, unsaid, qualifiers)
		}
		// Forgiveness runs one way. A candidate the vendor did not say in
		// full is reachable only because the catalog's own brackets were
		// forgiven, and letting it leave a vendor word unaccounted for too
		// forgives both sides at once - which is how "Hidden Arsenal 5
		// Booster Box" reaches the plain "Hidden Arsenal - Booster Box
		// [Unlimited Edition]" while the datastore carries Hidden Arsenal 5
		// under a name of its own. A candidate the vendor did say in full is
		// held to nothing extra, which is what keeps "The First Chapter 4
		// Booster Box Case" on the case it names.
		if answer != "" && (!forgiven[answer] || unexplained[answer] == 0) {
			return answer, nil
		}
	}
	return "", ErrCardDoesNotExist
}

// sealedHintBreaksTie picks the one candidate among those the sort left
// indistinguishable whose forgiven words the storefront's own filing says out
// loud - CardTrader shelves `Crucible of War Booster Box` under `Crucible of
// War - Unlimited`, which names the run the product name omits. A hint that
// speaks for none of them, or for more than one, leaves the tie standing.
func sealedHintBreaksTie(hint string, contained []string, unexplained, shared, unsaid map[string]int, qualifiers map[string]map[string]bool) (string, bool) {
	if hint == "" {
		return "", false
	}
	hintTokens := map[string]bool{}
	for _, tok := range sealedTokens(hint) {
		hintTokens[sealedPrintRunSynonym(tok)] = true
	}

	var tied []string
	for _, uuid := range contained {
		if unexplained[uuid] != unexplained[contained[0]] ||
			shared[uuid] != shared[contained[0]] ||
			unsaid[uuid] != unsaid[contained[0]] {
			continue
		}
		tied = append(tied, uuid)
	}

	// Only a word one of the tied candidates alone carries can speak for it.
	// A word they share tells them apart in neither direction: `[1st Edition]`
	// and `[Unlimited Edition]` are both spoken for by a shelf that says
	// `Edition`, which is how the datastore's own spelling of a print run
	// loses a tie-break that `- First` and `- Unlimited` win.
	carriers := map[string]int{}
	for _, uuid := range tied {
		for tok := range qualifiers[uuid] {
			carriers[tok]++
		}
	}

	var winner string
	var winners int
	for _, uuid := range tied {
		var spoken bool
		for tok := range qualifiers[uuid] {
			if hintTokens[tok] && carriers[tok] == 1 {
				spoken = true
				break
			}
		}
		if !spoken {
			continue
		}
		winner = uuid
		winners++
	}
	if winners != 1 {
		return "", false
	}
	return winner, true
}

// sealedPrintRunSynonym folds the spellings a storefront uses for a print run
// onto the one the datastore brackets: CardTrader's `Tales of Aria - First` is
// the datastore's `[1st Edition]`.
func sealedPrintRunSynonym(tok string) string {
	if tok == "first" {
		return "1st"
	}
	return tok
}

// sealedNarrowerWords returns the words that would name a more specific
// product than the given candidate on its own shelf: for every other sealed
// product of the same set that says everything this candidate says, the words
// it says on top.
//
// A word freed by the set vocabulary is free against every candidate in the
// game, and a game's set names pooled cover most of the language a storefront
// writes - some Pokemon set is called a Bundle, some a Blister, some names an
// Aqua. Freed that way, "Surging Sparks Booster Bundle" reads as the Booster
// Pack with a harmless extra word. But the set holds a Booster Bundle too, and
// a vendor word that picks it out of the shelf is not decoration on the
// product beside it: it says the vendor meant the other one, whether or not
// that one is reachable.
func sealedNarrowerWords(shelf []*sealedEntry, candidate *sealedEntry) map[string]bool {
	out := map[string]bool{}
	for _, sibling := range shelf {
		if len(sibling.tokens) <= len(candidate.tokens) {
			continue
		}
		covers := true
		for tok := range candidate.said {
			if !sibling.said[tok] {
				covers = false
				break
			}
		}
		if !covers {
			continue
		}
		for tok := range sibling.said {
			if !candidate.said[tok] {
				out[tok] = true
			}
		}
	}
	return out
}

// unexplainedTokens counts the vendor words that neither the candidate's
// own name nor the set it is filed under accounts for. The counts the name
// itself marks as counts, and the language noise sealedExtrasSafe tolerates,
// are not identity, so they do not count against a candidate.
func unexplainedTokens(vendor []string, candSet, setTokens, counts map[string]bool) int {
	var n int
	for _, tok := range vendor {
		if candSet[tok] || setTokens[tok] || counts[tok] {
			continue
		}
		switch tok {
		case "s", "en", "english":
			continue
		}
		n++
	}
	return n
}

// ResolveSealed resolves a storefront's name for a sealed product to its uuid,
// using the default datastore.
func ResolveSealed(name string) (string, error) {
	return defaultBackend.ResolveSealed(name)
}

// ResolveSealedWithHint resolves a storefront's name for a sealed product to
// its uuid, letting the phrase the storefront files it under settle a tie the
// name alone cannot, using the default datastore.
func ResolveSealedWithHint(name, hint string) (string, error) {
	return defaultBackend.ResolveSealedWithHint(name, hint)
}

// SealedNameSubsumed reports whether a storefront name says everything one of
// the names beside it says and at least one word more, with the shelf they all
// sit on discounted.
//
// Two entries of one catalog are two products, and the resolver cannot see
// that: it is asked about one name at a time and answers for the product that
// name describes best, so a catalog listing both "Marnie Premium Tournament
// Collection Box" and "Marnie Premium Tournament Collection Box Bundle" gets
// one answer twice. The longer name is the one that is wrong. Its extra word
// is what it sells - a bundle of the boxes, not a box - and left on the box's
// uuid it prices a case at a box's price. The caller passes the names that
// reached one product, and the set that product is filed under, and drops the
// ones this answers for.
//
// The shelf is discounted because a storefront prepends it freely, the same
// reason the resolver forgives it: Cardmarket lists the one Victini Box twice,
// once plainly and once as "Black & White Victini Box", and the set's own name
// is the whole of the difference. A name that adds nothing else is filing the
// product under its set, not naming something built on it.
//
// Only a strict superset counts. A catalog spells one product several ways -
// "Sun & Moon Booster Box" beside "Sun & Moon Booster Booster Box" - and those
// say the same words; two names that merely differ, "Promotion Pack 2022
// Vol.1" and "Vol.2", say nothing about which of them is the product either,
// and guessing between them is not this question.
//
// The names are split the way the resolver splits them, since a caller reading
// them any other way reads a difference the resolver never saw.
func SealedNameSubsumed(name string, beside []string, shelf string) bool {
	shelved := map[string]bool{}
	for _, word := range sealedTokens(shelf) {
		shelved[word] = true
	}
	words := sealedTokens(name)
	said := map[string]bool{}
	for _, word := range words {
		said[word] = true
	}
	for _, other := range beside {
		shorter := sealedTokens(other)
		// A name left saying nothing says nothing about this one either.
		if len(shorter) == 0 || len(shorter) >= len(words) {
			continue
		}
		if sealedNameSaysMore(said, words, shorter, shelved) {
			return true
		}
	}
	return false
}

// sealedNameSaysMore reports whether a name holds every word of a shorter one
// and at least one word more that the shelf does not account for.
func sealedNameSaysMore(said map[string]bool, words, shorter []string, shelved map[string]bool) bool {
	spoken := map[string]bool{}
	for _, word := range shorter {
		if !said[word] {
			return false
		}
		spoken[word] = true
	}
	for _, word := range words {
		if !spoken[word] && !shelved[word] {
			return true
		}
	}
	return false
}

// sealedLanguageWords mark a storefront product as a non-English variant,
// which the English-only datastores deliberately do not carry - and whose
// prices must not land on the English product's uuid.
var sealedLanguageWords = map[string]bool{
	"chinese": true, "simplified": true, "japanese": true, "french": true,
	"german": true, "italian": true, "spanish": true, "portuguese": true,
}

// sealedAsideRe matches the phrases a storefront sets aside from the run of a
// name: what it brackets, what it puts in parentheses, and what it quotes.
// The quotes are matched greedily on either side because Cardmarket doubles
// them, and a single pair read off the doubling would leave the phrase itself
// outside.
var sealedAsideRe = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|"+[^"]*"+`)

// sealedJapanMark is the short form a marketplace writes for a printing made
// for the Japanese market rather than naming its language: Cardmarket spells a
// whole run of Japanese boxes "<set> JP Booster Box" and nothing else about
// them says so.
const sealedJapanMark = "jp"

// sealedMarksJapanPrinting reports whether a name carries that mark where it
// speaks for the printing: in the run of the name itself, not inside a phrase
// the storefront set aside.
//
// The distinction is the whole of what makes the mark safe. Two letters are
// short enough to turn up saying something else, and where they do, they are
// always set aside: the datastore's own English rows spell "(JP Pokemon Center
// Exclusive)" and "(JP Raging Bolt)", one naming the storefront an edition was
// sold at and the other the deck a world champion played. Both name something
// beside the product; neither says the product is Japanese, and a storefront
// writing one of those names an English printing.
func sealedMarksJapanPrinting(name string) bool {
	bare := sealedAsideRe.ReplaceAllString(strings.ToLower(name), " ")
	for _, tok := range sealedTokenRe.FindAllString(bare, -1) {
		if tok == sealedJapanMark {
			return true
		}
	}
	return false
}

// SealedIsLanguageVariant reports whether a storefront's product name marks
// a non-English printing ("Origins Booster Box (Chinese, Slim)", "The First
// Chapter Japanese Booster Box", "Black Bolt JP Booster Box").
func SealedIsLanguageVariant(name string) bool {
	var sawNon bool
	for _, tok := range sealedTokenRe.FindAllString(strings.ToLower(name), -1) {
		if sealedLanguageWords[tok] {
			return true
		}
		// Cardmarket marks the ones it does not name a language for as
		// "(Non-English)", which says the same thing: not the printing the
		// datastore carries.
		if sawNon && tok == "english" {
			return true
		}
		sawNon = tok == "non"
	}
	return sealedMarksJapanPrinting(name)
}

// AddSealed files a sealed product in the sealed namespace: its uuid in
// AllSealedUUIDs and in its set's bucket, its name in the sealed name index,
// and the product id as an identifier for BuildSealedProductMap rather than
// in the external identifier index — which is how Magic keeps sealed out of
// MatchID's reach.
//
// A product whose set is unknown is dropped. A uuid a card already holds
// keeps that card: the set still lists the product, because that listing is
// what the sealed views read, but the name index and the uuid map answer for
// what was already there.
//
// A zero product id carries no identifier at all rather than the zero value,
// which would give BuildSealedProductMap one shared key for every unlinked
// listing to funnel onto.
func (b *Backend) AddSealed(uuid, name, setCode, image string, tcgplayerProductID int) {
	set := b.Sets[setCode]
	if set == nil {
		return
	}

	card := Card{
		UUID:    uuid,
		Name:    name,
		SetCode: setCode,
		Rarity:  "product",
		Images: map[string]string{
			"full":      image,
			"thumbnail": image,
		},
		Language: "English",
	}
	if tcgplayerProductID != 0 {
		card.Identifiers = map[string]string{
			"tcgplayerProductId": fmt.Sprint(tcgplayerProductID),
		}
	}

	set.SealedProduct = append(set.SealedProduct, SealedProduct{
		UUID:        uuid,
		Name:        name,
		SetCode:     setCode,
		Identifiers: card.Identifiers,
	})

	if _, found := b.UUIDs[uuid]; found {
		return
	}
	// The name lists are gated on their own contents rather than on bucket
	// existence: a card can already own the bucket, and the sealed name must
	// still be searchable. Each list is gated on what it holds rather than
	// on one of the others, for the reason AddName spells out: two spellings
	// can normalize to one string while staying two spellings, and asking
	// the wrong list drops one of them.
	if b.seenSealed == nil {
		b.seenSealed = map[string]bool{}
		b.seenLowerSealed = map[string]bool{}
		b.seenCanonicalSealed = map[string]bool{}
	}
	n := Normalize(name)
	if !b.seenSealed[n] {
		b.seenSealed[n] = true
		b.AllSealed = append(b.AllSealed, n)
	}
	if lower := strings.ToLower(name); !b.seenLowerSealed[lower] {
		b.seenLowerSealed[lower] = true
		b.AllLowerSealed = append(b.AllLowerSealed, lower)
	}
	if !b.seenCanonicalSealed[name] {
		b.seenCanonicalSealed[name] = true
		b.AllCanonicalSealed = append(b.AllCanonicalSealed, name)
	}
	b.Hashes[n] = append(b.Hashes[n], uuid)

	b.UUIDs[uuid] = &CardObject{
		Card:    card,
		Edition: set.Name,
		Sealed:  true,
	}
	b.AllSealedUUIDs = append(b.AllSealedUUIDs, uuid)
	b.SetSealedUUIDs[setCode] = append(b.SetSealedUUIDs[setCode], uuid)
}

// SortSealed puts the sealed indexes in order, once every product is filed.
// The lists are built in the datastore's order and read as sorted ones.
func (b *Backend) SortSealed() {
	sort.Strings(b.AllSealedUUIDs)
	for code := range b.SetSealedUUIDs {
		sort.Strings(b.SetSealedUUIDs[code])
	}
	sort.Strings(b.AllSealed)
	sort.Strings(b.AllCanonicalSealed)
	sort.Strings(b.AllLowerSealed)
	b.sealedIdx = b.buildSealedIndex()
}
