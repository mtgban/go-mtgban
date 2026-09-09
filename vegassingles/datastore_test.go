package vegassingles

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
)

// magicInstalled records whether TestMain found a Magic datastore. The package
// no longer refuses to run without one: the games this scraper is scheduled
// for are the other three, and their CI jobs carry their own datastore and
// not this one.
var magicInstalled bool

// withMagic skips a test that reads the Magic datastore where none is
// installed.
func withMagic(t *testing.T) {
	t.Helper()
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
