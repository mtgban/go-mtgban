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
	"’", "",
	",", "",
	"®", "",
	":", "",
	"꞉", "",
	"：", "",
	"~", "",
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

	// Accented characters
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
	if found {
		return cached.(string)
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

// Compare strings after both are Normalize-d.
func Equals(str1, str2 string) bool {
	return Normalize(str1) == Normalize(str2)
}

// Check if str1 contains str2 after both are Normalize-d.
func Contains(str1, str2 string) bool {
	return strings.Contains(Normalize(str1), Normalize(str2))
}

// Check if str2 is the prefix of str1 after both are Normalize-d.
func HasPrefix(str1, str2 string) bool {
	return strings.HasPrefix(Normalize(str1), Normalize(str2))
}

// Check if str2 is the suffix of str1 after both are Normalize-d.
func HasSuffix(str1, str2 string) bool {
	return strings.HasSuffix(Normalize(str1), Normalize(str2))
}
