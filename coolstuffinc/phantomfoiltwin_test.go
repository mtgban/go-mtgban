package coolstuffinc

import (
	"testing"
)

// TestMagicPhantomFoilTwin pins the preference, not a blanket skip. See
// magicPhantomFoilTwin and parseBL.
func TestMagicPhantomFoilTwin(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	// Luxury Suite (Marvel Super Heroes Commander) - MSC 482, nonfoil only.
	const uuid = "2eeef31e-a769-5ea5-a0c5-5affcc82b92b"
	co, err := b.GetUUID(uuid)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", uuid, err)
	}
	if co.Foil || co.Etched {
		t.Fatalf("fixture printing is no longer nonfoil-only: %q", co)
	}

	if !magicPhantomFoilTwin(co, 1, true) {
		t.Error("a foil-flagged row on a nonfoil-only printing, with a nonfoil sibling, should be skipped")
	}
	if magicPhantomFoilTwin(co, 1, false) {
		t.Error("a foil-flagged row with no nonfoil sibling is the only listing CSI publishes and must stand")
	}
	if magicPhantomFoilTwin(co, 0, true) {
		t.Error("a nonfoil row is never the phantom twin")
	}
}
