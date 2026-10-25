package cardgamecorner

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestGameInput(t *testing.T) {
	for _, tt := range []struct {
		game    mtgmatcher.Game
		card    Card
		variant Variant
		want    mtgmatcher.InputCard
	}{
		{
			game:    mtgmatcher.GameFleshAndBlood,
			card:    Card{Name: "Zero to Sixty (Blue) (Normal)", Edition: "Arcane Rising - Unlimited", Number: "ARC028"},
			variant: Variant{Print: "Unl.", Foil: "Non Foil"},
			want:    mtgmatcher.InputCard{Name: "Zero to Sixty (Blue)", Edition: "Arcane Rising - Unlimited", Variation: "ARC028", Finish: "Unlimited Edition Normal"},
		},
		{
			game:    mtgmatcher.GameFleshAndBlood,
			card:    Card{Name: "Drone of Brutality (Red) (Rainbow Foil)", Edition: "Welcome to Rathe - Unlimited", Number: "WTR164"},
			variant: Variant{Print: "Unl.", Foil: "Non Foil"},
			want:    mtgmatcher.InputCard{Name: "Drone of Brutality (Red)", Edition: "Welcome to Rathe - Unlimited", Variation: "WTR164", Foil: true, Finish: "Unlimited Edition Rainbow Foil"},
		},
		{
			game:    mtgmatcher.GameYuGiOh,
			card:    Card{Name: "Star Ryzeal", Edition: "Alliance Insight", Number: "022", Rarity: "Secret Rare"},
			variant: Variant{Print: "First Edition", Foil: "Non Foil"},
			want:    mtgmatcher.InputCard{Name: "Star Ryzeal", Edition: "Alliance Insight", Variation: "022 Secret Rare", Finish: "1st Edition"},
		},
		{
			game:    mtgmatcher.GameRiftbound,
			card:    Card{Name: "Miss Fortune, Buccaneer (V.2 - Showcase)", Edition: "Origins", Number: "193a"},
			variant: Variant{Foil: "Foil"},
			want:    mtgmatcher.InputCard{Name: "Miss Fortune, Buccaneer", Edition: "Origins", Variation: "193a", Foil: true},
		},
		{
			game:    mtgmatcher.GamePokemon,
			card:    Card{Name: "Gyarados d Delta Species ", Edition: "EX Holon Phantoms", Number: "8", Rarity: "Rare Holo"},
			variant: Variant{Print: "Unl.", Foil: "Non Foil"},
			want:    mtgmatcher.InputCard{Name: "Gyarados", Edition: "EX Holon Phantoms", Variation: "8", Finish: "Holofoil"},
		},
		{
			game:    mtgmatcher.GamePokemon,
			card:    Card{Name: "Wooloo", Edition: "Sword & Shield: Black Star Promos", Number: "011", Rarity: "Promo"},
			variant: Variant{Print: "Unl.", Foil: "Reverse Holo"},
			want:    mtgmatcher.InputCard{Name: "Wooloo", Edition: "Sword & Shield: Black Star Promos", Variation: "SWSH011", Finish: "Reverse Holofoil"},
		},
	} {
		t.Run(tt.card.Name, func(t *testing.T) {
			tt.card.Variants = []Variant{tt.variant}
			got := gameInput(tt.game, &tt.card, 0)
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}

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

func TestOldShelfSet(t *testing.T) {
	for _, tt := range []struct {
		edition  string
		language string
		want     string
	}{
		{edition: "Legend of Blue Eyes White Dragon ENG (LOB)", language: "US", want: "LOB"},
		{edition: "Legend of Blue Eyes White Dragon ENG (LOB)", language: "EN"},
		{edition: "Spell Ruler / Magic Ruler ENG (MRL)", language: "US", want: "MRL"},
		{edition: "Starter Deck: Yugi ENG (SDY)", language: "US", want: "SDY"},
		{edition: "Legend of Blue Eyes White Dragon ITA (LDD)", language: "US"},
		{edition: "Alliance Insight", language: "US"},
	} {
		got, found := oldShelfSet(tt.edition, tt.language)
		if got != tt.want || found != (tt.want != "") {
			t.Errorf("oldShelfSet(%q, %q) = %q, %t; want %q", tt.edition, tt.language, got, found, tt.want)
		}
	}
}
