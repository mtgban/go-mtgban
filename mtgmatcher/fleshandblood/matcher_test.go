package fleshandblood

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var updateFleshandblood = flag.Bool("update-fleshandblood", false, "regenerate the Flesh and Blood golden test data")

const fleshandbloodTestData = "testdata/fleshandblood_test_data.json"

type matchTest struct {
	Desc string               `json:"description"`
	In   mtgmatcher.InputCard `json:"input"`
	Id   string               `json:"uuid"`
	Err  string               `json:"error,omitempty"`
}

// The seeds cover the storefront shapes the scrapers feed: the canonical
// pitch-color parenthetical beside a full collector number (and its dashed
// and truncated spellings), the treatment and print-run wording that routes
// to its printing's entry without ever gating, cardtrader's marvel letter
// tail and print-run edition suffixes, the fused-card pair numbers, and the
// identifier index the TCGplayer feed resolves through.
var fleshandbloodSeeds = []matchTest{
	{
		Desc: "canonical pitch name with the full collector number",
		In:   mtgmatcher.InputCard{Name: "Sink Below (Red)", Variation: "WTR215"},
	},
	{
		Desc: "dashed pitch color normalizes onto the parenthetical name",
		In:   mtgmatcher.InputCard{Name: "Sink Below - Red", Variation: "WTR215"},
	},
	{
		Desc: "bare truncated name adopts the pitch parenthetical carrying the number",
		In:   mtgmatcher.InputCard{Name: "Breakneck Battery", Variation: "WTR011"},
	},
	{
		Desc: "rainbow wording routes to its plainest print run",
		In:   mtgmatcher.InputCard{Name: "Sink Below (Red)", Variation: "WTR215 Rainbow Foil", Edition: "Flesh & Blood: Welcome to Rathe Singles"},
	},
	{
		Desc: "cold foil wording picks the cold entry over the rainbow default",
		In:   mtgmatcher.InputCard{Name: "Frost Hex", Variation: "UPR126 Cold Foil", Foil: true},
	},
	{
		Desc: "tcgplayer printing name routes both finish axes",
		In:   mtgmatcher.InputCard{Name: "Sink Below (Red)", Variation: "WTR215 1st Edition Rainbow Foil", Foil: true},
	},
	{
		Desc: "storefront wording picks the variant it describes",
		In:   mtgmatcher.InputCard{Name: "Enigma, New Moon (Marvel)"},
	},
	{
		Desc: "cardtrader marvel letter tail demands the variant",
		In:   mtgmatcher.InputCard{Name: "Enigma, New Moon", Variation: "MST238m"},
	},
	{
		Desc: "bare number with the marvel tail still demands the variant",
		In:   mtgmatcher.InputCard{Name: "Enigma, New Moon", Variation: "238m"},
	},
	{
		Desc: "letter label beside the treatment wording picks its product",
		In:   mtgmatcher.InputCard{Name: "Lightning Flow (A)", Variation: "OMN203 Cold Foil", Foil: true},
	},
	{
		Desc: "treatment wording spelling a label defers to the named variant",
		In:   mtgmatcher.InputCard{Name: "Ash // Aether Ashwing (Marvel)", Variation: "UPR043 Cold Foil", Foil: true},
	},
	{
		Desc: "pair collector number resolves the fused card",
		In:   mtgmatcher.InputCard{Name: "Anothos // Bravo", Variation: "WTR040 // WTR039"},
	},
	{
		Desc: "front half of a pair number still matches",
		In:   mtgmatcher.InputCard{Name: "Anothos // Bravo", Variation: "WTR040"},
	},
	{
		Desc: "cardtrader print-run suffix strips off the edition",
		In:   mtgmatcher.InputCard{Name: "Barraging Beatdown (Yellow)", Variation: "18", Edition: "Welcome to Rathe - 1st Edition"},
	},
	{
		Desc: "alpha print run rides the 1st edition entry",
		In:   mtgmatcher.InputCard{Name: "Barraging Beatdown (Yellow)", Variation: "18", Edition: "Welcome to Rathe - Alpha Print Run"},
	},
	{
		Desc: "tcgplayer product id resolves through the identifier index",
		In:   mtgmatcher.InputCard{Id: "225309"},
	},
	{
		Desc: "the printing named beside the id prices that printing's entry",
		In:   mtgmatcher.InputCard{Id: "225015", Finish: "Unlimited Edition Normal"},
	},
	{
		Desc: "a bare treatment reaches the print run the product sold it in",
		In:   mtgmatcher.InputCard{Id: "225015", Finish: "Cold Foil"},
	},
	{
		Desc: "negative: a treatment the product was never priced in refuses the sibling",
		In:   mtgmatcher.InputCard{Id: "225501", Name: "Fyendal's Spring Tunic", Variation: "FAB001", Finish: "Rainbow Foil"},
	},
	{
		Desc: "negative: unknown card name",
		In:   mtgmatcher.InputCard{Name: "Nonexistent Imaginary Hero", Variation: "WTR001"},
	},
	{
		Desc: "negative: known name with wrong collector number",
		In:   mtgmatcher.InputCard{Name: "Sink Below (Red)", Variation: "WTR999"},
	},
	{
		Desc: "negative: a name reprinted across sets stays ambiguous unaided",
		In:   mtgmatcher.InputCard{Name: "Brutal Assault"},
	},
	{
		Desc: "negative: pitch variants without a base alias on a bare number",
		In:   mtgmatcher.InputCard{Name: "Dig In", Variation: "FAB384"},
	},
	// The only extended art of OMN133 wears the words behind a colour, so
	// the storefront naming the treatment names the second tag and nothing
	// else.
	{
		Desc: "a tag written behind another still names its printing",
		In:   mtgmatcher.InputCard{Name: "Tome of Quandaries", Variation: "OMN133 Extended Art"},
	},
}

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("FLESHANDBLOOD_PATH")
	if path == "" {
		t.Skip("FLESHANDBLOOD_PATH not set; skipping Flesh and Blood matcher suite")
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

func TestFleshandbloodMatch(t *testing.T) {
	b := loadBackend(t)

	data, err := os.ReadFile(fleshandbloodTestData)
	if err != nil {
		if *updateFleshandblood && os.IsNotExist(err) {
			data = []byte("[]")
		} else {
			t.Fatal(err)
		}
	}
	var tests []matchTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}

	if *updateFleshandblood {
		regenerateFleshandbloodTestData(t, b, tests)
		return
	}

	if len(tests) == 0 {
		t.Fatal("no Flesh and Blood test cases")
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
				t.Errorf("Match(%q num=%q) = (%q, %q), want (%q, %q)",
					tt.In.Name, tt.In.Variation, id, gotErr, tt.Id, tt.Err)
			}
		})
	}
}

