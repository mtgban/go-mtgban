package riftbound

import (
	"encoding/json"
	"flag"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// matchTest mirrors the Lorcana matcher harness: an input card and the
// expected Match() verdict (uuid or error string). The hand-authored seeds
// below pin the contract edges: number affixes (letters, stars, prefixes),
// parenthetical and dashed real names, edition narrowing, and error paths.
// Run with -update-riftbound to re-bake expected verdicts after a deliberate
// change; the regeneration refuses to flip a case between success and error,
// so a regression cannot be blessed silently.
type matchTest struct {
	Desc string               `json:"description"`
	In   mtgmatcher.InputCard `json:"input"`
	Id   string               `json:"uuid,omitempty"`
	Err  string               `json:"error,omitempty"`
}

const riftboundTestData = "testdata/riftbound_test_data.json"

var updateRiftbound = flag.Bool("update-riftbound", false,
	"re-run Match over every test input and rewrite "+riftboundTestData)

// riftboundSeeds are hand-authored cases ensured present by -update-riftbound.
// Expected verdicts are baked by the regeneration rather than hard-coded, so
// they always reflect what Match actually returns; a "negative:" description
// prefix declares the author's intent and the regeneration fails loudly if
// the outcome class does not match it. Concrete cards referenced here exist
// in the real datastore (checked against the card gallery at authoring time).
var riftboundSeeds = []matchTest{
	{
		Desc: "name and number resolve the base printing",
		In:   mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "66"},
	},
	{
		Desc: "zero-padded letter variant has its own number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "066a"},
	},
	{
		Desc: "foil resolves to its own uuid",
		In:   mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "66", Foil: true},
	},
	{
		Desc: "full public code works as a number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "OGN-066/298"},
	},
	{
		Desc: "starred variant number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Inquisitive", Variation: "227*"},
	},
	{
		Desc: "special prefixed number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Inquisitive", Variation: "SP3"},
	},
	{
		Desc: "token number with zero-padded letter prefix",
		In:   mtgmatcher.InputCard{Name: "Gold", Variation: "T05"},
	},
	{
		Desc: "parenthetical is part of the real name",
		In:   mtgmatcher.InputCard{Name: "Recruit (NX)", Variation: "272"},
	},
	{
		Desc: "parenthetical name resolves without a number",
		In:   mtgmatcher.InputCard{Name: "Recruit (DE)"},
	},
	{
		Desc: "dashed name stays intact",
		In:   mtgmatcher.InputCard{Name: "Dark Child - Starter"},
	},
	// Legends are exported title-only; storefronts prepend the champion.
	{
		Desc: "champion-prefixed legend resolves to its title",
		In:   mtgmatcher.InputCard{Name: "Kai'Sa - Daughter of the Void", Variation: "247"},
	},
	{
		Desc: "champion-prefixed signature legend",
		In:   mtgmatcher.InputCard{Name: "Ahri - Nine-Tailed Fox (Signature)", Variation: "303*"},
	},
	{
		Desc: "champion-prefixed overnumbered legend",
		In:   mtgmatcher.InputCard{Name: "Kai'Sa - Daughter of the Void", Variation: "299"},
	},
	{
		Desc: "shortened gallery champion resolves",
		In:   mtgmatcher.InputCard{Name: "Master Yi - Meditative", Variation: "4"},
	},
	{
		Desc: "starter legend resolves through its storefront name",
		In:   mtgmatcher.InputCard{Name: "Lux - Lady of Luminosity (Starter)", Variation: "21"},
	},
	// Edition narrowing: the Match skeleton restricts candidates to the sets
	// matching a supplied edition; without one (or with one that resolves to
	// no set name) the number-driven contract is unchanged.
	{
		Desc: "edition narrows a name printed in two sets",
		In:   mtgmatcher.InputCard{Name: "Recruit (NX)", Edition: "Vendetta"},
	},
	{
		Desc: "storefront-prefixed edition still narrows",
		In:   mtgmatcher.InputCard{Name: "Ahri, Inquisitive", Variation: "227", Edition: "Riftbound: Spiritforged"},
	},
	// Promotional printings live in their own promo-typed sets and only
	// match when the edition names one; every other input keeps resolving
	// to the main printings, storefront promo name shapes included.
	{
		Desc: "promo qualifier name resolves inside its promo set",
		In:   mtgmatcher.InputCard{Name: "Sett - The Boss (Metal) (Best Of)", Variation: "269", Edition: "Riftbound Organized Play Promotional Cards"},
	},
	{
		Desc: "promo edition disambiguates a number shared across promo sets",
		In:   mtgmatcher.InputCard{Name: "Jinx - Rebel", Variation: "202", Edition: "Riftbound Promotional Cards"},
	},
	{
		Desc: "same promo number under the other promo set",
		In:   mtgmatcher.InputCard{Name: "Jinx - Rebel", Variation: "202", Edition: "Riftbound Organized Play Promotional Cards"},
	},
	{
		Desc: "promo-shaped legend name still reaches the main set",
		In:   mtgmatcher.InputCard{Name: "Teemo - Swift Scout", Variation: "263", Edition: "Origins"},
	},
	{
		Desc: "promos never match without their edition",
		In:   mtgmatcher.InputCard{Name: "Teemo - Swift Scout", Variation: "263"},
	},
	{
		Desc: "base printing wins over its promos by default",
		In:   mtgmatcher.InputCard{Name: "Viktor - Leader", Variation: "246"},
	},
	// The datastore builder stamps every printing with its TCGplayer
	// product id (652842 is Ahri, Alluring OGN-066).
	{
		Desc: "tcgplayer product id resolves through the identifier index",
		In:   mtgmatcher.InputCard{Id: "652842", Foil: true},
	},
	// Error contract.
	{
		Desc: "negative: unknown card name",
		In:   mtgmatcher.InputCard{Name: "Nonexistent Imaginary Champion", Variation: "1"},
	},
	{
		Desc: "negative: known name with wrong collector number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Alluring", Variation: "99999"},
	},
	{
		Desc: "negative: name printed in two sets stays ambiguous without a number",
		In:   mtgmatcher.InputCard{Name: "Recruit (NX)"},
	},
	{
		Desc: "negative: edition excludes the printing carrying the number",
		In:   mtgmatcher.InputCard{Name: "Ahri, Inquisitive", Variation: "227", Edition: "Vendetta"},
	},
}

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("RIFTBOUND_PATH")
	if path == "" {
		t.Skip("RIFTBOUND_PATH not set; skipping Riftbound matcher suite")
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
	return b
}

