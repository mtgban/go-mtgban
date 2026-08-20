package fleshandblood

import "testing"

// TestPromoTypeLabels pins that a token can be read back as the words it was
// made from. The token is what a search query carries; the label is what a
// reader is shown. The builder folds the catalog's qualifier to lower case,
// so the words survive and only their case is lost - unlike Magic and
// Riftbound, whose tokens ran their words together and had to be written
// down one by one.
func TestPromoTypeLabels(t *testing.T) {
	b := loadBackend(t)

	if len(b.PromoTypeLabels) != len(b.AllPromoTypes) {
		t.Errorf("%d tags declared but %d labelled", len(b.AllPromoTypes), len(b.PromoTypeLabels))
	}
	for _, tag := range b.AllPromoTypes {
		if b.PromoTypeLabel(tag) == "" {
			t.Errorf("tag %q reads back as nothing", tag)
		}
	}
	for _, tt := range []struct{ promoType, want string }{
		{"alternateart", "Alternate Art"},
		{"extendedart", "Extended Art"},
		{"marvel", "Marvel"},
		{"treasure", "Treasure"},
		// The cases that say why the words are looked up rather than
		// guessed: title-casing gives "Cc Tag" and "Fab362".
		{"cctag", "CC Tag"},
		{"fab362", "FAB362"},
		{"jpnexclusive", "JPN Exclusive"},
	} {
		if got := b.PromoTypeLabel(tt.promoType); got != tt.want {
			t.Errorf("PromoTypeLabel(%q) = %q, want %q", tt.promoType, got, tt.want)
		}
	}
	// An unknown token still reads as something rather than empty.
	if got := b.PromoTypeLabel("nosuchtag"); got == "" {
		t.Error("an undeclared tag reads back as nothing")
	}
}
