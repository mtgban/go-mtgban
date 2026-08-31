package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestBoxEdition pins the box a storefront sells a set in, where the catalog
// files the cards under the set's own name. The name reaches no set as
// written, so the edition narrowed nothing and every printing the card ever
// had answered: each listing below was dropped as aliased rather than
// mispriced, which is why nothing but the name had to change.
func TestBoxEdition(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
	}{
		{
			desc:    "the box Duel Devastator is sold in",
			in:      mtgmatcher.InputCard{Name: "Abyss Dweller", Edition: "Duel Devastator Box", Variation: "Ultra Rare"},
			wantSet: "DUDE", wantNum: "DUDE-EN016",
		},
		{
			desc:    "and the set itself, which already reached it",
			in:      mtgmatcher.InputCard{Name: "Abyss Dweller", Edition: "Duel Devastator", Variation: "Ultra Rare"},
			wantSet: "DUDE", wantNum: "DUDE-EN016",
		},
		{
			desc:    "the box set Duel Overload is sold in",
			in:      mtgmatcher.InputCard{Name: "Ancient Gear Ballista", Edition: "Duel Overload Box Set", Variation: "Ultra Rare"},
			wantSet: "DUOV", wantNum: "DUOV-EN010",
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
