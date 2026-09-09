package mtgmatcher_test

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// testBackend is the datastore the core suite exercises. Core matching is
// game-agnostic but only Magic carries data rich enough to probe it (tokens,
// promo types, alternate names), so the tests load Magic and drive it through
// the package-level API. The Magic replay corpus itself lives with its rules,
// in mtgmatcher/magic.
var (
	testBackend   *mtgmatcher.Backend
	datastoreOnce sync.Once
	datastoreErr  error
)

func TestMain(m *testing.M) {
	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))
	os.Exit(m.Run())
}

// realDatastore skips a test that reads the datastore where the run carries
// none.
func realDatastore(t *testing.T) {
	t.Helper()
	datastoreOnce.Do(func() {
		if len(mtgmatcher.GetAllSets()) > 0 {
			testBackend = mtgmatcher.GlobalDatastore()
			return
		}
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		reader, err := datastore.Open(path)
		if err != nil {
			datastoreErr = err
			return
		}
		defer reader.Close()

		backend, err := magic.Load(reader)
		if err != nil {
			datastoreErr = err
			return
		}
		testBackend = backend
		mtgmatcher.SetGlobalDatastore(testBackend)
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if testBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}
