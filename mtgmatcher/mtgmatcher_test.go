package mtgmatcher_test

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastoretest"
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
	datastorePath := os.Getenv("ALLPRINTINGS5_PATH")
	if datastorePath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run this suite")
	}

	datastoreReader, err := datastoretest.Open(datastorePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer datastoreReader.Close()

	testBackend, err = magic.Load(datastoreReader)
	if err != nil {
		log.Fatalln(err)
	}

	mtgmatcher.SetGlobalDatastore(testBackend)
	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}
