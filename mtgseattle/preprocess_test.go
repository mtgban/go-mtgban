package mtgseattle

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

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

// TestMysteryBoosterCardsResolvesToPLSTNumber pins a Mystery Booster Cards
// listing to the PLST printing the MB1 retail booster bundles for that name.
func TestMysteryBoosterCardsResolvesToPLSTNumber(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		number   string
	}{
		{"Gonti, Lord of Luxury", "KLD-84"},
		{"Krosan Verge", "C18-263"},
	}
	for _, test := range tests {
		theCard, err := preprocess(b, test.cardName, "Mystery Booster Cards", "")
		if err != nil {
			t.Fatalf("%s: preprocess: %v", test.cardName, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s: Match: %v", test.cardName, err)
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatalf("%s: GetUUID: %v", test.cardName, err)
		}
		if co.SetCode != "PLST" || co.Number != test.number {
			t.Errorf("%s: matched %s %s, want PLST %s", test.cardName, co.SetCode, co.Number, test.number)
		}
	}
}
