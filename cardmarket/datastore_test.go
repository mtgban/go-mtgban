package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// installBackend installs a backend as the global datastore for the test
// and puts back the one that stood before, so a test's handful of rows is
// not what the next test matches against.
func installBackend(t *testing.T, b *mtgmatcher.Backend) {
	t.Helper()
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(b)
	t.Cleanup(func() {
		mtgmatcher.SetGlobalDatastore(previous)
	})
}

// installDatastore reads a datastore written inline as the game's and
// installs it for the test.
func installDatastore(t *testing.T, game, doc string) {
	t.Helper()
	b, err := mtgmatcher.Open(game, strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	installBackend(t, b)
}
