package coolstuffinc

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// datastores holds each game's backend once a test has read it: 48 tests
// here ask for one, nine of them for AllPrintings, and a Backend is never
// written to once loaded.
var (
	datastoresMu sync.Mutex
	datastores   = map[mtgmatcher.Game]*mtgmatcher.Backend{}
)

// readGameDatastore reads a game's datastore from the variable naming it,
// for a test to ask directly, or skips the test where the run carries none:
// this storefront's tests cover several games, and each runs under the job
// holding its own game's file.
func readGameDatastore(t *testing.T, game mtgmatcher.Game, env string) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	datastoresMu.Lock()
	defer datastoresMu.Unlock()
	if b, ok := datastores[game]; ok {
		return b
	}
	b, err := datastore.Read(game, path)
	if err != nil {
		t.Fatal(err)
	}
	datastores[game] = b
	return b
}
