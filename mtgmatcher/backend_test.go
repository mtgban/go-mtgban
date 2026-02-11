package mtgmatcher

import (
	"math/rand"
	"testing"
)

var NameToBeFound string
var ReturnWhenFound bool

var SliceOfObj []string
var SliceOfStr []string

func setupBenchmark() {
	for _, code := range GetAllSets() {
		set, _ := GetSet(code)
		if len(set.Cards) == 0 {
			continue
		}
		index := rand.Intn(len(set.Cards))
		NameToBeFound = set.Cards[index].Name
		break
	}

	SliceOfObj = backend.AllCanonicalNames
	SliceOfStr = backend.AllNames
}

func backendUUIDs(name string, doneWhenFound bool) (printings []string) {
	name = Normalize(name)
	for key := range backend.UUIDs {
		if Normalize(backend.UUIDs[key].Name) == name {
			printings = backend.UUIDs[key].Printings
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
	name = Normalize(name)
	for i := range SliceOfObj {
		if Normalize(SliceOfObj[i]) == name {
			printings, _ = Printings4Card(name)
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
	name = Normalize(name)
	for i := range SliceOfStr {
		if SliceOfStr[i] == name {
			printings, _ = Printings4Card(name)
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
