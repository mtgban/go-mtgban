package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPooledEdition pins the storefront name that spans two of the catalog's
// sets. Cool Stuff Inc files both Speed Duel starter decks under one
// "Starter Deck: Speed Dueling", so the edition reaches no set and every
// printing the card ever had used to answer. The two decks share no card,
// so narrowing to the pair is all the name needs to pick one.
func TestPooledEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
	}{
		{
			desc:    "a card the first deck prints",
			in:      mtgmatcher.InputCard{Name: "A Cat of Ill Omen", Edition: "Starter Deck: Speed Dueling", Variation: "Common"},
			wantSet: "SS01", wantNum: "SS01-ENB11",
		},
		{
			desc:    "a card the second one prints",
			in:      mtgmatcher.InputCard{Name: "Alligator's Sword", Edition: "Starter Deck: Speed Dueling", Variation: "Common"},
			wantSet: "SS02", wantNum: "SS02-ENB05",
		},
		{
			desc:    "and each deck named outright still reaches itself",
			in:      mtgmatcher.InputCard{Name: "A Cat of Ill Omen", Edition: "Speed Duel Decks: Destiny Masters", Variation: "Common"},
			wantSet: "SS01", wantNum: "SS01-ENB11",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}
}
