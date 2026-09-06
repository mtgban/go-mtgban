package cardkingdom

import (
	"errors"
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPreprocessPunchCard pins that a punch card reaches the Punchcard row
// of its sheet under the name the datastore files it with, and that a
// sheet without one refuses the row quietly.
func TestPreprocessPunchCard(t *testing.T) {
	theCard, err := Preprocess(cardkingdom.Product{
		SKU:     "THOU-014",
		Name:    "Hour of Devastation Punch Card",
		Edition: "Hour of Devastation",
	})
	if err != nil {
		t.Fatal(err)
	}
	cardID, err := mtgmatcher.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	co, _ := mtgmatcher.GetUUID(cardID)
	if co.Name != "Punchcard" || co.SetCode != "THOU" {
		t.Errorf("Match(%v) = %v, want the THOU Punchcard", theCard, co)
	}

	_, err = Preprocess(cardkingdom.Product{
		SKU:       "TNCC-006X",
		Name:      "Streets of New Capenna Commander Punch Card",
		Variation: "006 // 005",
		Edition:   "Streets of New Capenna Commander Decks",
	})
	if !errors.Is(err, mtgmatcher.ErrUnsupported) {
		t.Errorf("Preprocess of a punch card the sheet does not hold = %v, want ErrUnsupported", err)
	}
}
