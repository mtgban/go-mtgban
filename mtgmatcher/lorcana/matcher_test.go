package lorcana

import (
	"encoding/json"
	"flag"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// matchTest mirrors the Magic matcher harness: an input card and the expected
// Match() verdict (uuid or error string). Positive cases were baked from
// Match's verdicts over a sampled Lorcana corpus, so the suite guards that
// Match keeps resolving those cards the same way across changes. The
// hand-authored seeds below pin the contract edges the
// sampled corpus cannot reach: error paths, case-variant spellings,
// parenthetical suffixes, prefix names, and zero-numbered cards. Run with
// -update-lorcana to re-bake expected verdicts after a deliberate change;
// the regeneration refuses to flip a case between success and error, so a
// regression cannot be blessed silently.
type matchTest struct {
	Desc string               `json:"description"`
	In   mtgmatcher.InputCard `json:"input"`
	ID   string               `json:"uuid,omitempty"`
	Err  string               `json:"error,omitempty"`
}

const lorcanaTestData = "testdata/lorcana_test_data.json"

var updateLorcana = flag.Bool("update-lorcana", false,
	"re-run Match over every test input and rewrite "+lorcanaTestData)

// lorcanaSeeds are hand-authored cases ensured present by -update-lorcana.
// Expected verdicts are baked by the regeneration rather than hard-coded, so
// they always reflect what Match actually returns; a "negative:" description
// prefix declares the author's intent and the regeneration fails loudly if
// the outcome class does not match it. Concrete cards referenced here exist
// in the real datastore (checked against LorcanaJSON at authoring time).
var lorcanaSeeds = []matchTest{
	{
		// TCGplayer prices this printing as three skus; the id names the
		// printing and the finish beside it names which sku.
		Desc: "product id plus the sku's finish reaches the foil sub-type",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204", Finish: "Holofoil", Foil: true},
	},
	{
		// The same product id, each of the other skus TCGplayer prices it
		// as, so no two of them can answer with one uuid.
		Desc: "product id plus the plain sku's finish",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204", Finish: "Normal"},
	},
	{
		Desc: "product id plus the standard foil sku's finish",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204", Finish: "Cold Foil", Foil: true},
	},
	{
		// Promotion runs from any sibling: a sub-type's own uuid demotes
		// to the plain printing when the caller prices that sku.
		Desc: "a sub-type uuid plus a finish reaches its plain sibling",
		In:   mtgmatcher.InputCard{ID: "1951_rainbowpillars", Finish: "Normal"},
	},
	{
		// An id with no finish beside it is still the flag's question, so
		// the wording no longer overrides the printing the id names.
		Desc: "an id sent without a finish answers with the flag's foil",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204 Holofoil", Foil: true},
	},
	{
		// A storefront that sends no id still spells the sub-type, and
		// that path is untouched.
		Desc: "wording alone still picks the foil sub-type",
		In:   mtgmatcher.InputCard{Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204 Holofoil", Foil: true},
	},
	{
		// Satin is a finish this game sells and this printing does not, so
		// the contradiction stands rather than answering with a sibling.
		Desc: "negative: a finish the printing is not sold in",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204", Finish: "Satin", Foil: true},
	},
	{
		// A name the game cannot place at all is a vendor spelling nobody
		// has taught it yet, which says nothing about the printing: the
		// wording answers, exactly as it did before an id could name one.
		Desc: "a finish name the game does not know falls through to the wording",
		In:   mtgmatcher.InputCard{ID: "647652", Name: "Ariel - Singing Mermaid", Edition: "Fabled", Variation: "15/204 Holofoil", Finish: "Reverse Holofoil", Foil: true},
	},
	{
		Desc: "product id two cards claim falls back to the name",
		In:   mtgmatcher.InputCard{ID: "544501", Name: "Let It Go (Disney Lorcana Challenge Top 128)", Edition: "Disney Lorcana Promo Cards", Variation: "2 Holofoil", Foil: true},
	},
	{
		Desc: "foil-only promo listed without the flag",
		In:   mtgmatcher.InputCard{Name: "A Whole New World", Variation: "010B", Edition: "Disney Lorcana Promo Cards"},
	},
	{
		Desc: "the letter the data gives a sibling still selects it",
		In:   mtgmatcher.InputCard{Name: "Dalmatian Puppy - Tail Wagger", Variation: "4c"},
	},
	{
		// The truncated name and the missing flag at once: an enchanted
		// printing is sold foil only, so requiring a plain finish deleted
		// every candidate before the name could be adopted.
		Desc: "truncated name reaches a foil-only card listed without the flag",
		In:   mtgmatcher.InputCard{Name: "Hades - King of", Variation: "205", Edition: "The First Chapter"},
	},
	{
		// "Bolt" prefixes four distinct names, one of which also has two
		// foil-only printings: the tier set aside must not break the tie
		// the first tier could not.
		Desc: "negative: a foil-only sibling does not settle an ambiguous prefix",
		In:   mtgmatcher.InputCard{Name: "Bolt"},
	},
	{
		Desc: "name typeset with a hyphen character instead of a dash",
		In:   mtgmatcher.InputCard{Name: "Fix‐It Felix, Jr. - Delighted Sightseer", Variation: "17", Edition: "Shimmering Skies"},
	},
	// Case-variant spellings: pairs of distinct cards whose names differ only
	// in letter case, reachable from either spelling via the shared name hash.
	{
		Desc: "case variant: enchanted spelling with its own number",
		In:   mtgmatcher.InputCard{Name: "Cruella De Vil - Miserable As Usual", Variation: "4", Foil: true},
	},
	{
		Desc: "case variant: base spelling with its own number",
		In:   mtgmatcher.InputCard{Name: "Cruella De Vil - Miserable as Usual", Variation: "72"},
	},
	{
		Desc: "case variant: base spelling reaches the enchanted printing",
		In:   mtgmatcher.InputCard{Name: "Cruella De Vil - Miserable as Usual", Variation: "4", Foil: true},
	},
	{
		Desc: "case variant: song spelling with its own number",
		In:   mtgmatcher.InputCard{Name: "The Islands I Pulled from the Sea", Variation: "130"},
	},
	// Core Match splits parenthesized chunks off the name and appends them to
	// Variation; the number filter must keep working on decorated listings.
	{
		Desc: "parenthetical suffix does not poison the collector number",
		In:   mtgmatcher.InputCard{Name: "Hades - King of Olympus (Enchanted)", Variation: "205", Foil: true},
	},
	// Truncated names resolve through the prefix fallback plus the number.
	{
		Desc: "prefix name resolves via collector number",
		In:   mtgmatcher.InputCard{Name: "Stitch", Variation: "22"},
	},
	{
		Desc: "prefix name resolves among lettered variants",
		In:   mtgmatcher.InputCard{Name: "Dalmatian Puppy", Variation: "4c"},
	},
	// The one genuinely 0-numbered card, plus zero-padding tolerance.
	{
		Desc: "zero collector number is a real number",
		In:   mtgmatcher.InputCard{Name: "Bruno Madrigal - Undetected Uncle", Variation: "0/204"},
	},
	{
		Desc: "zero-padded collector number",
		In:   mtgmatcher.InputCard{Name: "99 Puppies", Variation: "024"},
	},
	{
		// Stripping the leading zeros must not take the digit with them
		// when a letter follows it.
		Desc: "zero collector number carrying a storefront letter",
		In:   mtgmatcher.InputCard{Name: "Bruno Madrigal - Undetected Uncle", Variation: "000B"},
	},
	{
		Desc: "negative: the restored zero is a number, not a wildcard",
		In:   mtgmatcher.InputCard{Name: "A Whole New World", Variation: "000B"},
	},
	// Error contract.
	{
		Desc: "negative: unknown card name",
		In:   mtgmatcher.InputCard{Name: "Nonexistent Imaginary Hero", Variation: "1"},
	},
	{
		Desc: "negative: known name with wrong collector number",
		In:   mtgmatcher.InputCard{Name: "Ariel - On Human Legs", Variation: "99999"},
	},
	{
		// A feed that never says "foil" is not claiming nonfoil, so a
		// foil-only printing still answers it; the opposite direction
		// below is a claim, and still refused.
		Desc: "foil-only printing listed without the flag",
		In:   mtgmatcher.InputCard{Name: "Hades - King of Olympus", Variation: "205"},
	},
	{
		Desc: "negative: nonfoil-only printing requested as foil",
		In:   mtgmatcher.InputCard{Name: "Anna - Ensnared Sister", Variation: "1", Foil: true},
	},
	{
		// The legacy single-uuid shortcut returned this card ignoring the
		// number; the unified pipeline deliberately validates it.
		Desc: "negative: single-printing name with wrong collector number",
		In:   mtgmatcher.InputCard{Name: "Anna - Ensnared Sister", Variation: "77777"},
	},
	{
		Desc: "negative: wrong number zero does not disable the filter",
		In:   mtgmatcher.InputCard{Name: "99 Puppies", Variation: "0"},
	},
	{
		Desc: "negative: same name and number across sets aliases",
		In:   mtgmatcher.InputCard{Name: "Let It Go", Variation: "163"},
	},
	// Edition narrowing: the Match skeleton restricts candidates to the sets
	// matching a supplied edition; without one (or with one that resolves to
	// no set name) the number-driven contract above is unchanged.
	{
		Desc: "edition disambiguates a name and number shared across sets",
		In:   mtgmatcher.InputCard{Name: "Let It Go", Variation: "163", Edition: "The First Chapter"},
	},
	{
		Desc: "edition disambiguates toward the later printing",
		In:   mtgmatcher.InputCard{Name: "Let It Go", Variation: "163", Edition: "Winterspell"},
	},
	{
		Desc: "storefront-prefixed edition still narrows",
		In:   mtgmatcher.InputCard{Name: "Let It Go", Variation: "163", Edition: "Disney Lorcana: Winterspell"},
	},
	{
		Desc: "unrecognized edition does not block a unique number",
		In:   mtgmatcher.InputCard{Name: "99 Puppies", Variation: "024", Edition: "Not A Real Set"},
	},
	{
		Desc: "negative: edition excludes the printing carrying the number",
		In:   mtgmatcher.InputCard{Name: "Let It Go", Variation: "10", Edition: "Winterspell"},
	},
	{
		Desc: "negative: prefix without a number stays ambiguous",
		In:   mtgmatcher.InputCard{Name: "Dalmatian Puppy"},
	},
	// Non-card products TCGplayer files under its "Cards" type, verbatim from
	// the live catalog. The lot spellings carry a real card's name or a real
	// card's promotion, so the last case pins that neither is dragged down
	// with them.
	{
		Desc: "negative: puzzle insert piece is not a card",
		In:   mtgmatcher.InputCard{Name: "Azurite Sea Puzzle Insert (Top Left)", Variation: "Normal", Edition: "Azurite Sea"},
	},
	{
		Desc: "negative: puzzle insert lot named after a card is not a card",
		In:   mtgmatcher.InputCard{Name: "Mickey Mouse - Brave Little Tailor Puzzle Insert (Set of 4)", Variation: "Normal", Edition: "The First Chapter"},
	},
	{
		Desc: "negative: multi-card promo lot is not a card",
		In:   mtgmatcher.InputCard{Name: "Disney Cruise Promos (Set of 5)", Variation: "Normal", Edition: "Disney Lorcana Promo Cards"},
	},
	{
		Desc: "the promotion behind the lot still names a real card",
		In:   mtgmatcher.InputCard{Name: "Mickey Mouse - True Friend (Disney Cruise Promo)", Variation: "10 Holofoil", Edition: "Disney Lorcana Promo Cards", Foil: true},
	},
}

func TestLorcanaMatch(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set; skipping Lorcana matcher suite")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(lorcanaTestData)
	if err != nil {
		t.Fatal(err)
	}
	var tests []matchTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}

	if *updateLorcana {
		regenerateLorcanaTestData(t, b, tests)
		return
	}

	if len(tests) == 0 {
		t.Fatal("no Lorcana test cases")
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			t.Parallel()
			in := tt.In
			id, err := b.Match(&in)
			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if id != tt.ID || gotErr != tt.Err {
				t.Errorf("Match(%q num=%q foil=%v) = (%q, %q), want (%q, %q)",
					tt.In.Name, tt.In.Variation, tt.In.Foil, id, gotErr, tt.ID, tt.Err)
			}
		})
	}
}

