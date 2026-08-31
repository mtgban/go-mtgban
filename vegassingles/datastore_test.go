package vegassingles

import (
	"io"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	"github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	"github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
)

// magicInstalled records whether TestMain found a Magic datastore. The package
// no longer refuses to run without one: the games this scraper is scheduled
// for are the other three, and their CI jobs carry their own datastore and
// not this one.
var magicInstalled bool

// magicDatastore is the datastore installMagic loaded, kept so a test that
// installs another game's can put this one back without reading the file a
// second time. A backend is 2.9GB resident and the reload held two.
var magicDatastore *mtgmatcher.Backend

// withMagic skips a test that reads the Magic datastore where none is
// installed.
func withMagic(t *testing.T) {
	t.Helper()
	if !magicInstalled {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}

// withGameDatastore installs another game's datastore for the duration of a
// test and puts the Magic one back afterwards, since the matcher holds a
// single datastore package-wide. The test is skipped where that game's
// datastore is not configured, which is how a run carrying only one of them
// sees the rest.
func withGameDatastore(t *testing.T, env string, load func(io.Reader) (*mtgmatcher.Backend, error)) {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	reader, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	ds, err := load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(ds)

	t.Cleanup(func() {
		if !magicInstalled {
			return
		}
		mtgmatcher.SetGlobalDatastore(magicDatastore)
	})
}

func withRiftbound(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "RIFTBOUND_PATH", riftbound.Load)
}

func withOnePiece(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "ONEPIECE_PATH", onepiece.Load)
}

func withPokemon(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "POKEMON_PATH", pokemon.Load)
}
