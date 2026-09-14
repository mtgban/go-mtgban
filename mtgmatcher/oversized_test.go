package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMatchOversized pins which listings the word oversize marks unsupported.
// A dungeon's oversized sheet (OAFR, OCLB) is filed under its parent set's
// name exactly where the ordinary token sheet is too, so a listing naming
// the parent and saying oversized only in the variation - the common shape,
// not the shelf's own "Oversize Cards" edition - used to read as the
// ordinary token filed beside it instead: the sheet went unrecognized (no
// candidate at all before it was carried), and once carried, the token
// sheet's own edition match won the tie unchallenged. Both are pinned here.
func TestMatchOversized(t *testing.T) {
	realDatastore(t)
	for _, probe := range []struct {
		name      string
		edition   string
		variation string
		setCode   string
	}{
		// Oversized sheets naming their parent set, disambiguated from the
		// ordinary token sheet filed under the very same name
		{"Undercity // The Initiative", "Commander Legends: Battle for Baldur's Gate", "Oversized", "OCLB"},
		{"Lost Mine of Phandelver", "Adventures in the Forgotten Realms", "Oversized", "OAFR"},
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

// TestMatchOversizedShelf pins the catalog's shelf for oversized cards being
// read as a shelf. It names no set - one group stands for a dozen at once -
// so the word narrows to the sets that printed the card oversized, and the
// collector number picks among those when one of them answers to it.
func TestMatchOversizedShelf(t *testing.T) {
	realDatastore(t)
	for _, probe := range []struct {
		name      string
		variation string
		setCode   string
	}{
		// One set printed each of these oversized, so the shelf is enough
		{"Comet Storm", "76", "P10"},
		{"Wurmcoil Engine", "223", "P10"},
		{"Lightning Bolt", "M10", "P09"},
		// Two did, and the number tells them apart: Duskmourn numbers its
		// reprint 331 where the schemes sheet numbers it 4★
		{"Choose Your Demise", "4", "OE01"},
		{"When Will You Learn?", "20", "OE01"},
		// Three did, and the wording names the one
		{"Tazeem", "Release Event Promo", "DCI"},
	} {
		in := mtgmatcher.InputCard{
			Name:      probe.name,
			Edition:   "Oversize Cards",
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
			t.Errorf("Match(%v) = %s (%s %s #%s), want a printing in %s", in, id, co.SetCode, co.Card.Name, co.Number, probe.setCode)
		}
	}
}

// TestMatchOversizedShelfKeepsTheSetsWeSkip pins the shelf being read from the
// edition alone. A variation saying oversized beside a set the datastore never
// built - the championship prizes - names a printing that is not carried, and
// answering it with whichever other set printed one would price the wrong card.
func TestMatchOversizedShelfKeepsTheSetsWeSkip(t *testing.T) {
	in := mtgmatcher.InputCard{
		Name:      "Lightning Bolt",
		Edition:   "Legacy Championship",
		Variation: "Oversized",
	}
	id, err := mtgmatcher.Match(&in)
	if err == nil {
		co, _ := mtgmatcher.GetUUID(id)
		t.Errorf("Match(%v) = %s (%v), want an error: that printing is not carried", in, id, co)
	}
}
