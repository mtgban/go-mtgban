package vegassingles

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
)

// withMagic skips a test that reads the Magic datastore where none is
// installed.
var (
	magicOnce      sync.Once
	magicErr       error
	magicInstalled bool
)

// withMagic installs AllPrintings the first time a test asks for it, and
// skips where the run carries none.
func withMagic(t *testing.T) {
	t.Helper()
	magicOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		reader, err := datastore.Open(path)
		if err != nil {
			magicErr = err
			return
		}
		ds, err := magic.Load(reader)
		reader.Close()
		if err != nil {
			magicErr = err
			return
		}
		mtgmatcher.SetGlobalDatastore(ds)
		magicInstalled = true
	})
	if magicErr != nil {
		t.Fatal(magicErr)
	}
	if !magicInstalled {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}

// withGameDatastore installs another game's datastore for the duration of a
// test and puts back what stood before, since the package-level matcher
// holds a single datastore and the rest of this package's tests are Magic ones.
// The test is skipped where that game's datastore is not configured, which
// is how the shared `go test ./...` run sees it.
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

func withRiftbound(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "riftbound", "RIFTBOUND_PATH")
}

func withOnePiece(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "onepiece", "ONEPIECE_PATH")
}

func withPokemon(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "pokemon", "POKEMON_PATH")
}
