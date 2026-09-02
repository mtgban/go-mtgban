package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The identifiers are shared by every finish sibling, so ConvertID's answer
// is a contract, not an accident of load order: a shared id resolves to the
// base sibling, and the etched product id to the etched one.
func TestConvertIDFilesTheBaseSibling(t *testing.T) {
	var checkedBase, checkedEtched bool
	for _, uuid := range testBackend.AllUUIDs {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}

		if !checkedBase {
			scryfallID := co.Identifiers["scryfallId"]
			nonfoil, hasNonfoil := co.FoilUUIDs[mtgmatcher.FinishNonfoil]
			foil, hasFoil := co.FoilUUIDs[mtgmatcher.FinishFoil]
			if scryfallID != "" && hasNonfoil && hasFoil && nonfoil != foil {
				got := testBackend.ConvertID(mtgmatcher.IDSpaceScryfall, scryfallID)
				if got != nonfoil {
					t.Errorf("shared scryfall id %s resolved to %s, want the nonfoil sibling %s", scryfallID, got, nonfoil)
				}
				checkedBase = true
			}
		}

		if !checkedEtched {
			etchedID := co.Identifiers["tcgplayerEtchedProductId"]
			etched, hasEtched := co.FoilUUIDs[mtgmatcher.FinishEtched]
			nonfoil, hasNonfoil := co.FoilUUIDs[mtgmatcher.FinishNonfoil]
			if etchedID != "" && hasEtched && hasNonfoil && etched != nonfoil {
				got := testBackend.ConvertID(mtgmatcher.IDSpaceTCGplayer, etchedID)
				if got != etched {
					t.Errorf("etched product id %s resolved to %s, want the etched sibling %s", etchedID, got, etched)
				}
				checkedEtched = true
			}
		}

		if checkedBase && checkedEtched {
			break
		}
	}
	if !checkedBase || !checkedEtched {
		t.Fatalf("datastore offered no card to check with (base %v, etched %v)", checkedBase, checkedEtched)
	}

	// An id asked for in the wrong space stays unknown.
	if uuid := testBackend.ConvertID(mtgmatcher.IDSpaceScryfall, "272554"); uuid != "" {
		t.Errorf("a TCGplayer id resolved through the scryfall space, to %q", uuid)
	}
}
