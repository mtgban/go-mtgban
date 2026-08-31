package mtgmatcher

import (
	"strings"
	"sync"
	"sync/atomic"
)

var replacerStrings = []string{
	// Remove a very common field, sometimes added with no reason
	// Needs the dashes to work with will-o'-the-wisp, which is why
	// it needs to be before removing the dash step
	" the ", "",
	"-the-", "",
	// Hopefully "of/from the" is specific enough
	"of the", "of the",
	"from the", "from the",

	// I can't even
	"jeong, the", "jeongthe",

	// Wrong escaping or conversion
	"&quot;", "",

	// Quotes and commas and whatnot
	"''", "",
	"“", "",
	"”", "",
	"\"", "",
	"'", "",
	"-", "",
	"—", "",
	// Storefronts typeset the hyphen a card is printed with: cardmarket
	// sends Lorcana's "Fix-It Felix, Jr." spelled with U+2010, and its
	// dashes with an en dash. Left standing they are one more letter, and
	// the name matches nothing.
	"‐", "",
	"–", "",
	"’", "",
	",", "",
	"®", "",
	":", "",
	"꞉", "",
	"：", "",
	"~", "",
	// A catalog numbers the cards that share a name, and the storefronts
	// spell that number without the mark: Yu-Gi-Oh files "Sasuke Samurai
	// #2" where a listing says "Sasuke Samurai 2". No Magic name carries
	// one at all, and across every game exactly one pair of names is drawn
	// together by dropping it - two spellings of Flying Kamakiri #1, which
	// is one card either way.
	"#", "",
	"(", "",
	")", "",
	".", "",
	"!", "",
	"?", "",
	"+", "",
	"…", "",

	// UNF blanks
	"___________", "_____",
	"__________", "_____",
	"_________", "_____",
	"________", "_____",
	"_______", "_____",
	"______", "_____",

	// Separators
	"goblin // soldier", "goblin // soldier",
	"/", "",
	"|", "",
	"trial and error", "trial and error",
	"welcome to", "welcometo",
	" and ", "",
	" to ", "",
	" & ", "",
	"reverse the polarity", "reverse the polarity",
	"glimpse, the unthinkable", "glimpse, the unthinkable",

	// Accented characters; see asciiReplacer, which folds the same set for
	// callers that tokenize rather than normalize.
	"â", "a",
	"á", "a",
	"à", "a",
	"ä", "a",
	"ā", "a",
	"é", "e",
	"í", "i",
	"ï", "i",
	"ö", "o",
	"ō", "o",
	"ó", "o",
	"ú", "u",
	"û", "u",
	"ü", "u",
	"ñ", "n",

	// Ancient ligature
	"æ", "ae",

	// Also plurals, just preserve 'blossom' that aliases 'lotus bloom'
	// and 'asp' for 'tangle asp'/'tanglesap', and ogress...
	// 'vs' is a key for determining duel decks
	// Any accented s need to be removed as well to behave like a normal s
	"asp", "asp",
	"lossom", "lossom",
	"ogress", "ogress",
	"slash", "slash",
	"squash", "squash",
	"sword", "sword",
	"kess", "kess",
	"kediss", "kediss",
	"vs", "vs",
	"pest", "pest",
	"š", "",
	"s", "",

	// Spaces are overrated, except when not
	"waste land", "waste land",
	" ", "",
}

// asciiStrings folds the letters that carry a mark down to the plain ascii
// they stand for, and nothing else, for the callers that split a name into
// words themselves and must not have the rest of the normalizing rewrites
// applied to it.
var asciiStrings = []string{
	"â", "a",
	"á", "a",
	"à", "a",
	"ä", "a",
	"ā", "a",
	"é", "e",
	"í", "i",
	"ï", "i",
	"ö", "o",
	"ō", "o",
	"ó", "o",
	"ú", "u",
	"û", "u",
	"ü", "u",
	"ñ", "n",
	"æ", "ae",
}

// asciiReplacer folds a marked letter to the plain ascii it stands for.
var asciiReplacer = strings.NewReplacer(asciiStrings...)

var replacer = strings.NewReplacer(replacerStrings...)

