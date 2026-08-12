package mtgmatcher

import (
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

func tokensSubset(sub, super []string) bool {
	supSet := map[string]bool{}
	for _, tok := range super {
		supSet[tok] = true
	}
	for _, tok := range sub {
		if !supSet[tok] {
			return false
		}
	}
	return true
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
		if tokensSubset(candidate, vendor) && sealedExtrasSafe(vendor, candidate, setTokens) {
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
	for _, tok := range sealedTokenRe.FindAllString(strings.ToLower(name), -1) {
		if sealedLanguageWords[tok] {
			return true
		}
	}
	return false
}
