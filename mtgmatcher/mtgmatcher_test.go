package mtgmatcher_test

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// testBackend is the datastore the core suite exercises. Core matching is
// game-agnostic but only Magic carries data rich enough to probe it (tokens,
// promo types, alternate names), so the tests load Magic and drive it through
// the Backend's methods. The Magic replay corpus itself lives with its rules,
// in mtgmatcher/magic.
var testBackend *mtgmatcher.Backend

// realDatastore skips a test that reads the datastore where the run carries
// none. The internal suite loads it, once for this whole binary.
func realDatastore(t *testing.T) {
	t.Helper()
	testBackend = mtgmatcher.RealDatastore(t)
	testBackend.Logger = log.New(os.Stderr, "", 0)
}

// testBackendOrEmpty is for the tests that degrade gracefully with no
// datastore loaded rather than skipping: they read the empty Backend
// currentBackend() used to fall back to, and simply find nothing to check.
func testBackendOrEmpty() *mtgmatcher.Backend {
	if testBackend != nil {
		return testBackend
	}
	return &mtgmatcher.Backend{}
}
