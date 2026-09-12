package pokemon

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// CandidateSets preserves Pokemon's promo-shelf fallback. An unresolved
// heading such as Plasma Storm Promos with Holo Promo wording must not widen
// straight onto every ordinary printing sharing the collector number. Exact
// editions still win; in the loose pass, promo shelves join edition matches.
func (r Rules) CandidateSets(b *mtgmatcher.Backend, in *mtgmatcher.InputCard, editions []string) []string {
	if len(editions) <= 1 || in.PromoWildcard || !b.IsGenericPromo(in) {
		return r.DefaultRules.CandidateSets(b, in, editions)
	}
	var exact, loose []string
	for _, code := range editions {
		set := b.Sets[code]
		if mtgmatcher.Equals(set.Name, in.Edition) {
			exact = append(exact, code)
		}
		if mtgmatcher.Contains(set.Name, in.Edition) || strings.HasSuffix(set.Name, "Promos") {
			loose = append(loose, code)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(loose) > 0 {
		return loose
	}
	return editions
}
