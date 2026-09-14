package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMatchAltArtNotTheList pins Ertai, the Corrupted's alternate-art
// printing (Planeshift's booster-foil mechanic, told apart from the plain
// copy only by a starred number) resolving to its own set rather than
// aliasing against The List's plain reprint, which cannot represent the
// distinction at all. A listing that does not ask for the alternate art,
// or that names The List outright, is unaffected either way.
func TestMatchAltArtNotTheList(t *testing.T) {
	realDatastore(t)
	for _, probe := range []struct {
		name      string
		edition   string
		variation string
		foil      bool
		setCode   string
		number    string
	}{
		{"Ertai, the Corrupted", "Planeshift", "Alt. Art Foil", true, "PLS", "107★"},
		{"Ertai, the Corrupted", "Planeshift", "", false, "PLS", "107"},
		{"Ertai, the Corrupted", "The List", "", false, "PLST", "PLS-107"},
		// Planeshift's other two starred pairs, same mechanic, no List
		// reprint to collide with - pinned so a future one does not regress
		{"Tahngarth, Talruum Hero", "Planeshift", "Alt. Art Foil", true, "PLS", "74★"},
		{"Skyship Weatherlight", "Planeshift", "Alt. Art Foil", true, "PLS", "133★"},
	} {
		in := mtgmatcher.InputCard{
			Name:      probe.name,
			Edition:   probe.edition,
			Variation: probe.variation,
			Foil:      probe.foil,
		}
		id, err := mtgmatcher.Match(&in)
		if err != nil {
			t.Errorf("Match(%v) = %v", in, err)
			continue
		}
		co, err := mtgmatcher.GetUUID(id)
		if err != nil {
			t.Errorf("GetUUID(%s) = %v", id, err)
			continue
		}
		if co.SetCode != probe.setCode || co.Number != probe.number {
			t.Errorf("Match(%v) = %s (%s #%s), want %s #%s", in, id, co.SetCode, co.Number, probe.setCode, probe.number)
		}
	}
}
