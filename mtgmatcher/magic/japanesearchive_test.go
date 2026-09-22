package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestJapaneseArchive pins the Mystical Archive sets filing one card under an
// English printing and a Japanese one, and the listing that says which being
// read for it. Secrets of Strixhaven files each card three times - English,
// Japanese, and the Japanese silver scroll foil - and a listing naming the
// Japanese art was answered with the English printing, whose price is a
// different card's, until the set was read the way its 2021 sibling is.
//
// The silver scroll printings were already reachable, by the treatment the
// listing spells beside the art, so they say here that naming the art does
// not cost the finish that names them.
func TestJapaneseArchive(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
	}{
		{
			desc:    "the art the listing names is the Japanese printing",
			in:      mtgmatcher.InputCard{Name: "Ad Nauseam", Edition: "Secrets of Strixhaven Mystical Archive", Variation: "JP Alternate Art"},
			wantSet: "SOA", wantNum: "90",
		},
		{
			desc:    "and silence about it keeps the English one",
			in:      mtgmatcher.InputCard{Name: "Ad Nauseam", Edition: "Secrets of Strixhaven Mystical Archive"},
			wantSet: "SOA", wantNum: "25",
		},
		{
			desc:    "the English printing is still the foil one it is sold in",
			in:      mtgmatcher.InputCard{Name: "Force of Will", Edition: "Secrets of Strixhaven Mystical Archive", Foil: true},
			wantSet: "SOA", wantNum: "19",
		},
		{
			desc:    "the silver scroll foil is the Japanese printing wearing it",
			in:      mtgmatcher.InputCard{Name: "Armageddon", Edition: "Secrets of Strixhaven Mystical Archive", Variation: "JP Alternate Art Silver Scroll Foil", Foil: true},
			wantSet: "SOA", wantNum: "133",
		},
		{
			desc:    "the 2021 set reads the same wording the same way",
			in:      mtgmatcher.InputCard{Name: "Brainstorm", Edition: "Strixhaven Mystical Archive", Variation: "JP ALT ART"},
			wantSet: "STA", wantNum: "76",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}
}
