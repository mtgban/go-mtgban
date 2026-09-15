package magic

import (
	"testing"
)

// TestEditions pins that no spelling in the table is already an official set
// code: a name that resolves on its own must not be rewritten to another set.
func TestEditions(t *testing.T) {
	realDatastore(t)

	for edition := range EditionTable {
		_, err := testBackend.GetSet(edition)
		if err == nil {
			t.Errorf("FAIL: %s is already an official set code", edition)
			continue
		}
	}
}
