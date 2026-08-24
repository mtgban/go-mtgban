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
