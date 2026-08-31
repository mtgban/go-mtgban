package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFoilPrintingID pins which listings the foil flag may move. CardTrader
// files a blueprint's foil listings under the plain printing's id, so the
// flag has to reach the foil sibling; an etched listing raises the very same
// flag, and answering it there would file the etched price under the foil
// printing. The ids are drawn from the datastore so the test holds across
// its releases.
func TestFoilPrintingID(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the datastore-backed cases")
	}

	// Strixhaven Mystical Archive sells one printing in all three finishes,
	// which is what lets the flag cross between two of them.
	plain := mtgmatcher.ExternalUUID("235648")
	if plain == "" {
		t.Fatal("datastore carries no TCGplayer id 235648")
	}
	foil, err := mtgmatcher.MatchID(plain, true)
	if err != nil {
		t.Fatal(err)
	}
	etched, err := mtgmatcher.MatchID(plain, false, true)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc   string
		cardID string
		name   string
		want   string
	}{
		{"plain printing reaches its foil", plain, "Tainted Pact", foil},
		{"foil printing stays put", foil, "Tainted Pact", foil},
		{"etched printing stays put", etched, "Tainted Pact", etched},
		{"unknown id is left alone", "not-a-uuid", "Tainted Pact", "not-a-uuid"},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := foilPrintingID(test.cardID, test.name)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
