package starcitygames

import "testing"

// The catalog carries supplies, sealed and bulk lots alongside singles,
// and each scraper has to keep to its own.
func TestProductTypeConstants(t *testing.T) {
	if ProductTypeSingles != "Singles" {
		t.Errorf("singles type = %q", ProductTypeSingles)
	}
	if ProductTypeSealed != "Sealed" {
		t.Errorf("sealed type = %q", ProductTypeSealed)
	}
}
