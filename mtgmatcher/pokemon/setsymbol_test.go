package pokemon

import (
	"strings"
	"testing"
)

// TestSetSymbol pins the mark a set's cards print, which the builder carries
// from tcgdex and fills from pokemontcg.io where tcgdex has none. Not every
// set has one, so anything rendering these has to be ready to draw the set
// code instead: a missing symbol is a state to handle, not a gap to fill.
//
// Only the tcgdex-sourced sets are pinned by value. The pokemontcg.io half
// is best-effort in the builder - a build made while that API is down keeps
// none of it, and it answered 500 twice while this was written - so pinning
// one would fail on the datastore rather than on a defect.
func TestSetSymbol(t *testing.T) {
	b := loadBackend(t)

	var carried int
	for _, set := range b.Sets {
		if set.Symbol != "" {
			carried++
		}
	}
	if carried == 0 {
		t.Skip("this datastore predates the published set symbol")
	}

	for _, tt := range []struct {
		code string
		want string
	}{
		{"JU", "https://assets.tcgdex.net/en/base/base2/symbol.png"},
		{"N1", "https://assets.tcgdex.net/en/neo/neo1/symbol.png"},
		{"PRC", "https://assets.tcgdex.net/en/xy/xy5/symbol.png"},
		{"SVI", "https://assets.tcgdex.net/en/sv/sv01/symbol.png"},
	} {
		set, found := b.Sets[tt.code]
		if !found {
			t.Errorf("%s is not a set", tt.code)
			continue
		}
		if set.Symbol != tt.want {
			t.Errorf("%s: Symbol is %q, want %q", tt.code, set.Symbol, tt.want)
		}
	}

	// Whatever a set does carry has to be an asset that can be fetched, from
	// one of the two sources the builder reads: tcgdex serves png under its
	// en locale, and pokemontcg.io, which fills the sets tcgdex holds no
	// symbol for, serves png too. Neither path may be built from a set id -
	// a wrong tcgdex path answers with an HTML 404 typed image/png, the same
	// shape it answered in when this build still asked for webp under the
	// universal path tcgdex has since stopped serving anything under.
	for code, set := range b.Sets {
		if set.Symbol == "" {
			continue
		}
		fromTcgdex := strings.HasPrefix(set.Symbol, "https://assets.tcgdex.net/en/") &&
			strings.HasSuffix(set.Symbol, "/symbol.png")
		fromPokemontcg := strings.HasPrefix(set.Symbol, "https://images.pokemontcg.io/") &&
			strings.HasSuffix(set.Symbol, "/symbol.png")
		if !fromTcgdex && !fromPokemontcg {
			t.Errorf("%s: Symbol is %q, want a tcgdex or a pokemontcg.io png", code, set.Symbol)
		}
	}
}
