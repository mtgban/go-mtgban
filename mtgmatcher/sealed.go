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
func sealedQuantityTokens(name string) map[string]bool {
	out := map[string]bool{}
	lower := strings.ToLower(asciiReplacer.Replace(name))
	for _, tok := range sealedTokenRe.FindAllString(lower, -1) {
		if sealedMultiplierRe.MatchString(tok) {
			out[tok] = true
		}
	}
	for _, match := range sealedParenRe.FindAllStringSubmatch(lower, -1) {
		toks := sealedTokenRe.FindAllString(match[1], -1)
		if len(toks) < 2 || !sealedNumberRe.MatchString(toks[0]) {
			continue
		}
		if sealedContainerWords[toks[len(toks)-1]] {
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
	"bandai": true,
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
	// are folded the same way before they are looked up.
	name = asciiReplacer.Replace(name)
	for _, tok := range sealedTokenRe.FindAllString(strings.ToLower(name), -1) {
		if sealedFiller[tok] {
			continue
		}
		switch tok {
		case "box", "boxes":
			tok = "display"
		case "booster", "boosters", "packs":
			tok = "pack"
		case "decks":
			tok = "deck"
		case "versus":
			tok = "vs"
		case "volume":
			tok = "vol"
		}
		set[tok] = true
	}
	out := make([]string, 0, len(set))
	for tok := range set {
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

// sealedExtrasSafe reports whether every vendor token missing from the
// candidate is harmless: a count, a set-name word (storefronts file
// products under the set they belong to), or noise. Anything else means
// the vendor is naming a different product - "Prerelease Pack" and
// "Participation Booster" must not resolve to the plain Booster Pack.
func sealedExtrasSafe(vendor, candidate []string, setTokens map[string]bool) bool {
	candSet := map[string]bool{}
	for _, tok := range candidate {
		candSet[tok] = true
	}
	for _, tok := range vendor {
		if candSet[tok] || setTokens[tok] || sealedCountRe.MatchString(tok) {
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

func (b *Backend) resolveSealed(name, hint string) (string, error) {
	if b.UUIDs == nil {
		return "", ErrDatastoreEmpty
	}

	setTokens := map[string]bool{}
	for _, set := range b.Sets {
		for _, tok := range sealedTokens(set.Name) {
			setTokens[tok] = true
		}
	}

	counts := sealedQuantityTokens(name)
	vendor := sealedTokens(name)
	var exact, contained []string
	shared := map[string]int{}
	unsaid := map[string]int{}
	unexplained := map[string]int{}
	forgiven := map[string]bool{}
	qualifiers := map[string]map[string]bool{}
	for _, uuid := range b.AllSealedUUIDs {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		candidate := sealedTokens(co.Name)
		if tokensEqual(candidate, vendor) {
			exact = append(exact, uuid)
			continue
		}
		groups := b.sealedQualifierGroups(co.Name)
		free := sealedQualifierTokens(groups)
		subset, matched := tokensSubsetModulo(candidate, vendor, free)
		if subset && sealedExtrasSafe(vendor, candidate, setTokens) {
			contained = append(contained, uuid)
			// Specificity counts the words the vendor and the candidate
			// actually share. A forgiven word is one the vendor never said,
			// so letting it lengthen the candidate would rank a product by
			// how much of it went unsaid.
			shared[uuid] = matched
			unsaid[uuid] = sealedUnsaidGroups(groups, vendor)
			unexplained[uuid] = unexplainedTokens(vendor, candidate, b.setNameTokens(co.SetCode), counts)
			forgiven[uuid] = matched < len(candidate)
			qualifiers[uuid] = free
		}
	}

	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
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

	var winner string
	var winners int
	for _, uuid := range contained {
		if unexplained[uuid] != unexplained[contained[0]] ||
			shared[uuid] != shared[contained[0]] ||
			unsaid[uuid] != unsaid[contained[0]] {
			continue
		}
		var spoken bool
		for tok := range qualifiers[uuid] {
			if hintTokens[tok] {
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

// setNameTokens returns the identity tokens of a set's name, which a
// storefront may prepend to any product filed under it.
func (b *Backend) setNameTokens(setCode string) map[string]bool {
	out := map[string]bool{}
	set, found := b.Sets[setCode]
	if !found {
		return out
	}
	for _, tok := range sealedTokens(set.Name) {
		out[tok] = true
	}
	return out
}

// unexplainedTokens counts the vendor words that neither the candidate's
// own name nor the set it is filed under accounts for. The counts the name
// itself marks as counts, and the language noise sealedExtrasSafe tolerates,
// are not identity, so they do not count against a candidate.
func unexplainedTokens(vendor, candidate []string, setTokens, counts map[string]bool) int {
	candSet := map[string]bool{}
	for _, tok := range candidate {
		candSet[tok] = true
	}
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

// sealedLanguageWords mark a storefront product as a non-English variant,
// which the English-only datastores deliberately do not carry - and whose
// prices must not land on the English product's uuid.
var sealedLanguageWords = map[string]bool{
	"chinese": true, "simplified": true, "japanese": true, "french": true,
	"german": true, "italian": true, "spanish": true, "portuguese": true,
}

// SealedIsLanguageVariant reports whether a storefront's product name marks
// a non-English printing ("Origins Booster Box (Chinese, Slim)", "The First
// Chapter Japanese Booster Box").
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
	return false
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
}
