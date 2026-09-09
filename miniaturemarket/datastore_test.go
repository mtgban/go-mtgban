package miniaturemarket

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// withGameDatastore installs a game's datastore for the test, read from the
// variable naming it, and puts back whatever stood before once it ends. The
// test is skipped where the run carries no such file.
func withGameDatastore(t *testing.T, game, env string) {
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
	t.Cleanup(func() {
		mtgmatcher.SetGlobalDatastore(previous)
	})
}
