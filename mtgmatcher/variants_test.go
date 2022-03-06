package mtgmatcher

import (
	"strings"
	"testing"
)

func TestVariants(t *testing.T) {
	for edition, tables := range VariantsTable {
		table := tables

		for card, variants := range table {
			for key, _ := range variants {
				if key != strings.ToLower(key) {
					t.Errorf("FAIL: [%s] %s (%s) is not lowercase", edition, card, key)
				}
			}
		}
	}
}
