package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A Secret Lair listing silent about the flavor name is the printing sold
// under the card's own name, where the drop has just one.
func TestUnflavoredPrinting(t *testing.T) {
	realDatastore(t)

	for _, tc := range []struct {
		name, variation string
		foil            bool
		want            string
	}{
		{"Laboratory Maniac", "Borderless", false, "1097"},
		{"Laboratory Maniac", "Borderless", true, "1097"},
		{"Liliana of the Dark Realms", "Borderless", false, "1107"},
		{"Liliana of the Dark Realms", "Borderless", true, "1107"},
		{"Dictate of Erebos", "Rainbow Foil", true, "1713★"},
		// Naming the flavor still reaches it
		{"Laboratory Maniac", "Chaotic Chaotician", false, "1394"},
	} {
		id, err := testBackend.Match(&mtgmatcher.InputCard{Name: tc.name, Variation: tc.variation, Edition: "Secret Lair", Foil: tc.foil})
		if err != nil {
			t.Errorf("%s (%s) foil=%v: %v", tc.name, tc.variation, tc.foil, err)
			continue
		}
		co, _ := testBackend.GetUUID(id)
		if co.Number != tc.want || co.Foil != tc.foil {
			t.Errorf("%s (%s) foil=%v: got %s, want %s", tc.name, tc.variation, tc.foil, co, tc.want)
		}
	}

	// Two printings of Windfall carry no flavor name, so silence about it
	// cannot pick between them
	id, err := testBackend.Match(&mtgmatcher.InputCard{Name: "Windfall", Variation: "Rainbow Foil", Edition: "Secret Lair", Foil: true})
	co, _ := testBackend.GetUUID(id)
	if err == nil && co.Number == "2571" {
		t.Errorf("Windfall (Rainbow Foil) landed on %s", co)
	}
}
