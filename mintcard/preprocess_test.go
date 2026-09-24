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
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore loads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		b, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = b
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestPreprocessInserts pins that the inserts a booster carries beside its
// cards - theme cards, the helper cards of a set the datastore files none
// for - and the emblems and signed cards are refused quietly: 364 of the
// 411 lines of the night of 2026-09-06.
func TestPreprocessInserts(t *testing.T) {
	// No datastore lookup is reached on this path, so an empty backend is
	// enough.
	b := &mtgmatcher.Backend{}
	for _, name := range []string{
		"Angels Theme Card",
		"Helper Card",
		"Emblem Garruk, Caller of Beasts",
		"Chrome Mox (Manuel Bevand Signature)",
	} {
		_, err := preprocess(b, name, "", "Regular", "English", "Foundations Jumpstart", "J25")
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("preprocess(%q) = %v, want ErrUnsupported", name, err)
		}
	}
}

// TestPreprocessShelves pins the storefront's own spellings against the
// datastore's: a duel deck code of its own, a Final Fantasy buy-a-box
// promo listed under its flavor name with the card's own in parentheses,
// the helper card of a set whose substitute cards the datastore files, the
// two cards whose names read like inserts, and a Phyrexian printing filed
// as English beside a List card that only names New Phyrexia.
func TestPreprocessShelves(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		name, edition, code, language string
		wantSet, wantNumber           string
	}{
		{"Coalition Relic", "Duel Decks: Phyrexia vs. Coalition", "PVC", "English", "DDE", "54"},
		{"Fatalism (Arcane Denial) (Final Fantasy Buy-a-Box)", "Promo: Buy-A-Box", "PBAB", "Japanese", "RFIN", "J2"},
		{"Helper Card (9/9)", "Kaldheim", "KHM", "English", "SKHM", "9"},
		{"Signature Slam", "Modern Horizons 3", "MH3", "English", "MH3", "168"},
		{"Emblem of the Warmind", "Future Sight", "FUT", "English", "FUT", "112"},
		{"Plains (267) (Phyrexian)", "Phyrexia: All Will Be One", "ONE", "English", "ONE", "267"},
		{"Beast Within (New Phyrexian)", "Mystery Booster/The List", "MYS", "English", "PLST", "NPH-103"},
	} {
		theCard, err := preprocess(b, tt.name, "", "Regular", tt.language, tt.edition, tt.code)
		if err != nil {
			t.Fatalf("preprocess(%q) = %v", tt.name, err)
		}
		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("Match(%q) = %v", theCard, err)
		}
		co, _ := b.GetUUID(cardID)
		if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
			t.Errorf("Match(%q) = %s %s, want %s %s", theCard, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
		}
	}
}

// TestPreprocessLanguage pins that the feed's blanket Language "English" is
// dropped before it reaches mtgmatcher: an Arabic prerelease foil shelved
// under that tag, with no language of its own beside "English" to pass,
// would otherwise demand an English candidate and refuse.
func TestPreprocessLanguage(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Stone-Tongue Basilisk (Odyssey Prerelease)(Arabic)", "", "Foil Prerelease", "English", "Odyssey", "ODY")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "PODY" || co.Number != "276" || !co.Foil {
		t.Errorf("Match(%q) = %s %s foil=%v, want PODY 276 foil", theCard, co.SetCode, co.Number, co.Foil)
	}
}

// TestPreprocessSignatureSpellbook pins that a Signature Spellbook card, a
// real set, escapes the insert guard the word "Signature" otherwise trips.
func TestPreprocessSignatureSpellbook(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Brainstorm (Signature Spellbook: Jace)", "", "Regular", "English", "Mystery Booster/The List", "MYS")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "PLST" || co.Number != "SS1-3" {
		t.Errorf("Match(%q) = %s %s, want PLST SS1-3", theCard, co.SetCode, co.Number)
	}
}

// TestPreprocessTokenFlavorSwap pins that a token's own parenthetical names
// the set it comes from, not a flavor: "Goblin Soldier Token (Apocalypse)"
// keeps its own name rather than swapping to Apocalypse, also a card.
func TestPreprocessTokenFlavorSwap(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Goblin Soldier Token (Apocalypse) (Player Rewards)", "", "Regular", "English", "Promo: Magic Player Rewards", "PMPR")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	if theCard.Name != "Goblin Soldier Token" {
		t.Errorf("preprocess() name = %q, want %q", theCard.Name, "Goblin Soldier Token")
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "MPR" || co.Number != "6" {
		t.Errorf("Match(%q) = %s %s, want MPR 6", theCard, co.SetCode, co.Number)
	}
}
