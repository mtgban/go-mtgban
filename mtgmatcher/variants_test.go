package mtgmatcher

import (
	"strings"
	"testing"
)

func TestVariants(t *testing.T) {
	for edition, table := range VariantsTable {
		set, err := GetSetByName(edition)
		if err != nil {
			t.Errorf("FAIL: [%s] %s", edition, err.Error())
			continue
		}

		for cardName, variants := range table {
			found := false
			for _, card := range set.Cards {
				if card.Name == cardName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("FAIL: [%s] '%s' name not found", edition, cardName)
				continue
			}

			for key := range variants {
				if key != strings.ToLower(key) {
					t.Errorf("FAIL: [%s] %s (%s) is not lowercase", edition, cardName, key)
				}
			}
		}
	}
}
