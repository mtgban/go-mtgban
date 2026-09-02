package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMatchOversized pins which listings the word oversize marks unsupported.
// The sets naming an oversized product used to be recognized by the words in
// their titles, so "Commander Legends: Battle for Baldur's Gate" read as an
// oversized Commander product and its dungeon was priced as the ordinary
// token filed beside it.
func TestMatchOversized(t *testing.T) {
	for _, probe := range []struct {
		name      string
		edition   string
		variation string
		setCode   string
	}{
		// The oversized sheets the datastore never built
		{"Undercity // The Initiative", "Commander Legends: Battle for Baldur's Gate", "Oversized", ""},
		{"Lost Mine of Phandelver", "Adventures in the Forgotten Realms", "Oversized", ""},
		// The oversized printings it does carry, however they are addressed:
		// by the set the sheet was printed beside, and by a set name the
		// storefront leaves the year off
		{"All in Good Time", "Archenemy", "Oversized", "OARC"},
		{"Feeding Grounds", "Planechase", "Plane Oversized", "OHOP"},
		{"Lightning Bolt", "Magic Player Rewards", "Oversize", "P09"},
	} {
		in := mtgmatcher.InputCard{
			Name:      probe.name,
			Edition:   probe.edition,
			Variation: probe.variation,
		}
		id, err := mtgmatcher.Match(&in)
		if probe.setCode == "" {
			if err == nil {
				co, _ := mtgmatcher.GetUUID(id)
				t.Errorf("Match(%v) = %s (%v), want an error: no oversized printing is carried", in, id, co)
			}
			continue
		}
		if err != nil {
			t.Errorf("Match(%v) = %v", in, err)
			continue
		}
		co, err := mtgmatcher.GetUUID(id)
		if err != nil {
			t.Errorf("GetUUID(%s) = %v", id, err)
			continue
		}
		if co.SetCode != probe.setCode {
			t.Errorf("Match(%v) = %s (%s %s), want a printing in %s", in, id, co.SetCode, co.Card.Name, probe.setCode)
		}
	}
}
