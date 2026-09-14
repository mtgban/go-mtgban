package mtgban

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestGameWireFormat pins the string every game is published as, Magic
// included now that it is a named value rather than the zero value.
// ScraperInfo is marshalled into the dumps the site reads, so a game's
// value is a wire format and not an internal label: renaming one here
// renames it in every dump already written.
func TestGameWireFormat(t *testing.T) {
	for _, tt := range []struct {
		game Game
		want string
	}{
		{GameMagic, "Magic"},
		{GameLorcana, "Lorcana"},
		{GameRiftbound, "Riftbound"},
		{GameOnePiece, "OnePiece"},
		{GameYuGiOh, "YuGiOh"},
		{GameFleshAndBlood, "FleshAndBlood"},
		{GamePokemon, "Pokemon"},
		{GameGundam, "Gundam"},
		{GamePalworld, "Palworld"},
	} {
		if string(tt.game) != tt.want {
			t.Errorf("game is %q, want %q", string(tt.game), tt.want)
		}
		if !slices.Contains(AllGames, tt.game) {
			t.Errorf("%q is not in AllGames", tt.game)
		}
	}
	if len(AllGames) != 9 {
		t.Errorf("AllGames holds %d games, want 9; add the new one to this test too", len(AllGames))
	}
}

// TestScraperInfoGameJSON pins that every game, Magic included, now writes
// an explicit game field rather than Magic relying on an omitted zero value.
func TestScraperInfoGameJSON(t *testing.T) {
	for _, tt := range []struct {
		game Game
		want string
	}{
		{GameMagic, `{"name":"","shorthand":"","game":"Magic"}`},
		{GamePokemon, `{"name":"","shorthand":"","game":"Pokemon"}`},
	} {
		blob, err := json.Marshal(ScraperInfo{Game: tt.game})
		if err != nil {
			t.Fatalf("marshalling %q: %v", tt.game, err)
		}
		if string(blob) != tt.want {
			t.Errorf("marshalled %q as %s, want %s", tt.game, blob, tt.want)
		}
		var back ScraperInfo
		if err := json.Unmarshal([]byte(tt.want), &back); err != nil {
			t.Fatalf("unmarshalling %s: %v", tt.want, err)
		}
		if back.Game != tt.game {
			t.Errorf("read %s back as %q, want %q", tt.want, back.Game, tt.game)
		}
	}
}
