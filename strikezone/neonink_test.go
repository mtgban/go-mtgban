package strikezone

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore when one is configured. Some of this
// package's suites read no cards, so a checkout without it still runs them;
// the ones that do ask say so and skip.
var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		backend, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = backend
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestNeonInkWording pins the four Neon Ink colours to their own printings.
// The buylist writes them without the word the treatment is named for, and
// that shorter wording names no treatment at all: every colour answered with
// the plain printing that stands beside them, which prices a $300 card at the
// bulk one's id.
func TestNeonInkWording(t *testing.T) {
	b := realDatastore(t)

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
			card, err := preprocess(b, test.name, "Kamigawa Neon Dynasty", "")
			if err != nil {
				t.Fatal(err)
			}
			card.Foil = true
			id, err := b.Match(card)
			if err != nil {
				t.Fatal(err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "NEO" || co.Number != test.number {
				t.Errorf("got %s #%s, want NEO #%s", co.SetCode, co.Number, test.number)
			}
		})
	}
}
