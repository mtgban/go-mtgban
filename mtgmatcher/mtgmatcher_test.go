package mtgmatcher_test

import (
	"log"
	"os"
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
var testBackend *mtgmatcher.Backend

func TestMain(m *testing.M) {
	// The tests inside the package cannot install this themselves: a loader
	// would import mtgmatcher and this file is mtgmatcher, so TestMain is the
	// only place that can hand them a datastore. A run without one is no
	// longer refused, it just reaches the guards below.
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path != "" {
		reader, err := datastore.Open(path)
		if err != nil {
			log.Fatalln(err)
		}
		testBackend, err = magic.Load(reader)
		reader.Close()
		if err != nil {
			log.Fatalln(err)
		}
		mtgmatcher.SetGlobalDatastore(testBackend)
	}
	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))
	os.Exit(m.Run())
}

// realDatastore skips a test that reads the datastore where the run carries
// none.
func realDatastore(t *testing.T) {
	t.Helper()
	if testBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}