// regenerateLorcanaTestData re-runs Match over every committed input plus the
// hand-authored seeds, bakes the resulting uuid/error, and rewrites the golden
// file sorted by description for a stable diff. The input set is curated (it
// is not derived from Match), so this only refreshes expectations after a
// deliberate logic change. A committed case may change verdict detail (a
// different uuid or message) — each such change is logged — but flipping
// between success and error aborts the rewrite: acknowledging a behavior
// change of that magnitude requires editing the entry or seed by hand.
func regenerateLorcanaTestData(t *testing.T, b *mtgmatcher.Backend, tests []matchTest) {
	// Seeds are authoritative for their inputs: drop any committed entry
	// sharing a seed description, then re-add the seeds, so editing a seed
	// takes effect instead of being masked by a stale committed entry.
	isSeed := map[string]bool{}
	for _, seed := range lorcanaSeeds {
		isSeed[seed.Desc] = true
	}
	kept := tests[:0]
	for _, tt := range tests {
		if !isSeed[tt.Desc] {
			kept = append(kept, tt)
		}
	}
	tests = append(kept, lorcanaSeeds...)

	for i := range tests {
		in := tests[i].In
		id, err := b.Match(&in)
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}

		wantNegative := tests[i].Err != ""
		if isSeed[tests[i].Desc] {
			wantNegative = strings.HasPrefix(tests[i].Desc, "negative:")
		}
		if wantNegative != (gotErr != "") {
			t.Errorf("refusing to flip %q: (%q, %q) -> (%q, %q); edit the entry or seed by hand",
				tests[i].Desc, tests[i].ID, tests[i].Err, id, gotErr)
			continue
		}

		if tests[i].ID != id || tests[i].Err != gotErr {
			t.Logf("updating %q: (%q, %q) -> (%q, %q)",
				tests[i].Desc, tests[i].ID, tests[i].Err, id, gotErr)
		}
		tests[i].ID = id
		tests[i].Err = gotErr
	}
	if t.Failed() {
		t.Fatal("verdict-class flips detected; golden file left untouched")
	}

	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Desc < tests[j].Desc
	})

	out, err := os.Create(lorcanaTestData)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	// The committed file spells "&" as itself, so the ampersand in two
	// card names does not churn into an escape on every regeneration.
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	if err := enc.Encode(tests); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d Lorcana test cases to %s", len(tests), lorcanaTestData)
}

