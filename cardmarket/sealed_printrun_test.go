package cardmarket

import (
	"slices"
	"testing"
)

// TestSealedNameTokens pins what makes two names comparable: the print-run
// mark and the count go, so a marketplace's "(Pre-Errata)" boxes and a
// datastore's "(Wave 1 - Blue)" ones come down to the same words.
func TestSealedNameTokens(t *testing.T) {
	base := sealedBaseKey("Romance Dawn Booster Box")
	// The mark a marketplace uses does not change what the name says.
	if got := sealedBaseKey("Romance Dawn Booster Box (Pre-Errata)"); got != base {
		t.Errorf("the marked box reduced to %q, want %q", got, base)
	}
	// Nor does saying a word twice, which the case name does.
	caseName := sealedBaseKey("Romance Dawn Booster Box Case (12x Booster Box)")
	if caseName != "booster box case dawn romance" {
		t.Errorf("the case reduced to %q", caseName)
	}
	// A case is not its box.
	if caseName == base {
		t.Error("a case reduced to the same words as its box")
	}
}

func TestNamesEarlyPrintRun(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Romance Dawn Booster Box (Pre-Errata)", true},
		{"Romance Dawn Booster Box Case (12x Booster Box) (Pre-Errata)", true},
		{"Romance Dawn Booster Box", false},
		// An ordinal is not a run mark: storefronts write plenty for other
		// reasons.
		{"1st Anniversary Tournament Pack", false},
		{"3rd Anniversary Tournament Pack", false},
	} {
		if got := namesEarlyPrintRun(tt.name); got != tt.want {
			t.Errorf("namesEarlyPrintRun(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestPrintRunIndexResolve pins the choice itself, on a stand-in index so the
// test says what it means without a datastore.
func TestPrintRunIndexResolve(t *testing.T) {
	key := sealedBaseKey("Romance Dawn Booster Box")
	index := printRunIndex{
		key: []printRunEntry{{uuid: "wave1", wave: 1}, {uuid: "wave2", wave: 2}},
		// A product held in a single run is not a choice at all.
		sealedBaseKey("Romance Dawn Booster Pack"): []printRunEntry{{uuid: "pack", wave: 1}},
	}

	for _, tt := range []struct {
		name  string
		early bool
		want  string
	}{
		{"Romance Dawn Booster Box", false, "wave2"},
		{"Romance Dawn Booster Box (Pre-Errata)", true, "wave1"},
		{"Romance Dawn Booster Pack", false, ""},
		{"Nothing Like It", false, ""},
	} {
		got, found := index.resolve(tt.name, tt.early)
		if tt.want == "" {
			if found {
				t.Errorf("resolve(%q) = %q, want no run", tt.name, got)
			}
			continue
		}
		if !found || got != tt.want {
			t.Errorf("resolve(%q, early=%v) = %q (%v), want %q", tt.name, tt.early, got, found, tt.want)
		}
	}
	_ = slices.Contains([]string{}, "")
}
