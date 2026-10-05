package mtgmatcher

import (
	"testing"
)

// mtgjson lists sibling printings in Variations, and some of those uuids
// belong to cards this datastore does not carry (a set filtered out at
// load, most often). Reading one back out of the UUIDs map yields a nil
// pointer rather than an empty card, so every consumer of that array has
// to check before dereferencing: MatchID walks it for any card whose
// finish does not match the request, which is an ordinary lookup.
func TestMatchIDOverAbsentVariations(t *testing.T) {
	b := realDatastore(t)

	var withAbsent, unmatched int
	for _, uuid := range b.GetUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed {
			continue
		}

		var absent bool
		for _, variation := range co.Variations {
			if _, found := b.UUIDs[variation]; !found {
				absent = true
				break
			}
		}
		if !absent {
			continue
		}
		withAbsent++

		// Ask for each finish in turn: the ones the card does not carry
		// are what send MatchID into the Variations walk
		for _, finishes := range [][]bool{{false, false}, {true, false}, {false, true}} {
			_, err = b.MatchID(uuid, finishes...)
			if err != nil {
				unmatched++
			}
		}
	}

	if withAbsent == 0 {
		t.Fatal("no card in this datastore lists an absent variation")
	}
	t.Logf("exercised %d cards listing an absent variation, %d finish requests unmatched", withAbsent, unmatched)
}

// A finish request is answered by the card's own twin, never by the
// alt-art printing in another language that shares its collector number.
func TestMatchIDFinishTwinKeepsLanguage(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		name, id string
		foil     bool
		want     string
	}{
		{"10E Drudge Skeletons", "d9902515-6292-5a4a-9bde-a7616ad5f0bc", false, "2d080e75-7e1b-5f75-9c8f-18d02a4710dd"},
		{"7ED Raise Dead", "25fdc9a4-fe3c-5a51-b09c-d2b3420ec619", true, "f7731317-6e2c-587a-80b0-304df2d2973b"},
	}
	for _, test := range tests {
		got, err := b.MatchID(test.id, test.foil)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got != test.want {
			t.Errorf("%s: MatchID(foil=%v) = %s, want %s", test.name, test.foil, got, test.want)
		}
	}

	// The Chinese Simplified 139s is not the twin of the English 139★
	english, err := b.GetUUID("d9902515-6292-5a4a-9bde-a7616ad5f0bc")
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := b.GetUUID("1bd70275-068f-5ec2-bc8e-4c5817a1e625")
	if err != nil {
		t.Fatal(err)
	}
	if finishTwins(english, chinese) {
		t.Errorf("%s %s (%s) reads as a twin of %s %s (%s)", english.SetCode, english.Number, english.Language, chinese.SetCode, chinese.Number, chinese.Language)
	}
}
