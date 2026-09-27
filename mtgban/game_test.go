package mtgban

import (
	"encoding/json"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestScraperInfoGameJSON pins that every game, Magic included, now writes
// an explicit game field rather than Magic relying on an omitted zero value.
func TestScraperInfoGameJSON(t *testing.T) {
	for _, tt := range []struct {
		game mtgmatcher.Game
		want string
	}{
		{mtgmatcher.GameMagic, `{"name":"","shorthand":"","game":"magic"}`},
		{mtgmatcher.GamePokemon, `{"name":"","shorthand":"","game":"pokemon"}`},
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
