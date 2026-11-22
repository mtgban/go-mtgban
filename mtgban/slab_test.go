package mtgban

import (
	"errors"
	"slices"
	"strconv"
	"testing"
)

// Every label the grade table holds folds through Slab.Fold to the grade
// SlabCondition gives it, read the way a scraper fills a Slab: a number
// as the score, a word as the label.
func TestFoldAgreesWithSlabCondition(t *testing.T) {
	for grader, scores := range slabGrades {
		for key := range scores {
			want, _ := SlabCondition(grader, key)
			slab := Slab{Grader: grader, Label: key}
			score, err := strconv.ParseFloat(key, 64)
			if err == nil {
				slab = Slab{Grader: grader, Score: score}
			}
			got, found := slab.Fold()
			if !found || got != want {
				t.Errorf("%s %s: Fold gave %q, %v; SlabCondition gives %q", grader, key, got, found, want)
			}
		}
	}
}

func TestFold(t *testing.T) {
	for _, tt := range []struct {
		slab  Slab
		want  Condition
		found bool
	}{
		{Slab{Grader: "BGS", Score: 9.5}, NM, true},
		{Slab{Grader: "BGS", Score: 8.5}, SP, true},
		{Slab{Grader: "CGC", Score: 10, Label: "Pristine"}, NM, true},
		{Slab{Grader: "CGC", Label: "Authentic Altered"}, PO, true},
		{Slab{Grader: "PSA", Label: "Authentic"}, "", false},
		{Slab{Grader: "SGC", Score: 10}, "", false},
		{Slab{}, "", false},
	} {
		got, found := tt.slab.Fold()
		if got != tt.want || found != tt.found {
			t.Errorf("%+v: got %q, %v; want %q, %v", tt.slab, got, found, tt.want, tt.found)
		}
	}
}

func TestUnreadSlab(t *testing.T) {
	got := UnreadSlab("PSA", "PSA Authentic")
	if *got != (Slab{Grader: "PSA", Label: "PSA Authentic"}) {
		t.Errorf("supported grader: got %+v", *got)
	}
	got = UnreadSlab("TAG", "TAG 10")
	if *got != (Slab{Grader: "TAG", Label: "TAG 10"}) {
		t.Errorf("grader with no fold table: got %+v", *got)
	}
	got = UnreadSlab("SGC", "SGC 10")
	if *got != (Slab{Label: "SGC 10"}) {
		t.Errorf("unsupported grader: got %+v", *got)
	}
}

// A slab carries the grade its scraper chose: one without is refused rather
// than defaulted to NM, and so is a grader or score outside the supported
// ones.
func TestAddRefusesASlabItCannotPublish(t *testing.T) {
	for _, entry := range []InventoryEntry{
		{Price: 100, Slab: &Slab{Grader: "PSA", Score: 10}},
		{Conditions: NM, Price: 100, Slab: &Slab{Grader: "SGC", Score: 10}},
		{Conditions: NM, Price: 100, Slab: &Slab{Grader: "PSA", Score: 9.3}},
		{Conditions: NM, Price: 100, Slab: &Slab{Grader: "PSA", Score: 11}},
		{Conditions: NM, Price: 100, Slab: &Slab{Grader: "PSA", Score: 0.5}},
	} {
		inv := InventoryRecord{}
		err := inv.Add("uuid", &entry)
		if !errors.Is(err, ErrInvalidCondition) {
			t.Errorf("%+v: got %v, want ErrInvalidCondition", *entry.Slab, err)
		}
		if len(inv) != 0 {
			t.Errorf("%+v: was added", *entry.Slab)
		}
	}

}

// Copies at one grade and price stay apart when their slabs differ, a raw
// copy among them, even with no id telling them apart; only the same slab
// twice merges.
func TestAddKeepsDifferentSlabsApart(t *testing.T) {
	slabs := []*Slab{
		{Grader: "PSA", Score: 10},
		{Grader: "BGS", Score: 9.5},
		{Grader: "CGC", Score: 10, Label: "Pristine"},
		{Label: "SGC 10"},
		{},
		nil,
	}
	inv := InventoryRecord{}
	for _, slab := range append(slices.Clone(slabs), &Slab{Grader: "PSA", Score: 10}) {
		entry := InventoryEntry{Conditions: NM, Price: 100, Slab: slab, URL: "u"}
		err := inv.AddRelaxed("uuid", &entry)
		if err != nil {
			t.Fatalf("%v: %v", slab, err)
		}
	}
	entries := inv["uuid"]
	if len(entries) != len(slabs) {
		t.Fatalf("got %d entries, want %d", len(entries), len(slabs))
	}
	for i, slab := range slabs {
		if !sameSlab(entries[i].Slab, slab) {
			t.Errorf("entry %d has slab %v, want %v", i, entries[i].Slab, slab)
		}
	}
	if entries[0].Quantity != 2 {
		t.Errorf("the two PSA 10s: got quantity %d, want 2", entries[0].Quantity)
	}

	unique := InventoryRecord{}
	for _, slab := range slabs {
		entry := InventoryEntry{Conditions: NM, Price: 100, Slab: slab}
		err := unique.AddUnique("uuid", &entry)
		if err != nil {
			t.Errorf("AddUnique %v: %v", slab, err)
		}
	}
}
