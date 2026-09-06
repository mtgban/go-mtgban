package mtgban

import "testing"

// TestAddCheapestKeepsTheLowerPrice pins that a card and grade priced by more
// than one product is published once, at the lower price, whichever product
// is recorded first - and that a different grade or seller is its own row.
func TestAddCheapestKeepsTheLowerPrice(t *testing.T) {
	inv := InventoryRecord{}
	for _, e := range []InventoryEntry{
		{Conditions: "NM", Price: 25, URL: "first"},
		{Conditions: "NM", Price: 15, URL: "cheaper"},
		{Conditions: "NM", Price: 40, URL: "dearer"},
		{Conditions: "SP", Price: 9, URL: "other-grade"},
		{Conditions: "NM", Price: 1, URL: "other-seller", SellerName: "B"},
	} {
		e := e
		if err := inv.AddCheapest("card", &e); err != nil {
			t.Fatal(err)
		}
	}
	got := map[string]InventoryEntry{}
	for _, e := range inv["card"] {
		got[e.Conditions+"/"+e.SellerName] = e
	}
	if len(got) != 3 {
		t.Fatalf("published %d rows, want NM, SP and NM/B: %+v", len(got), got)
	}
	if nm := got["NM/"]; nm.Price != 15 || nm.URL != "cheaper" {
		t.Errorf("NM is %.0f from %q, want 15 from the cheaper product", nm.Price, nm.URL)
	}
	if sp := got["SP/"]; sp.Price != 9 {
		t.Errorf("SP is %.0f, want 9", sp.Price)
	}
	if b := got["NM/B"]; b.Price != 1 {
		t.Errorf("seller B's NM is %.0f, want its own 1", b.Price)
	}
}
