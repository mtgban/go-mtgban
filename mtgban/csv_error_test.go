package mtgban

import (
	"errors"
	"testing"
)

// failAfter accepts n bytes and then fails, standing in for a destination
// that dies partway through: with a small record set the failure lands only
// on the final flush, which used to be deferred and its error dropped.
type failAfter struct {
	budget int
}

func (f *failAfter) Write(p []byte) (int, error) {
	if f.budget <= 0 {
		return 0, errors.New("destination went away")
	}
	if len(p) > f.budget {
		n := f.budget
		f.budget = 0
		return n, errors.New("destination went away")
	}
	f.budget -= len(p)
	return len(p), nil
}

func TestWriteInventoryToCSVReportsAFailedDestination(t *testing.T) {
	inv := InventoryRecord{}
	for _, id := range []string{"a|Card A|SET|1", "b|Card B|SET|2", "c|Card C|SET|3"} {
		inv[id] = []InventoryEntry{{Conditions: "NM", Price: 1, Quantity: 1}}
	}

	// enough for the header, not for everything after it
	if err := WriteInventoryToCSV(inv, &failAfter{budget: 40}); err == nil {
		t.Error("a destination that failed mid-write reported success")
	}
}

func TestWriteBuylistToCSVReportsAFailedDestination(t *testing.T) {
	bl := BuylistRecord{
		"a|Card A|SET|1": []BuylistEntry{{Conditions: "NM", BuyPrice: 1, Quantity: 1}},
	}
	if err := WriteBuylistToCSV(bl, 1, &failAfter{budget: 40}); err == nil {
		t.Error("a destination that failed mid-write reported success")
	}
}

func TestWriteInventoryToCSVStillSucceeds(t *testing.T) {
	inv := InventoryRecord{
		"a|Card A|SET|1": []InventoryEntry{{Conditions: "NM", Price: 1, Quantity: 1}},
	}
	if err := WriteInventoryToCSV(inv, &failAfter{budget: 1 << 20}); err != nil {
		t.Errorf("a healthy destination reported %v", err)
	}
}
