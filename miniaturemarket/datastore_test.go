package miniaturemarket

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// withGameDatastore reads a game's datastore for the test, from the variable
// naming it. The test is skipped where the run carries no such file.
func withGameDatastore(t *testing.T, game, env string) *mtgmatcher.Backend {
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
