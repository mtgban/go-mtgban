package tcgplayer

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCrossSetProductIDs pins the id-vs-set collisions this datastore
// carries, so a fix or a fresh upstream id assignment shows up here rather
// than silently changing what the price scraper skips.
func TestCrossSetProductIDs(t *testing.T) {
	realDatastore(t)

	collisions := crossSetProductIDs()

	want := map[string][]string{
		// AFR's ordinary Dungeon of the Mad Mage is filed under its
		// derived token set (TAFR), not AFR itself, once loaded - the
		// same reclassification every Dungeon-type card gets.
		"245106": {"OAFR", "TAFR"},
		"38221":  {"PJSE", "PSUS"},
	}
	for id, sets := range want {
		got, ok := collisions[id]
		if !ok {
			t.Errorf("crossSetProductIDs()[%s] missing, want sets %v", id, sets)
			continue
		}
		if len(got) != len(sets) {
			t.Errorf("crossSetProductIDs()[%s] = %v, want %v", id, got, sets)
			continue
		}
		for i := range sets {
			if got[i] != sets[i] {
				t.Errorf("crossSetProductIDs()[%s] = %v, want %v", id, got, sets)
				break
			}
		}
	}

	// A double-faced card's two faces share one id and one set - not a
	// collision this guard should ever flag.
	co, err := mtgmatcher.GetUUID("d514cf1d-cbbb-5c26-8d17-83439facfe19")
	if err != nil {
		t.Fatal(err)
	}
	dfcID := co.Identifiers["tcgplayerProductId"]
	if sets, found := collisions[dfcID]; found {
		t.Errorf("crossSetProductIDs()[%s] = %v, want no entry: a DFC's own faces share a set", dfcID, sets)
	}
}