// regenerateFleshandbloodTestData re-runs Match over every committed input
// plus the hand-authored seeds, bakes the resulting uuid/error, and
// rewrites the golden file sorted by description. Flipping a case between
// success and error aborts the rewrite unless its description owns the
// error with a "negative:" prefix: acknowledging a change of that magnitude
// requires editing the entry by hand.
func regenerateFleshandbloodTestData(t *testing.T, b *mtgmatcher.Backend, tests []matchTest) {
	isSeed := map[string]bool{}
	for _, seed := range fleshandbloodSeeds {
		isSeed[seed.Desc] = true
	}
	kept := tests[:0]
	for _, tt := range tests {
		if !isSeed[tt.Desc] {
			kept = append(kept, tt)
		}
	}
	tests = append(kept, fleshandbloodSeeds...)

	for i := range tests {
		in := tests[i].In
		id, err := b.Match(&in)
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}
		wasError := tests[i].Err != ""
		isError := gotErr != ""
		if (tests[i].Id != "" || tests[i].Err != "") && wasError != isError &&
			!strings.HasPrefix(tests[i].Desc, "negative:") {
			t.Fatalf("%s: flipped between success and error (%q/%q -> %q/%q); edit the entry by hand",
				tests[i].Desc, tests[i].Id, tests[i].Err, id, gotErr)
		}
		if tests[i].Id != id || tests[i].Err != gotErr {
			t.Logf("%s: (%q, %q) -> (%q, %q)", tests[i].Desc, tests[i].Id, tests[i].Err, id, gotErr)
		}
		tests[i].Id = id
		tests[i].Err = gotErr
	}

	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Desc < tests[j].Desc
	})
	// Encoded rather than marshalled so the ampersands in a set name stay
	// ampersands: MarshalIndent escapes them for HTML, and re-baking an
	// otherwise unchanged golden would rewrite those lines every time.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	err := enc.Encode(tests)
	data := bytes.TrimRight(buf.Bytes(), "\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fleshandbloodTestData, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("rewrote %s with %d cases", fleshandbloodTestData, len(tests))
}

// TestFleshandbloodSealed pins the sealed namespace: products load, resolve
// by name, and never shadow the card index.
func TestFleshandbloodSealed(t *testing.T) {
	b := loadBackend(t)

	if len(b.AllSealedUUIDs) == 0 {
		t.Fatal("no sealed products loaded")
	}
	productMap := b.BuildSealedProductMap("tcgplayerProductId")
	if len(productMap) == 0 {
		t.Fatal("no sealed product ids")
	}
	for id, uuids := range productMap {
		if len(uuids) != 1 {
			t.Errorf("tcgplayer id %d shared by %d sealed products", id, len(uuids))
		}
	}

	uuid, err := b.ResolveSealed("Welcome to Rathe Booster Pack [1st Edition]")
	if err != nil {
		t.Fatalf("bracket-edition booster pack did not resolve: %s", err)
	}
	co, err := b.GetUUID(uuid)
	if err != nil || !co.Sealed {
		t.Fatalf("resolved uuid %s is not a sealed product", uuid)
	}

	// The WTR booster pack is two real products (the 1st Edition and
	// Unlimited Edition print runs); a wording naming neither must stay
	// unresolved rather than pick one.
	if uuid, err := b.ResolveSealed("Welcome to Rathe Booster Pack"); err == nil {
		t.Fatalf("print-run-ambiguous booster pack resolved to %s", uuid)
	}
}
