package mtgban

import (
	"errors"
	"math/rand"
	"testing"
)

func TestAddRelaxed(t *testing.T) {
	entryNM := InventoryEntry{
		Quantity:   5,
		Conditions: "NM",
		Price:      20.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	entrySP := InventoryEntry{
		Quantity:   4,
		Conditions: "SP",
		Price:      10.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	inventory := InventoryRecord{}

	// Empty inventory, add an entry
	err := inventory.AddRelaxed("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) == 0 {
		t.Error("FAIL: inventory does not contain entries for A")
		return
	}

	// Add entry with same ID, but it's a different conditions
	err = inventory.AddRelaxed("A", &entrySP)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory only contains %d entries for A", len(inventory["A"]))
		return
	}

	// Same entry, same id
	err = inventory.AddRelaxed("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory contains a differen number of entries (%d) than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 10 {
		t.Error("FAIL: inventory did not merge quantities")
		return
	}

	// Similar but different
	entryNM.SellerName = "NOTBANNED"
	err = inventory.AddRelaxed("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 3 {
		t.Errorf("FAIL: inventory contains a differen number of entries %d than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 10 {
		t.Error("FAIL: inventory merged quantities")
		return
	}

	// Similar but different
	entryNM.SellerName = "NOTBANNED"
	err = inventory.AddRelaxed("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 3 {
		t.Errorf("FAIL: inventory contains a differen number of entries %d than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 10 {
		t.Error("FAIL: inventory merged quantities")
		return
	}

	t.Log("PASS: AddRelaxed")
}

func TestAdd(t *testing.T) {
	entryNM := InventoryEntry{
		Quantity:   5,
		Conditions: "NM",
		Price:      20.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	entrySP := InventoryEntry{
		Quantity:   4,
		Conditions: "SP",
		Price:      10.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	inventory := InventoryRecord{}

	// Empty inventory, add an entry
	err := inventory.Add("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) == 0 {
		t.Error("FAIL: inventory does not contain entries for A")
		return
	}

	// Add entry with same ID, but it's a different conditions
	err = inventory.Add("A", &entrySP)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory only contains %d entries for A", len(inventory["A"]))
		return
	}

	// Same entry, same id
	err = inventory.Add("A", &entryNM)
	if err == nil {
		t.Error("FAIL: Tried to add the same entry twice")
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory contains a differen number of entries (%d) than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 5 {
		t.Errorf("FAIL: inventory merged quantities (got %d)", inventory["A"][0].Quantity)
		return
	}

	// Similar but different
	entryNM.SellerName = "NOTBANNED"
	err = inventory.Add("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 3 {
		t.Errorf("FAIL: inventory contains a differen number of entries %d than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 5 {
		t.Errorf("FAIL: inventory merged quantities (got %d)", inventory["A"][0].Quantity)
		return
	}

	t.Log("PASS: Add")
}

func TestAddStrict(t *testing.T) {
	entryNM := InventoryEntry{
		Quantity:   5,
		Conditions: "NM",
		Price:      20.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	entrySP := InventoryEntry{
		Quantity:   4,
		Conditions: "SP",
		Price:      10.0,
		URL:        "https://mtgban.com",
		SellerName: "BANNED",
	}
	inventory := InventoryRecord{}

	// Empty inventory, add an entry
	err := inventory.AddStrict("A", &entryNM)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) == 0 {
		t.Error("FAIL: inventory does not contain entries for A")
		return
	}

	// Add entry with same ID, but it's a different conditions
	err = inventory.AddStrict("A", &entrySP)
	if err != nil {
		t.Errorf("FAIL: Unexpected error: %s", err.Error())
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory only contains %d entries for A", len(inventory["A"]))
		return
	}

	// Same entry, same id
	err = inventory.AddStrict("A", &entryNM)
	if err == nil {
		t.Error("FAIL: Tried to add the same entry twice")
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory contains a differen number of entries (%d) than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 5 {
		t.Errorf("FAIL: inventory merged quantities (got %d)", inventory["A"][0].Quantity)
		return
	}

	// Similar but different
	err = inventory.AddStrict("A", &entryNM)
	if err == nil {
		t.Error("FAIL: Tried to add a similar entry twice")
		return
	}
	if len(inventory["A"]) != 2 {
		t.Errorf("FAIL: inventory contains a differen number of entries (%d) than expected for A", len(inventory["A"]))
		return
	}
	if inventory["A"][0].Quantity != 5 {
		t.Errorf("FAIL: inventory merged quantities (got %d)", inventory["A"][0].Quantity)
		return
	}

	t.Log("PASS: AddStrict")
}

func TestSort(t *testing.T) {
	testEntries := []InventoryEntry{
		{
			Quantity:   5,
			Conditions: "NM",
			Price:      20.0,
			URL:        "https://mtgban.com",
			SellerName: "BANNED",
		},
		{
			Quantity:   4,
			Conditions: "SP",
			Price:      8.0,
			URL:        "https://mtgban.com",
			SellerName: "BANNED",
		},
		{
			Quantity:   4,
			Conditions: "SP",
			Price:      10.0,
			URL:        "https://mtgban.com",
			SellerName: "BANNED",
		},
		{
			Quantity:   1,
			Conditions: "SP",
			Price:      10.0,
			URL:        "https://mtgban.com",
			SellerName: "BANNED_TWO",
		},
	}

	rand.Shuffle(len(testEntries), func(i, j int) {
		testEntries[i], testEntries[j] = testEntries[j], testEntries[i]
	})

	expectedCond := []string{"NM", "SP", "SP", "SP"}
	expectedPrice := []float64{20.0, 8.0, 10.0, 10.0}
	expectedQty := []int{5, 4, 4, 1}

	inventory := InventoryRecord{}

	// Add all the entries in wrong order
	for _, testEntry := range testEntries {
		err := inventory.AddStrict("A", &testEntry)
		if err != nil {
			t.Errorf("FAIL: Unexpected error: %s", err.Error())
			return
		}
	}
	if len(inventory["A"]) != len(testEntries) {
		t.Errorf("FAIL: inventory contains a differen number of entries (%d) than expected (%d) for A", len(inventory["A"]), len(testEntries))
		return
	}

	for _, entries := range inventory {
		for i := range entries {
			if entries[i].Conditions != expectedCond[i] {
				t.Errorf("FAIL: array not sorted: condition of %d is not %s (got %s)", i, expectedCond[i], entries[i].Conditions)
				return
			}
			if entries[i].Price != expectedPrice[i] {
				t.Errorf("FAIL: array not sorted: price of %d is not %f (got %f)", i, expectedPrice[i], entries[i].Price)
				return
			}
			if entries[i].Quantity != expectedQty[i] {
				t.Errorf("FAIL: array not sorted: quantity of %d is not %d (got %d)", i, expectedQty[i], entries[i].Quantity)
				return
			}
		}
	}

	t.Log("PASS: Sort")
}

// TestDuplicateEntryIsRecognisable pins that a record refusing a duplicate
// says so in a way a caller can test for. A storefront listing one card under
// two products produces the refusal by the thousand, and a scraper that knows
// its storefront does that has to tell it from a real failure.
func TestDuplicateEntryIsRecognisable(t *testing.T) {
	t.Run("buylist", func(t *testing.T) {
		bl := BuylistRecord{}
		entry := &BuylistEntry{Conditions: "NM", BuyPrice: 4.34}
		if err := bl.Add("id", entry); err != nil {
			t.Fatalf("first add: %v", err)
		}
		err := bl.Add("id", entry)
		if !errors.Is(err, ErrDuplicateEntry) {
			t.Errorf("second add: %v, want ErrDuplicateEntry", err)
		}
		if len(bl["id"]) != 1 {
			t.Errorf("the record kept %d entries, want the first one alone", len(bl["id"]))
		}
	})

	t.Run("a differing price is not a duplicate", func(t *testing.T) {
		bl := BuylistRecord{}
		if err := bl.Add("id", &BuylistEntry{Conditions: "NM", BuyPrice: 4.34}); err != nil {
			t.Fatalf("first add: %v", err)
		}
		if err := bl.Add("id", &BuylistEntry{Conditions: "NM", BuyPrice: 9.99}); err != nil {
			t.Errorf("a second price was refused: %v", err)
		}
		if len(bl["id"]) != 2 {
			t.Errorf("the record kept %d entries, want both prices", len(bl["id"]))
		}
	})

	t.Run("inventory", func(t *testing.T) {
		inv := InventoryRecord{}
		entry := &InventoryEntry{Conditions: "NM", Price: 4.34, Quantity: 3, URL: "u"}
		if err := inv.Add("id", entry); err != nil {
			t.Fatalf("first add: %v", err)
		}
		err := inv.Add("id", entry)
		if !errors.Is(err, ErrDuplicateEntry) {
			t.Errorf("second add: %v, want ErrDuplicateEntry", err)
		}
		if len(inv["id"]) != 1 || inv["id"][0].Quantity != 3 {
			t.Errorf("the record holds %+v, want the first entry with its own quantity", inv["id"])
		}
	})

	t.Run("a bad condition is still its own error", func(t *testing.T) {
		bl := BuylistRecord{}
		err := bl.Add("id", &BuylistEntry{Conditions: "XX"})
		if errors.Is(err, ErrDuplicateEntry) {
			t.Errorf("an invalid condition reported %v", err)
		}
		if !errors.Is(err, ErrInvalidCondition) {
			t.Errorf("got %v, want ErrInvalidCondition", err)
		}
	})
}
