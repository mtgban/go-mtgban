package vegassingles

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
	_ "github.com/mtgban/go-mtgban/matcher/onepiece"
	_ "github.com/mtgban/go-mtgban/matcher/pokemon"
	_ "github.com/mtgban/go-mtgban/matcher/riftbound"
)

// withMagic skips a test that reads the Magic datastore where none is
// installed.
var (
	magicOnce sync.Once
	magicErr  error
	magicB    *matcher.Backend
)

// withMagic installs AllPrintings as the global the first time a test asks
// for it, and skips where the run carries none.
func withMagic(t *testing.T) *matcher.Backend {
	t.Helper()
	magicOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		b, err := datastore.Read("magic", path)
		if err != nil {
			magicErr = err
			return
		}
		magicB = b
	})
	if magicErr != nil {
		t.Fatal(magicErr)
	}
	if magicB == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return magicB
}

// withGameDatastore loads another game's datastore for a test, where the
// rest of this package's tests are Magic ones. The test is skipped where
// that game's datastore is not configured, which is how the shared
// `go test ./...` run sees it.
func withGameDatastore(t *testing.T, game matcher.Game, env string) *matcher.Backend {
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

func withRiftbound(t *testing.T) *matcher.Backend {
	t.Helper()
	return withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")
}

func withOnePiece(t *testing.T) *matcher.Backend {
	t.Helper()
	return withGameDatastore(t, "onepiece", "ONEPIECE_PATH")
}

func withPokemon(t *testing.T) *matcher.Backend {
	t.Helper()
	return withGameDatastore(t, "pokemon", "POKEMON_PATH")
}
