package cardgamecorner

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestGameFinish(t *testing.T) {
	for _, tt := range []struct {
		game    mtgmatcher.Game
		rarity  string
		variant Variant
		rainbow bool
		want    string
	}{
		{game: mtgmatcher.GameYuGiOh, variant: Variant{Print: "First Edition"}, want: "1st Edition"},
		{game: mtgmatcher.GameYuGiOh, variant: Variant{Print: "Unl."}, want: ""},
		{game: mtgmatcher.GamePokemon, variant: Variant{Print: "Unl.", Foil: "Reverse Holo"}, want: "Reverse Holofoil"},
		{game: mtgmatcher.GamePokemon, rarity: "Rare Holo", variant: Variant{Print: "First Edition"}, want: "1st Edition Holofoil"},
		{game: mtgmatcher.GamePokemon, rarity: "Common", variant: Variant{Print: "Unl."}, want: ""},
		{game: mtgmatcher.GameFleshAndBlood, variant: Variant{Print: "Unl."}, want: "Unlimited Edition Normal"},
		{game: mtgmatcher.GameFleshAndBlood, variant: Variant{Print: "Unl."}, rainbow: true, want: "Unlimited Edition Rainbow Foil"},
		{game: mtgmatcher.GameFleshAndBlood, rainbow: true, want: "Rainbow Foil"},
		{game: mtgmatcher.GameLorcana, variant: Variant{Print: "Steel", Foil: "Foil"}, want: ""},
	} {
		got := gameFinish(tt.game, &Card{Rarity: tt.rarity}, tt.variant, tt.rainbow)
		if got != tt.want {
			t.Errorf("%s %+v rainbow %v: got %q, want %q", tt.game, tt.variant, tt.rainbow, got, tt.want)
		}
	}
}
