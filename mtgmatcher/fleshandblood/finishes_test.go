package fleshandblood

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPrintingUUIDs walks every product and asks each printing it carries
// for the uuid pricing it. Each has to answer with the entry stamped with
// that combination, no two printings of a product and no two products may
// answer with one uuid, and every stored entry has to be some product's
// printing - a combination silently overwritten in FoilUUIDs would leave a
// uuid nothing can reach. The flag slots carry a printing the product is
// really sold in.
func TestPrintingUUIDs(t *testing.T) {
	b := loadBackend(t)

	owner := map[string]string{}
	var products, printings int
	for _, code := range b.AllSets {
		for _, card := range b.Sets[code].Cards {
			products++

			printingOf := map[string]string{}
			for key, target := range card.FoilUUIDs {
				// A flag slot names a printing sold under another name;
				// the plain printing is keyed nonfoil itself.
				if key == mtgmatcher.FinishNonfoil || key == mtgmatcher.FinishFoil {
					if co, err := b.GetUUID(target); err == nil && co.Finish != key {
						continue
					}
				}
				if other, found := printingOf[target]; found {
					t.Errorf("%s: finishes %q and %q share uuid %s", card.UUID, other, key, target)
				}
				printingOf[target] = key
				if prev, found := owner[target]; found && prev != card.UUID {
					t.Errorf("uuid %s answers for products %s and %s", target, prev, card.UUID)
				}
				owner[target] = card.UUID

				co, err := b.GetUUID(target)
				if err != nil {
					t.Errorf("%s: finish %q names unknown uuid %s", card.UUID, key, target)
					continue
				}
				if co.Finish != key {
					t.Errorf("%s: finish %q names uuid %s carrying finish %q",
						card.UUID, key, target, co.Finish)
				}
				got, err := b.MatchIDFinish(card.UUID, key)
				if err != nil || got != target {
					t.Errorf("MatchIDFinish(%s, %q) = (%q, %v), want %q",
						card.UUID, key, got, err, target)
				}
				printings++
			}

			// Only the foilness classes the product is sold in are
			// registered, and the slot has to land on a printing it carries.
			for _, slot := range []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil} {
				target := card.FoilUUIDs[slot]
				if target == "" {
					continue
				}
				if _, found := printingOf[target]; !found {
					t.Errorf("%s: flag slot %q names uuid %s that no finish carries",
						card.UUID, slot, target)
				}
			}
			if len(card.Finishes) == 0 {
				t.Errorf("%s: product is sold in no foilness class at all", card.UUID)
			}

		}
	}

	var orphans int
	for uuid, co := range b.UUIDs {
		if co.Sealed {
			continue
		}
		if _, found := owner[uuid]; !found {
			orphans++
			t.Errorf("uuid %s carries finish %q but no product names it", uuid, co.Finish)
		}
	}
	if products == 0 || printings == 0 {
		t.Fatalf("datastore covers only part of the table: %d products, %d printings",
			products, printings)
	}
	t.Logf("%d products, %d printings, %d orphaned entries", products, printings, orphans)
}
