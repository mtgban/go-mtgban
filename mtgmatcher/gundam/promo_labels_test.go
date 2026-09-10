package gundam

import (
	"slices"
	"testing"
)

// TestPromoTypeLabels pins the label table against the datastore both
// ways: every token the loader declares reads back as words, and every
// row of the table spells a token the datastore declares - a row for a
// token it declares nowhere is a spelling nothing will ever read, and the
// rot is invisible to the vocabulary check, which audits the tokens the
// loader declares.
func TestPromoTypeLabels(t *testing.T) {
	b := loadBackend(t)

	for _, tag := range b.AllPromoTypes {
		if b.PromoTypeLabel(tag) == "" {
			t.Errorf("tag %q reads back as nothing", tag)
		}
	}
	for token := range promoTypeLabels {
		if !slices.Contains(b.AllPromoTypes, token) {
			t.Errorf("promoTypeLabels spells %q, a token the datastore declares nowhere", token)
		}
	}
}
