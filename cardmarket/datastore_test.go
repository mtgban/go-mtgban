package cardmarket

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// installBackend installs a backend as the global datastore for legacy tests
// that still exercise package-level matcher wrappers.
func installBackend(t *testing.T, b *mtgmatcher.Backend) {
	t.Helper()
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(b)
	t.Cleanup(func() { mtgmatcher.SetGlobalDatastore(previous) })
}

// datastoreBackend reads a datastore written inline as the game's.
func datastoreBackend(t *testing.T, game, doc string) *mtgmatcher.Backend {
	t.Helper()
	b, err := mtgmatcher.Open(game, strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var (
	realBackend     *mtgmatcher.Backend
	realBackendOnce sync.Once
	realBackendErr  error
)

// realDatastore installs the real, full Magic datastore for a test - loaded
// once per test binary run from ALLPRINTINGS5_PATH and reused, the way
// mtgmatcher's own tests do - skipping the test when that env var is unset
// rather than needing a checked-in fixture.
func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	realBackendOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		reader, err := datastore.Open(path)
		if err != nil {
			realBackendErr = err
			return
		}
		defer reader.Close()
		realBackend, realBackendErr = magic.Load(reader)
	})
	if realBackendErr != nil {
		t.Fatal(realBackendErr)
	}
	if realBackend == nil {
		t.Skip("no ALLPRINTINGS5_PATH")
	}
	installBackend(t, realBackend)
	return realBackend
}
