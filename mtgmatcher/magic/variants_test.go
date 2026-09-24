package magic

import (
	"slices"
	"strings"
	"testing"
)

func TestVariants(t *testing.T) {
	realDatastore(t)
	for edition, table := range VariantsTable {
		set, err := testBackend.GetSetByName(edition)
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

// TestEd4VariantsNumbersAreReal pins what TestVariants above does not check:
// that a VariantsTable entry's value is an actual card.Number in the set it
// is registered for, not just that the card name exists there. The three
// editions share ed4Variants, so each is checked against its own numbers.
func TestEd4VariantsNumbersAreReal(t *testing.T) {
	realDatastore(t)

	tests := []struct {
		edition string
		table   map[string]map[string]string
	}{
		{"Fourth Edition", ed4Variants},
		{"Fourth Edition Foreign Black Border", ed4Variants},
		{"Alternate Fourth Edition", ed4Variants},
	}
	for _, tt := range tests {
		set, err := testBackend.GetSetByName(tt.edition)
		if err != nil {
			t.Fatalf("[%s] %v", tt.edition, err)
		}
		for cardName, variants := range tt.table {
			var numbers []string
			for _, card := range set.Cards {
				if card.Name == cardName {
					numbers = append(numbers, card.Number)
				}
			}
			for key, num := range variants {
				if !slices.Contains(numbers, num) {
					t.Errorf("FAIL: [%s] %q[%q] = %q, not one of its real numbers %v", tt.edition, cardName, key, num, numbers)
				}
			}
		}
	}
}

func TestMultiPromoVariants(t *testing.T) {
	var allKeys []string
	for _, table := range multiPromosVariants {
		for key := range table {
			for subkey := range strings.FieldsSeq(key) {
				// This is a trick just to deal with IXL and RIX aliasing
				switch subkey {
				case "prerelease", "phyrexian", "alternate":
					continue
				}
				if slices.Contains(allKeys, subkey) {
					continue
				}
				allKeys = append(allKeys, subkey)
			}
		}
	}

	var baseKeys []string
	for key := range MultiPromosTable {
		baseKeys = append(baseKeys, strings.ToLower(key))
	}

	for _, key := range allKeys {
		found := false
		for _, base := range baseKeys {
			if strings.Contains(base, key) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FAIL: [%s] is not found in MultiPromosTable", key)
		}
	}
}
