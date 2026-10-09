package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPooledEdition pins the storefront name that spans two of the catalog's
// sets. Cool Stuff Inc files both Speed Duel starter decks under one
// "Starter Deck: Speed Dueling", so the edition reaches no set and, left alone,
// every printing the card ever had would answer. The two decks share no card,
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

// TestPooledEditionYuyaDeclan pins that the 2-Player Starter Deck Yuya &
// Declan is refused outright rather than folded onto Saber Force and
// Dark Legion. The shelf is a real Europe/Oceania product (Cardmarket's
// YS15, TCGplayer never listed it) with its own two 21-card halves, and the
// datastore this loader builds from - TCGplayer's own catalog - has no row
// for either one; a fold would land every name on whichever of the two
// unrelated decks also prints it, at that deck's own number and
// rarity. See unsupportedEditions.
func TestPooledEditionYuyaDeclan(t *testing.T) {
	b := loadBackend(t)
	for _, name := range []string{
		"Fabled Ashenveil",       // Declan's deck (Dark Legion) prints this
		"The Calculator",         // Yuya's deck (Saber Force) prints this
		"Mystical Space Typhoon", // both decks print this
		"Ancient Dragon",         // neither English deck prints this; the fold
		"Bright Star Dragon",     // would land it on Galactic Overlord instead
	} {
		in := mtgmatcher.InputCard{Name: name, Edition: "2-Player Starter Deck Yuya & Declan", Variation: "Common"}
		if id, err := b.Match(&in); err != mtgmatcher.ErrUnsupported {
			t.Errorf("Match(%q) = (%q, %v), want %v", name, id, err, mtgmatcher.ErrUnsupported)
		}
	}
}

// TestOTSPastRun pins that an OTS Tournament Pack listing numbered past the
// set's English run is skipped as a printing the datastore has no row for,
// where the same number would otherwise coincide with a card of another set
// ("Gladiator Beast Darius" 031 is Premium Gold's PTDN-EN031), and that a
// listing inside the run still matches.
func TestOTSPastRun(t *testing.T) {
	b := loadBackend(t)

	past := mtgmatcher.InputCard{Name: "Gladiator Beast Darius", Edition: "OTS Tournament Pack 12", Variation: "031 Common"}
	if id, err := b.Match(&past); err != mtgmatcher.ErrUnsupported {
		t.Errorf("Match(%v) = (%q, %v), want %v", past, id, err, mtgmatcher.ErrUnsupported)
	}

	inRun := mtgmatcher.InputCard{Name: "Tenyi Spirit - Vishuda", Edition: "OTS Tournament Pack 12", Variation: "010 Super Rare"}
	id, err := b.Match(&inRun)
	if err != nil {
		t.Fatalf("Match(%v) = %v", inRun, err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", id, err)
	}
	if co.Number != "OP12-EN010" {
		t.Errorf("Match(%v) = %s, want OP12-EN010", inRun, co.Number)
	}
}
