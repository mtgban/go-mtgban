package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestSecondStock pins the bucket decision in both stream orders, with no
// datastore behind it: the catalog does not promise which record of a pair
// arrives first, and the fold must not care.
func TestSecondStock(t *testing.T) {
	const plain = "SGL-FAB-APS-017-ENN"
	const marked = "SGL-FAB-APS-017_CC-ENN"

	scg := &Starcitygames{buckets: map[string]struct{}{}}
	if scg.secondStock(plain) {
		t.Error("a plain sku with no pair on record read as a second bucket")
	}
	if !scg.secondStock(marked) {
		t.Error("the marked sku did not read as a second bucket")
	}

	scg = &Starcitygames{buckets: map[string]struct{}{}}
	if !scg.secondStock(marked) {
		t.Error("marked-first: the marked sku did not read as a second bucket")
	}
	if !scg.secondStock(plain) {
		t.Error("marked-first: the plain twin did not fold into the pair")
	}

	scg = &Starcitygames{buckets: map[string]struct{}{}}
	if scg.secondStock("SGL-FAB-APS-019-ENN") {
		t.Error("an unrelated sku read as a second bucket")
	}
}

// TestStockFold pins the strictness the bucket decision picks: the first
// bucket must report a same-price duplicate, the second must fold silently,
// on the retail and the buylist side both.
func TestStockFold(t *testing.T) {
	scg := &Starcitygames{
		inventory: mtgban.InventoryRecord{},
		buylist:   mtgban.BuylistRecord{},
	}
	inv := func(qty int, url string) *mtgban.InventoryEntry {
		return &mtgban.InventoryEntry{Conditions: "NM", Price: 1.5, Quantity: qty, URL: url}
	}
	buy := func(qty int) *mtgban.BuylistEntry {
		return &mtgban.BuylistEntry{Conditions: "NM", BuyPrice: 1.0, Quantity: qty}
	}

	if err := scg.addInventoryStock(false, "id", inv(3, "a")); err != nil {
		t.Fatalf("first bucket refused outright: %v", err)
	}
	if err := scg.addInventoryStock(false, "id", inv(2, "b")); err == nil {
		t.Error("a duplicate first bucket folded instead of reporting")
	}
	if err := scg.addInventoryStock(true, "id", inv(2, "b")); err != nil {
		t.Errorf("the second bucket reported instead of folding: %v", err)
	}
	if qty := scg.inventory["id"][0].Quantity; qty != 5 {
		t.Errorf("folded quantity = %d, want 5", qty)
	}

	// A buylist duplicate is the very same record - grade, price and
	// quantity alike - which is what the second stock bucket produces.
	if err := scg.addBuylistStock(false, "id", buy(3)); err != nil {
		t.Fatalf("first buylist bucket refused outright: %v", err)
	}
	if err := scg.addBuylistStock(false, "id", buy(3)); err == nil {
		t.Error("a duplicate first buylist bucket folded instead of reporting")
	}
	if err := scg.addBuylistStock(true, "id", buy(3)); err != nil {
		t.Errorf("the second buylist bucket reported instead of folding: %v", err)
	}
	if qty := scg.buylist["id"][0].Quantity; qty != 6 {
		t.Errorf("folded buylist quantity = %d, want 6", qty)
	}
}

// TestSiblingCandidate pins the filter that walks a sku marker from its base
// printing to the one it names, over hand-built printings: the same slot
// spelled longer qualifies, another physical format or another slot never
// does - whatever a datastore happens to carry.
func TestSiblingCandidate(t *testing.T) {
	base := &mtgmatcher.CardObject{}
	base.Name = "Mulan - Imperial Soldier"
	base.SetCode = "1"
	base.Number = "118"

	sibling := func(name, set, number string) *mtgmatcher.CardObject {
		co := &mtgmatcher.CardObject{}
		co.Name = name
		co.SetCode = set
		co.Number = number
		return co
	}

	for _, tt := range []struct {
		desc string
		co   *mtgmatcher.CardObject
		want bool
	}{
		{"the errata spelling beside the base",
			sibling("Mulan - Imperial Soldier (Errata Version)", "1", "118"), true},
		{"a jumbo print is another product, not the marker's printing",
			sibling("Mulan - Imperial Soldier (Oversized)", "1", "118"), false},
		{"another slot in the set says nothing about this one",
			sibling("Mulan - Imperial Soldier (Errata Version)", "1", "119"), false},
		{"another set's printing says nothing either",
			sibling("Mulan - Imperial Soldier (Errata Version)", "2", "118"), false},
		{"the base's own spelling is not a sibling",
			sibling("Mulan - Imperial Soldier", "1", "118"), false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := siblingCandidate(base, tt.co); got != tt.want {
				t.Errorf("siblingCandidate() = %v, want %v", got, tt.want)
			}
		})
	}
}
