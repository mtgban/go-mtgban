package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// readGameDatastore reads a game's datastore from the variable naming it,
// for a test to ask directly, or skips the test where the run carries none:
// this storefront's tests cover five games, and each runs under the job
// holding its own game's file.
func readGameDatastore(t *testing.T, game, env string) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	b, err := datastore.Read(game, path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// withGameDatastore installs a game's datastore for the test and puts back
// whatever stood before once it ends.
func withGameDatastore(t *testing.T, game, env string) {
	t.Helper()
	b := readGameDatastore(t, game, env)
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(b)
	t.Cleanup(func() {
		mtgmatcher.SetGlobalDatastore(previous)
	})
}
