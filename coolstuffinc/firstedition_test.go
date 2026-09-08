package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestConditionRunReachesTheRun pins the print run this storefront names
// where a condition goes. The catalog files a run as a finish - 945 rows
// carry a first-edition one - so the wording reaches the printing rather
// than being refused.
//
// The spelling has to be exact, which is what matchRun is for: asking for
// the plain run of a card printed in holo answers with the holo of the
// other run, so an answer not carrying the run is refused instead of being
// published as the ordinary printing at a fraction of the price.
func TestConditionRunReachesTheRun(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("Need POKEMON_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name, edition string
		wantID        string
	}{
		{"Alakazam - 1/102", "Base Set", "001-102_42346_1steditionholofoil"},
		{"Lapras - 10/62", "Fossil", "10-62_44419_1steditionholofoil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			finishes := conditionRun("1st Edition  1st Edition ")
			if finishes == nil {
				t.Fatal("the wording named no run")
			}
			card := pokemonListing(tt.name, tt.edition, "", false)
			if card == nil {
				t.Fatal("the listing preprocessed to nothing")
			}
			id, err := matchRun(card, finishes)
			if err != nil {
				t.Fatalf("matchRun(%q) = %v", tt.name, err)
			}
			if id != tt.wantID {
				t.Errorf("matchRun(%q) = %q, want %q", tt.name, id, tt.wantID)
			}
		})
	}
}

// TestMatchRunRefusesTheOtherRun pins the refusal: a run the card was not
// printed in must not answer with the run it was.
func TestMatchRunRefusesTheOtherRun(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("Need POKEMON_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	// This card has no first-edition printing, so the wording names nothing
	// the catalog can answer with.
	card := pokemonListing("Pikachu V - 43/172", "SWSH08: Brilliant Stars", "", false)
	if card == nil {
		t.Skip("the listing preprocessed to nothing")
	}
	id, err := matchRun(card, conditionRun("1st Edition"))
	if err == nil {
		co, _ := mtgmatcher.GetUUID(id)
		t.Errorf("matchRun = %q (finish %q), want a refusal", id, co.Finish)
	}
}
