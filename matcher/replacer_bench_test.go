package matcher

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// A scrape's listings: every string new, each read a couple of times,
// which is the shape the match-set benchmarks never exercise.
func uniqueKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("Listing %d (Foil) - Near Mint", i)
	}
	return keys
}

func BenchmarkNormalizeUniqueTwice(b *testing.B) {
	keys := uniqueKeys(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(keys[i])
		Normalize(keys[i])
	}
}

func BenchmarkNormalizeUniqueTwiceParallel(b *testing.B) {
	keys := uniqueKeys(b.N)
	var next atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(next.Add(1) - 1)
			Normalize(keys[i])
			Normalize(keys[i])
		}
	})
}

// The hot keys of a match: a few strings read millions of times.
func BenchmarkNormalizeHotParallel(b *testing.B) {
	keys := []string{"Foil", "Etched", "Oversize", "Lightning Bolt", "Secret Lair Drop", "Prerelease"}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Normalize(keys[i%len(keys)])
			i++
		}
	})
}
