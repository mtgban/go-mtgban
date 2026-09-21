package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestFirstEditionShelfReachesTheRun pins the run a shelf names in its title.
// The storefront sells the first-edition run as a shelf of its own, and the
// catalog files the run as a finish of the set, so the listings matched the
// unlimited printing instead - silently, since the match succeeded, it just
// answered with the other run.
func TestFirstEditionShelfReachesTheRun(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	for _, tt := range []struct {
		name, edition, wantID string
	}{
		{"Lapras - 10/62", "1st Edition Fossil", "10-62_44419_1steditionholofoil"},
		{"Vileplume - 15/64", "1st Edition Jungle", "15-64_45126_1steditionholofoil"},
		// Base Set's run is a set of its own, and both its finishes have to
		// be reached there: the holo run had one card filed under the
		// shelf's own set, and the plain run had none at all.
		{"Alakazam - 1/102", "1st Edition Base Set", "001-102_106996_1steditionholofoil"},
		{"Abra - 43/102", "1st Edition Base Set", "043-102_107040_1stedition"},
	} {
		t.Run(tt.edition+" "+tt.name, func(t *testing.T) {
			shelf, run := firstEditionShelf(tt.edition)
			if run == nil {
				t.Fatalf("firstEditionShelf(%q) named no run", tt.edition)
			}
			card := pokemonListing(b, tt.name, shelf, "", false)
			if card == nil {
				t.Fatal("the listing preprocessed to nothing")
			}
			id, err := matchRun(b, card, run)
			if err != nil {
				t.Fatalf("matchRun = %v", err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("matchRun = %q (finish %q), want %q", id, co.Finish, tt.wantID)
			}
		})
	}
}

// TestFirstEditionShelfRefusesTheOtherRun pins the refusal that keeps the fix
// safe: a card the set has no first-edition row for is refused rather than
// answered with the unlimited printing, which is what used to be published.
func TestFirstEditionShelfRefusesTheOtherRun(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	// Machamp is the one Base Set card with no shadowless printing - its
	// first-edition stamp sits on a shadowed card - so the run names nothing
	// the catalog can answer with.
	shelf, run := firstEditionShelf("1st Edition Base Set")
	card := pokemonListing(b, "Machamp - 8/102", shelf, "", false)
	if card == nil {
		t.Skip("the listing preprocessed to nothing")
	}
	id, err := matchRun(b, card, run)
	if err == nil {
		co, _ := b.GetUUID(id)
		t.Errorf("matchRun = %q (finish %q), want a refusal", id, co.Finish)
	}
}

// A shelf naming no run is left exactly as it was.
func TestFirstEditionShelfLeavesOtherShelves(t *testing.T) {
	shelf, run := firstEditionShelf("Fossil")
	if shelf != "Fossil" || run != nil {
		t.Errorf("firstEditionShelf(Fossil) = %q, %v; want it untouched", shelf, run)
	}
}

// TestFirstEditionShelfNamesTheRunSet pins which set a run shelf sells. Every
// shelf but one names the set beside it; Base Set's runs are filed as "Base
// Set (Shadowless)", and asking the shelf's own set for them answered with
// the shadowed unlimited printing at the first edition's price.
func TestFirstEditionShelfNamesTheRunSet(t *testing.T) {
	for _, tt := range []struct{ edition, want string }{
		{"1st Edition Base Set", "Base Set (Shadowless)"},
		{"1st Edition Fossil", "Fossil"},
		{"1st Edition Team Rocket", "Team Rocket"},
		{"Base Set", "Base Set"},
	} {
		shelf, _ := firstEditionShelf(tt.edition)
		if shelf != tt.want {
			t.Errorf("firstEditionShelf(%q) = %q, want %q", tt.edition, shelf, tt.want)
		}
	}
}
