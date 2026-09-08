package lorcana

import "testing"

// TestPromoTypeLabels pins that a token can be read back as the words it was
// made from. The token is what a search query carries; the label is what a
// reader is shown, and title-casing the token cannot put back the spaces it
// dropped.
func TestPromoTypeLabels(t *testing.T) {
	b := loadDatastore(t)

	if len(b.PromoTypeLabels) != len(b.AllPromoTypes) {
		t.Errorf("%d tags declared but %d labelled", len(b.AllPromoTypes), len(b.PromoTypeLabels))
	}
	for _, tag := range b.AllPromoTypes {
		if b.PromoTypeLabel(tag) == "" {
			t.Errorf("tag %q reads back as nothing", tag)
		}
	}
	// The datastore publishes a slug and nothing else, so the words come
	// from this side. A title-caser cannot put back the boundaries the slug
	// dropped, which is what the table is for.
	if got := b.PromoTypeLabel("organizedplay"); got != "Organized Play" {
		t.Errorf("PromoTypeLabel(%q) = %q, want %q", "organizedplay", got, "Organized Play")
	}
	// An initialism the datastore still spells with capitals of its own is
	// shown as written rather than title-cased into "Pd1".
	if got := b.PromoTypeLabel("pd1"); got != "PD1" {
		t.Errorf("PromoTypeLabel(%q) = %q, want %q", "pd1", got, "PD1")
	}
	// An unknown token still reads as something rather than empty.
	if got := b.PromoTypeLabel("nosuchtag"); got == "" {
		t.Error("an undeclared tag reads back as nothing")
	}
}
