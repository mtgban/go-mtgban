package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTokenFaceKeepsCardName pins the alternate lookups a token's face must
// not steal. The MID and VOW day-night helper is spelled "Day // Night", so
// its faces repeat the names of Apocalypse's "Night // Day" card; whoever
// registers "Night" last owns the lookup, and it has to be the card.
func TestTokenFaceKeepsCardName(t *testing.T) {
	for _, tt := range []struct {
		face, edition, want string
	}{
		{"Night", "Apocalypse", "0a18d581-5298-5a3a-9608-236e52a15ad6"},
		{"Night", "Dominaria Remastered", "b048e268-99f2-50d9-bccb-68ecf7ec3e38"},
	} {
		t.Run(tt.face, func(t *testing.T) {
			in := mtgmatcher.InputCard{Name: tt.face, Edition: tt.edition}
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", in, err)
			}
			if id != tt.want {
				co, _ := testBackend.GetUUID(id)
				t.Errorf("Match(%v) = %s (%v), want the card at %s", in, id, co, tt.want)
			}
		})
	}
}
