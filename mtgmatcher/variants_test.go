package mtgmatcher

import (
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

var cursedListCodes = []string{"MB1", "FMB1", "PLIST", "PHED", "PCTB", "PAGL"}

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
			if !found && !slices.Contains(cursedListCodes, set.Code) {
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
