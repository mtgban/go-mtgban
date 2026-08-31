package strikezone

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastoretest"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestNeonInkWording pins the four Neon Ink colours to their own printings.
// The buylist writes them without the word the treatment is named for, and
// that shorter wording names no treatment at all: every colour answered with
// the plain printing that stands beside them, which prices a $300 card at the
// bulk one's id.
func TestNeonInkWording(t *testing.T) {
	datastorePath := os.Getenv("ALLPRINTINGS5_PATH")
	if datastorePath == "" {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}
	reader, err := datastoretest.Open(datastorePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	backend, err := magic.Load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(backend)

	tests := []struct {
		name   string
		number string
	}{
		{"Hidetsugu Devouring Chaos (Neon Red)", "429"},
		{"Hidetsugu Devouring Chaos (Neon Green)", "430"},
		{"Hidetsugu Devouring Chaos (Neon Blue)", "431"},
		{"Hidetsugu Devouring Chaos (Neon Yellow) (WPN Exclusive)", "432"},
		{"Hidetsugu Devouring Chaos", "99"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card, err := preprocess(test.name, "Kamigawa Neon Dynasty", "")
			if err != nil {
				t.Fatal(err)
			}
			card.Foil = true
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatal(err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "NEO" || co.Number != test.number {
				t.Errorf("got %s #%s, want NEO #%s", co.SetCode, co.Number, test.number)
			}
		})
	}
}
