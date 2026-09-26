package mtgmatcher_test

import (
	"fmt"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestNamesForms pins the list each form answers with, of cards and of
// sealed products.
func TestNamesForms(t *testing.T) {
	b := &mtgmatcher.Backend{
		AllNames:           []string{"n"},
		AllLowerNames:      []string{"l"},
		AllCanonicalNames:  []string{"c"},
		AllSealed:          []string{"sn"},
		AllLowerSealed:     []string{"sl"},
		AllCanonicalSealed: []string{"sc"},
	}
	for _, tt := range []struct {
		form   mtgmatcher.NameForm
		sealed bool
		want   string
	}{
		{mtgmatcher.NameFormNormalized, false, "n"},
		{mtgmatcher.NameFormLowercase, false, "l"},
		{mtgmatcher.NameFormCanonical, false, "c"},
		{mtgmatcher.NameFormNormalized, true, "sn"},
		{mtgmatcher.NameFormLowercase, true, "sl"},
		{mtgmatcher.NameFormCanonical, true, "sc"},
	} {
		got := b.Names(tt.form, tt.sealed)
		if len(got) != 1 || got[0] != tt.want {
			t.Errorf("Names(%d, %v) = %v, want [%s]", tt.form, tt.sealed, got, tt.want)
		}
	}
	got := b.Names(mtgmatcher.NameForm(9), false)
	if got != nil {
		t.Errorf("Names of an unknown form = %v, want nothing", got)
	}
}

func BenchmarkSearchEquals(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		testBackend.SearchEquals(NameToBeFound)
	}
}

func BenchmarkSearchHasPrefix(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[:len(NameToBeFound)/2]

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		testBackend.SearchHasPrefix(name)
	}
}

func BenchmarkSearchContains(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[len(NameToBeFound)/4 : len(NameToBeFound)/2]
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		testBackend.SearchContains(name)
	}
}

func BenchmarkSearchRegexp(b *testing.B) {
	if NameToBeFound == "" {
		setupBenchmark()
	}

	name := NameToBeFound[len(NameToBeFound)/4:] + ".*"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		testBackend.SearchRegexp(name)
	}
}

func TestSearchRegexp(t *testing.T) {
	realDatastore(t)
	b := testBackend
	hashes, err := b.SearchRegexp("Lotus$")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var found bool
	for _, hash := range hashes {
		co, err := b.GetUUID(hash)
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
	realDatastore(t)
	b := testBackend
	hashes, err := b.SearchEquals("Stay with Me")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var count int
	for _, hash := range hashes {
		co, err := b.GetUUID(hash)
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
	realDatastore(t)
	b := testBackend
	hashes, err := b.SearchEquals("Jonathan Harker")
	if err != nil {
		t.Error("FAIL: Unexpected", err)
		return
	}

	var count int
	for _, hash := range hashes {
		co, err := b.GetUUID(hash)
		if err != nil {
			t.Error("FAIL: Unexpected", err)
			return
		}
		if co.HasPromoType(magic.PromoTypeDracula) {
			count++
		}
	}
	if count != 2 {
		t.Error("FAIL: Search should return exactly 2 results, got " + fmt.Sprint(count))
	} else {
		t.Log("PASS: half name")
	}
}

func TestPrintings(t *testing.T) {
	realDatastore(t)
	b := testBackend
	setCodes, _ := b.Printings4Card("Black Lotus")
	if len(setCodes) != 6 {
		t.Error("FAIL: Printings should be exactly 6 results, got " + fmt.Sprint(setCodes))
	} else {
		t.Log("PASS: Printings4Card")
	}
}
