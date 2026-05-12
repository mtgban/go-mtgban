package mtgmatcher_test

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// BenchmarkMatchQuiet runs the match set with the logger left as
// production leaves it - discarding - so the measurement is of the
// matching rather than of writing the matcher's narration to stderr.
func BenchmarkMatchQuiet(b *testing.B) {
	// The logger is only reachable through its setter from out here, so
	// the cleanup reinstates the one TestMain installs rather than a
	// saved copy of it.
	mtgmatcher.SetGlobalLogger(log.New(io.Discard, "", log.LstdFlags))
	b.Cleanup(func() { mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0)) })

	b.ReportAllocs()
	for b.Loop() {
		for _, testSet := range MatchTestSet {
			backend := testSet.Backend
			for _, test := range testSet.MatchTests {
				runMatch(backend, test)
			}
		}
	}
}