// TestLorcanaFinishNames checks the invariants of the per-uuid Finish field:
// foil entries carry their exported foil-type name, nonfoil entries carry
// "nonfoil", and a sub-type suffixed uuid agrees with the name that derived
// it.
func TestLorcanaFinishNames(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set; skipping Lorcana matcher suite")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}

	subTypes := map[string]int{}
	for uuid, co := range b.UUIDs {
		// Sealed products live outside the finish machinery: no finish,
		// no finish-suffixed uuid, nothing to derive
		if co.Sealed {
			continue
		}
		if co.Foil {
			if co.Finish == "" || co.Finish == mtgmatcher.FinishNonfoil {
				t.Errorf("%s: foil entry with finish %q", uuid, co.Finish)
			}
		} else if co.Finish != mtgmatcher.FinishNonfoil {
			t.Errorf("%s: nonfoil entry with finish %q", uuid, co.Finish)
		}
		// A sub-type uuid is suffixed with the finish's canonical name, so
		// the entry's own finish spells the suffix outright.
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 && uuid[idx:] != suffixFoil {
			if co.Finish != uuid[idx+1:] {
				t.Errorf("%s: finish %q does not spell suffix %q", uuid, co.Finish, uuid[idx+1:])
			}
			subTypes[co.Finish]++
		}
	}
	if len(subTypes) == 0 {
		t.Fatal("no foil sub-type entries found in the datastore")
	}
	t.Logf("sub-type finishes: %v", subTypes)
}
