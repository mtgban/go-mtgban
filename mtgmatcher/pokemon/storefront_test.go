package pokemon

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoPseudoExpansionAlias pins the storefront promo pseudo-expansions
// resolving to the set the catalog files those printings in. None of the
// names shares a tail with its set's, so without the table the edition
// narrows nothing and every printing of the name stands.
func TestPromoPseudoExpansionAlias(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"wizards black star names the wotc promos", mtgmatcher.InputCard{
			Name: "Mewtwo", Edition: "Wizards Black Star Promos", Variation: "014 14"},
			"14-53_87417"},
		{"mcdonald's collection names its promo set", mtgmatcher.InputCard{
			Name: "Froakie", Edition: "McDonald's Collection 25th Anniversary", Variation: "022h Holo Promo | 022/025"},
			"022-025_232336_holo"},
		// The prize pack series and the SV promos are absent from the
		// table on purpose: both sets file two products at one collector
		// number - a holo sibling per series, and SV pairs only an id can
		// tell apart - so an aliased edition turns a refusal into a coin
		// flip. Replayed against the products whose TCGplayer id names
		// the answer, the flip lands wrong for 69 of 260 SV products and
		// 73 prize pack cards, so the edition stays unaliased and the
		// name refuses instead.
		{"prize pack series refuses the sibling coin flip", mtgmatcher.InputCard{
			Name: "Grotle", Edition: "Play! Pokémon Prize Pack Series", Variation: "007 BRS 007"},
			""},
		{"sv black star refuses the sibling coin flip", mtgmatcher.InputCard{
			Name: "Charizard ex", Edition: "SV Black Star Promos", Variation: "196 SVP 196"},
			""},
		{"w promos reach the w-stamped printing", mtgmatcher.InputCard{
			Name: "Misty's Psyduck", Edition: "W Promos", Variation: "054 W Promo | 54/132"},
			"054-132_166296"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("Match(%v) = %s (%v), want an error", tt.in, id, b.UUIDs[id])
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestPooledBlisterDeckHeading pins the one storefront name spanning two
// sets: the candidates narrow to the Blister Exclusives / Deck Exclusives
// pair, the collector number picks within it, and a card the pool does not
// carry refuses rather than falling through to a same-numbered printing in
// some other set.
func TestPooledBlisterDeckHeading(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a build and battle kit promo lands in deck exclusives", mtgmatcher.InputCard{
			Name: "Chi-Yu", Edition: "Theme Deck & Blisters Exclusives", Variation: "029h Holo Build & Battle Kit | 029/182"},
			"029-182_528309"},
		{"the version's treatment picks the plain half of the pool", mtgmatcher.InputCard{
			Name: "Zacian", Edition: "Theme Deck & Blisters Exclusives", Variation: "045 Non-Holo | 045/094"},
			"045-094_664005"},
		{"a card the pool does not carry refuses", mtgmatcher.InputCard{
			Name: "Maschiff", Edition: "Theme Deck & Blisters Exclusives", Variation: "146 Cosmos Holo | 142/193"},
			""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("Match(%v) = %s (%v), want an error", tt.in, id, b.UUIDs[id])
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}
}

// TestStorefrontNameTails pins the decorations cardtrader writes into the
// name itself coming off - and the real catalog names wearing the same
// shapes staying whole.
func TestStorefrontNameTails(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"a numeric level strips", mtgmatcher.InputCard{
			Name: "Moltres Lv.33", Edition: "Wizards Black Star Promos", Variation: "021 21"},
			"21-53_87557"},
		{"the delta species tail strips", mtgmatcher.InputCard{
			Name: "Deoxys δ Delta Species", Edition: "POP Series 4", Variation: "02/17h Holo Promo | 2/17"},
			"002-017_84765_holo"},
		{"a bracketed letter strips", mtgmatcher.InputCard{
			Name: "Unown [J]", Edition: "Wizards Black Star Promos", Variation: "038 38"},
			"38-53_90215"},
		{"a dash subtitle respells as the bracketed supporter", mtgmatcher.InputCard{
			Name: "Boss's Orders - Cyrus", Edition: "Play! Pokémon Prize Pack Series", Variation: "132 132/172"},
			"132-172_515539"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			if id != tt.want {
				t.Errorf("Match(%v) = %s (%v), want %s", tt.in, id, b.UUIDs[id], tt.want)
			}
		})
	}

	// The catalog's own LV.X names wear the level shape with an X, and the
	// strip must never take them apart.
	in := mtgmatcher.InputCard{Name: "Heatran LV.X", Variation: "97"}
	Rules{}.Prefilter(b, &in)
	if in.Name != "Heatran LV.X" {
		t.Errorf("Prefilter rewrote %q to %q", "Heatran LV.X", in.Name)
	}
}
