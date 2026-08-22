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

// sealedQualifierTokens returns the words a catalog name carries only inside
// brackets. Those name what a product is decorated with rather than the
// product: the deck the catalog files as `Theme Deck - "Storm Rider"
// [Zapdos]` is the Storm Rider deck whatever Zapdos is doing on the box, and
// a storefront naming it says nothing about Zapdos.
//
// A bracket naming a print run is not a decoration - two runs of one product
// are two products and the bracket is the only thing telling them apart - so
// it is left in. A word the name also carries outside its brackets is left
// in too, since there it is doing the product's own work.
func (b *Backend) sealedQualifierTokens(name string) map[string]bool {
	groups := bracketRe.FindAllStringSubmatch(name, -1)
	if len(groups) == 0 {
		return nil
	}

	inside := map[string]bool{}
	for _, group := range groups {
		// Only a card of this game's own is decoration. Everything else a
		// bracket holds is the product's identity - the print run, the
		// number of copies, the placing a promo was handed out for - and
		// forgiving those would merge products that differ by nothing else.
		if _, found := b.CanonicalNames[Normalize(group[1])]; !found {
			continue
		}
		for _, tok := range sealedTokens(group[1]) {
			inside[tok] = true
		}
	}
	if len(inside) == 0 {
		return nil
	}
	for _, tok := range sealedTokens(bracketRe.ReplaceAllString(name, " ")) {
		delete(inside, tok)
	}
	return inside
}

// tokensSubsetModulo reports whether every word of sub is one that super says
// or one of the free words beside it, with something super said still left
// over. The floor matters: a candidate forgiven down to nothing would answer
// for every name in the game.
func tokensSubsetModulo(sub, super []string, free map[string]bool) bool {
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
			return false
		}
	}
	return matched > 0
}

// ResolveSealed returns the uuid of the single sealed product the given
// storefront name describes. An exact token match wins; failing that, a
// candidate fully contained in the vendor's wording wins when its extras
// are safe and it is the most specific such candidate. No match, or more
// than one equally good, is ErrCardDoesNotExist: unique or nothing.
func (b *Backend) ResolveSealed(name string) (string, error) {
	if b.UUIDs == nil {
		return "", ErrDatastoreEmpty
	}

	setTokens := map[string]bool{}
	for _, set := range b.Sets {
		for _, tok := range sealedTokens(set.Name) {
			setTokens[tok] = true
		}
	}

	vendor := sealedTokens(name)
	var exact, contained []string
	containedLen := map[string]int{}
	unexplained := map[string]int{}
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
		if tokensSubsetModulo(candidate, vendor, b.sealedQualifierTokens(co.Name)) && sealedExtrasSafe(vendor, candidate, setTokens) {
			contained = append(contained, uuid)
			containedLen[uuid] = len(candidate)
			unexplained[uuid] = unexplainedTokens(vendor, candidate, b.setNameTokens(co.SetCode))
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
		// explains, then by specificity. Counting tokens alone reads a
		// product's own identity as filler when its words happen to be a
		// rearrangement of another's - "Paramount War Box Promotion
		// Booster" names the Box Promotion Pack, but has every word of the
		// Paramount War Booster Box too, and that one is longer. A tie on
		// both stays unresolved.
		sort.Slice(contained, func(i, j int) bool {
			if unexplained[contained[i]] != unexplained[contained[j]] {
				return unexplained[contained[i]] < unexplained[contained[j]]
			}
			if containedLen[contained[i]] != containedLen[contained[j]] {
				return containedLen[contained[i]] > containedLen[contained[j]]
			}
			return contained[i] < contained[j]
		})
		if len(contained) == 1 ||
			unexplained[contained[0]] < unexplained[contained[1]] ||
			containedLen[contained[0]] > containedLen[contained[1]] {
			return contained[0], nil
		}
	}
	return "", ErrCardDoesNotExist
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
// own name nor the set it is filed under accounts for. Counts and the
// language noise sealedExtrasSafe tolerates are not identity, so they do
// not count against a candidate.
func unexplainedTokens(vendor, candidate []string, setTokens map[string]bool) int {
	candSet := map[string]bool{}
	for _, tok := range candidate {
		candSet[tok] = true
	}
	var n int
	for _, tok := range vendor {
		if candSet[tok] || setTokens[tok] || sealedCountRe.MatchString(tok) {
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
