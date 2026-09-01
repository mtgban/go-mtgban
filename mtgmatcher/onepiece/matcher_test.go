package onepiece

import (
	"encoding/json"
	"flag"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var updateOnepiece = flag.Bool("update-onepiece", false, "regenerate the One Piece golden test data")

const onepieceTestData = "testdata/onepiece_test_data.json"

type matchTest struct {
	Desc string               `json:"description"`
	In   mtgmatcher.InputCard `json:"input"`
	ID   string               `json:"uuid"`
	Err  string               `json:"error,omitempty"`
}

// The seeds cover the storefront shapes the scrapers feed: cardtrader's
// clean name beside a full collector number (with the letter tail its
// alternate arts carry), cardmarket's "(V.n)" positional wording,
// coolstuffinc's bare-tail numbers with qualifier parentheticals, and the
// identifier index the TCGplayer feed resolves through.
var onepieceSeeds = []matchTest{
	{
		Desc: "clean name with the full collector number",
		In:   mtgmatcher.InputCard{Name: "Roronoa Zoro", Variation: "OP01-025"},
	},
	{
		Desc: "epithet parenthetical is part of the name",
		In:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham)", Variation: "OP01-084"},
	},
	{
		Desc: "storefront wording picks the variant it describes",
		In:   mtgmatcher.InputCard{Name: "Nami (OP01-016) (Manga)"},
	},
	{
		Desc: "cardmarket V.1 keeps the base printing",
		In:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham) (OP01-084) (V.1)"},
	},
	{
		Desc: "cardmarket V.2 demands the variant",
		In:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham) (OP01-084) (V.2)"},
	},
	{
		Desc: "cardtrader letter tail demands a variant",
		In:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham)", Variation: "OP01-084a"},
	},
	{
		Desc: "bare tail number narrows within the edition",
		In:   mtgmatcher.InputCard{Name: "Mr.2.Bon.Kurei (Bentham)", Variation: "084", Edition: "Romance Dawn"},
	},
	{
		Desc: "tcgplayer product id resolves through the identifier index",
		In:   mtgmatcher.InputCard{ID: "454615"},
	},
	{
		Desc: "coolstuffinc edition wears the set code before a dash",
		In:   mtgmatcher.InputCard{Name: "Brannew", Edition: "OP03 - Pillars of Strength", Variation: "OP03-089"},
	},
	{
		Desc: "dash-number promo name resolves through its qualifier",
		In:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy - P-043 (Convention Promo 2024)", Edition: "Promo"},
	},
	{
		Desc: "set-qualified number refuses another set's same tail",
		In:   mtgmatcher.InputCard{Name: "Ain", Variation: "OP07-002"},
	},
	{
		Desc: "promo line names the set's event printings",
		In:   mtgmatcher.InputCard{Name: "Adio (OP03-002)", Variation: "002", Edition: "Promos: Pillars of Strength"},
	},
	{
		Desc: "a promo marker naming no set keeps the regular printing",
		In:   mtgmatcher.InputCard{Name: "Dogura", Variation: "OP02-010", Edition: "Promos:"},
	},
	{
		Desc: "a promo marker trailing punctuation names no set either",
		In:   mtgmatcher.InputCard{Name: "Dogura", Variation: "OP02-010", Edition: "Promos: -"},
	},
	{
		Desc: "a promo marker trailing a possessive names no set either",
		In:   mtgmatcher.InputCard{Name: "Dogura", Variation: "OP02-010", Edition: "Promo: 's"},
	},
	{
		Desc: "the set itself keeps the regular printing",
		In:   mtgmatcher.InputCard{Name: "Adio (OP03-002)", Variation: "002", Edition: "Pillars of Strength"},
	},
	{
		Desc: "an edition naming neither set keeps the regular printing",
		In:   mtgmatcher.InputCard{Name: "Ain (OP07-002)", Variation: "002", Edition: "500 Years into the Future"},
	},
	{
		Desc: "the set code picks the volume its family shares a name with",
		In:   mtgmatcher.InputCard{Name: "Roronoa Zoro", Variation: "OP06-118", Edition: "PRB-02: Premium Booster"},
	},
	{
		Desc: "negative: a family name without its volume stays ambiguous",
		In:   mtgmatcher.InputCard{Name: "Roronoa Zoro", Variation: "OP06-118", Edition: "Premium Booster"},
	},
	{
		Desc: "the set code outranks a wording naming the other volume",
		In:   mtgmatcher.InputCard{Name: "Roronoa Zoro", Variation: "OP06-118", Edition: "PRB-02: Premium Booster -The Best-"},
	},
	{
		Desc: "an event set spelled a word off selects the event printing",
		In:   mtgmatcher.InputCard{Name: "Sterry", Variation: "OP07-006", Edition: "500 Years into the Future Pre-Release Cards"},
	},
	{
		Desc: "a code short of the wording does not outrank it",
		In:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "OP14 - The Azure Sea's Seven Release"},
	},
	{
		Desc: "the same code with no marker spelled keeps the base set",
		In:   mtgmatcher.InputCard{Name: "Bepo", Variation: "OP14-012", Edition: "OP14 - The Azure Sea's Seven"},
	},
	{
		Desc: "a code naming another set yields to a wording naming one",
		In:   mtgmatcher.InputCard{Name: "Kouzuki Oden", Variation: "EB01-001", Edition: "EB-02 - Extra Booster: Memorial Collection"},
	},
	{
		Desc: "a word appended behind the coded name keeps the coded set",
		In:   mtgmatcher.InputCard{Name: "Adio", Variation: "OP03-002", Edition: "OP03 - Pillars of Strength Cards"},
	},
	{
		Desc: "the same set reached by spelling its marker in order",
		In:   mtgmatcher.InputCard{Name: "Adio", Variation: "OP03-002", Edition: "OP03 - Pillars of Strength Pre-Release"},
	},
	{
		Desc: "a word appended past a family name keeps the coded volume",
		In:   mtgmatcher.InputCard{Name: "Monkey.D.Luffy", Variation: "OP01-024", Edition: "PRB-01 - Premium Booster -The Best- Promo"},
	},
	{
		Desc: "a family name cut short still resolves through its code",
		In:   mtgmatcher.InputCard{Name: "Lucky.Roux", Variation: "PRB02-003", Edition: "PRB-02: Premium Booster"},
	},
	{
		Desc: "an event set missing its deck ordinal still selects it",
		In:   mtgmatcher.InputCard{Name: "Usopp", Variation: "ST01-002", Edition: "Super Pre-Release Starter Deck: Straw Hat Crew"},
	},
	{
		Desc: "a wording spelling no event marker keeps the base set",
		In:   mtgmatcher.InputCard{Name: "Blast Breath", Variation: "ST04-016", Edition: "Animal Kingdom Pirates: Starter Deck 4"},
	},
	{
		Desc: "storefront drops the deck ordinal out of the set name",
		In:   mtgmatcher.InputCard{Name: "Blast Breath (ST04-016)", Variation: "016", Edition: "Starter Deck: Animal Kingdom Pirates"},
	},
	{
		Desc: "a set name spelled a word off still selects its set",
		In:   mtgmatcher.InputCard{Name: "Basil Hawkins (OP07-029)", Variation: "029", Edition: "500 Years into the Future"},
	},
	{
		Desc: "a promo line spelled a word off still selects the event set",
		In:   mtgmatcher.InputCard{Name: "Aladine (OP07-020)", Variation: "020", Edition: "Promos: 500 Years into the Future"},
	},
	{
		Desc: "bare number hung off the name after a dash",
		In:   mtgmatcher.InputCard{Name: `"Buddha" Sengoku - 077`, Variation: "OP16-077", Edition: "OP16 - The Time Of Battle"},
	},
	{
		Desc: "bare number hung off the name before a qualifier",
		In:   mtgmatcher.InputCard{Name: "Trafalgar Law - 047 (Parallel)", Edition: "OP01 - Romance Dawn"},
	},
	{
		Desc: "a qualifier carrying an ordinal does not become the number",
		In:   mtgmatcher.InputCard{Name: "Killer - 106 (Judge Pack Vol. 5)", Edition: "One Piece Promotion Cards"},
	},
	{
		Desc: "the ordinal a promo label opens with is not the number either",
		In:   mtgmatcher.InputCard{Name: "Sanji - 013 (English Version 1st Anniversary Set)", Edition: "One Piece Promotion Cards"},
	},
	{
		Desc: "negative: a vendor bucket edition selects no set",
		In:   mtgmatcher.InputCard{Name: "Arlong (OP06-023)", Variation: "023", Edition: "One Piece Products"},
	},
	{
		Desc: "negative: a foreign set code refuses another set's tail",
		In:   mtgmatcher.InputCard{Name: "Dogura", Variation: "ST05-010", Edition: "OP02 - Paramount War"},
	},
	{
		Desc: "negative: unknown card name",
		In:   mtgmatcher.InputCard{Name: "Nonexistent Imaginary Pirate", Variation: "OP01-001"},
	},
	{
		Desc: "negative: known name with wrong collector number",
		In:   mtgmatcher.InputCard{Name: "Roronoa Zoro", Variation: "OP01-999"},
	},
	{
		Desc: "negative: reprinted base printings stay ambiguous on a bare input",
		In:   mtgmatcher.InputCard{Name: "Nami", Variation: "OP01-016"},
	},
	{
		Desc: "the place a family differs by answers when the wording names it",
		In: mtgmatcher.InputCard{Name: "Capone\"Gang\"Bege",
			Variation: "OP04-100w Championship 2023 | Winner",
			Edition:   "Championships Promo"},
	},
	{
		Desc: "negative: two events awarding one place settle nothing",
		In: mtgmatcher.InputCard{Name: "Gum-Gum Jet Pistol",
			Variation: "ST01-015f Regional 2024 | Finalist",
			Edition:   "Store Tournaments Promos"},
	},
	{
		Desc: "a letter tail no label answers falls to the edition",
		In: mtgmatcher.InputCard{Name: "Four Thousand-Brick Fist",
			Variation: "OP05-020s 1st Anniversary Stamp",
			Edition:   "OP-05: 1st Anniversary Tournament Cards"},
	},
	{
		Desc: "the same, for a starter deck's pre-release printings",
		In: mtgmatcher.InputCard{Name: "Edward Weevil",
			Variation: "ST03-002p Super Pre-Release Stamp",
			Edition:   "ST-03: Starter Deck: The Seven Warlords of the Sea"},
	},
	{
		Desc: "corrected run named by the letter on the number",
		In: mtgmatcher.InputCard{Name: "Trafalgar Law",
			Variation: `OP01-047α Alpha Pre-Errata Card | "1 Character to your hand: Play 1 Character card"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "the art word picks the treated printing of that same run",
		In: mtgmatcher.InputCard{Name: "Trafalgar Law",
			Variation: `OP01-047αa Alternate Art | Alpha Pre-Errata Card | "1 Character to your hand: Play 1 Character card"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "a corrected printing is not the base card at its number",
		In: mtgmatcher.InputCard{Name: "Officer Agents",
			Variation: `OP01-087α Alpha Pre-Errata Card | "Play 1 {Baroque Works} type card"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "the letter tail does not say which art: the plain one carries it",
		In: mtgmatcher.InputCard{Name: "Dracule Mihawk",
			Variation: `OP01-070ae Pre-Errata Card | Missing Text "Up to"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "and the alternate art at that number carries none",
		In: mtgmatcher.InputCard{Name: "Dracule Mihawk",
			Variation: `OP01-070e Alternate Art | Pre-Errata Card | Missing Text "Up to"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "the alternate art the catalog files as a box topper",
		In: mtgmatcher.InputCard{Name: "Alvida",
			Variation: `OP01-064a Alternate Art | Pre-Errata Card | Missing text "Up to"`,
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "a treatment the wording spells out answers before the art word",
		In: mtgmatcher.InputCard{Name: "Monkey.D.Luffy",
			Variation: "OP01-003d Pre-Errata Card | Demo Deck Map Design",
			Edition:   "Pre-Errata Cards"},
	},
	{
		Desc: "the word's other sense keeps the base printing it names",
		In: mtgmatcher.InputCard{Name: "Marshall.D.Teach",
			Variation: `ST03-014 1st Print Errata Card | "Return 1 Character"`,
			Edition:   "ST03 - The Seven Warlords of The Sea"},
	},
}

// datastoreOnce loads the datastore the first time a test asks for it. The
// suite used to read and parse the file again on every call.
var datastoreOnce = sync.OnceValues(func() (*mtgmatcher.Backend, error) {
	path := os.Getenv("ONEPIECE_PATH")
	if path == "" {
		return nil, nil
	}
	f, err := datastore.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
})

func loadBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := datastoreOnce()
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Skip("ONEPIECE_PATH not set; skipping One Piece matcher suite")
	}
	return b
}

func TestOnepieceMatch(t *testing.T) {
	b := loadBackend(t)

	data, err := os.ReadFile(onepieceTestData)
	if err != nil {
		if *updateOnepiece && os.IsNotExist(err) {
			data = []byte("[]")
		} else {
			t.Fatal(err)
		}
	}
	var tests []matchTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}

	if *updateOnepiece {
		regenerateOnepieceTestData(t, b, tests)
		return
	}

	if len(tests) == 0 {
		t.Fatal("no One Piece test cases")
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
				t.Errorf("Match(%q num=%q) = (%q, %q), want (%q, %q)",
					tt.In.Name, tt.In.Variation, id, gotErr, tt.ID, tt.Err)
			}
		})
	}
}

// regenerateOnepieceTestData re-runs Match over every committed input plus
// the hand-authored seeds, bakes the resulting uuid/error, and rewrites the
// golden file sorted by description. Flipping a case between success and
// error aborts the rewrite unless its description owns the error with a
// "negative:" prefix: acknowledging a change of that magnitude requires
// editing the entry by hand.
func regenerateOnepieceTestData(t *testing.T, b *mtgmatcher.Backend, tests []matchTest) {
	isSeed := map[string]bool{}
	for _, seed := range onepieceSeeds {
		isSeed[seed.Desc] = true
	}
	kept := tests[:0]
	for _, tt := range tests {
		if !isSeed[tt.Desc] {
			kept = append(kept, tt)
		}
	}
	tests = append(kept, onepieceSeeds...)

	for i := range tests {
		in := tests[i].In
		id, err := b.Match(&in)
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}
		wasError := tests[i].Err != ""
		isError := gotErr != ""
		if (tests[i].ID != "" || tests[i].Err != "") && wasError != isError &&
			!strings.HasPrefix(tests[i].Desc, "negative:") {
			t.Fatalf("%s: flipped between success and error (%q/%q -> %q/%q); edit the entry by hand",
				tests[i].Desc, tests[i].ID, tests[i].Err, id, gotErr)
		}
		if tests[i].ID != id || tests[i].Err != gotErr {
			t.Logf("%s: (%q, %q) -> (%q, %q)", tests[i].Desc, tests[i].ID, tests[i].Err, id, gotErr)
		}
		tests[i].ID = id
		tests[i].Err = gotErr
	}

	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Desc < tests[j].Desc
	})
	data, err := json.MarshalIndent(tests, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(onepieceTestData, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("rewrote %s with %d cases", onepieceTestData, len(tests))
}

// TestOnepieceDashTailNames pins the datastore invariant the bare-tail
// rule rests on: no card is named with a dash and a number at the end, so
// a tail found there is always the collector number a storefront hung off
// the name. Two names do carry a full code of their own ("Monkey.D.Luffy
// - OP14-34"), which is why Prefilter leaves a name that is canonical as
// it stands alone, both before the parentheticals are split off and after.
func TestOnepieceDashTailNames(t *testing.T) {
	b := loadBackend(t)

	for _, name := range b.AllCanonicalNames {
		if dashTailRe.MatchString(name) {
			t.Errorf("card name ends in a dash number: %q", name)
		}
	}
}

// TestOnepieceSealed pins the sealed namespace: products load, resolve by
// name, and never shadow the card index.
func TestOnepieceSealed(t *testing.T) {
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

	uuid, err := b.ResolveSealed("One Piece Card Game Romance Dawn Booster Pack")
	if err != nil {
		t.Fatalf("booster pack did not resolve: %s", err)
	}
	co, err := b.GetUUID(uuid)
	if err != nil || !co.Sealed {
		t.Fatalf("resolved uuid %s is not a sealed product", uuid)
	}

	// The OP01 booster box is two real products (the Wave 1 - Blue and
	// Wave 2 - White print waves); a wording naming neither must stay
	// unresolved rather than pick one.
	if uuid, err := b.ResolveSealed("One Piece Card Game Romance Dawn Booster Box"); err == nil {
		t.Fatalf("wave-ambiguous booster box resolved to %s", uuid)
	}
}

// TestOnepieceNumbered pins the datastore invariant the DON!! rules rest
// on: the resource card is the one name in the game whose every printing is
// filed under a code rather than a number, so a storefront's number field
// can say nothing about which of them it means. A name with even one
// numbered printing is not that case, and Monkey.D.Luffy is the proof: a
// handful of his event printings wear "LEADER" while the rest wear real
// numbers, and those still have to be told apart by number.
func TestOnepieceNumbered(t *testing.T) {
	b := loadBackend(t)

	// CanonicalNames holds the card names alone; the qualified spellings
	// naming one printing apiece are searchable but never canonical, and a
	// qualified name reaching a "LEADER" printing has no number to compare
	// either.
	for _, name := range b.CanonicalNames {
		want := !strings.HasPrefix(name, "DON!! Card")
		if got := numbered(b, name); got != want {
			t.Errorf("numbered(%q) = %v, want %v", name, got, want)
		}
	}
}
