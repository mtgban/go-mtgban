package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestLorcanaSpelling pins the Lorcana names this storefront misspells. Each
// one names a card the catalog has and reaches nothing as typed, so the
// listing goes unpriced; each is also checked as typed, because a pair that
// stopped being a misspelling - the catalog renames a card, the storefront
// fixes its own spelling - is a pair that should leave the table rather than
// sit there rewriting a name that now means something.
//
// The set and number are asserted beside the name: a character's titles are
// the whole of what tells one of its printings from another here, and Rise
// of the Floodborn sells three Basils at 138, 139 and 140.
func TestLorcanaSpelling(t *testing.T) {
	b := readGameDatastore(t, "lorcana", "LORCANA_PATH")

	tests := []struct {
		name    string
		edition string
		number  string
		setCode string
	}{
		{"Basil - Perspective Investigator", "Rise of the Floodborn", "140/204", "2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spelled := lorcanaSpelling(test.name)
			if spelled == test.name {
				t.Fatalf("lorcanaSpelling(%q) corrected nothing", test.name)
			}
			asTyped := &mtgmatcher.InputCard{Name: test.name, Edition: test.edition, Variation: test.number}
			if id, err := b.Match(asTyped); err == nil {
				t.Errorf("Match(%q) = %q, want the name to reach nothing before it is corrected",
					test.name, id)
			}
			card := &mtgmatcher.InputCard{Name: spelled, Edition: test.edition, Variation: test.number}
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
			if co.SetCode != test.setCode {
				t.Errorf("Match(%q) set = %q, want %q", spelled, co.SetCode, test.setCode)
			}
			if want := mtgmatcher.ExtractNumber(test.number); co.Number != want {
				t.Errorf("Match(%q) number = %q, want %q", spelled, co.Number, want)
			}
		})
	}
}

// TestLorcanaVariationRainbowFoil pins "Rainbow Foil" to the catalog's own
// name for the finish, "Rainbow Pillars" - a Starter Deck Exclusive's
// rainbow foil otherwise names no finish the matcher knows and lands on the
// set's ordinary cold foil instead, at a fraction of its price.
func TestLorcanaVariationRainbowFoil(t *testing.T) {
	b := readGameDatastore(t, "lorcana", "LORCANA_PATH")

	card := &mtgmatcher.InputCard{
		Name:      "Ariel - Singing Mermaid",
		Edition:   "Fabled",
		Variation: lorcanaVariation("15/204, Rainbow Foil Starter Deck Exclusive"),
		Foil:      true,
	}
	id, err := b.Match(card)
	if err != nil {
		t.Fatalf("Match(%q) = %v", card, err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if co.Finish != "holofoil" {
		t.Errorf("Match(%q) finish = %q, want holofoil (Rainbow Pillars)", card, co.Finish)
	}

	// The raw wording, unrewritten, lands on the plain cold foil instead.
	plain := &mtgmatcher.InputCard{Name: card.Name, Edition: card.Edition, Variation: "15/204, Rainbow Foil Starter Deck Exclusive", Foil: true}
	plainID, err := b.Match(plain)
	if err != nil {
		t.Fatalf("Match(%q) = %v", plain, err)
	}
	if plainID == id {
		t.Error("unrewritten wording already reached the rainbow pillars uuid; the fixture no longer demonstrates the bug")
	}
}
