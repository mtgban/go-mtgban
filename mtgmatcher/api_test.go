package mtgmatcher

import (
	"fmt"
	"testing"
)

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
	} else {
		t.Log("PASS: regexp")
	}
}

func TestSearchFlavor(t *testing.T) {
	hashes, err := SearchEquals("Stay with Me")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var count int
	for _, hash := range hashes {
		co, err := GetUUID(hash)
		if err != nil {
			t.Error("FAIL: Unexpected", err)
			return
		}
		if co.SetCode == "FCA" {
			count++
		}
	}
	if count != 2 {
		t.Error("FAIL: Search should return exactly 2 results, got " + fmt.Sprint(count))
	} else {
		t.Log("PASS: flavor")
	}
}

func TestSearchHalfName(t *testing.T) {
	hashes, err := SearchEquals("Jonathan Harker")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var count int
	for _, hash := range hashes {
		co, err := GetUUID(hash)
		if err != nil {
			t.Error("FAIL: Unexpected", err)
			return
		}
		if co.HasPromoType(PromoTypeDracula) {
			count++
		}
	}
	if count != 2 {
		t.Error("FAIL: Search should return exactly 2 results, got " + fmt.Sprint(count))
	} else {
		t.Log("PASS: half name")
	}
}
