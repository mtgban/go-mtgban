package cardmarket

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
)

// datastoreBackend reads a datastore written inline as the game's.
func datastoreBackend(t *testing.T, game matcher.Game, doc string) *matcher.Backend {
	t.Helper()
	b, err := matcher.Open(game, strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var (
	realBackend     *matcher.Backend
	realBackendOnce sync.Once
	realBackendErr  error
)

// realDatastore installs the real, full Magic datastore for a test - loaded
// once per test binary run from ALLPRINTINGS5_PATH and reused, the way
// matcher's own tests do - skipping the test when that env var is unset
// rather than needing a checked-in fixture.
func realDatastore(t *testing.T) *matcher.Backend {
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
	return realBackend
}
