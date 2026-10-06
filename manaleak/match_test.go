package manaleak

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

func TestMatchAmbiguousID(t *testing.T) {
	b := realDatastore(t)
	ml := NewScraper(b)

	tests := []struct {
		name    string
		product MLProduct
		set     string
		number  string
		finish  string
	}{
		{
			name:    "double-faced card by its front face",
			product: MLProduct{Name: "Bruna, The Fading Light", TCGProductID: "119686", AmbiguousID: true},
			set:     "EMN", number: "15", finish: "nonfoil",
		},
		{
			name:    "double-faced card, foil",
			product: MLProduct{Name: "Bruna, The Fading Light - Foil", TCGProductID: "119686", AmbiguousID: true},
			set:     "EMN", number: "15", finish: "foil",
		},
		{
			name:    "misspelled name",
			product: MLProduct{Name: "Dread Sanctuary", TCGProductID: "121810", AmbiguousID: true},
			set:     "CN2", number: "217", finish: "nonfoil",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := ml.match(tc.product)
			if err != nil {
				t.Fatal(err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tc.set || co.Number != tc.number || co.Finish != tc.finish {
				t.Errorf("landed on %s %s %s, want %s %s %s",
					co.SetCode, co.Number, co.Finish, tc.set, tc.number, tc.finish)
			}
		})
	}
}

func TestNameAgrees(t *testing.T) {
	b := realDatastore(t)
	ml := NewScraper(b)

	id := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "119686")
	cardID, err := b.MatchID(id, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !ml.nameAgrees(cardID, "Bruna, The Fading Light") {
		t.Error("a double-faced card's front face should agree")
	}
	if ml.nameAgrees(cardID, "Lightning Bolt") {
		t.Error("an unrelated name should not agree")
	}
	if ml.nameAgrees(cardID, "") {
		t.Error("an empty name should not agree")
	}
}
