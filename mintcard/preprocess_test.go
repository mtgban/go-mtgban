package mintcard

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var (
	datastoreOnce sync.Once
	datastoreErr  error
	datastoreOK   bool
)

// realDatastore installs the Magic datastore the first time a test asks for
// it, and skips where the run carries none.
func realDatastore(t *testing.T) {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		err := datastore.Load("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreOK = true
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if !datastoreOK {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
}

// TestPreprocessInserts pins that the inserts a booster carries beside its
// cards - theme cards, the helper cards of a set the datastore files none
// for - and the emblems and signed cards are refused quietly: 364 of the
// 411 lines of the night of 2026-09-06.
func TestPreprocessInserts(t *testing.T) {
	for _, name := range []string{
		"Angels Theme Card",
		"Helper Card",
		"Emblem Garruk, Caller of Beasts",
		"Chrome Mox (Manuel Bevand Signature)",
	} {
		_, err := preprocess(name, "", "Regular", "English", "Foundations Jumpstart", "J25")
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("preprocess(%q) = %v, want ErrUnsupported", name, err)
		}
	}
}

// TestPreprocessShelves pins the storefront's own spellings against the
// datastore's: a duel deck code of its own, a Final Fantasy buy-a-box
// promo listed under its flavor name with the card's own in parentheses,
// the helper card of a set whose substitute cards the datastore files, and
// the two cards whose names read like inserts.
func TestPreprocessShelves(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		name, edition, code, language string
		wantSet, wantNumber           string
	}{
		{"Coalition Relic", "Duel Decks: Phyrexia vs. Coalition", "PVC", "English", "DDE", "54"},
		{"Fatalism (Arcane Denial) (Final Fantasy Buy-a-Box)", "Promo: Buy-A-Box", "PBAB", "Japanese", "RFIN", "J2"},
		{"Helper Card (9/9)", "Kaldheim", "KHM", "English", "SKHM", "9"},
		{"Signature Slam", "Modern Horizons 3", "MH3", "English", "MH3", "168"},
		{"Emblem of the Warmind", "Future Sight", "FUT", "English", "FUT", "112"},
	} {
		theCard, err := preprocess(tt.name, "", "Regular", tt.language, tt.edition, tt.code)
		if err != nil {
			t.Fatalf("preprocess(%q) = %v", tt.name, err)
		}
		cardID, err := mtgmatcher.Match(theCard)
		if err != nil {
			t.Fatalf("Match(%q) = %v", theCard, err)
		}
		co, _ := mtgmatcher.GetUUID(cardID)
		if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
			t.Errorf("Match(%q) = %s %s, want %s %s", theCard, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
		}
	}
}
