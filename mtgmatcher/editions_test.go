package mtgmatcher

import (
	"testing"
)

func TestEditions(t *testing.T) {
	for edition := range EditionTable {
		_, err := GetSet(edition)
		if err == nil {
			t.Errorf("FAIL: %s is already an official set code", edition)
			continue
		}
	}
}
