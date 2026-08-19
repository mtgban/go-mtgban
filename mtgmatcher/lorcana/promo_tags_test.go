package lorcana

import (
	"slices"
	"testing"
)

// TestPromoTagsAndQualifiedNames pins the same contract the other games
// carry. Lorcana writes none of this into the name - not one card name holds
// a parenthesis - so the tags come from the datastore's own promo fields.
func TestPromoTagsAndQualifiedNames(t *testing.T) {
	b := loadDatastore(t)

	for _, tag := range []string{"Organized Play", "D23", "HighGloss"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("promo type %q is not declared, so nothing will print it", tag)
		}
	}

	bare, err := b.SearchEquals("Mickey Mouse - Brave Little Tailor")
	if err != nil {
		t.Fatal(err)
	}
	qualified, err := b.SearchEquals("Mickey Mouse - Brave Little Tailor (D23)")
	if err != nil {
		t.Fatalf("qualified spelling: %v", err)
	}
	if len(qualified) >= len(bare) {
		t.Errorf("qualified spelling reached %d printings against the name's %d, expected fewer",
			len(qualified), len(bare))
	}
	for _, uuid := range qualified {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(co.PromoTypes, "D23") {
			t.Errorf("qualified spelling reached a printing tagged %v, want one carrying D23", co.PromoTypes)
		}
	}
}
