package riftbound

import (
	"slices"
	"testing"
)

// TestPromoTagsAndQualifiedNames pins what tells sibling promos apart. Three
// printings share the name "Teemo, Swift Scout" and the number 263, and only
// the qualifiers separate them; the site prints a qualifier only when the
// backend declares it, so both halves have to hold.
func TestPromoTagsAndQualifiedNames(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range []string{"metal", "best of", "prize wall", "alternate art"} {
		if !slices.Contains(b.AllPromoTypes, tag) {
			t.Errorf("promo type %q is not declared, so nothing will print it", tag)
		}
	}

	// A qualifier that only repeats the collector number describes nothing:
	// the catalog names a rune variant "Fury Rune (R01c)" and the number
	// already says it.
	for _, tag := range b.AllPromoTypes {
		if CanonicalNumber(tag) == tag && len(tag) > 1 && tag[0] == 'r' && tag[1] >= '0' && tag[1] <= '9' {
			t.Errorf("promo type %q is a collector number, not a description", tag)
		}
	}

	bare, err := b.SearchEquals("Teemo, Swift Scout")
	if err != nil {
		t.Fatal(err)
	}
	if len(bare) < 2 {
		t.Errorf("bare name reached %d printings, expected every printing of the name", len(bare))
	}

	for _, tt := range []struct{ query, promoType string }{
		{"Teemo, Swift Scout (Metal Best Of)", "best of"},
		{"Teemo, Swift Scout (Metal Prize Wall)", "prize wall"},
	} {
		got, err := b.SearchEquals(tt.query)
		if err != nil {
			t.Errorf("%q: %v", tt.query, err)
			continue
		}
		if len(got) != 1 {
			t.Errorf("%q reached %d printings, want the one it names", tt.query, len(got))
			continue
		}
		co, err := b.GetUUID(got[0])
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(co.PromoTypes, tt.promoType) {
			t.Errorf("%q reached a printing tagged %v, want one carrying %q", tt.query, co.PromoTypes, tt.promoType)
		}
	}
}
