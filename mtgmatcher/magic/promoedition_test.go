package magic

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"testing"
)

func TestPromoEditionDescriptions(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		name, edition, variation, set, number string
		foil                                  bool
	}{
		{"Fauna Shaman", "Secret Lair Promo", "Secret Lair", "SLP", "41", false},
		{"Swords to Plowshares", "Promo", "MagicCon 2025", "PF25", "12", true},
		{"Ugin, the Spirit Dragon", "Promo", "MagicCon Retro Frame", "PF25", "6", true},
		{"Arcane Signet", "Promo", "Commandfest 2025", "PF25", "10", false},
		{"Dark Ritual", "Teenage Mutant Ninja Turtles Eternal", "Borderless Pizza Bundle Promo 131", "TMC", "131", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.variation, Foil: tt.foil}
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatal(err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.set || co.Number != tt.number || co.Foil != tt.foil {
				t.Fatalf("got %s, want %s #%s foil=%t", co, tt.set, tt.number, tt.foil)
			}
		})
	}
}
