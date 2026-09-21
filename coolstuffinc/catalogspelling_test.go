package coolstuffinc

import (
	"strings"
	"testing"

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
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	tests := []struct {
		name      string
		edition   string
		variation string
	}{
		{"Belial - Marqis of Darkness", "Structure Deck Gates of the Underworld", "Common"},
		{"Compulsory Evactuation Device", "Rarity Collection 5", "Stamped Version - Ultra Rare Ultra Rare"},
		{"Doube-Edged Sword Technique", "Structure Deck Samurai Warlords", "Common"},
		{"Fearl Imp", "Dark Beginning 1", "Common"},
		{"Fiendish Engine Ω", "Legendary Collection 4", "Common"},
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
			if id, err := b.Match(asTyped); err == nil {
				t.Errorf("Match(%q) = %q, want the name to reach nothing before it is corrected",
					test.name, id)
			}
			card := &mtgmatcher.InputCard{Name: spelled, Edition: test.edition, Variation: test.variation}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", spelled, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.Name != spelled {
				t.Errorf("Match(%q) = %q, want %q", spelled, co.Name, spelled)
			}
		})
	}
}

// TestCatalogSpellingBracketed covers the listings that hang the printing
// they mean behind the name, where the name in front of the bracket is typed
// the same wrong way. The correction has to reach the head of the line, and
// has to leave alone a head no pair in the table names - including the head
// of a card the catalog really does write with a bracket.
func TestCatalogSpellingBracketed(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			"Compulsory Evactuation Device (Stamped Version Ultra Rare)",
			"Compulsory Evacuation Device (Stamped Version Ultra Rare)",
		},
		{"Raigeki (No Stamp Ultimate Rare)", "Raigeki (No Stamp Ultimate Rare)"},
		{"Number 39: Utopia (Astral Language)", "Number 39: Utopia (Astral Language)"},
		{"Compulsory Evacuation Device", "Compulsory Evacuation Device"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := catalogSpelling(test.name); got != test.want {
				t.Errorf("catalogSpelling(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

// TestCatalogSpellingHeads states what lets the correction read a head: no
// name the catalog carries begins with a pair the table corrects, so a head
// that matches one is the typo and never the opening of a longer name.
func TestCatalogSpellingHeads(t *testing.T) {
	b := readGameDatastore(t, "yugioh", "YUGIOH_PATH")

	for typed := range csiSpellings {
		for _, code := range b.GetAllSets() {
			set, err := b.GetSet(code)
			if err != nil {
				t.Fatal(err)
			}
			for _, card := range set.Cards {
				if strings.HasPrefix(card.Name, typed) {
					t.Errorf("%q begins %q in %s, so the table may not read a head",
						typed, card.Name, code)
				}
			}
		}
	}
}