func TestRiftboundMatch(t *testing.T) {
	b := loadBackend(t)

	data, err := os.ReadFile(riftboundTestData)
	if err != nil {
		if *updateRiftbound && os.IsNotExist(err) {
			data = []byte("[]")
		} else {
			t.Fatal(err)
		}
	}
	var tests []matchTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}

	if *updateRiftbound {
		regenerateRiftboundTestData(t, b, tests)
		return
	}

	if len(tests) == 0 {
		t.Fatal("no Riftbound test cases")
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

// regenerateRiftboundTestData re-runs Match over every committed input plus
// the hand-authored seeds, bakes the resulting uuid/error, and rewrites the
// golden file sorted by description for a stable diff. A committed case may
// change verdict detail — each such change is logged — but flipping between
// success and error aborts the rewrite: acknowledging a behavior change of
// that magnitude requires editing the entry or seed by hand.
func regenerateRiftboundTestData(t *testing.T, b *mtgmatcher.Backend, tests []matchTest) {
	// Seeds are authoritative for their inputs: drop any committed entry
	// sharing a seed description, then re-add the seeds, so editing a seed
	// takes effect instead of being masked by a stale committed entry.
	isSeed := map[string]bool{}
	for _, seed := range riftboundSeeds {
		isSeed[seed.Desc] = true
	}
	kept := tests[:0]
	for _, tt := range tests {
		if !isSeed[tt.Desc] {
			kept = append(kept, tt)
		}
	}
	tests = append(kept, riftboundSeeds...)

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

	out, err := os.Create(riftboundTestData)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "    ")
	if err := enc.Encode(tests); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d Riftbound test cases to %s", len(tests), riftboundTestData)
}

// TestRiftboundIdentifiers checks the TCGplayer identifier index the
// datastore builder stamps into the datastore: the external map and the
// per-card identifier round-trip to each other, and the id lookup path
// resolves finishes.
func TestRiftboundIdentifiers(t *testing.T) {
	b := loadBackend(t)

	var n int
	for uuid, co := range b.UUIDs {
		pid := co.Identifiers["tcgplayerProductId"]
		if pid == "" {
			continue
		}
		if strings.HasSuffix(uuid, "_"+mtgmatcher.FinishFoil) {
			continue
		}
		n++
		if got := b.ExternalIdentifiers[pid]; got != uuid {
			t.Errorf("%s: external identifier %s resolves to %q", uuid, pid, got)
			continue
		}
		if id, err := b.MatchId(pid, true); err != nil || id != co.FoilUUIDs[mtgmatcher.FinishFoil] {
			t.Errorf("%s: MatchId(%s, foil) = (%q, %v)", uuid, pid, id, err)
		}
	}
	if n == 0 {
		t.Fatal("no tcgplayer identifiers loaded; rebuild the datastore with github.com/mtgban/riftbound-datastore")
	}
	if len(b.ExternalIdentifiers) != n {
		t.Errorf("external identifiers (%d) do not mirror the identified cards (%d)",
			len(b.ExternalIdentifiers), n)
	}
	t.Logf("%d cards carry a tcgplayer product id", n)
}

// TestRiftboundFinishUUIDs checks the datastore invariants of the two-finish
// scheme: every uuid spells its finish out, each nonfoil has a foil sibling,
// both carry the right Finish and Foil markers, and the FoilUUIDs map
// round-trips to them.
func TestRiftboundFinishUUIDs(t *testing.T) {
	b := loadBackend(t)

	var bases int
	for uuid, co := range b.UUIDs {
		if strings.HasSuffix(uuid, "_"+mtgmatcher.FinishFoil) {
			if !co.Foil || co.Finish != mtgmatcher.FinishFoil {
				t.Errorf("%s: foil entry with foil=%v finish=%q", uuid, co.Foil, co.Finish)
			}
			continue
		}
		if !strings.HasSuffix(uuid, "_"+mtgmatcher.FinishNonfoil) {
			t.Errorf("%s: uuid does not spell out its finish", uuid)
			continue
		}
		bases++
		if co.Foil || co.Finish != mtgmatcher.FinishNonfoil {
			t.Errorf("%s: nonfoil entry with foil=%v finish=%q", uuid, co.Foil, co.Finish)
		}
		sibling := strings.TrimSuffix(uuid, "_"+mtgmatcher.FinishNonfoil) + "_" + mtgmatcher.FinishFoil
		foil, found := b.UUIDs[sibling]
		if !found {
			t.Errorf("%s: missing foil sibling", uuid)
			continue
		}
		if co.FoilUUIDs[mtgmatcher.FinishNonfoil] != uuid ||
			co.FoilUUIDs[mtgmatcher.FinishFoil] != foil.UUID {
			t.Errorf("%s: FoilUUIDs do not round-trip: %v", uuid, co.FoilUUIDs)
		}
	}
	if bases == 0 {
		t.Fatal("no cards loaded")
	}
	t.Logf("%d cards, %d uuids", bases, len(b.UUIDs))
}
