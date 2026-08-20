package riftbound

import "testing"

// TestPromoTypeLabels pins that a token can be read back as the words it was
// made from. The token is what a search query carries; the label is what a
// reader is shown, and title-casing the token cannot put back the spaces it
// dropped.
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
		{"bestof", "Best Of"},
		{"prizewall", "Prize Wall"},
		// The case that says why the spelling is looked up rather than
		// guessed: title-casing this token gives "Gg Ez".
		{"ggez", "GG EZ"},
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
