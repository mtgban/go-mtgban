package mintcard

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

func TestMain(m *testing.M) {
	if path := os.Getenv("ALLPRINTINGS5_PATH"); path != "" {
		if err := datastore.Load(path); err != nil {
			log.Fatalln(err)
		}
	}
	os.Exit(m.Run())
}

// TestPreprocessInserts pins that the inserts a booster carries beside its
// cards - theme cards, helper cards - and the emblems and signed cards are
// refused quietly: 364 of the 411 lines of the night of 2026-09-06.
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
// datastore's: a duel deck code of its own, and a Final Fantasy buy-a-box
// promo listed under its flavor name with the card's own in parentheses.
func TestPreprocessShelves(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set")
	}
	for _, tt := range []struct {
		name, edition, code, language string
		wantSet, wantNumber           string
	}{
		{"Coalition Relic", "Duel Decks: Phyrexia vs. Coalition", "PVC", "English", "DDE", "54"},
		{"Fatalism (Arcane Denial) (Final Fantasy Buy-a-Box)", "Promo: Buy-A-Box", "PBAB", "Japanese", "RFIN", "J2"},
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
