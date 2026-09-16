package mtgmatcher_test

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// withGame installs a game's datastore for one test, or skips.
func withGame(t *testing.T, game, env string) {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	b, err := datastore.Read(game, path)
	if err != nil {
		t.Fatal(err)
	}
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(b)
	t.Cleanup(func() { mtgmatcher.SetGlobalDatastore(previous) })
}

// The split is lossless: a finish is its print run and its treatment, in that
// order and nothing else. A game that welds the two into one name is the
// reason the field pair exists, so the halves have to put the name back
// together for every printing it sells.
func TestFinishSplitRejoins(t *testing.T) {
	for _, tt := range []struct{ game, env string }{
		{"fleshandblood", "FLESHANDBLOOD_PATH"},
		{"pokemon", "POKEMON_PATH"},
		{"yugioh", "YUGIOH_PATH"},
	} {
		t.Run(tt.game, func(t *testing.T) {
			withGame(t, tt.game, tt.env)

			var checked, broken int
			for _, uuid := range mtgmatcher.GetUUIDs() {
				co, err := mtgmatcher.GetUUID(uuid)
				if err != nil || co.Sealed || co.Finish == "" {
					continue
				}
				checked++
				if co.PrintRun+co.Treatment != co.Finish {
					broken++
					if broken < 4 {
						t.Errorf("%s: %q + %q is not %q",
							uuid, co.PrintRun, co.Treatment, co.Finish)
					}
				}
			}
			if checked == 0 {
				t.Fatal("the datastore loaded no printings")
			}
			if broken != 0 {
				t.Errorf("%d of %d printings do not rejoin", broken, checked)
			}
		})
	}
}

// And the split is what it is for: the names a game welds stand for fewer
// treatments than there are names, which is what makes a treatment worth
// asking about on its own. Flesh and Blood sells three treatments under eight
// names; Yu-Gi-Oh names only runs, so every treatment it has is empty.
func TestFinishSplitCollapsesTheNames(t *testing.T) {
	for _, tt := range []struct {
		game, env  string
		treatments []string
		runs       []string
	}{
		{
			game: "fleshandblood", env: "FLESHANDBLOOD_PATH",
			treatments: []string{"coldfoil", "normal", "rainbowfoil"},
			runs:       []string{"", "1stedition", "unlimitededition"},
		},
		{
			game: "yugioh", env: "YUGIOH_PATH",
			treatments: []string{""},
			runs:       []string{"1stedition", "limited", "unlimited"},
		},
	} {
		t.Run(tt.game, func(t *testing.T) {
			withGame(t, tt.game, tt.env)

			treatments := map[string]bool{}
			runs := map[string]bool{}
			for _, uuid := range mtgmatcher.GetUUIDs() {
				co, err := mtgmatcher.GetUUID(uuid)
				if err != nil || co.Sealed || co.Finish == "" {
					continue
				}
				treatments[co.Treatment] = true
				runs[co.PrintRun] = true
			}
			if got := sorted(treatments); !equal(got, tt.treatments) {
				t.Errorf("treatments = %v, want %v", got, tt.treatments)
			}
			if got := sorted(runs); !equal(got, tt.runs) {
				t.Errorf("print runs = %v, want %v", got, tt.runs)
			}
		})
	}
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func equal(got, want []string) bool {
	return strings.Join(got, "|") == strings.Join(want, "|")
}
