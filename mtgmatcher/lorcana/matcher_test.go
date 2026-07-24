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
// Match() verdict (uuid or error string). Positive cases were seeded from the
// agreement between the legacy SimpleSearch and the unified Match, so the
// suite guards that Match keeps resolving Lorcana cards the way the old
// per-game path did. The hand-authored seeds below pin the contract edges the
// sampled corpus cannot reach: error paths, case-variant spellings,
// parenthetical suffixes, prefix names, and zero-numbered cards. Run with
// -update-lorcana to re-bake expected verdicts after a deliberate change;
// the regeneration refuses to flip a case between success and error, so a
// regression cannot be blessed silently.
type matchTest struct {
	Desc string               `json:"description"`
	In   mtgmatcher.InputCard `json:"input"`
	Id   string               `json:"uuid,omitempty"`
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
		Desc: "negative: foil-only printing requested as nonfoil",
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
	{
		Desc: "negative: prefix without a number stays ambiguous",
		In:   mtgmatcher.InputCard{Name: "Dalmatian Puppy"},
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
		tt := tt
		t.Run(tt.Desc, func(t *testing.T) {
			t.Parallel()
			in := tt.In
			id, err := b.Match(&in)
			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if id != tt.Id || gotErr != tt.Err {
				t.Errorf("Match(%q num=%q foil=%v) = (%q, %q), want (%q, %q)",
					tt.In.Name, tt.In.Variation, tt.In.Foil, id, gotErr, tt.Id, tt.Err)
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
				tests[i].Desc, tests[i].Id, tests[i].Err, id, gotErr)
			continue
		}

		if tests[i].Id != id || tests[i].Err != gotErr {
			t.Logf("updating %q: (%q, %q) -> (%q, %q)",
				tests[i].Desc, tests[i].Id, tests[i].Err, id, gotErr)
		}
		tests[i].Id = id
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
	enc.SetIndent("", "    ")
	if err := enc.Encode(tests); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d Lorcana test cases to %s", len(tests), lorcanaTestData)
}
