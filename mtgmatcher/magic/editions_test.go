package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestEditions pins that no spelling in the table is already an official set
// code: a name that resolves on its own must not be rewritten to another set.
func TestEditions(t *testing.T) {
	for edition := range EditionTable {
		_, err := mtgmatcher.GetSet(edition)
		if err == nil {
			t.Errorf("FAIL: %s is already an official set code", edition)
			continue
		}
	}
}
