package mtgmatcher_test

import (
	"math/rand"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var NameToBeFound string
var ReturnWhenFound bool

var SliceOfObj []string
var SliceOfStr []string

func setupBenchmark() {
	b := MatchTestSet[0].Backend

	for _, code := range b.AllSets {
		set, _ := b.GetSet(code)
		if len(set.Cards) == 0 {
			continue
		}
		index := rand.Intn(len(set.Cards))
		NameToBeFound = set.Cards[index].Name
		break
	}

	SliceOfObj = b.AllCanonicalNames
	SliceOfStr = b.AllNames
}

func backendUUIDs(name string, doneWhenFound bool) (printings []string) {
	b := MatchTestSet[0].Backend
	name = mtgmatcher.Normalize(name)
	for key := range b.UUIDs {
		if mtgmatcher.Normalize(b.UUIDs[key].Name) == name {
			printings = b.UUIDs[key].Printings
			if doneWhenFound {
				return
			}
		}
	}
	return
}

func BenchmarkSearchWithUUIDs(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		backendUUIDs(NameToBeFound, ReturnWhenFound)
	}
}

func backendSlice(name string, doneWhenFound bool) (printings []string) {
	b := MatchTestSet[0].Backend
	name = mtgmatcher.Normalize(name)
	for i := range SliceOfObj {
		if mtgmatcher.Normalize(SliceOfObj[i]) == name {
			printings, _ = b.Printings4Card(name)
			if doneWhenFound {
				return
			}
		}
	}
	return
}

func BenchmarkSearchWithSlice(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		backendSlice(NameToBeFound, ReturnWhenFound)
	}
}

func backendHybrid(name string, doneWhenFound bool) (printings []string) {
	b := MatchTestSet[0].Backend
	name = mtgmatcher.Normalize(name)
	for i := range SliceOfStr {
		if SliceOfStr[i] == name {
			printings, _ = b.Printings4Card(name)
			if doneWhenFound {
				return
			}
		}
	}
	return
}

func BenchmarkSearchWithSliceAndMap(b *testing.B) {
	for n := 0; n < b.N; n++ {
		backendHybrid(NameToBeFound, ReturnWhenFound)
	}
}
