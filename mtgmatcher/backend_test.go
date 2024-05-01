package mtgmatcher

import (
	"math/rand"
	"sort"
	"testing"
)

var NameToBeFound string
var ReturnWhenFound bool

var SliceOfObj []cardinfo
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

	sliceOfObj := make([]cardinfo, 0, len(backend.Cards))
	sliceOfStr := make([]string, 0, len(backend.Cards))
	for _, card := range backend.Cards {
		sliceOfObj = append(sliceOfObj, card)
		sliceOfStr = append(sliceOfStr, Normalize(card.Name))
	}
	sort.Slice(sliceOfObj, func(i, j int) bool {
		return sliceOfObj[i].Name > sliceOfObj[j].Name
	})
	sort.Slice(sliceOfStr, func(i, j int) bool {
		return sliceOfStr[i] > sliceOfStr[j]
	})
	SliceOfObj = sliceOfObj
	SliceOfStr = sliceOfStr
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

func backendInfo(name string, doneWhenFound bool) (printings []string) {
	name = Normalize(name)
	for key := range backend.Cards {
		if key == name {
			printings = backend.Cards[key].Printings
			if doneWhenFound {
				return
			}
		}
	}
	return
}

func BenchmarkSearchWithInfo(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		backendInfo(NameToBeFound, ReturnWhenFound)
	}
}

func backendSlice(name string, doneWhenFound bool) (printings []string) {
	name = Normalize(name)
	for i := range SliceOfObj {
		if Normalize(SliceOfObj[i].Name) == name {
			printings = SliceOfObj[i].Printings
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
			printings = backend.Cards[name].Printings
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
