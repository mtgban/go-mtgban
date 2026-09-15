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
	realDatastore(t)

	// Strixhaven Mystical Archive sells one printing in all three finishes,
	// which is what lets the flag cross between two of them.
	plain := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "235648")
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

	// A derived token pairing's own combined name is deliberately excluded
	// from every name index (mtgmatcher/magic/tokenpairs.go), so the plain
	// HasFoilPrinting path below can never find one's foil sibling by name -
	// resolved instead by the pairing's own tcgplayerProductId plus the
	// foil flag. AFR's dungeon-card pairing sells in foil; a pairing where
	// one face never did (Goblin // Giant Teddy Bear, TC21) is refused
	// rather than silently kept at its nonfoil price.
	pairedFoilCapable := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "244297")
	if pairedFoilCapable == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 244297")
	}
	pairedFoilCapableFoil, err := mtgmatcher.MatchID(pairedFoilCapable, true)
	if err != nil {
		t.Fatal(err)
	}
	pairedNonfoilOnly := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, "209922")
	if pairedNonfoilOnly == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 209922")
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
		{"a derived pairing sold in foil reaches its own foil sibling", pairedFoilCapable, "Dungeon of the Mad Mage // Lost Mine of Phandelver", pairedFoilCapableFoil},
		{"a derived pairing never sold in foil is refused, not kept nonfoil", pairedNonfoilOnly, "Goblin // Giant Teddy Bear", ""},
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
