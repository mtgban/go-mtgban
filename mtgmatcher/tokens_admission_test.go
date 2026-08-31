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

// TestMatchTokenSetVariation pins the word token reading the same wherever a
// storefront writes it. The rule refusing it fires on the edition and on the
// variation alike, so asking only the edition whether its tokens are carried
// left every row that named the set plainly and said token beside it refused.
func TestMatchTokenSetVariation(t *testing.T) {
	for _, probe := range []struct {
		name      string
		edition   string
		variation string
		setCode   string
	}{
		{"Bat Token", "Bloomburrow", "Token", "TBLB"},
		{"Angel", "Dominaria United", "Token", "TDMU"},
		{"Wolf", "Innistrad: Midnight Hunt", "Token 13", "TMID"},
	} {
		in := mtgmatcher.InputCard{
			Name:      probe.name,
			Edition:   probe.edition,
			Variation: probe.variation,
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
		if co.SetCode != probe.setCode {
			t.Errorf("Match(%v) = %s (%s %s), want a printing in %s", in, id, co.SetCode, co.Card.Name, probe.setCode)
		}
	}
}

// TestMatchTokenNameUnderTokenEdition pins which of two names sharing a
// normalized form a token edition means. A token carrying no Token in its
// name is filed under a key that says it, so the plain key is left to the
// card that normalizes the same way - "Bat" reached the Unsanctioned "Bat-"
// and "Rhino" the Unstable "Rhino-", whatever edition asked.
func TestMatchTokenNameUnderTokenEdition(t *testing.T) {
	for _, probe := range []struct {
		name      string
		edition   string
		variation string
		setCode   string
	}{
		// The token the edition names, not the card sharing its bucket
		{"Bat", "Bloomburrow Tokens", "10", "TBLB"},
		{"Rhino", "Return to Ravnica Tokens", "9", "TRTR"},
		{"Cat Warrior", "Dominaria United Tokens", "17", "TDMU"},
		// The cards those buckets hold stay reachable by their own names
		{"Bat-", "Unsanctioned", "32", "UND"},
		{"Rhino-", "Unstable", "18", "UST"},
		{"Cat Warriors", "Legends", "", "LEG"},
	} {
		in := mtgmatcher.InputCard{
			Name:      probe.name,
			Edition:   probe.edition,
			Variation: probe.variation,
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
		if co.SetCode != probe.setCode {
			t.Errorf("Match(%v) = %s (%s %s), want a printing in %s", in, id, co.SetCode, co.Card.Name, probe.setCode)
		}
	}
}

// TestMatchTokenNameKeepsTheEditionFilter pins the other direction: answering
// with the token key skips the name fixup, and it is that fixup's " Token"
// suffix which asks for the edition filter further down. Without asking for
// the filter here, a token carrying a single printing was served for whatever
// token edition a listing named.
func TestMatchTokenNameKeepsTheEditionFilter(t *testing.T) {
	for _, probe := range []struct {
		name    string
		edition string
	}{
		{"Aetherborn", "Bloomburrow Tokens"},
		{"Acorn Stash", "Bloomburrow Tokens"},
		{"Alien Angel", "Bloomburrow Tokens"},
		{"Bat", "Doctor Who Tokens"},
		{"Rhino", "Dominaria United Tokens"},
	} {
		in := mtgmatcher.InputCard{Name: probe.name, Edition: probe.edition}
		id, err := mtgmatcher.Match(&in)
		if err == nil {
			co, _ := mtgmatcher.GetUUID(id)
			t.Errorf("Match(%v) = %s (%v), want an error: no such token is filed there", in, id, co)
		}
	}
}
