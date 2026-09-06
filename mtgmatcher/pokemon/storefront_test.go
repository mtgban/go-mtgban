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
		// The SV promos are absent from both tables on purpose: the set
		// pairs printings only an id can tell apart, and replayed against
		// the products whose TCGplayer id names the answer the flip lands
		// wrong for 69 of 260, so the edition stays unresolved and the
		// name refuses instead.
		//
		// The prize pack series was out for the same reason and is back,
		// pooled rather than aliased. Its siblings carry the treatment
		// labels now - every one of the 130 numbers the set files more
		// than one product at is told apart by a label - so restricting
		// to the set no longer flips a coin. Replayed the same way over
		// the whole set, the pool answers 937 right against 2 wrong where
		// the unrestricted name managed 191 against 4, and the 4 it drops
		// were printings in other sets entirely: a prize pack Zekrom ex
		// priced as the Black Bolt one. Grotle is filed as a single
		// product, so refusing it was never the sibling case anyway.
		{"prize pack series reaches its own set", mtgmatcher.InputCard{
			Name: "Grotle", Edition: "Play! Pokémon Prize Pack Series", Variation: "007 BRS 007"},
			"007-172_489316"},
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
		{"the cosmos holo of an ex box lands with the miscellaneous cards", mtgmatcher.InputCard{
			Name: "Maschiff", Edition: "Theme Deck & Blisters Exclusives", Variation: "146 Cosmos Holo | 142/193"},
			"142-193_544444_holo"},
		{"a card the pool does not carry refuses", mtgmatcher.InputCard{
			Name: "Chansey", Edition: "Theme Deck & Blisters Exclusives", Variation: "003 Cosmos Holo | 003/102"},
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

// TestVersionWording pins what the ride-along Version buys and what its
// guards refuse: an agreeing set total outranks a bare verbatim number, a
// contradicting total vetoes the candidate, a wording spelling both
// treatments stays ambiguous, and the stamp, sequin and cosmos wordings
// never land on a plain printing.
func TestVersionWording(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{"an agreeing total beats the bare promo number", mtgmatcher.InputCard{
			Name: "Venusaur", Edition: "Supreme Victors Promos", Variation: "013/147 Non-Holo Theme Deck | 13/147"},
			"013-147_125073"},
		{"a wording spelling both treatments stays ambiguous", mtgmatcher.InputCard{
			Name: "Counter Catcher", Edition: "Play! Pokémon Prize Pack Series", Variation: "160 Non-Holo / Cosmos Holo | PAR 160"},
			""},
		{"a stamp wording refuses the unstamped printing", mtgmatcher.InputCard{
			Name: "Rapidash", Edition: "Wizards Black Star Promos", Variation: "051pc Pokemon Center Stamped | 51"},
			""},
		{"a stamp wording still reaches a stamped sibling", mtgmatcher.InputCard{
			Name: "Buddy-Buddy Poffin", Edition: "League Promos", Variation: "144staff STAFF | Reverse Holo 144",
			Finish: "Reverse Holofoil", Foil: true},
			"144-162_638139_reverse"},
		{"a sequin wording reaches the general mills printing", mtgmatcher.InputCard{
			Name: "Jangmo-o", Edition: "SM Black Star Promos", Variation: "SM40sq Sequin Holo Promo | SM40"},
			"sm40_161466_holo"},
		{"a cosmos wording refuses a printing that cannot be one", mtgmatcher.InputCard{
			Name: "Fezandipiti", Edition: "Theme Deck & Blisters Exclusives", Variation: "096 Cosmos Holo | 096/167"},
			""},
		{"a cosmos wording keeps a printing selling the holo", mtgmatcher.InputCard{
			Name: "Eevee", Edition: "Wizards Black Star Promos", Variation: "011 Cosmos Holo 11"},
			"11-53_85074_holo"},
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

// TestTotalDisagrees pins the veto's edges: only two spelled-out totals can
// contradict, an agreeing field clears a decorated one, and a wording with
// no total for the numerator vetoes nothing.
func TestTotalDisagrees(t *testing.T) {
	for _, tt := range []struct {
		variation, number string
		want              bool
	}{
		{"Non-Holo | 053/167", "053/094", true},
		{"Holo Promo | 013/025", "013/025", false},
		// The blueprint's own decorated field disagrees, the version's
		// clean one agrees, and agreement wins.
		{"02/17h Holo Promo | 2/17", "002/017", false},
		// No slashed field for the numerator: nothing to compare.
		{"144staff STAFF | Reverse Holo 144", "144/162", false},
		// A card number with no total cannot be contradicted.
		{"Non-Holo | 053/167", "053", false},
	} {
		if got := totalDisagrees(tt.variation, tt.number); got != tt.want {
			t.Errorf("totalDisagrees(%q, %q) = %v, want %v", tt.variation, tt.number, got, tt.want)
		}
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
		// Named by the set rather than by the storefront's prize pack
		// wording, which is unaliased on purpose and refuses: what is
		// under test is the name, and an edition that cannot narrow
		// leaves the case resting on there being one candidate. There
		// were two the day the catalog filed this card in Deck
		// Exclusives as well.
		{"a dash subtitle respells as the bracketed supporter", mtgmatcher.InputCard{
			Name: "Boss's Orders - Cyrus", Edition: "Prize Pack Series Cards", Variation: "132 132/172"},
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

// TestUnresolvedEditionDoesNotWiden pins the widening gate: a Japanese
// catalog's set name resolves to nothing, and with no edition to gate on
// the qualified-name widening must not walk the listing onto an English
// printing that happens to carry the number.
func TestUnresolvedEditionDoesNotWiden(t *testing.T) {
	b := loadBackend(t)

	in := mtgmatcher.InputCard{
		Name: "Moltres ex", Edition: "Plasma Storm Promos", Variation: "014/135 Holo Promo | 14/135"}
	if id, err := b.Match(&in); err == nil {
		t.Fatalf("Match(%v) = %s (%v), want an error", in, id, b.UUIDs[id])
	}

	// A resolved edition still widens onto the qualified spelling.
	in = mtgmatcher.InputCard{
		Name: "Cherrim", Edition: "SWSH Black Star Promos", Variation: "088 Prerelease Promo | SWSH088"}
	id, err := b.Match(&in)
	if err != nil {
		t.Fatalf("Match(%v) = %v", in, err)
	}
	if id != "swsh088_234276_holo" {
		t.Errorf("Match(%v) = %s (%v), want swsh088_234276_holo", in, id, b.UUIDs[id])
	}
}

// TestExtractNumbersSkipsYears pins that a bare year in the wording is not
// read as a collector number: the league energies are unnumbered and
// labelled by year, and "2006 Non-Holo Promo" names the label.
func TestExtractNumbersSkipsYears(t *testing.T) {
	for _, tt := range []struct {
		wording string
		want    []string
	}{
		{"FFE-9JT-SUX | 2006 Non-Holo Promo", nil},
		{"2006 Unnumbered", nil},
		{"Cosmos Holo | 024/167", []string{"024"}},
		{"SVP 205", []string{"205"}},
	} {
		got := extractNumbers(tt.wording)
		if len(got) != len(tt.want) {
			t.Errorf("extractNumbers(%q) = %v, want %v", tt.wording, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractNumbers(%q) = %v, want %v", tt.wording, got, tt.want)
			}
		}
	}
}
