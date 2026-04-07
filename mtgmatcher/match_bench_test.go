package mtgmatcher

import (
	"io"
	"log"
	"testing"
)

// BenchmarkMatchQuiet runs the match set with the logger left as
// production leaves it - discarding - so the measurement is of the
// matching rather than of writing the matcher's narration to stderr.
func BenchmarkMatchQuiet(b *testing.B) {
	saved := logger
	logger = log.New(io.Discard, "", log.LstdFlags)
	b.Cleanup(func() { logger = saved })

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
