package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestFirstEditionShelfReachesTheRun pins the run a shelf names in its title.
// The storefront sells the first-edition run as a shelf of its own, and the
// catalog files the run as a finish of the set, so the listings matched the
// unlimited printing instead - silently, since the match succeeded, it just
// answered with the other run.
func TestFirstEditionShelfReachesTheRun(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("Need POKEMON_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name, edition, wantID string
	}{
		{"Lapras - 10/62", "1st Edition Fossil", "10-62_44419_1eholo"},
		{"Vileplume - 15/64", "1st Edition Jungle", "15-64_45126_1eholo"},
		{"Alakazam - 1/102", "1st Edition Base Set", "001-102_42346_1eholo"},
	} {
		t.Run(tt.edition+" "+tt.name, func(t *testing.T) {
			shelf, run := firstEditionShelf(tt.edition)
			if run == nil {
				t.Fatalf("firstEditionShelf(%q) named no run", tt.edition)
			}
			card := pokemonListing(tt.name, shelf, "", false)
			if card == nil {
				t.Fatal("the listing preprocessed to nothing")
			}
			id, err := matchRun(card, run)
			if err != nil {
				t.Fatalf("matchRun = %v", err)
			}
			if id != tt.wantID {
				co, _ := mtgmatcher.GetUUID(id)
				t.Errorf("matchRun = %q (finish %q), want %q", id, co.Finish, tt.wantID)
			}
		})
	}
}

// TestFirstEditionShelfRefusesTheOtherRun pins the refusal that keeps the fix
// safe: a card the set has no first-edition row for is refused rather than
// answered with the unlimited printing, which is what used to be published.
func TestFirstEditionShelfRefusesTheOtherRun(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("Need POKEMON_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	// Base Set carries one first-edition row, Alakazam; the rest of the set
	// has none, so this names a run the catalog cannot answer with.
	shelf, run := firstEditionShelf("1st Edition Base Set")
	card := pokemonListing("Venusaur - 15/102", shelf, "", false)
	if card == nil {
		t.Skip("the listing preprocessed to nothing")
	}
	id, err := matchRun(card, run)
	if err == nil {
		co, _ := mtgmatcher.GetUUID(id)
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
