package gundam

import (
	"slices"
	"strings"
	"testing"
)

// TestLoadFoldsMixedCaseSetCode guards the regression the fold exists for.
// TCGplayer abbreviates Edition Beta "GD01_b", the one mixed-case
// abbreviation in the category, and the builder carried the case into the
// set code. Every lookup folds the caller's spelling up before it reads the
// map, so the set was listed by GetAllSets, worn by its cards, and found by
// nothing: the website's "s:GD01-b" came back empty and every link into the
// set was dead.
func TestLoadFoldsMixedCaseSetCode(t *testing.T) {
	b, err := Load(strings.NewReader(qualifiedFixture))
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(b.AllSets, "GD01-B") {
		t.Fatalf("AllSets = %v, want it to list GD01-B", b.AllSets)
	}

	// Every spelling of the code reaches the set, which is what GetSet
	// promises and what a link, a query and a card all rely on.
	for _, spelling := range []string{"GD01-B", "GD01-b", "gd01-b"} {
		set, err := b.GetSet(spelling)
		if err != nil {
			t.Errorf("GetSet(%q) = %v, want Edition Beta", spelling, err)
			continue
		}
		if set.Code != "GD01-B" {
			t.Errorf("GetSet(%q).Code = %q, want GD01-B", spelling, set.Code)
		}
		if got := len(b.GetUUIDsInSet(spelling)); got != 2 {
			t.Errorf("GetUUIDsInSet(%q) holds %d uuids, want 2", spelling, got)
		}
	}

	// The cards are folded with the map, or the two disagree and the set
	// lists printings its own uuids are not filed under.
	set, err := b.GetSet("GD01-B")
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range set.Cards {
		if card.SetCode != "GD01-B" {
			t.Errorf("%s carries SetCode %q, want GD01-B", card.UUID, card.SetCode)
		}
	}
}

// TestEverySetInTheDatastoreCanBeLookedUp sweeps the shipped datastore for
// any other code that is listed and cannot be found, the shape the mixed-case
// one had.
func TestEverySetInTheDatastoreCanBeLookedUp(t *testing.T) {
	b := loadBackend(t)

	for _, code := range b.AllSets {
		if _, err := b.GetSet(code); err != nil {
			t.Errorf("%s is listed, and GetSet(%q) says %v", code, code, err)
		}
	}
}
