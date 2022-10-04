package mtgmatcher

import "testing"

func BenchmarkSearchEquals(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		SearchEquals(NameToBeFound)
	}
}

func BenchmarkSearchHasPrefix(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[:len(NameToBeFound)/2]

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		SearchHasPrefix(name)
	}
}

func BenchmarkSearchContains(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[len(NameToBeFound)/4 : len(NameToBeFound)/2]
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		SearchContains(name)
	}
}

func BenchmarkSearchRegexp(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[len(NameToBeFound)/4:] + ".*"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		SearchRegexp(name)
	}
}

func TestSearchRegexp(t *testing.T) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	hashes, err := SearchRegexp("Lotus$")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var found bool
	for _, hash := range hashes {
		co, err := GetUUID(hash)
		if err != nil {
			t.Error("FAIL: Unexpected", err)
			return
		}
		if co.Name == "Black Lotus" {
			found = true
			break
		}
	}
	if !found {
		t.Error("FAIL: Not found")
	}
	t.Log("PASS: regexp")
}
