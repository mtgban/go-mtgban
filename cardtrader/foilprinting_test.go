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
	b := realDatastore(t)

	// Strixhaven Mystical Archive sells one printing in all three finishes,
	// which is what lets the flag cross between two of them.
	plain := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "235648")
	if plain == "" {
		t.Fatal("datastore carries no TCGplayer id 235648")
	}
	foil, err := b.MatchID(plain, true)
	if err != nil {
		t.Fatal(err)
	}
	etched, err := b.MatchID(plain, false, true)
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
	pairedFoilCapable := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "244297")
	if pairedFoilCapable == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 244297")
	}
	pairedFoilCapableFoil, err := b.MatchID(pairedFoilCapable, true)
	if err != nil {
		t.Fatal(err)
	}
	pairedNonfoilOnly := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "209922")
	if pairedNonfoilOnly == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 209922")
	}

	// Card Trader spells this one "Lord of Ulvenwald", which no printing
	// carries, so only the resolved card's own name finds its foil.
	dfc := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "247917")
	if dfc == "" {
		t.Fatal("datastore carries no TCGplayer id 247917")
	}
	dfcFoil, err := b.MatchID(dfc, true)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc   string
		cardID string
		want   string
	}{
		{"plain printing reaches its foil", plain, foil},
		{"a double-faced printing reaches its foil", dfc, dfcFoil},
		{"foil printing stays put", foil, foil},
		{"etched printing stays put", etched, etched},
		{"unknown id is left alone", "not-a-uuid", "not-a-uuid"},
		{"a derived pairing sold in foil reaches its own foil sibling", pairedFoilCapable, pairedFoilCapableFoil},
		{"a derived pairing never sold in foil is refused, not kept nonfoil", pairedNonfoilOnly, ""},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := foilPrintingID(b, test.cardID)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
