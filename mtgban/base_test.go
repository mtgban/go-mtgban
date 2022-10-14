package mtgban

import (
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
