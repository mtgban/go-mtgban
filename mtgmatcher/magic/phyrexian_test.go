package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A Secret Lair listing naming the Phyrexian language is the printing in it,
// even where the drop sells an English one of the card too.
func TestPhyrexianPrinting(t *testing.T) {
	realDatastore(t)

	for _, tc := range []struct {
		name, variation string
		foil            bool
		want            string
	}{
		{"Inkmoth Nexus", "Phyrexian", false, "1207"},
		{"Inkmoth Nexus", "Phyrexian", true, "1207"},
		{"K'rrik, Son of Yawgmoth", "Phyrexian", false, "1204"},
		{"K'rrik, Son of Yawgmoth", "Phyrexian", true, "1204"},
		{"Elesh Norn, Grand Cenobite", "Phyrexian", false, "209"},
		{"Vorinclex, Voice of Hunger", "Phyrexian", true, "213"},
		// Silence about the language still reaches the English printing
		{"Inkmoth Nexus", "", false, "45"},
		{"K'rrik, Son of Yawgmoth", "", false, "1186"},
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
}