// Normalize is called millions of times, and the generic Replacer
// allocates on every call even when nothing matches, so results are
// memoized. The gain comes from repetition in what callers pass in, not
// just from the datastore names: a scrape normalizes the same handful of
// store tags across every listing. Caller input therefore has to be
// cached to be worth anything, which means arbitrary strings reach this
// map (the site hands it search queries), and the two bounds below exist
// to keep that from being a liability.
const (
	normalizeCacheCap = 1 << 17

	// The longest name the datastore produces is 141 bytes. Anything
	// past this is caller input too long to repeat, and caching it only
	// buys a way to retain megabytes of someone else's text.
	normalizeCacheMaxKey = 160
)

var (
	normalizeCache     atomic.Pointer[sync.Map] // string -> string
	normalizeCacheSize atomic.Int64
)

func init() {
	normalizeCache.Store(&sync.Map{})
}

// Normalize uses the rules defined in Replacer to replace uncommon elements of
// card names, dropping all the spaces and producing a lowercase string.
func Normalize(str string) string {
	cache := normalizeCache.Load()
	cached, found := cache.Load(str)
	normalized, isString := cached.(string)
	if found && isString {
		return normalized
	}

	out := strings.TrimSpace(str)
	out = strings.ToLower(out)
	out = replacer.Replace(out)

	if len(str) > normalizeCacheMaxKey {
		return out
	}

	if normalizeCacheSize.Load() >= normalizeCacheCap {
		// Start over rather than stop: freezing a full cache lets a
		// flood of one-off keys lock it shut for the rest of the
		// process, so that a later datastore reload would never cache
		// its new names. The entries worth having are re-filled by the
		// calls that follow. Two goroutines racing here only costs a
		// second reset.
		normalizeCache.Store(&sync.Map{})
		normalizeCacheSize.Store(0)
		cache = normalizeCache.Load()
	}

	// Clone the key so the cache cannot pin a larger buffer the
	// input may be slicing (the output is always freshly built)
	_, loaded := cache.LoadOrStore(strings.Clone(str), out)
	if !loaded {
		normalizeCacheSize.Add(1)
	}
	return out
}

// Equals reports whether the two strings are the same after both are
// Normalize-d, which folds case and drops punctuation.
func Equals(str1, str2 string) bool {
	return Normalize(str1) == Normalize(str2)
}

// Contains reports whether str1 contains str2 after both are Normalize-d.
func Contains(str1, str2 string) bool {
	return strings.Contains(Normalize(str1), Normalize(str2))
}

// HasPrefix reports whether str2 is a prefix of str1 after both are
// Normalize-d.
func HasPrefix(str1, str2 string) bool {
	return strings.HasPrefix(Normalize(str1), Normalize(str2))
}

// HasSuffix reports whether str2 is a suffix of str1 after both are
// Normalize-d.
func HasSuffix(str1, str2 string) bool {
	return strings.HasSuffix(Normalize(str1), Normalize(str2))
}

// CloseName reports whether got is want with a piece missing off one end or
// up to two letters wrong, both read Normalize-d. It is the misspelling test
// a game's rules gate a rename on, once something else - a collector number,
// an edition - has already picked the one printing the new name may come
// from: the closeness is what stops that pick from renaming a card into an
// unrelated one. Which evidence licenses the rename is each game's own
// business; what counts as close is not.
func CloseName(got, want string) bool {
	got, want = Normalize(got), Normalize(want)
	if got == "" || want == "" {
		return false
	}
	if strings.HasPrefix(want, got) || strings.HasSuffix(want, got) {
		return true
	}
	return editDistance(got, want, 2) <= 2
}

// editDistance is the Levenshtein distance between two strings, giving up at
// limit: the caller only cares whether the two are within a couple of edits,
// and a name pair that is not stops being measured once the whole row of the
// table is past the limit.
func editDistance(a, b string, limit int) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) > len(br) {
		ar, br = br, ar
	}
	if len(br)-len(ar) > limit {
		return limit + 1
	}
	prev := make([]int, len(ar)+1)
	cur := make([]int, len(ar)+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= len(br); j++ {
		cur[0] = j
		best := cur[0]
		for i := 1; i <= len(ar); i++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[i] = min(prev[i]+1, cur[i-1]+1, prev[i-1]+cost)
			best = min(best, cur[i])
		}
		if best > limit {
			return limit + 1
		}
		prev, cur = cur, prev
	}
	return prev[len(ar)]
}
