// Package vocabulary checks that a loader reads the promo types a datastore
// publishes, and nothing else.
//
// The datastore states a printing's facts and the loader reads them. Every
// way that can go wrong has gone wrong at least once: a loader invented
// tokens the datastore never published, one renamed a published token
// through its own label table, and one showed a reader the slug instead of
// the words. None of it failed a test, because the tests named tokens by
// hand and a token nobody had listed could do as it liked.
package vocabulary

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// ordinalCaps matches an ordinal a title-caser has capitalised: "1St
// Place", "3Rd Anniversary". mtgmatcher.Title puts them back down, and a
// label carrying one was cased by something else.
var ordinalCaps = regexp.MustCompile(`[0-9](St|Nd|Rd|Th)\b`)

// slugRe is everything a token is.
var slugRe = regexp.MustCompile(`^[a-z0-9]+$`)

// Backend is the part of a loaded datastore these checks read.
type Backend struct {
	// Declared is the vocabulary a query is written against.
	Declared []string

	// Labels are the words each declared token reads as, empty for a token
	// the loader kept no spelling of.
	Labels map[string]string

	// Worn are the tokens the loader put on cards, which may hold more than
	// Declared: a mark, a quoted rarity or a date is a fact a listing names
	// without being a promotion, and the games filter their declarations.
	Worn []string
}

// Published is the part of a datastore these checks read: every token it
// states, and the facts a loader is allowed to make a token out of.
type Published struct {
	// Tokens are the promo types the datastore states, slugged.
	Tokens []string

	// Facts are the other things it states that a printing can answer a
	// listing with - its mark, its rarity, its date - slugged. A loader may
	// carry these on a card without declaring them.
	Facts []string

	// Words are the catalog's own wording for each token: the run of words
	// in a carrying printing's variant whose slug is that token. A token
	// the catalog never writes in words has none, and nothing is said
	// about it.
	Words map[string]string

	// Marked says whether this datastore publishes the mark saying which
	// copy of a number a printing is. A datastore that does has had the
	// subjects and the artwork letters taken out of the variant, so a token
	// derived from the variant is one it never stated; a datastore that
	// does not is one a loader still falls back on the variant for, which
	// is what every loader here is written to do.
	Marked bool
}

// Problems are every way a loader's vocabulary departs from the
// datastore's.
type Problems struct {
	// Unstated are tokens the loader declares or carries that the datastore
	// states nowhere.
	//
	// It is measured against every fact a datastore states and not only
	// against its promo types, because which of those facts a game declares
	// is that game's own business: Lorcana declares the pool a promotional
	// printing was numbered within, because storefronts write it behind the
	// number, and One Piece declares no mark at all while carrying several.
	// Both are reading what is published. Inventing is the other thing: a
	// token folded, renamed, or derived from the variant, which the
	// datastore never said.
	Unstated []string

	// NotSlugs are declared tokens a query could not carry.
	NotSlugs []string

	// Unlabelled are declared tokens the loader kept no words for, so a
	// reader is shown the slug.
	Unlabelled []string

	// Mangled are labels whose ordinals were capitalised by a title-caser
	// that does not know what an ordinal is.
	Mangled []string

	// RunTogether are labels the loader kept no words for, shown to a
	// reader as the slug title-cased: "Legendarybattledeck" for a token the
	// catalog writes "Legendary Battle Deck", "Prerelease" for one it
	// writes "Pre-Release".
	//
	// The words are not guessed at: they are the catalog's own, read off
	// the variants of the printings that carry the token.
	//
	// Only a label that is exactly what title-casing the token gives is
	// read as one nobody wrote. Anything else is somebody's, and theirs to
	// spell: "Special" for the tag sp, "Championship Series" for cs, and
	// Toys "R" Us where the catalog writes Toys R Us. Case is not read
	// either, so a table may write "Bandai Card Games Fest" where the
	// catalog shouts it.
	RunTogether []string
}

// Any reports whether anything was found.
func (p Problems) Any() bool {
	return len(p.Unstated)+len(p.NotSlugs)+len(p.Unlabelled)+len(p.Mangled)+len(p.RunTogether) > 0
}

// Lines are the problems as one line each.
func (p Problems) Lines() []string {
	var out []string
	say := func(what string, found []string) {
		if len(found) == 0 {
			return
		}
		shown := found
		if len(shown) > 8 {
			shown = shown[:8]
		}
		out = append(out, fmt.Sprintf("%d %s: %s", len(found), what, strings.Join(shown, ", ")))
	}
	say("tokens the datastore states nowhere", p.Unstated)
	say("declared tokens are not slugs", p.NotSlugs)
	say("declared tokens read back as their own slug", p.Unlabelled)
	say("labels have an ordinal a title-caser capitalised", p.Mangled)
	say("labels read as one word for a token the catalog writes as several", p.RunTogether)
	return out
}

// Check holds a loader to the datastore it read.
func Check(loaded Backend, stated Published) Problems {
	var found Problems
	statable := set(append(append([]string(nil), stated.Tokens...), stated.Facts...))

	for _, token := range sorted(append(append([]string(nil), loaded.Declared...), loaded.Worn...)) {
		if !statable[token] {
			found.Unstated = append(found.Unstated, token)
		}
	}
	for _, token := range sorted(loaded.Declared) {
		if !slugRe.MatchString(token) {
			found.NotSlugs = append(found.NotSlugs, token)
		}
		label := loaded.Labels[token]
		if label == "" {
			found.Unlabelled = append(found.Unlabelled, token)
			continue
		}
		if ordinalCaps.MatchString(label) {
			found.Mangled = append(found.Mangled, token+" = "+label)
		}
		words := stated.Words[token]
		if words != "" && label == mtgmatcher.Title(token) && !strings.EqualFold(label, words) {
			found.RunTogether = append(found.RunTogether, fmt.Sprintf("%s = %q, written %q", token, label, words))
		}
	}
	return found
}

// set is a list as the lookup of it.
func set(list []string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, item := range list {
		out[item] = true
	}
	return out
}

// sorted is a list in order, without repeats, leaving the caller's alone.
func sorted(list []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

// Slug is a label as the token a query would carry, for telling a label the
// loader kept from one it title-cased out of the token.
func Slug(label string) string {
	return mtgmatcher.PromoTypeSlug(label)
}
