package mtgmatcher

import (
	"testing"
)

// mtgjson lists sibling printings in Variations, and some of those uuids
// belong to cards this datastore does not carry (a set filtered out at
// load, most often). Reading one back out of the UUIDs map yields a nil
// pointer rather than an empty card, so every consumer of that array has
// to check before dereferencing: MatchId walks it for any card whose
// finish does not match the request, which is an ordinary lookup.
func TestMatchIdOverAbsentVariations(t *testing.T) {
	if len(GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}

	var withAbsent int
	for _, uuid := range GetUUIDs() {
		co, err := GetUUID(uuid)
		if err != nil || co.Sealed {
			continue
		}

		var absent bool
		for _, variation := range co.Variations {
			if _, found := defaultBackend.UUIDs[variation]; !found {
				absent = true
				break
			}
		}
		if !absent {
			continue
		}
		withAbsent++

		// Ask for each finish in turn: the ones the card does not carry
		// are what send MatchId into the Variations walk
		for _, finishes := range [][]bool{{false, false}, {true, false}, {false, true}} {
			MatchId(uuid, finishes...)
		}
		if variantInCommanderDeck(&InputCard{Name: co.Name}, &co.Card) {
			continue
		}
	}

	if withAbsent == 0 {
		t.Skip("no card in this datastore lists an absent variation")
	}
	t.Logf("exercised %d cards listing an absent variation", withAbsent)
}
