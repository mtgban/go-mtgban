package mtgban

import (
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
		t.Errorf("FAIL: inventory does not contain entries for A")
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
		t.Errorf("FAIL: inventory did not merge quantities")
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
		t.Errorf("FAIL: inventory merged quantities")
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
		t.Errorf("FAIL: inventory merged quantities")
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
		t.Errorf("FAIL: inventory does not contain entries for A")
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
		t.Errorf("FAIL: Tried to add the same entry twice")
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
		t.Errorf("FAIL: inventory does not contain entries for A")
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
		t.Errorf("FAIL: Tried to add the same entry twice")
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
		t.Errorf("FAIL: Tried to add a similar entry twice")
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
