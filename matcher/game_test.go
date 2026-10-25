package matcher_test

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/games"
)

// TestGameNames pins the string every game is registered, flagged and
// published under. ScraperInfo carries it into every dump, so renaming a
// game renames it in every dump already written.
func TestGameNames(t *testing.T) {
	for _, tt := range []struct {
		game matcher.Game
		want string
	}{
		{matcher.GameMagic, "magic"},
		{matcher.GameLorcana, "lorcana"},
		{matcher.GameRiftbound, "riftbound"},
		{matcher.GameOnePiece, "onepiece"},
		{matcher.GameYuGiOh, "yugioh"},
		{matcher.GameFleshAndBlood, "fleshandblood"},
		{matcher.GamePokemon, "pokemon"},
		{matcher.GameGundam, "gundam"},
		{matcher.GamePalworld, "palworld"},
	} {
		if string(tt.game) != tt.want {
			t.Errorf("game is %q, want %q", string(tt.game), tt.want)
		}
		if !slices.Contains(matcher.AllGames, tt.game) {
			t.Errorf("%q is not in AllGames", tt.game)
		}
	}
	if len(matcher.AllGames) != 9 {
		t.Errorf("AllGames holds %d games, want 9; add the new one to this test too", len(matcher.AllGames))
	}
}

// TestEveryGameIsRegistered pins AllGames to the loaders the games register:
// a constant with no loader behind it, or a loader under a name AllGames
// lacks, would open no datastore or build no scraper.
func TestEveryGameIsRegistered(t *testing.T) {
	registered := matcher.RegisteredGames()
	slices.Sort(registered)
	want := slices.Clone(matcher.AllGames)
	slices.Sort(want)
	if !slices.Equal(registered, want) {
		t.Errorf("registered %v, want %v", registered, want)
	}
}
