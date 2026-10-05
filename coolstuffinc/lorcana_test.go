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

// TestLorcanaShelfFollowsTheQuestNote pins the prize cards filed on the shelf
// of the set whose card they repeat: the frame the note names is the quest.
func TestLorcanaShelfFollowsTheQuestNote(t *testing.T) {
	b := readGameDatastore(t, "lorcana", "LORCANA_PATH")

	for _, tt := range []struct {
		name, shelf, notes, wantSet string
	}{
		{"Yen Sid - Powerful Sorcerer", "Ursula's Return", "223/204, Ink Tentacles Card Frame", "Q1"},
		{"Pinocchio - Strings Attached", "Reign of Jafar", "224/204, Foil Shifting Sands Version", "Q2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			shelf := lorcanaShelf(b, tt.name, tt.shelf, tt.notes)
			card := &mtgmatcher.InputCard{Name: tt.name, Edition: shelf, Variation: lorcanaVariation(tt.notes), Foil: true}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			co, _ := b.GetUUID(id)
			if co.SetCode != tt.wantSet {
				t.Errorf("Match = %q (%s), want set %s", id, co.SetCode, tt.wantSet)
			}
		})
	}
	got := lorcanaShelf(b, "Yen Sid - Powerful Sorcerer", "Ursula's Return", "223/204")
	if got != "Ursula's Return" {
		t.Errorf("a note naming no quest moved the listing to %q", got)
	}
	got = lorcanaShelf(b, "Ariel - On Human Legs", "The First Chapter", "1/204, Ink Tentacles Card Frame")
	if got != "The First Chapter" {
		t.Errorf("a card absent from the quest moved the listing to %q", got)
	}
}
