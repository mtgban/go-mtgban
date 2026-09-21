package magic

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPunchcardIsACard pins the one-word Punchcard the token sheets carry
// against the two-word "Punch Card" insert vendors sell.
//
// The clause refusing the insert was written with mtgmatcher.Contains, which
// drops spaces before comparing, so the two wordings read alike and all
// thirteen real rows were swallowed with the insert. The refusal is silent,
// so nothing in a run's log said so.
func TestPunchcardIsACard(t *testing.T) {
	realDatastore(t)

	// Every sheet the datastore files a punch card on, with the number it
	// files it at. A sheet carrying two, as Streets of New Capenna's does,
	// cannot be told apart by name alone and is left to the id.
	for _, tt := range []struct{ setCode, number string }{
		{"TAKH", "26"},
		{"TC20", "20"},
		{"TDSK", "19"},
		{"TECL", "13"},
		{"TEOE", "12"},
		{"TFIN", "37"},
		{"THOU", "13"},
		{"TIKO", "14"},
		{"TLCI", "19"},
		{"TMKM", "22"},
		{"TSOS", "14"},
	} {
		t.Run(tt.setCode, func(t *testing.T) {
			assertNumber(t, mtgmatcher.InputCard{
				Name: "Punchcard", Edition: tt.setCode,
			}, tt.setCode, tt.number)
		})
	}

	// The wording the inserts are sold under still reaches no printing:
	// Cardmarket's own listings are titled this way and none of them is a
	// card.
	for _, name := range []string{
		"Punch Card",
		"Amonkhet Punch Card",
		"Streets of New Capenna Commander Punch Card",
	} {
		t.Run(name, func(t *testing.T) {
			in := mtgmatcher.InputCard{Name: name, Edition: "Amonkhet"}
			_, err := testBackend.Match(&in)
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				t.Errorf("Match(%q) = %v, want ErrUnsupported", name, err)
			}
		})
	}
}
