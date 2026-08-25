package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMatchTokenSetEdition pins the edition gate admitting an edition that
// names a carried token set: the token is filed right there, so the edition
// is not a leak to be refused but the exact address of the printing.
func TestMatchTokenSetEdition(t *testing.T) {
	in := mtgmatcher.InputCard{
		Name:      "Wolf",
		Edition:   "Innistrad: Midnight Hunt Tokens",
		Variation: "13",
	}
	id, err := mtgmatcher.Match(&in)
	if err != nil {
		t.Fatalf("Match(%v) = %v", in, err)
	}
	co, err := mtgmatcher.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if co.SetCode != "TMID" || co.Card.Name != "Wolf" {
		t.Errorf("Match(%v) = %s (%v), want the TMID Wolf token", in, id, co)
	}
}

// TestParseCommanderEditionKeepsTokenSets pins the guard that stops a
// carried token set's own name being parsed down to the commander set it
// stems from, which would lose the tokens filed under it.
func TestParseCommanderEditionKeepsTokenSets(t *testing.T) {
	for _, edition := range []string{
		"Commander 2019 Tokens",
		"March of the Machine Commander Tokens",
	} {
		if got := testBackend.ParseCommanderEdition(edition, ""); got != "" {
			t.Errorf("ParseCommanderEdition(%q) = %q, want it left alone", edition, got)
		}
	}
}

// TestMatchTokenSetParentPicksNamedSheet pins how a parent set's name is
// read when the set printed more than one token sheet: the sheet it names as
// its own answers first, and a sheet it merely stems from answers only for
// the tokens the named one never carried.
func TestMatchTokenSetParentPicksNamedSheet(t *testing.T) {
	for _, probe := range []struct {
		name    string
		edition string
		setCode string
	}{
		// Dominaria United printed TDMU, PTDMU and WDMU, and all three
		// carry an Angel
		{"Angel", "Dominaria United", "TDMU"},
		{"Soldier", "Dominaria United", "TDMU"},
		{"Zombie", "Dominaria United", "TDMU"},
		// TWOE has no Goblin and TMKM no Treasure, so the promo sheet is
		// the only address left
		{"Goblin", "Wilds of Eldraine", "WWOE"},
		{"Treasure", "Murders at Karlov Manor", "WMKM"},
	} {
		in := mtgmatcher.InputCard{
			Name:    probe.name,
			Edition: probe.edition,
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
		if co.SetCode != probe.setCode || co.Card.Name != probe.name {
			t.Errorf("Match(%v) = %s (%s %s), want the %s %s", in, id, co.SetCode, co.Card.Name, probe.setCode, probe.name)
		}
	}
}
