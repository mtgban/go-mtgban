package cardmarket

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// TestPokemonFinishCell pins the projection every real finish name collapses
// onto - confirmed live that Cardmarket exposes no flag for holo-ness on its
// own, so Normal, Holofoil, Unlimited and Unlimited Holofoil all share the
// (false, false) cell a Holo Rare and its Normal counterpart are sold under
// as one product.
func TestPokemonFinishCell(t *testing.T) {
	tests := []struct {
		finish          string
		wantFirstEd     bool
		wantReverseHolo bool
	}{
		{"nonfoil", false, false},
		{"holofoil", false, false},
		{"unlimited", false, false},
		{"unlimitedholofoil", false, false},
		{"reverseholofoil", false, true},
		{"1stedition", true, false},
		{"1steditionholofoil", true, false},
		// A human-written spelling normalizes the same as the stored form.
		{"Reverse Holofoil", false, true},
	}
	for _, tt := range tests {
		firstEd, reverseHolo := pokemonFinishCell(tt.finish)
		if firstEd != tt.wantFirstEd || reverseHolo != tt.wantReverseHolo {
			t.Errorf("pokemonFinishCell(%q) = (%v, %v), want (%v, %v)",
				tt.finish, firstEd, reverseHolo, tt.wantFirstEd, tt.wantReverseHolo)
		}
	}
}

var (
	pokemonBackendOnce sync.Once
	pokemonBackendErr  error
	pokemonBackend     *mtgmatcher.Backend
)

// loadPokemonBackend installs the real Pokemon datastore POKEMON_PATH
// names, once per test binary run, for the tests below that need real
// finish siblings rather than synthetic ones.
func loadPokemonBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	pokemonBackendOnce.Do(func() {
		path := os.Getenv("POKEMON_PATH")
		if path == "" {
			return
		}
		pokemonBackend, pokemonBackendErr = datastore.Read("pokemon", path)
	})
	if pokemonBackendErr != nil {
		t.Fatal(pokemonBackendErr)
	}
	if os.Getenv("POKEMON_PATH") == "" {
		t.Skip("Need POKEMON_PATH set to run this test")
	}
	return pokemonBackend
}

// TestPokemonFinishPlanTeamRocket pins the case this whole fix exists for:
// Team Rocket's Dark Charizard predates reverse holo entirely, so its two
// real printings - Unlimited Holofoil and 1st Edition Holofoil - both
// project to a cell with no reverse-holo component, and both must appear
// as their own query target rather than one hiding behind the other.
func TestPokemonFinishPlanTeamRocket(t *testing.T) {
	b := loadPokemonBackend(t)

	cardID, err := b.Match(&mtgmatcher.InputCard{
		Name: "Dark Charizard", Edition: "Team Rocket", Variation: "4",
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}

	targets := pokemonFinishPlan(b, cardID)
	if len(targets) != 2 {
		t.Fatalf("pokemonFinishPlan(%q) = %d targets, want 2: %+v", cardID, len(targets), targets)
	}

	got := map[string]pokemonFinishTarget{}
	for _, target := range targets {
		got[target.cardID] = target
	}

	unlimited, ok := got[cardID]
	if !ok {
		t.Fatalf("plan does not target the resolved cardID %q at all: %+v", cardID, targets)
	}
	if unlimited.isFirstEd || unlimited.isReverseHolo {
		t.Errorf("Unlimited Holofoil target = %+v, want both flags false", unlimited)
	}

	firstEdID, err := b.MatchIDFinish(cardID, "1st Edition Holofoil")
	if err != nil {
		t.Fatalf("MatchIDFinish(1st Edition Holofoil): %v", err)
	}
	firstEd, ok := got[firstEdID]
	if !ok {
		t.Fatalf("plan does not target the 1st Edition Holofoil sibling %q: %+v", firstEdID, targets)
	}
	if !firstEd.isFirstEd || firstEd.isReverseHolo {
		t.Errorf("1st Edition Holofoil target = %+v, want isFirstEd true, isReverseHolo false", firstEd)
	}
}
