package mtgmatcher

import (
	"strings"
	"testing"

	"golang.org/x/exp/slices"
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

func TestMultiPromoVariants(t *testing.T) {
	var allKeys []string
	for _, table := range multiPromosVariants {
		for key := range table {
			for _, subkey := range strings.Fields(key) {
				// This is a trick just to deal with IXL and RIX aliasing
				if subkey == "prerelease" {
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
	for key := range multiPromosTable {
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
			t.Errorf("FAIL: [%s] is not found in multiPromosTable", key)
		}
	}
}
