package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestCatalogSpelling pins the Yu-Gi-Oh names this storefront misspells. Each
// one names a card the catalog has and reaches nothing as typed, so the
// listing goes unpriced; each is also checked as typed, because a pair that
// stopped being a misspelling - the catalog renames a card, the storefront
// fixes its own spelling - is a pair that should leave the table rather than
// sit there rewriting a name that now means something.
func TestCatalogSpelling(t *testing.T) {
	path := os.Getenv("YUGIOH_PATH")
	if path == "" {
		t.Skip("Need YUGIOH_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		edition   string
		variation string
	}{
		{"Belial - Marqis of Darkness", "Structure Deck Gates of the Underworld", "Common"},
		{"Compulsory Evactuation Device", "Rarity Collection 5", "Stamped Version - Ultra Rare Ultra Rare"},
		{"Fearl Imp", "Dark Beginning 1", "Common"},
		{"Homumculus the Alchemic Being", "Rise of Destiny", "Common"},
		{"Miracle Jurrassic Egg", "Structure Deck Dinosaurs Rage", "Common"},
		{"Perfect Synch - A-Un", "Phantom Rage", "Super Rare"},
		{"Rush Recklessely", "Dark Beginning 1", "Common"},
		{"Sealing Ceremony of Mokuten", "Extreme Victory", "Common"},
		{"Sealing Cermony of Raiton", "Galactic Overlord", "Common"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spelled := catalogSpelling(test.name)
			if spelled == test.name {
				t.Fatalf("catalogSpelling(%q) corrected nothing", test.name)
			}
			asTyped := &mtgmatcher.InputCard{Name: test.name, Edition: test.edition, Variation: test.variation}
			if id, err := mtgmatcher.Match(asTyped); err == nil {
				t.Errorf("Match(%q) = %q, want the name to reach nothing before it is corrected",
					test.name, id)
			}
			card := &mtgmatcher.InputCard{Name: spelled, Edition: test.edition, Variation: test.variation}
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", spelled, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.Name != spelled {
				t.Errorf("Match(%q) = %q, want %q", spelled, co.Name, spelled)
			}
		})
	}
}
