package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// FinishSiblings must gather every uuid a card sells under: the registered
// finishes whichever sibling it is asked from, and the foils the old sets
// filed as cards of their own, which no FoilUUIDs map links.
func TestFinishSiblings(t *testing.T) {
	var checkedSplit, checkedTwin bool
	for _, uuid := range testBackend.AllUUIDs {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}

		if !checkedSplit {
			nonfoil, hasNonfoil := co.FoilUUIDs[mtgmatcher.FinishNonfoil]
			foil, hasFoil := co.FoilUUIDs[mtgmatcher.FinishFoil]
			if hasNonfoil && hasFoil && nonfoil != foil {
				fromBase := testBackend.FinishSiblings(nonfoil)
				fromFoil := testBackend.FinishSiblings(foil)
				if !slices.Contains(fromBase, foil) || !slices.Contains(fromFoil, nonfoil) {
					t.Errorf("split card %s: base saw %v, foil saw %v", co.Name, fromBase, fromFoil)
				}
				if len(fromBase) > 0 && fromBase[0] != nonfoil {
					t.Errorf("split card %s: siblings open with %s, want the base %s", co.Name, fromBase[0], nonfoil)
				}
				checkedSplit = true
			}
		}

		if !checkedTwin {
			// A single-finish card whose foil lives among its variations.
			if len(co.FoilUUIDs) == 1 {
				for _, variation := range co.Variations {
					altCo, err := testBackend.GetUUID(variation)
					if err != nil {
						continue
					}
					if mtgmatcher.ExtractNumberValue(co.Number) != mtgmatcher.ExtractNumberValue(altCo.Number) ||
						co.HasFinish(mtgmatcher.FinishNonfoil) == altCo.HasFinish(mtgmatcher.FinishNonfoil) {
						continue
					}
					siblings := testBackend.FinishSiblings(co.UUID)
					if !slices.Contains(siblings, altCo.UUID) {
						t.Errorf("twin card %s (%s): %v misses the separate-printing twin %s",
							co.Name, co.SetCode, siblings, altCo.UUID)
					}
					checkedTwin = true
					break
				}
			}
		}

		if checkedSplit && checkedTwin {
			break
		}
	}
	if !checkedSplit || !checkedTwin {
		t.Fatalf("datastore offered no card to check with (split %v, twin %v)", checkedSplit, checkedTwin)
	}

	// An external id enumerates the same card its conversion names.
	siblings := testBackend.FinishSiblings("272554")
	if len(siblings) < 2 {
		t.Errorf("tcg id 272554 enumerated %v, want the nonfoil and foil Hadoken bolts", siblings)
	}
}
