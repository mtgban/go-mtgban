package miniaturemarket

import "testing"

func TestSealedNameOnePiece(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		// The deck number moves to the front, where the canonical names
		// lead with it, and the storefront decorations fall away.
		{
			"One Piece TCG: BLUE Kuzan [ST-33] - Starter Deck (Preorder)",
			"Starter Deck 33: BLUE Kuzan",
		},
		{
			"One Piece TCG: RED Monkey.D.Luffy [ST-31] - Starter Deck (Preorder)",
			"Starter Deck 31: RED Monkey.D.Luffy",
		},
		// A non-deck code restates a set the name already spells, so it
		// only falls away.
		{
			"One Piece TCG: Extra Booster Heroines Edition [EB-03] - Booster Box (24 Packs) (Preorder)",
			"Extra Booster Heroines Edition - Booster Box",
		},
		{
			"One Piece TCG: The Time of Battle [OP-16] - Booster Pack",
			"The Time of Battle - Booster Pack",
		},
		// A name with nothing to rewrite passes through whole.
		{
			"One Piece TCG: Heroines Gift Collection",
			"Heroines Gift Collection",
		},
	} {
		got := sealedName(GameOnePiece, tt.in)
		if got != tt.want {
			t.Errorf("%q:\n got  %q\n want %q", tt.in, got, tt.want)
		}
	}
}

// The other games' names already read like their canon: nothing moves.
func TestSealedNameOtherGamesUntouched(t *testing.T) {
	name := "Riftbound: League of Legends TCG - Origins Booster Box (Preorder)"
	if got := sealedName(GameRiftbound, name); got != name {
		t.Errorf("riftbound name rewritten: %q", got)
	}
}
